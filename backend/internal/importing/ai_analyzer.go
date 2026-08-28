package importing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"wms-backend/internal/ia"
)

// HeaderResolver é o único ponto de contato entre a Importação
// Inteligente e um modelo de IA. Interface pequena e isolada de
// propósito: permite testar todo o resto do pacote (normalizer,
// validator, service) sem precisar de um Ollama real rodando, e deixa
// explícito que a IA é usada SOMENTE para esta responsabilidade
// pontual — resolver cabeçalhos de coluna ambíguos (item 3/19/29 da
// especificação: a IA nunca vê a planilha inteira, nunca conta linhas,
// nunca valida datas, nunca decide quantidades).
type HeaderResolver interface {
	// ResolveHeaders recebe os cabeçalhos que o mapeamento determinístico
	// não conseguiu associar a nenhum campo conhecido, junto de algumas
	// amostras de valor de cada coluna (para dar contexto sem enviar a
	// planilha inteira), e devolve um mapeamento best-effort de cabeçalho
	// para campo lógico. Cabeçalhos que a IA também não conseguir
	// resolver simplesmente não aparecem no mapa de retorno — não é
	// tratado como erro, apenas como "ficou sem mapear" (ver
	// service.go: UnmappedHeaders).
	ResolveHeaders(ctx context.Context, ambiguous []AmbiguousHeader, knownFields []FieldKey) (map[string]FieldKey, error)
}

// OllamaHeaderResolver implementa HeaderResolver usando o mesmo cliente
// Ollama já configurado para o Otis (ver internal/ia/ollama.go) — não
// duplicamos configuração de conexão/modelo, apenas reaproveitamos o
// client existente com um prompt de sistema diferente e focado.
type OllamaHeaderResolver struct {
	client *ia.OllamaClient
}

func NewOllamaHeaderResolver(client *ia.OllamaClient) *OllamaHeaderResolver {
	return &OllamaHeaderResolver{client: client}
}

const headerResolverSystemPrompt = `Você é um assistente que ajuda a interpretar cabeçalhos de colunas de planilhas de estoque de um sistema de almoxarifado escolar chamado ANT-Stock.

Você receberá uma lista de cabeçalhos de coluna que não puderam ser reconhecidos automaticamente, cada um com até 3 valores de exemplo da própria coluna, e a lista de campos lógicos possíveis do sistema.

Sua única tarefa é decidir, para cada cabeçalho, qual campo lógico ele representa — ou nenhum, se não corresponder a nada da lista.

Responda APENAS com um objeto JSON válido, sem nenhum texto antes ou depois, no formato exato:
{"mapeamentos": [{"cabecalho": "texto exato do cabeçalho recebido", "campo": "um dos campos lógicos, ou null se não corresponder a nenhum"}]}

Não invente campos que não estão na lista fornecida. Se um cabeçalho parecer ser uma informação irrelevante para estoque (ex.: "Fornecedor", "Comentário Interno"), responda "campo": null para ele.`

type headerResolverResponseItem struct {
	Cabecalho string  `json:"cabecalho"`
	Campo     *string `json:"campo"`
}

type headerResolverResponse struct {
	Mapeamentos []headerResolverResponseItem `json:"mapeamentos"`
}

func (r *OllamaHeaderResolver) ResolveHeaders(ctx context.Context, ambiguous []AmbiguousHeader, knownFields []FieldKey) (map[string]FieldKey, error) {
	if len(ambiguous) == 0 {
		return map[string]FieldKey{}, nil
	}

	prompt := buildHeaderResolverPrompt(ambiguous, knownFields)
	raw, err := r.client.Chat(ctx, headerResolverSystemPrompt, []ia.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar IA para resolver cabeçalhos: %w", err)
	}

	parsed, err := parseHeaderResolverResponse(raw)
	if err != nil {
		// Resposta da IA fora do formato esperado: tratado como "IA não
		// ajudou desta vez", não como falha da importação inteira (item
		// 20 da especificação: a Importação Inteligente deve continuar
		// funcionando mesmo quando um componente de IA falha).
		return map[string]FieldKey{}, nil
	}

	validFields := map[FieldKey]bool{}
	for _, f := range knownFields {
		validFields[f] = true
	}

	result := make(map[string]FieldKey, len(parsed.Mapeamentos))
	for _, m := range parsed.Mapeamentos {
		if m.Campo == nil {
			continue
		}
		field := FieldKey(*m.Campo)
		if !validFields[field] {
			continue // IA sugeriu um campo que não existe — ignorado, não é erro fatal
		}
		result[m.Cabecalho] = field
	}
	return result, nil
}

func buildHeaderResolverPrompt(ambiguous []AmbiguousHeader, knownFields []FieldKey) string {
	var sb strings.Builder
	sb.WriteString("Campos lógicos possíveis: ")
	names := make([]string, len(knownFields))
	for i, f := range knownFields {
		names[i] = string(f)
	}
	sb.WriteString(strings.Join(names, ", "))
	sb.WriteString("\n\nCabeçalhos a resolver:\n")
	for _, h := range ambiguous {
		sb.WriteString(fmt.Sprintf("- %q (exemplos: %s)\n", h.Header, strings.Join(h.SampleValues, "; ")))
	}
	return sb.String()
}

// parseHeaderResolverResponse extrai o JSON da resposta do modelo,
// tolerando cercas de código markdown (```json ... ```) que modelos
// pequenos às vezes adicionam mesmo quando instruídos a não fazer isso.
func parseHeaderResolverResponse(raw string) (headerResolverResponse, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var parsed headerResolverResponse
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return headerResolverResponse{}, fmt.Errorf("resposta da IA não é um JSON válido: %w", err)
	}
	return parsed, nil
}

// resolvableFieldsForAI é o subconjunto de FieldKey que faz sentido
// oferecer à IA como opção — deliberadamente exclui campos que o
// normalizer determinístico já cobre com alta confiança em praticamente
// toda planilha real (nome, quantidade, lote, validade raramente vêm
// com nomes exóticos o suficiente para chegar à IA), mantendo o prompt
// focado nos campos "secundários" mais propensos a nomenclatura variada.
var resolvableFieldsForAI = []FieldKey{
	FieldName, FieldQuantity, FieldLotNumber, FieldExpiryDate,
	FieldSKU, FieldBrand, FieldMinQuantity, FieldCategory, FieldNotes, FieldUnit,
}
