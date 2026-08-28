package importing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"wms-backend/internal/domain"
	"wms-backend/internal/services"
)

// ErrTooManyAmbiguousColumns limita quantos cabeçalhos ambíguos são
// enviados de uma vez para a IA (item 29 da especificação: performance
// — nunca mandar a planilha inteira, e aqui, adicionalmente, nunca
// mandar uma quantidade desproporcional de colunas de uma vez só, que
// indicaria uma planilha fora do formato esperado, não uma ambiguidade
// legítima de nomenclatura).
const maxAmbiguousColumnsForAI = 15

// ImportService orquestra o fluxo completo da Importação Inteligente:
// parse → normalize → validate → IA (só para ambiguidade de cabeçalho)
// → preview, e depois preview revisado pelo usuário → Services
// existentes → banco (item 16/24 da especificação: "IA/Importador →
// Service existente → Repository existente → PostgreSQL", nunca SQL
// direto nem acesso ao banco a partir deste pacote).
type ImportService struct {
	inventory      *services.InventoryService
	preProducts    *services.PreProductService
	categories     *services.CategoryService
	headerResolver HeaderResolver // pode ser nil: ver Preview, funciona sem IA
}

func NewImportService(
	inventory *services.InventoryService,
	preProducts *services.PreProductService,
	categories *services.CategoryService,
	headerResolver HeaderResolver,
) *ImportService {
	return &ImportService{
		inventory:      inventory,
		preProducts:    preProducts,
		categories:     categories,
		headerResolver: headerResolver,
	}
}

// Preview interpreta uma planilha e devolve a prévia completa para o
// usuário revisar — nada é persistido nesta chamada (item 1/8 da
// especificação). filename é usado só para a detecção de formato
// forjado (ver parser.go); a detecção real do conteúdo é sempre feita
// pelos bytes do arquivo, nunca pela extensão isoladamente.
func (s *ImportService) Preview(ctx context.Context, filename string, data []byte, target TargetKind) (Preview, error) {
	start := time.Now()

	table, err := ParseSpreadsheet(filename, data)
	if err != nil {
		return Preview{}, err
	}

	mapped, ambiguous := MapColumnsDeterministic(table)

	aiUsed := false
	if len(ambiguous) > 0 && len(ambiguous) <= maxAmbiguousColumnsForAI && s.headerResolver != nil {
		// A IA só é chamada quando sobra pelo menos um cabeçalho que o
		// mapeamento determinístico não resolveu sozinho (item 3/19/29
		// da especificação) — a esmagadora maioria das planilhas reais
		// nunca chega a esta chamada.
		resolved, aiErr := s.headerResolver.ResolveHeaders(ctx, ambiguous, resolvableFieldsForAI)
		if aiErr != nil {
			// Falha na IA nunca derruba a importação inteira (item 20:
			// funciona independente do RAG/IA) — registrado em log,
			// os cabeçalhos seguem como não mapeados.
			log.Printf("importacao_inteligente: falha ao consultar IA para cabecalhos ambiguos: %v", aiErr)
		} else if len(resolved) > 0 {
			aiUsed = true
			mapped, ambiguous = applyAIResolution(mapped, ambiguous, resolved)
		}
	}

	items := BuildItems(table, mapped, target)
	items = resolveCategoryIDs(items, s.loadCategoryLookup())

	unmappedHeaders := make([]string, 0, len(ambiguous))
	for _, a := range ambiguous {
		unmappedHeaders = append(unmappedHeaders, a.Header)
	}

	summary := summarize(items)

	log.Printf("importacao_inteligente: preview concluido - linhas=%d itens=%d prontos=%d atencao=%d ia_usada=%v duracao=%s",
		len(table.Rows), summary.TotalItems, summary.ReadyCount, summary.AttentionCount, aiUsed, time.Since(start))

	return Preview{
		Items:           items,
		Summary:         summary,
		ColumnMapping:   mapped,
		UnmappedHeaders: unmappedHeaders,
		AIUsed:          aiUsed,
	}, nil
}

// applyAIResolution incorpora ao mapeamento os cabeçalhos que a IA
// conseguiu resolver, e devolve a lista atualizada de ambiguidades
// restantes (cabeçalhos que nem o normalizer nem a IA resolveram).
func applyAIResolution(mapped []ColumnMapping, ambiguous []AmbiguousHeader, resolved map[string]FieldKey) ([]ColumnMapping, []AmbiguousHeader) {
	alreadyMapped := map[FieldKey]bool{}
	for _, m := range mapped {
		alreadyMapped[m.Field] = true
	}

	var stillAmbiguous []AmbiguousHeader
	for _, a := range ambiguous {
		field, ok := resolved[a.Header]
		if !ok || alreadyMapped[field] {
			// Não resolvido pela IA, ou a IA sugeriu um campo que já
			// tinha sido mapeado deterministicamente por outra coluna —
			// neste segundo caso, mantemos a decisão determinística
			// (mais confiável) e a coluna segue como não mapeada, para
			// não sobrescrever silenciosamente uma correspondência já
			// resolvida com confiança maior.
			stillAmbiguous = append(stillAmbiguous, a)
			continue
		}
		mapped = append(mapped, ColumnMapping{ColumnIndex: a.ColumnIndex, Header: a.Header, Field: field, Source: "ai"})
		alreadyMapped[field] = true
	}
	return mapped, stillAmbiguous
}

func summarize(items []ImportItem) PreviewSummary {
	s := PreviewSummary{TotalItems: len(items)}
	for _, item := range items {
		s.TotalUnits += item.Quantity
		if item.Ready {
			s.ReadyCount++
		} else {
			s.AttentionCount++
		}
	}
	return s
}

// loadCategoryLookup carrega as categorias existentes uma única vez por
// Preview (nunca por item — item 29 da especificação: performance) para
// resolver CategoryRaw → CategoryID por correspondência exata de nome
// (sem acentuação, case-insensitive). Falha ao listar categorias não
// derruba a importação: os itens simplesmente ficam sem categoria
// resolvida automaticamente, e o usuário pode escolher manualmente na
// edição — é uma conveniência, não um requisito bloqueante.
func (s *ImportService) loadCategoryLookup() map[string]string {
	lookup := map[string]string{}
	categories, err := s.categories.List()
	if err != nil {
		log.Printf("importacao_inteligente: falha ao carregar categorias para resolucao automatica: %v", err)
		return lookup
	}
	for _, c := range categories {
		lookup[normalizeForComparison(c.Name)] = c.ID
	}
	return lookup
}

func resolveCategoryIDs(items []ImportItem, lookup map[string]string) []ImportItem {
	for i := range items {
		if items[i].CategoryRaw == "" {
			continue
		}
		if id, ok := lookup[normalizeForComparison(items[i].CategoryRaw)]; ok {
			items[i].CategoryID = &id
		}
		// Sem correspondência exata: CategoryID fica nil. Não é um erro
		// nem um campo obrigatório ausente (categoria nunca é
		// obrigatória nos Services existentes) — o item segue Ready se
		// os demais campos estiverem OK.
	}
	return items
}

// Commit grava os itens já revisados pelo usuário, um a um, usando
// exclusivamente os Services existentes (item 16 da especificação:
// nunca SQL direto). Uma falha em um item NUNCA interrompe os
// seguintes (item 26: "não deixar o sistema em estado inconsistente" +
// "o usuário precisa saber quais falharam") — cada item é uma operação
// independente, com seu próprio resultado de sucesso/erro reportado.
func (s *ImportService) Commit(ctx context.Context, user domain.User, req CommitRequest) CommitResult {
	start := time.Now()
	result := CommitResult{
		TotalAnalyzed: len(req.Items),
		Results:       make([]ItemResult, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		itemResult := s.commitOne(user, req.DepositID, req.Target, item)
		result.Results = append(result.Results, itemResult)
		if itemResult.Success {
			result.TotalImported++
		} else {
			result.TotalFailed++
		}
	}

	log.Printf("importacao_inteligente: commit concluido - usuario=%s deposito=%s destino=%s analisados=%d importados=%d falharam=%d duracao=%s",
		user.ID, req.DepositID, req.Target, result.TotalAnalyzed, result.TotalImported, result.TotalFailed, time.Since(start))

	return result
}

func (s *ImportService) commitOne(user domain.User, depositID string, target TargetKind, item CommitItemInput) ItemResult {
	base := ItemResult{RowIndex: item.RowIndex, Name: item.Name}

	name := strings.TrimSpace(item.Name)
	if name == "" {
		base.Error = "nome do produto é obrigatório"
		return base
	}

	if target == TargetPreProduct {
		created, err := s.preProducts.Create(user.ID, services.PreProductInput{
			Name:       name,
			CategoryID: item.CategoryID,
			Brand:      item.Brand,
			Unit:       item.Unit,
			Notes:      item.Notes,
		})
		if err != nil {
			base.Error = err.Error()
			return base
		}
		base.Success = true
		base.ItemID = created.ID
		return base
	}

	// Destino padrão: item de estoque real. Segue exatamente o mesmo
	// padrão em duas etapas já usado pelo frontend hoje (ver
	// api.js: createInventoryItem + moveStock) — InventoryService.Create
	// cadastra o item (quantidade inicial zero), e a quantidade da
	// planilha é lançada como uma movimentação de entrada em seguida,
	// preservando o mesmo rastro de auditoria (StockMovement) que uma
	// entrada manual geraria. Nenhum acesso a depósito fora do escopo
	// do usuário é possível aqui: tanto Create quanto MoveStock
	// revalidam DepositService.CanAccess internamente (item 14/25 da
	// especificação).
	created, err := s.inventory.Create(user, depositID, services.ItemInput{
		Name:        name,
		SKU:         item.SKU,
		Brand:       item.Brand,
		MinQuantity: item.MinQuantity,
		ExpiryDate:  item.ExpiryDate,
		LotNumber:   item.LotNumber,
		CategoryID:  item.CategoryID,
		Notes:       item.Notes,
	})
	if err != nil {
		base.Error = err.Error()
		return base
	}
	base.ItemID = created.ID

	if item.Quantity <= 0 {
		// Sem quantidade positiva não há o que lançar como entrada — o
		// item já foi cadastrado no catálogo do depósito, mas fica
		// registrado como falha PARCIAL nesta linha para o usuário
		// perceber que a quantidade não entrou, em vez de presumir
		// silenciosamente sucesso total (item 9: "não permitir
		// importação silenciosa de dados inválidos"). Na prática, isso
		// só deveria acontecer se um item marcado como Ready=false
		// (issue de quantidade) for enviado ao commit mesmo assim — o
		// que o frontend não deveria permitir, mas o backend não confia
		// nisso (item 25: nunca confiar nos dados enviados).
		base.Error = fmt.Sprintf("produto '%s' foi cadastrado, mas a quantidade (%d) é inválida e não foi lançada como entrada de estoque — ajuste manualmente", name, item.Quantity)
		return base
	}

	note := "Importação Inteligente"
	if item.LotNumber != "" {
		note = fmt.Sprintf("Importação Inteligente — lote %s", item.LotNumber)
	}
	_, _, err = s.inventory.MoveStock(user, created.ID, domain.MovementIn, item.Quantity, note)
	if err != nil {
		base.Error = fmt.Sprintf("produto '%s' foi cadastrado, mas houve falha ao lançar a quantidade em estoque: %v", name, err)
		return base
	}

	base.Success = true
	return base
}
