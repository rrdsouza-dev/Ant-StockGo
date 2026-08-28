package importing

import (
	"testing"
	"time"
)

func specExampleTable() RawTable {
	// Exatamente o exemplo dos itens 2/8/9 da especificação.
	return RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows: [][]string{
			{"Toddy 370g", "25", "TOD-001", "30/06/2027"},
			{"Feijão 1kg", "40", "FEJ-002", "15/08/2027"},
			{"Leite 1L", "30", "LEI-003", ""},
			{"Biscoito", "50", "", ""},
		},
	}
}

func TestBuildItems_SpecExample(t *testing.T) {
	table := specExampleTable()
	mapping, ambiguous := MapColumnsDeterministic(table)
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none", ambiguous)
	}
	items := BuildItems(table, mapping, TargetInventoryItem)
	if len(items) != 4 {
		t.Fatalf("got %d items, want 4", len(items))
	}

	// Toddy: tudo presente → pronto
	toddy := items[0]
	if toddy.Name != "Toddy 370g" || toddy.Quantity != 25 || toddy.LotNumber != "TOD-001" || toddy.ExpiryDate != "30/06/2027" {
		t.Errorf("toddy = %+v, campos não batem com a planilha", toddy)
	}
	if !toddy.Ready {
		t.Errorf("toddy.Ready = false, want true (todos os campos obrigatórios presentes); issues=%+v", toddy.Issues)
	}

	// Feijão: tudo presente → pronto
	if !items[1].Ready {
		t.Errorf("feijão.Ready = false, want true; issues=%+v", items[1].Issues)
	}

	// Leite: validade ausente → não pronto, com issue de campo ausente em ExpiryDate
	leite := items[2]
	if leite.Ready {
		t.Errorf("leite.Ready = true, want false (validade ausente)")
	}
	if !hasIssue(leite.Issues, FieldExpiryDate, IssueMissing) {
		t.Errorf("leite.Issues = %+v, want issue de validade ausente", leite.Issues)
	}
	// Lote de leite está presente, não deve gerar issue de lote.
	if hasIssue(leite.Issues, FieldLotNumber, IssueMissing) {
		t.Errorf("leite não deveria ter issue de lote ausente (LEI-003 está presente)")
	}

	// Biscoito: lote E validade ausentes → duas issues
	biscoito := items[3]
	if biscoito.Ready {
		t.Errorf("biscoito.Ready = true, want false")
	}
	if !hasIssue(biscoito.Issues, FieldLotNumber, IssueMissing) {
		t.Errorf("biscoito deveria ter issue de lote ausente")
	}
	if !hasIssue(biscoito.Issues, FieldExpiryDate, IssueMissing) {
		t.Errorf("biscoito deveria ter issue de validade ausente")
	}
}

func hasIssue(issues []FieldIssue, field FieldKey, code FieldIssueCode) bool {
	for _, i := range issues {
		if i.Field == field && i.Code == code {
			return true
		}
	}
	return false
}

func TestBuildItems_PreProductTarget_NameAndUnitRequired(t *testing.T) {
	// Mesma planilha, mas destino = pré-produto: lote/validade não são
	// exigidos (item 13 da especificação — pré-produto não tem estoque),
	// mas Unit é obrigatório (PreProductService.Create rejeita unidade
	// vazia) — sem coluna de unidade na planilha, nenhum item deve estar
	// Ready para esse destino.
	table := specExampleTable()
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetPreProduct)

	biscoito := items[3] // sem lote nem validade, e a planilha não tem coluna de unidade
	if biscoito.Ready {
		t.Errorf("biscoito.Ready = true, want false para target=pre_product sem coluna de unidade (Unit é obrigatório)")
	}
	if !hasIssue(biscoito.Issues, FieldUnit, IssueMissing) {
		t.Errorf("issues = %+v, want issue de unidade ausente", biscoito.Issues)
	}
	// Mas não deve mais reclamar de lote/validade, que não são
	// obrigatórios para pré-produto.
	if hasIssue(biscoito.Issues, FieldLotNumber, IssueMissing) || hasIssue(biscoito.Issues, FieldExpiryDate, IssueMissing) {
		t.Errorf("pré-produto não deveria exigir lote/validade: %+v", biscoito.Issues)
	}
}

func TestBuildItems_PreProductTarget_ReadyWithUnitColumn(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Unidade"},
		Rows:    [][]string{{"Caneta Azul", "unidade"}},
	}
	mapping, ambiguous := MapColumnsDeterministic(table)
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none", ambiguous)
	}
	items := BuildItems(table, mapping, TargetPreProduct)
	if !items[0].Ready {
		t.Errorf("item com nome e unidade preenchidos deveria estar Ready para pré-produto; issues=%+v", items[0].Issues)
	}
}

func TestBuildItems_MissingProductName(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"", "10", "LOT-1", "01/01/2028"}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if items[0].Ready {
		t.Error("item sem nome não deveria estar Ready")
	}
	if !hasIssue(items[0].Issues, FieldName, IssueMissing) {
		t.Errorf("issues = %+v, want issue de nome ausente", items[0].Issues)
	}
}

func TestBuildItems_InvalidQuantity(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item Estranho", "abc", "LOT-1", "01/01/2028"}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if items[0].Ready {
		t.Error("item com quantidade não numérica não deveria estar Ready")
	}
	if !hasIssue(items[0].Issues, FieldQuantity, IssueMissing) {
		t.Errorf("issues = %+v, want issue de quantidade (tratada como ausente quando não numérica)", items[0].Issues)
	}
}

func TestBuildItems_InvalidDateFormat(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item X", "5", "LOT-1", "31/02/2027"}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if items[0].Ready {
		t.Error("item com data inválida (31/02) não deveria estar Ready")
	}
	if !hasIssue(items[0].Issues, FieldExpiryDate, IssueInvalidFormat) {
		t.Errorf("issues = %+v, want issue de formato de data inválido (não 'ausente' — a diferença importa para a mensagem ao usuário)", items[0].Issues)
	}
}

func TestBuildItems_DuplicateDetection_SpecExample(t *testing.T) {
	// Exatamente o exemplo do item 12 da especificação.
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows: [][]string{
			{"Toddy 370g", "10", "L1", "01/01/2028"},
			{"Toddy Chocolate 370g", "10", "L2", "01/01/2028"},
			{"Toddy Chocolate em Pó 370g", "10", "L3", "01/01/2028"},
			{"Feijão 1kg", "10", "L4", "01/01/2028"}, // não deve ser marcado como duplicata de nada
		},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)

	for i := 0; i < 3; i++ {
		if len(items[i].DuplicateOfRowIndexes) == 0 {
			t.Errorf("items[%d] (%q) deveria ter sido sinalizado como possível duplicidade", i, items[i].Name)
		}
		if !hasIssue(items[i].Issues, FieldName, IssuePossibleDup) {
			t.Errorf("items[%d] (%q) deveria ter issue de possível duplicidade", i, items[i].Name)
		}
	}
	if len(items[3].DuplicateOfRowIndexes) != 0 {
		t.Errorf("Feijão não deveria ser marcado como duplicata: %+v", items[3])
	}

	// Duplicidade NUNCA remove o item automaticamente (item 12: "não
	// excluir automaticamente, mostrar ao usuário para decidir").
	if len(items) != 4 {
		t.Fatalf("got %d items, want 4 (nenhum item deve ser removido por causa de duplicidade)", len(items))
	}
}

func TestBuildItems_DuplicateDetection_UnrelatedNamesNotFlagged(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows: [][]string{
			{"Arroz 5kg", "10", "L1", "01/01/2028"},
			{"Feijão 1kg", "10", "L2", "01/01/2028"},
			{"Leite Integral 1L", "10", "L3", "01/01/2028"},
		},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	for _, item := range items {
		if len(item.DuplicateOfRowIndexes) != 0 {
			t.Errorf("%q não deveria ser sinalizado como duplicidade de nada: %+v", item.Name, item.DuplicateOfRowIndexes)
		}
	}
}

func TestBuildItems_ExpiredProductWarning(t *testing.T) {
	past := time.Now().AddDate(0, 0, -10).Format("02/01/2006")
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item Vencido", "5", "L1", past}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if !hasIssue(items[0].Issues, FieldExpiryDate, IssueExpired) {
		t.Errorf("issues = %+v, want issue de produto vencido", items[0].Issues)
	}
}

func TestBuildItems_ExpiringSoonWarning(t *testing.T) {
	soon := time.Now().AddDate(0, 0, 10).Format("02/01/2006")
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item Próximo do Vencimento", "5", "L1", soon}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if !hasIssue(items[0].Issues, FieldExpiryDate, IssueExpiringSoon) {
		t.Errorf("issues = %+v, want issue de validade próxima", items[0].Issues)
	}
	// Um item "próximo do vencimento" ainda é válido para importação —
	// não é um campo ausente/formato inválido, então continua Ready.
	if !items[0].Ready {
		t.Errorf("item próximo do vencimento deveria continuar Ready (aviso informativo, não bloqueante)")
	}
}

func TestBuildItems_FarFutureExpiryNoWarning(t *testing.T) {
	future := time.Now().AddDate(2, 0, 0).Format("02/01/2006")
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item Normal", "5", "L1", future}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if hasIssue(items[0].Issues, FieldExpiryDate, IssueExpired) || hasIssue(items[0].Issues, FieldExpiryDate, IssueExpiringSoon) {
		t.Errorf("item com validade distante não deveria ter aviso de vencimento: %+v", items[0].Issues)
	}
	if !items[0].Ready {
		t.Error("item com todos os campos válidos deveria estar Ready")
	}
}

func TestBuildItems_NegativeQuantity(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows:    [][]string{{"Item Negativo", "-5", "L1", "01/01/2028"}},
	}
	mapping, _ := MapColumnsDeterministic(table)
	items := BuildItems(table, mapping, TargetInventoryItem)
	if items[0].Ready {
		t.Error("quantidade negativa não deveria ser Ready")
	}
	if !hasIssue(items[0].Issues, FieldQuantity, IssueInvalidQuantity) {
		t.Errorf("issues = %+v, want issue de quantidade inválida (negativa)", items[0].Issues)
	}
}
