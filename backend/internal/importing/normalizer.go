package importing

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"wms-backend/internal/validation"
)

// excelEpoch é o dia zero do sistema de datas seriais do Excel
// (1900-based; a base real usada por praticamente todas as
// implementações é 1899-12-30, não 1900-01-01, por causa do conhecido
// "bug" histórico do Excel tratando 1900 como ano bissexto). Usado só
// para interpretar células de data que o Excel armazenou como número
// serial em vez de texto (ver xlsx_reader.go: resolveCellValue).
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// knownHeaderAliases mapeia variações comuns e já esperadas de nome de
// coluna (minúsculas, sem acento, sem espaço) para o campo lógico
// correspondente. Esta é a camada DETERMINÍSTICA (item 3/19 da
// especificação): resolve a esmagadora maioria das planilhas reais sem
// nunca precisar chamar a IA. A IA só é consultada para cabeçalhos que
// não batem com nenhuma entrada aqui (ver ai_analyzer.go).
var knownHeaderAliases = map[string]FieldKey{
	// nome do produto
	"produto":          FieldName,
	"nomedoproduto":    FieldName,
	"nome":             FieldName,
	"descricao":        FieldName,
	"item":             FieldName,
	"produtodescricao": FieldName,
	"mercadoria":       FieldName,

	// quantidade
	"quantidade": FieldQuantity,
	"qtd":        FieldQuantity,
	"qtde":       FieldQuantity,
	"quant":      FieldQuantity,
	"estoque":    FieldQuantity,
	"qtdatual":   FieldQuantity,

	// lote
	"lote":    FieldLotNumber,
	"nlote":   FieldLotNumber,
	"numlote": FieldLotNumber,
	"batch":   FieldLotNumber,

	// validade
	"validade":         FieldExpiryDate,
	"datadevalidade":   FieldExpiryDate,
	"vencimento":       FieldExpiryDate,
	"datadevencimento": FieldExpiryDate,
	"dtvalidade":       FieldExpiryDate,
	"expiracao":        FieldExpiryDate,

	// código / SKU
	"codigo":       FieldSKU,
	"cod":          FieldSKU,
	"sku":          FieldSKU,
	"codbarras":    FieldSKU,
	"codigobarras": FieldSKU,
	"referencia":   FieldSKU,

	// marca
	"marca":      FieldBrand,
	"fabricante": FieldBrand,

	// estoque mínimo
	"estoqueminimo":    FieldMinQuantity,
	"qtdminima":        FieldMinQuantity,
	"minimo":           FieldMinQuantity,
	"quantidademinima": FieldMinQuantity,

	// categoria
	"categoria": FieldCategory,
	"grupo":     FieldCategory,
	"tipo":      FieldCategory,

	// unidade (relevante só quando target = pré-produto)
	"unidade":       FieldUnit,
	"unidademedida": FieldUnit,
	"un":            FieldUnit,

	// observações
	"observacao":  FieldNotes,
	"observacoes": FieldNotes,
	"obs":         FieldNotes,
	"notas":       FieldNotes,
}

// RequiredFieldsFor lista, para cada destino (Produto/estoque real ou
// Pré-produto), quais campos lógicos são obrigatórios para um item ser
// considerado "pronto para importar" — espelha exatamente as regras já
// aplicadas por InventoryService.Create e PreProductService.Create (ver
// domain/inventory.go, domain/pre_product.go e services/pre_product_service.go),
// sem inventar novas regras (item 13 da especificação).
//
// Pré-produto exige Name E Unit (PreProductService.Create rejeita
// unidade de medida vazia) — não é só o nome, como uma leitura
// apressada do domínio (pré-produto "não tem quantidade/lote/validade")
// poderia sugerir.
func RequiredFieldsFor(target TargetKind) []FieldKey {
	switch target {
	case TargetPreProduct:
		return []FieldKey{FieldName, FieldUnit}
	default: // TargetInventoryItem
		return []FieldKey{FieldName, FieldQuantity, FieldLotNumber, FieldExpiryDate}
	}
}

// normalizeHeaderKey reduz um cabeçalho de coluna a uma chave comparável:
// minúsculas, sem acentuação, sem espaços/pontuação. "Nº Lote", "N Lote"
// e "numero_lote" caem todas na mesma chave — na prática cobrimos as
// variações mais comuns explicitamente na tabela acima em vez de tentar
// um algoritmo de distância de edição, que arriscaria mapear colunas
// erradas silenciosamente.
func normalizeHeaderKey(header string) string {
	h := strings.ToLower(strings.TrimSpace(header))
	h = stripDiacritics(h)
	var sb strings.Builder
	for _, r := range h {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

var diacriticsReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func stripDiacritics(s string) string {
	return diacriticsReplacer.Replace(s)
}

// MapColumnsDeterministic tenta associar cada cabeçalho a um campo
// lógico conhecido, sem envolver IA. Cabeçalhos que não batem com
// nenhum alias conhecido são devolvidos separadamente como candidatos a
// resolução por IA (ver ai_analyzer.go) — mas só entram nessa lista se
// forem colunas com conteúdo real (colunas totalmente vazias de dados
// são ignoradas, não geram ambiguidade nem chamada de IA à toa).
func MapColumnsDeterministic(table RawTable) (mapped []ColumnMapping, ambiguous []AmbiguousHeader) {
	seen := map[FieldKey]bool{}
	for i, header := range table.Headers {
		trimmed := strings.TrimSpace(header)
		if trimmed == "" {
			continue // coluna sem nome — ignorada silenciosamente, não é ambiguidade
		}
		key := normalizeHeaderKey(trimmed)
		field, ok := knownHeaderAliases[key]
		if ok && !seen[field] {
			// Em caso de colunas duplicadas mapeando pro mesmo campo
			// (planilha malformada), só a primeira ocorrência é usada —
			// evita sobrescrever silenciosamente na hora de montar os itens.
			mapped = append(mapped, ColumnMapping{ColumnIndex: i, Header: trimmed, Field: field, Source: "deterministic"})
			seen[field] = true
			continue
		}
		if columnHasAnyData(table, i) {
			ambiguous = append(ambiguous, AmbiguousHeader{
				ColumnIndex:  i,
				Header:       trimmed,
				SampleValues: sampleColumnValues(table, i, 3),
			})
		}
	}
	return mapped, ambiguous
}

func columnHasAnyData(table RawTable, col int) bool {
	for _, row := range table.Rows {
		if col < len(row) && strings.TrimSpace(row[col]) != "" {
			return true
		}
	}
	return false
}

func sampleColumnValues(table RawTable, col int, max int) []string {
	var out []string
	for _, row := range table.Rows {
		if col >= len(row) {
			continue
		}
		v := strings.TrimSpace(row[col])
		if v == "" {
			continue
		}
		out = append(out, v)
		if len(out) >= max {
			break
		}
	}
	return out
}

// quantityRegex aceita dígitos com separadores de milhar/decimal
// opcionais (ponto ou vírgula) — ex.: "1.234", "1234", " 25 un" (extrai
// só a parte numérica inicial).
var quantityRegex = regexp.MustCompile(`-?\d[\d.,]*`)

// NormalizeQuantity interpreta o texto de uma célula de quantidade como
// um inteiro. Aceita formatos comuns de planilha (separador de milhar,
// casas decimais que são truncadas — estoque é sempre unidade inteira
// neste sistema, ver domain.InventoryItem.Quantity) e texto
// acompanhando o número (ex.: "25 unidades"). Retorna (0, false) quando
// não há nenhum número interpretável — célula vazia e número inválido
// são tratados de forma diferente pelo validator (ver validator.go).
func NormalizeQuantity(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	match := quantityRegex.FindString(raw)
	if match == "" {
		return 0, false
	}
	cleaned := cleanNumericString(match)
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false
	}
	return int(f), true // trunca casas decimais — estoque é sempre inteiro
}

// cleanNumericString resolve qual símbolo (ponto ou vírgula) é o
// separador decimal quando ambos aparecem, e remove separadores de
// milhar, para então poder usar strconv.ParseFloat.
//
// Quando só um dos dois símbolos aparece, a regra é: se houver
// exatamente 3 dígitos após o único símbolo (padrão de agrupamento de
// milhar em pt-BR e en-US: "1.234", "1,234"), ele é tratado como
// separador de MILHAR e removido. Caso contrário (ex.: "25,5", "3.14"),
// é tratado como separador DECIMAL. Isso cobre corretamente tanto
// "1.234" (planilha com milhar) quanto "25,5" (planilha com quantidade
// fracionária, que de qualquer forma é truncada depois — estoque é
// inteiro).
func cleanNumericString(s string) string {
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	switch {
	case hasDot && hasComma:
		lastDot := strings.LastIndex(s, ".")
		lastComma := strings.LastIndex(s, ",")
		if lastComma > lastDot {
			// vírgula é o separador decimal: "1.234,50" → "1234.50"
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			// ponto é o separador decimal: "1,234.50" → "1234.50"
			s = strings.ReplaceAll(s, ",", "")
		}
	case hasDot:
		s = resolveThousandsOrDecimal(s, '.')
	case hasComma:
		s = resolveThousandsOrDecimal(s, ',')
	}
	return s
}

// resolveThousandsOrDecimal decide, para um único símbolo separador
// presente na string, se ele é milhar (remove) ou decimal (converte
// para ".", formato aceito por strconv.ParseFloat).
func resolveThousandsOrDecimal(s string, sep byte) string {
	idx := strings.LastIndexByte(s, sep)
	digitsAfter := len(s) - idx - 1
	if digitsAfter == 3 {
		// separador de milhar: remove
		return strings.ReplaceAll(s, string(sep), "")
	}
	if sep == ',' {
		return strings.Replace(s, ",", ".", 1)
	}
	return s // já é '.', formato aceito nativamente por ParseFloat
}

// NormalizeExpiryDate converte o valor bruto de uma célula de validade
// para o formato DD/MM/AAAA usado pelo restante do sistema (ver
// validation.ParseBRDate). Aceita três formas de entrada:
//  1. já no formato DD/MM/AAAA (ou D/M/AAAA — normaliza para 2 dígitos)
//  2. número serial de data do Excel (célula de data real, não texto)
//  3. formato ISO AAAA-MM-DD (comum em exports de CSV de outros sistemas)
//
// Retorna (candidato, true) sempre que reconhece a FORMA do valor como
// data, mesmo que a data em si seja inválida (ex.: "31/02/2027") — nesse
// caso o candidato formatado é devolvido para que o validator gere uma
// mensagem específica de "data inválida" em vez de "ausente" (ver
// validator.go, que reaplica validation.ParseBRDate para decidir isso).
// Retorna ("", false) apenas quando o valor não tem formato de data
// reconhecível.
func NormalizeExpiryDate(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	// Caso 2: número serial do Excel — só dígitos, sem barra nem hífen,
	// dentro de um intervalo plausível (evita interpretar um ano como
	// "2027" sozinho como serial; datas seriais de estoque real caem
	// tipicamente entre ~25000 [ano ~1968] e ~80000 [ano ~2119]).
	if isAllDigits(raw) {
		if serial, err := strconv.Atoi(raw); err == nil && serial > 25000 && serial < 80000 {
			d := excelEpoch.AddDate(0, 0, serial)
			return d.Format("02/01/2006"), true
		}
	}

	// Caso 1: DD/MM/AAAA ou D/M/AAAA
	if parts := strings.Split(raw, "/"); len(parts) == 3 {
		day, err1 := strconv.Atoi(parts[0])
		month, err2 := strconv.Atoi(parts[1])
		year, err3 := strconv.Atoi(parts[2])
		if err1 == nil && err2 == nil && err3 == nil {
			if year < 100 {
				year += 2000 // planilhas com ano de 2 dígitos, ex.: "30/06/27"
			}
			return zeroPadDate(day, month, year), true
		}
	}

	// Caso 3: ISO AAAA-MM-DD
	if parts := strings.Split(raw, "-"); len(parts) == 3 && len(parts[0]) == 4 {
		year, err1 := strconv.Atoi(parts[0])
		month, err2 := strconv.Atoi(parts[1])
		day, err3 := strconv.Atoi(parts[2])
		if err1 == nil && err2 == nil && err3 == nil {
			return zeroPadDate(day, month, year), true
		}
	}

	return "", false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func zeroPadDate(day, month, year int) string {
	return padInt(day) + "/" + padInt(month) + "/" + strconv.Itoa(year)
}

func padInt(n int) string {
	s := strconv.Itoa(n)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// IsValidBRDate confere se uma string DD/MM/AAAA representa uma data
// real, reutilizando exatamente a mesma validação usada pelo restante
// do sistema (InventoryService), em vez de duplicar a regra aqui.
func IsValidBRDate(value string) bool {
	_, err := validation.ParseBRDate(value)
	return err == nil
}

// NormalizeText colapsa espaços internos múltiplos e remove espaços nas
// pontas — normalização simples de texto (nome de produto, marca,
// observações) que o parser não faz.
func NormalizeText(raw string) string {
	fields := strings.Fields(raw)
	return strings.Join(fields, " ")
}
