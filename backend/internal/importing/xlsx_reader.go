package importing

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseXLSX lê a primeira planilha (worksheet) de um arquivo .xlsx.
//
// Implementado manualmente sobre archive/zip + encoding/xml (stdlib),
// sem depender de nenhuma biblioteca externa de terceiros: um arquivo
// .xlsx é, no formato OOXML, um container ZIP com XMLs internos. Para o
// propósito desta funcionalidade — ler texto/números de uma grade
// simples de produtos — só precisamos de três partes desse container:
//
//   - xl/worksheets/sheet1.xml  → células e suas posições (A1, B2, ...)
//   - xl/sharedStrings.xml      → tabela de strings (Excel referencia
//     texto por índice nessa tabela em vez de embuti-lo na célula,
//     como otimização de espaço)
//   - xl/workbook.xml           → nomes/ordem das planilhas (usado só
//     para escolher a primeira aba, caso existam várias)
//
// Não há suporte a fórmulas, formatação condicional, imagens, ou
// múltiplas planilhas além da primeira — fora do escopo desta
// funcionalidade (uma planilha simples de produtos).
func parseXLSX(data []byte) (RawTable, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return RawTable{}, fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}

	sharedStrings, err := readSharedStrings(zr)
	if err != nil {
		return RawTable{}, err
	}

	sheetPath, err := firstSheetPath(zr)
	if err != nil {
		return RawTable{}, err
	}

	cells, err := readSheetCells(zr, sheetPath, sharedStrings)
	if err != nil {
		return RawTable{}, err
	}
	if len(cells) == 0 {
		return RawTable{}, ErrNoDataRows
	}

	table := cellsToTable(cells)
	if len(table.Rows) == 0 {
		return RawTable{}, ErrNoDataRows
	}
	if len(table.Rows) > MaxDataRows {
		return RawTable{}, fmt.Errorf("a planilha tem %d linhas de dados; o máximo suportado por importação é %d", len(table.Rows), MaxDataRows)
	}
	return table, nil
}

func openZipFile(zr *zip.Reader, name string) ([]byte, bool, error) {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, true, fmt.Errorf("%w: %v", ErrCorruptFile, err)
		}
		defer rc.Close()
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(rc); err != nil {
			return nil, true, fmt.Errorf("%w: %v", ErrCorruptFile, err)
		}
		return buf.Bytes(), true, nil
	}
	return nil, false, nil
}

// ── sharedStrings.xml ──────────────────────────────────────────────

type sharedStringsXML struct {
	XMLName xml.Name       `xml:"sst"`
	Items   []sharedItemXML `xml:"si"`
}

type sharedItemXML struct {
	// Texto simples: <si><t>valor</t></si>
	Text *string `xml:"t"`
	// Texto com formatação rica (rich text runs): <si><r><t>parte</t></r>...</si>
	// Concatenamos as partes — não precisamos preservar negrito/itálico
	// para os fins desta importação.
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func readSharedStrings(zr *zip.Reader) ([]string, error) {
	raw, found, err := openZipFile(zr, "xl/sharedStrings.xml")
	if err != nil {
		return nil, err
	}
	if !found {
		// Planilhas muito simples podem não ter sharedStrings.xml (todas
		// as células são numéricas, por exemplo) — não é erro.
		return nil, nil
	}
	var parsed sharedStringsXML
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("%w: sharedStrings.xml inválido: %v", ErrCorruptFile, err)
	}
	out := make([]string, len(parsed.Items))
	for i, item := range parsed.Items {
		if item.Text != nil {
			out[i] = *item.Text
			continue
		}
		var sb strings.Builder
		for _, r := range item.Runs {
			sb.WriteString(r.Text)
		}
		out[i] = sb.String()
	}
	return out, nil
}

// ── workbook.xml (para achar a primeira planilha) ──────────────────

type workbookXML struct {
	Sheets struct {
		Sheet []struct {
			Name    string `xml:"name,attr"`
			SheetID string `xml:"sheetId,attr"`
			RID     string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheet"`
	} `xml:"sheets"`
}

type workbookRelsXML struct {
	Relationships []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// firstSheetPath resolve o caminho dentro do ZIP da primeira planilha
// declarada em workbook.xml, seguindo a relação até o arquivo físico
// (workbook.xml.rels). Se qualquer parte dessa cadeia estiver ausente
// (planilhas muito simples geradas por outras ferramentas às vezes
// omitem rels), cai de volta para o caminho convencional
// "xl/worksheets/sheet1.xml", que é o que praticamente todo gerador de
// XLSX usa por padrão.
func firstSheetPath(zr *zip.Reader) (string, error) {
	const fallback = "xl/worksheets/sheet1.xml"

	wbRaw, found, err := openZipFile(zr, "xl/workbook.xml")
	if err != nil || !found {
		return fallback, nil
	}
	var wb workbookXML
	if err := xml.Unmarshal(wbRaw, &wb); err != nil || len(wb.Sheets.Sheet) == 0 {
		return fallback, nil
	}
	firstRID := wb.Sheets.Sheet[0].RID

	relsRaw, found, err := openZipFile(zr, "xl/_rels/workbook.xml.rels")
	if err != nil || !found || firstRID == "" {
		return fallback, nil
	}
	var rels workbookRelsXML
	if err := xml.Unmarshal(relsRaw, &rels); err != nil {
		return fallback, nil
	}
	for _, rel := range rels.Relationships {
		if rel.ID != firstRID {
			continue
		}
		target := strings.TrimPrefix(rel.Target, "/")
		if !strings.HasPrefix(target, "xl/") {
			target = "xl/" + target
		}
		if _, ok, _ := openZipFile(zr, target); ok {
			return target, nil
		}
	}
	return fallback, nil
}

// ── sheet1.xml (células) ────────────────────────────────────────────

type sheetXML struct {
	SheetData struct {
		Rows []struct {
			RowNum string `xml:"r,attr"`
			Cells  []struct {
				Ref  string `xml:"r,attr"` // ex.: "B7"
				Type string `xml:"t,attr"` // "s" = shared string, "" = número, "str"/"inlineStr" = texto direto
				// Valor numérico ou índice da shared string.
				Value *string `xml:"v"`
				// Texto inline: <c t="inlineStr"><is><t>valor</t></is></c>
				InlineStr *struct {
					Text string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"row"`
	} `xml:"sheetData"`
}

type cell struct {
	col, row int // 0-based
	value    string
}

var cellRefRegex = regexp.MustCompile(`^([A-Z]+)(\d+)$`)

// columnIndexFromLetters converte a parte alfabética de uma referência
// de célula ("A", "B", ..., "Z", "AA", "AB", ...) para um índice 0-based,
// seguindo a mesma base-26 (sem zero) usada pelo próprio Excel.
func columnIndexFromLetters(letters string) int {
	idx := 0
	for _, ch := range letters {
		idx = idx*26 + int(ch-'A'+1)
	}
	return idx - 1
}

func parseCellRef(ref string) (col, row int, ok bool) {
	m := cellRefRegex.FindStringSubmatch(ref)
	if m == nil {
		return 0, 0, false
	}
	col = columnIndexFromLetters(m[1])
	rowNum, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, false
	}
	return col, rowNum - 1, true
}

func readSheetCells(zr *zip.Reader, sheetPath string, sharedStrings []string) ([]cell, error) {
	raw, found, err := openZipFile(zr, sheetPath)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: planilha '%s' não encontrada dentro do arquivo", ErrCorruptFile, sheetPath)
	}

	var sheet sheetXML
	if err := xml.Unmarshal(raw, &sheet); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}

	var cells []cell
	for _, row := range sheet.SheetData.Rows {
		for _, c := range row.Cells {
			col, rowIdx, ok := parseCellRef(c.Ref)
			if !ok {
				continue // célula sem referência de posição válida — ignorada
			}
			value := resolveCellValue(c.Type, c.Value, c.InlineStr, sharedStrings)
			cells = append(cells, cell{col: col, row: rowIdx, value: value})
		}
	}
	return cells, nil
}

func resolveCellValue(cellType string, v *string, inline *struct {
	Text string `xml:"t"`
}, sharedStrings []string) string {
	if inline != nil {
		return inline.Text
	}
	if v == nil {
		return ""
	}
	raw := *v
	if cellType == "s" {
		idx, err := strconv.Atoi(raw)
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return ""
		}
		return sharedStrings[idx]
	}
	// Datas no Excel são armazenadas como número serial (dias desde
	// 1899-12-30). Sem acesso ao styles.xml associando cada célula a um
	// formato de data, não é seguro decidir aqui se um número é uma data
	// ou uma quantidade — isso é resolvido depois, no normalizer, que já
	// sabe qual campo lógico aquela coluna representa (ver
	// normalizer.go: tryParseExcelSerialDate só é tentado para colunas
	// mapeadas como FieldExpiryDate).
	return raw
}

// cellsToTable converte a lista esparsa de células em uma grade densa
// (RawTable), usando a primeira linha ocupada como cabeçalho.
func cellsToTable(cells []cell) RawTable {
	maxCol, maxRow := -1, -1
	for _, c := range cells {
		if c.col > maxCol {
			maxCol = c.col
		}
		if c.row > maxRow {
			maxRow = c.row
		}
	}
	if maxCol < 0 || maxRow < 0 {
		return RawTable{}
	}

	grid := make([][]string, maxRow+1)
	for i := range grid {
		grid[i] = make([]string, maxCol+1)
	}
	for _, c := range cells {
		grid[c.row][c.col] = c.value
	}

	// A primeira linha da grade que não está inteiramente em branco vira
	// o cabeçalho — planilhas com uma linha de título/logo acima da
	// tabela real não são o caso comum aqui, então não tentamos pular
	// linhas em branco no meio do conteúdo, só no topo.
	headerIdx := 0
	for headerIdx < len(grid) && rowIsBlank(grid[headerIdx]) {
		headerIdx++
	}
	if headerIdx >= len(grid) {
		return RawTable{}
	}

	headers := grid[headerIdx]
	dataRows := dropFullyEmptyRows(grid[headerIdx+1:])

	// grid já é indexado pelo número de linha absoluto do XML, então a
	// ordem das linhas já sai naturalmente crescente — nenhuma ordenação
	// adicional é necessária aqui.
	return normalizeRowWidths(RawTable{Headers: headers, Rows: dataRows})
}
