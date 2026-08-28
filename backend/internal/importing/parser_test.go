package importing

import (
	"errors"
	"os"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("não foi possível ler fixture %s: %v", name, err)
	}
	return data
}

func TestParseSpreadsheet_XLSXBasic(t *testing.T) {
	data := readFixture(t, "sample_basic.xlsx")
	table, err := ParseSpreadsheet("sample_basic.xlsx", data)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	wantHeaders := []string{"Produto", "Quantidade", "Lote", "Validade"}
	if len(table.Headers) != len(wantHeaders) {
		t.Fatalf("headers = %v, want %v", table.Headers, wantHeaders)
	}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if len(table.Rows) != 4 {
		t.Fatalf("got %d data rows, want 4", len(table.Rows))
	}

	// Primeira linha: Toddy 370g / 25 / TOD-001 / 30/06/2027
	first := table.Rows[0]
	if first[0] != "Toddy 370g" {
		t.Errorf("rows[0][0] = %q, want %q", first[0], "Toddy 370g")
	}
	if first[1] != "25" {
		t.Errorf("rows[0][1] = %q, want %q (quantidade deve vir como texto do número)", first[1], "25")
	}
	if first[3] != "30/06/2027" {
		t.Errorf("rows[0][3] = %q, want %q", first[3], "30/06/2027")
	}

	// Última linha: Biscoito, sem lote nem validade (células vazias)
	last := table.Rows[3]
	if last[0] != "Biscoito" {
		t.Errorf("rows[3][0] = %q, want %q", last[0], "Biscoito")
	}
	if last[2] != "" {
		t.Errorf("rows[3][2] (lote) = %q, want vazio", last[2])
	}
	if last[3] != "" {
		t.Errorf("rows[3][3] (validade) = %q, want vazio", last[3])
	}
}

func TestParseSpreadsheet_XLSXAltHeadersAndBlankRow(t *testing.T) {
	data := readFixture(t, "sample_alt_headers.xlsx")
	table, err := ParseSpreadsheet("sample_alt_headers.xlsx", data)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	wantHeaders := []string{"Nome do Produto", "Qtd", "Nº Lote", "Data de Validade", "Observação"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	// A linha totalmente vazia no meio da planilha deve ter sido
	// descartada — só 2 linhas de dados reais (Arroz, Açúcar).
	if len(table.Rows) != 2 {
		t.Fatalf("got %d data rows, want 2 (linha em branco no meio deve ser descartada); rows=%v", len(table.Rows), table.Rows)
	}
	if table.Rows[0][0] != "Arroz 5kg" {
		t.Errorf("rows[0][0] = %q, want %q", table.Rows[0][0], "Arroz 5kg")
	}
	if table.Rows[1][0] != "Açúcar 1kg" {
		t.Errorf("rows[1][0] = %q, want %q (acentuação/shared strings)", table.Rows[1][0], "Açúcar 1kg")
	}
}

func TestParseSpreadsheet_XLSXEmpty(t *testing.T) {
	data := readFixture(t, "sample_empty.xlsx")
	_, err := ParseSpreadsheet("sample_empty.xlsx", data)
	if !errors.Is(err, ErrNoDataRows) {
		t.Fatalf("err = %v, want ErrNoDataRows", err)
	}
}

func TestParseSpreadsheet_XLSXEdgeCases(t *testing.T) {
	data := readFixture(t, "sample_edge_cases.xlsx")
	table, err := ParseSpreadsheet("sample_edge_cases.xlsx", data)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(table.Rows))
	}
	// Quantidade textual "abc" deve chegar como está — validação de
	// "é numérico?" é responsabilidade do validator, não do parser.
	if table.Rows[0][1] != "abc" {
		t.Errorf("rows[0][1] = %q, want %q (parser não deve tentar validar)", table.Rows[0][1], "abc")
	}
	// Célula de data real do Excel (não string) chega como o número
	// serial em texto — o normalizer é responsável por convertê-lo.
	if table.Rows[1][3] == "" {
		t.Errorf("rows[1][3] (data serial do Excel) veio vazia, esperava algum valor numérico serial")
	}
	t.Logf("valor bruto da célula de data serial do Excel: %q", table.Rows[1][3])
}

func TestParseSpreadsheet_EmptyFile(t *testing.T) {
	_, err := ParseSpreadsheet("qualquer.xlsx", []byte{})
	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("err = %v, want ErrEmptyFile", err)
	}
}

func TestParseSpreadsheet_ForgedExtension(t *testing.T) {
	// Conteúdo que não é nem ZIP nem texto CSV plausível, mas com nome
	// de arquivo .xlsx — deve ser rejeitado como corrompido, nunca
	// "adivinhado" como CSV.
	garbage := []byte("isto nao e um xlsx de verdade, apenas texto qualquer")
	_, err := ParseSpreadsheet("planilha.xlsx", garbage)
	if !errors.Is(err, ErrCorruptFile) {
		t.Fatalf("err = %v, want ErrCorruptFile", err)
	}
}

func TestParseSpreadsheet_CSVBasicComma(t *testing.T) {
	csv := []byte("Produto,Quantidade,Lote,Validade\nToddy 370g,25,TOD-001,30/06/2027\nFeijão 1kg,40,FEJ-002,15/08/2027\n")
	table, err := ParseSpreadsheet("produtos.csv", csv)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(table.Rows))
	}
	if table.Rows[0][0] != "Toddy 370g" {
		t.Errorf("rows[0][0] = %q, want %q", table.Rows[0][0], "Toddy 370g")
	}
}

func TestParseSpreadsheet_CSVSemicolonDelimiter(t *testing.T) {
	// Formato comum de export do Excel em locale pt-BR: ponto-e-vírgula.
	csv := []byte("Produto;Quantidade;Lote;Validade\nArroz 5kg;12;ARZ-01;01/01/2028\n")
	table, err := ParseSpreadsheet("produtos.csv", csv)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(table.Headers) != 4 {
		t.Fatalf("headers = %v, want 4 colunas (delimitador ; não foi detectado)", table.Headers)
	}
	if table.Rows[0][0] != "Arroz 5kg" {
		t.Errorf("rows[0][0] = %q, want %q", table.Rows[0][0], "Arroz 5kg")
	}
}

func TestParseSpreadsheet_CSVWithUTF8BOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	csv := append(bom, []byte("Produto,Quantidade\nAçúcar,20\n")...)
	table, err := ParseSpreadsheet("produtos.csv", csv)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if table.Headers[0] != "Produto" {
		t.Errorf("headers[0] = %q, want %q (BOM não removido corretamente)", table.Headers[0], "Produto")
	}
}

func TestParseSpreadsheet_CSVEmpty(t *testing.T) {
	_, err := ParseSpreadsheet("vazio.csv", []byte("Produto,Quantidade\n"))
	if !errors.Is(err, ErrNoDataRows) {
		t.Fatalf("err = %v, want ErrNoDataRows", err)
	}
}

func TestParseSpreadsheet_UnsupportedExtensionButValidCSVContent(t *testing.T) {
	// Um .txt com conteúdo tabular ainda é aceito, já que a detecção é
	// por conteúdo — não há necessidade de rejeitar aqui; o handler HTTP
	// é quem decide, por política de produto, quais extensões anunciar
	// como aceitas na UI (ver item 4 da especificação).
	csv := []byte("Produto,Quantidade\nItem,1\n")
	table, err := ParseSpreadsheet("dados.txt", csv)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(table.Rows))
	}
}

func TestParseSpreadsheet_FileTooLarge(t *testing.T) {
	huge := make([]byte, MaxUploadBytes+1)
	_, err := ParseSpreadsheet("grande.csv", huge)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
}
