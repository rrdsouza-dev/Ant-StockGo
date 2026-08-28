package importing

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

var (
	ErrEmptyFile         = errors.New("o arquivo está vazio")
	ErrUnsupportedFormat = errors.New("formato de arquivo não suportado: envie um arquivo .xlsx ou .csv")
	ErrNoDataRows        = errors.New("a planilha não contém nenhuma linha de dados")
	ErrCorruptFile       = errors.New("não foi possível ler o arquivo: ele pode estar corrompido ou em um formato inesperado")
	ErrFileTooLarge      = errors.New("arquivo excede o tamanho máximo permitido")
)

// MaxUploadBytes limita o tamanho do arquivo aceito pela Importação
// Inteligente (item 4/25 da especificação: nunca confiar apenas na
// extensão, e limitar tamanho de processamento). 10 MiB é generoso para
// uma planilha de estoque (mesmo com milhares de linhas) sem abrir
// margem para upload de arquivos desproporcionais.
const MaxUploadBytes = 10 * 1024 * 1024

// MaxDataRows limita quantas linhas de dados uma única importação pode
// processar de uma vez (item 29: performance). Acima disso, o usuário é
// orientado a dividir a planilha — processar dezenas de milhares de
// linhas em uma única requisição síncrona não é o objetivo desta V1.
const MaxDataRows = 5000

// zipMagic e as assinaturas OOXML abaixo permitem identificar um XLSX
// pelo conteúdo real do arquivo, não pela extensão informada pelo
// cliente (que pode estar errada ou ser forjada) — ver item 4 da
// especificação: "nunca confiar apenas na extensão do arquivo".
var zipMagic = []byte{0x50, 0x4B, 0x03, 0x04}

// ParseSpreadsheet interpreta o conteúdo bruto de um upload e devolve a
// tabela crua (cabeçalhos + linhas de texto), sem nenhuma normalização
// ainda. filename é usado apenas como dica auxiliar quando a detecção
// por conteúdo for inconclusiva (ex.: CSV vazio de cabeçalho only).
func ParseSpreadsheet(filename string, data []byte) (RawTable, error) {
	if len(data) == 0 {
		return RawTable{}, ErrEmptyFile
	}
	if len(data) > MaxUploadBytes {
		return RawTable{}, ErrFileTooLarge
	}

	if isZipContainer(data) {
		return parseXLSX(data)
	}

	// Não é um container ZIP/OOXML. Se o nome sugere .xlsx mas o
	// conteúdo não bate, é um arquivo forjado ou corrompido — não
	// tentamos "adivinhar" e arriscar interpretar lixo como CSV.
	lowerName := strings.ToLower(filename)
	if strings.HasSuffix(lowerName, ".xlsx") || strings.HasSuffix(lowerName, ".xls") {
		return RawTable{}, ErrCorruptFile
	}

	return parseCSV(data)
}

func isZipContainer(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], zipMagic)
}

// parseCSV lê um CSV tentando separadores comuns (vírgula e ponto-e-vírgula
// — este último frequente em planilhas exportadas com localidade
// pt-BR/Excel) e valida encoding UTF-8 básico. Linhas com menos campos que
// o cabeçalho são preenchidas com string vazia (célula "em branco"); o
// FieldsPerRecord relaxado evita rejeitar toda a planilha por causa de
// uma linha mal formatada — a validação de dados ausentes acontece depois,
// por campo, não aqui no parser.
func parseCSV(data []byte) (RawTable, error) {
	if !utf8.Valid(data) {
		// Tenta remover um BOM UTF-8 residual antes de desistir — bastante
		// comum em CSVs exportados pelo Excel no Windows.
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
		if !utf8.Valid(data) {
			return RawTable{}, fmt.Errorf("%w: o arquivo precisa estar em UTF-8", ErrCorruptFile)
		}
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	delimiter := detectCSVDelimiter(data)

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1 // relaxado: linhas de tamanho diferente não derrubam o parser inteiro
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return RawTable{}, fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}
	records = dropFullyEmptyRows(records)
	if len(records) == 0 {
		return RawTable{}, ErrNoDataRows
	}

	headers := records[0]
	rows := records[1:]
	if len(rows) == 0 {
		return RawTable{}, ErrNoDataRows
	}
	if len(rows) > MaxDataRows {
		return RawTable{}, fmt.Errorf("a planilha tem %d linhas de dados; o máximo suportado por importação é %d", len(rows), MaxDataRows)
	}

	return normalizeRowWidths(RawTable{Headers: headers, Rows: rows}), nil
}

// detectCSVDelimiter escolhe entre vírgula e ponto-e-vírgula contando
// ocorrências na primeira linha — heurística simples, mas suficiente
// para os dois formatos realmente comuns de exportação de planilha.
func detectCSVDelimiter(data []byte) rune {
	firstLine := data
	if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
		firstLine = data[:idx]
	}
	if bytes.Count(firstLine, []byte{';'}) > bytes.Count(firstLine, []byte{','}) {
		return ';'
	}
	return ','
}

func dropFullyEmptyRows(records [][]string) [][]string {
	out := make([][]string, 0, len(records))
	for _, row := range records {
		if rowIsBlank(row) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func rowIsBlank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// normalizeRowWidths preenche linhas mais curtas que o cabeçalho com
// células vazias, e trunca linhas mais longas — protege o resto do
// pipeline (normalizer/validator) de indexação fora do intervalo sem
// precisar checar o tamanho de cada linha em todo lugar.
func normalizeRowWidths(t RawTable) RawTable {
	width := len(t.Headers)
	for i, row := range t.Rows {
		if len(row) == width {
			continue
		}
		fixed := make([]string, width)
		copy(fixed, row) // se row for mais curta, o resto fica "" (zero value); se mais longa, trunca
		t.Rows[i] = fixed
	}
	return t
}

// discardReader é um io.Writer que só conta bytes, usado para validar
// limite de tamanho durante leitura em streaming sem materializar o
// arquivo inteiro antes de saber que ele é grande demais.
type discardCounter struct{ n int64 }

func (d *discardCounter) Write(p []byte) (int, error) {
	d.n += int64(len(p))
	if d.n > MaxUploadBytes {
		return 0, ErrFileTooLarge
	}
	return len(p), nil
}

// ReadUploadLimited lê um io.Reader até MaxUploadBytes+1, retornando
// ErrFileTooLarge se o limite for excedido. Usado pelo handler HTTP para
// nunca materializar em memória um upload maior do que o permitido,
// mesmo que o Content-Length declarado minta.
func ReadUploadLimited(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, MaxUploadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxUploadBytes {
		return nil, ErrFileTooLarge
	}
	return data, nil
}
