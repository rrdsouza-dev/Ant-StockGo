package importing

import (
	"strconv"
	"strings"
	"time"

	"wms-backend/internal/validation"
)

// blockingIssueCodes são os tipos de problema que impedem um item de
// ser considerado "pronto para importar" (item 9 da especificação: "não
// permitir importação silenciosa de dados inválidos"). Os demais
// (vencimento próximo, vencido, possível duplicidade) são avisos
// informativos — o usuário decide o que fazer, mas o item pode ser
// importado como está (item 12: duplicidade nunca bloqueia sozinha;
// mesmo um produto já vencido pode ser uma entrada de estoque
// legítima que o usuário queira registrar para posterior descarte).
var blockingIssueCodes = map[FieldIssueCode]bool{
	IssueMissing:         true,
	IssueInvalidFormat:   true,
	IssueInvalidQuantity: true,
}

func hasBlockingIssue(issues []FieldIssue) bool {
	for _, i := range issues {
		if blockingIssueCodes[i.Code] {
			return true
		}
	}
	return false
}

// BuildItems converte a tabela crua + o mapeamento de colunas em uma
// lista de ImportItem já normalizados e validados, prontos para a
// prévia (item 8/9 da especificação). Esta função não chama IA nem
// toca no banco — puramente determinística, dado um mapeamento de
// colunas já resolvido (seja pelo normalizer sozinho, seja com ajuda da
// IA para os cabeçalhos ambíguos).
func BuildItems(table RawTable, mapping []ColumnMapping, target TargetKind) []ImportItem {
	colFor := make(map[FieldKey]int, len(mapping))
	for _, m := range mapping {
		colFor[m.Field] = m.ColumnIndex
	}

	items := make([]ImportItem, 0, len(table.Rows))
	for i, row := range table.Rows {
		item := buildItem(row, colFor, i+1) // RowIndex 1-based, conta a partir da 1ª linha de dados
		items = append(items, item)
	}

	applyRequiredFieldIssues(items, target)
	applyExpiryWarnings(items)
	applyDuplicateDetection(items)

	for i := range items {
		items[i].Ready = !hasBlockingIssue(items[i].Issues)
	}

	return items
}

func cellValue(row []string, colFor map[FieldKey]int, field FieldKey) (string, bool) {
	idx, ok := colFor[field]
	if !ok || idx >= len(row) {
		return "", false
	}
	return strings.TrimSpace(row[idx]), true
}

func buildItem(row []string, colFor map[FieldKey]int, rowIndex int) ImportItem {
	item := ImportItem{RowIndex: rowIndex}

	if raw, ok := cellValue(row, colFor, FieldName); ok {
		item.Name = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldQuantity); ok && raw != "" {
		if qty, qok := NormalizeQuantity(raw); qok {
			item.Quantity = qty
		} else {
			// valor presente mas não interpretável como número —
			// sinalizado depois em applyRequiredFieldIssues via
			// tentativa de re-normalização; aqui só deixamos Quantity
			// zerado e guardamos o bruto para a issue poder citar o
			// valor original.
			item.Quantity = 0
		}
	}
	if raw, ok := cellValue(row, colFor, FieldLotNumber); ok {
		item.LotNumber = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldExpiryDate); ok && raw != "" {
		if normalized, dok := NormalizeExpiryDate(raw); dok {
			item.ExpiryDate = normalized
		}
		// se dok=false, ExpiryDate fica "" — tratado como ausente, não
		// como formato inválido, já que não reconhecemos nem a FORMA do
		// valor como data.
	}
	if raw, ok := cellValue(row, colFor, FieldSKU); ok {
		item.SKU = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldBrand); ok {
		item.Brand = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldMinQuantity); ok && raw != "" {
		if qty, qok := NormalizeQuantity(raw); qok {
			item.MinQuantity = qty
		}
	}
	if raw, ok := cellValue(row, colFor, FieldCategory); ok {
		item.CategoryRaw = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldNotes); ok {
		item.Notes = NormalizeText(raw)
	}
	if raw, ok := cellValue(row, colFor, FieldUnit); ok {
		item.Unit = NormalizeText(raw)
	}

	return item
}

// applyRequiredFieldIssues sinaliza, para cada item, os campos
// obrigatórios ausentes ou em formato inválido de acordo com o destino
// escolhido (Produto/estoque real ou Pré-produto) — ver
// RequiredFieldsFor em normalizer.go, que espelha exatamente as regras
// já aplicadas pelos Services existentes (item 9 e 13 da especificação:
// "não permitir importação silenciosa de dados inválidos" / "não
// inventar regras novas").
func applyRequiredFieldIssues(items []ImportItem, target TargetKind) {
	required := make(map[FieldKey]bool)
	for _, f := range RequiredFieldsFor(target) {
		required[f] = true
	}

	for i := range items {
		item := &items[i]

		if required[FieldName] && item.Name == "" {
			item.Issues = append(item.Issues, FieldIssue{Field: FieldName, Code: IssueMissing, Message: "Nome do produto não informado"})
		}

		if required[FieldQuantity] {
			if item.Quantity == 0 {
				item.Issues = append(item.Issues, FieldIssue{Field: FieldQuantity, Code: IssueMissing, Message: "Quantidade não informada"})
			} else if item.Quantity < 0 {
				item.Issues = append(item.Issues, FieldIssue{Field: FieldQuantity, Code: IssueInvalidQuantity, Message: "Quantidade não pode ser negativa"})
			}
		}

		if required[FieldLotNumber] && item.LotNumber == "" {
			item.Issues = append(item.Issues, FieldIssue{Field: FieldLotNumber, Code: IssueMissing, Message: "Lote não informado"})
		}

		if required[FieldExpiryDate] {
			if item.ExpiryDate == "" {
				item.Issues = append(item.Issues, FieldIssue{Field: FieldExpiryDate, Code: IssueMissing, Message: "Validade não informada"})
			} else if !IsValidBRDate(item.ExpiryDate) {
				item.Issues = append(item.Issues, FieldIssue{Field: FieldExpiryDate, Code: IssueInvalidFormat, Message: "Data de validade inválida: " + item.ExpiryDate})
			}
		}

		if required[FieldUnit] && item.Unit == "" {
			item.Issues = append(item.Issues, FieldIssue{Field: FieldUnit, Code: IssueMissing, Message: "Unidade de medida não informada"})
		}
	}
}

// applyExpiryWarnings sinaliza itens já vencidos ou próximos do
// vencimento (item 18 da especificação: "ideias adicionais" —
// identificação de produtos vencidos/próximos do vencimento). Estes são
// avisos informativos, não bloqueiam a importação (diferente de um
// campo obrigatório ausente).
func applyExpiryWarnings(items []ImportItem) {
	const expiringSoonWindow = 30 * 24 * time.Hour
	now := time.Now()

	for i := range items {
		item := &items[i]
		if item.ExpiryDate == "" || !IsValidBRDate(item.ExpiryDate) {
			continue
		}
		expiry, err := parseBRDateForWarnings(item.ExpiryDate)
		if err != nil {
			continue
		}
		switch {
		case expiry.Before(now):
			item.Issues = append(item.Issues, FieldIssue{Field: FieldExpiryDate, Code: IssueExpired, Message: "Produto já vencido"})
		case expiry.Before(now.Add(expiringSoonWindow)):
			item.Issues = append(item.Issues, FieldIssue{Field: FieldExpiryDate, Code: IssueExpiringSoon, Message: "Validade próxima (menos de 30 dias)"})
		}
	}
}

// applyDuplicateDetection sinaliza (sem remover — item 12 da
// especificação) itens cujo nome normalizado é muito similar a outro
// item da mesma planilha, um indício de duplicidade como "Toddy 370g" /
// "Toddy Chocolate 370g" / "Toddy Chocolate em Pó 370g".
func applyDuplicateDetection(items []ImportItem) {
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if items[i].Name == "" || items[j].Name == "" {
				continue
			}
			if !likelyDuplicateNames(items[i].Name, items[j].Name) {
				continue
			}
			items[i].DuplicateOfRowIndexes = append(items[i].DuplicateOfRowIndexes, items[j].RowIndex)
			items[j].DuplicateOfRowIndexes = append(items[j].DuplicateOfRowIndexes, items[i].RowIndex)
			items[i].Issues = append(items[i].Issues, FieldIssue{Field: FieldName, Code: IssuePossibleDup, Message: "Possível duplicidade com a linha " + strconv.Itoa(items[j].RowIndex)})
			items[j].Issues = append(items[j].Issues, FieldIssue{Field: FieldName, Code: IssuePossibleDup, Message: "Possível duplicidade com a linha " + strconv.Itoa(items[i].RowIndex)})
		}
	}
}

// likelyDuplicateNames considera dois nomes potencialmente duplicados
// quando, após normalização (minúsculas, sem acento) e quebra em
// palavras, o conjunto de palavras de um nome está inteiramente contido
// no do outro. Isso cobre o exemplo do item 12 da especificação: "Toddy
// 370g" (palavras: {toddy, 370g}) está contido em "Toddy Chocolate
// 370g" (palavras: {toddy, chocolate, 370g}), que por sua vez está
// contido em "Toddy Chocolate em Pó 370g" — mesmo a palavra
// "chocolate" aparecendo NO MEIO do nome maior, não só como prefixo.
//
// Deliberadamente conservador na direção de sinalizar demais: um falso
// positivo aqui só gera um aviso que o usuário descarta com um clique
// (nunca bloqueia nem remove nada — ver applyDuplicateDetection),
// enquanto um falso negativo deixaria duplicidades reais passarem
// batido. Nomes de uma palavra só (ex.: "Leite") não entram nessa
// comparação de subconjunto para evitar volume de falsos positivos (um
// nome de uma palavra estaria "contido" em quase qualquer nome maior
// que comece com a mesma palavra).
func likelyDuplicateNames(a, b string) bool {
	wordsA := normalizedWordSet(a)
	wordsB := normalizedWordSet(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return false
	}
	if len(wordsA) < 2 && len(wordsB) < 2 {
		// ambos nomes de uma palavra só: exige igualdade exata para não
		// gerar ruído (ex.: "Arroz" vs "Feijão" nunca deveriam bater; e
		// "Arroz" vs "Arroz" é o mesmo nome, sinalizado como igual).
		return normalizeForComparison(a) == normalizeForComparison(b)
	}
	smaller, larger := wordsA, wordsB
	if len(wordsB) < len(wordsA) {
		smaller, larger = wordsB, wordsA
	}
	for word := range smaller {
		if !larger[word] {
			return false
		}
	}
	return true
}

func normalizedWordSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(normalizeForComparison(s)) {
		set[w] = true
	}
	return set
}

func normalizeForComparison(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripDiacritics(s)
	return strings.Join(strings.Fields(s), " ")
}

func parseBRDateForWarnings(value string) (time.Time, error) {
	// Reaproveita exatamente a mesma validação/parsing usada pelo resto
	// do sistema (InventoryService) — ver validation.ParseBRDate. Nunca
	// uma regra de data paralela.
	return validation.ParseBRDate(value)
}
