package importing

import "testing"

func TestApplyAIResolution_ResolvesAmbiguousHeader(t *testing.T) {
	mapped := []ColumnMapping{
		{ColumnIndex: 0, Header: "Produto", Field: FieldName, Source: "deterministic"},
	}
	ambiguous := []AmbiguousHeader{
		{ColumnIndex: 1, Header: "Marca do Item"},
	}
	resolved := map[string]FieldKey{"Marca do Item": FieldBrand}

	newMapped, stillAmbiguous := applyAIResolution(mapped, ambiguous, resolved)

	if len(stillAmbiguous) != 0 {
		t.Fatalf("stillAmbiguous = %v, want none", stillAmbiguous)
	}
	if len(newMapped) != 2 {
		t.Fatalf("newMapped = %+v, want 2 entries", newMapped)
	}
	found := false
	for _, m := range newMapped {
		if m.Header == "Marca do Item" {
			found = true
			if m.Field != FieldBrand || m.Source != "ai" {
				t.Errorf("mapping da IA = %+v, want Field=brand Source=ai", m)
			}
		}
	}
	if !found {
		t.Error("mapeamento de 'Marca do Item' não foi incorporado")
	}
}

func TestApplyAIResolution_DoesNotOverrideDeterministicMapping(t *testing.T) {
	// Cenário adversarial: a IA sugere um campo que uma coluna
	// determinística já ocupa. A decisão determinística deve prevalecer
	// — a coluna ambígua permanece não mapeada em vez de sobrescrever
	// silenciosamente.
	mapped := []ColumnMapping{
		{ColumnIndex: 0, Header: "Produto", Field: FieldName, Source: "deterministic"},
	}
	ambiguous := []AmbiguousHeader{
		{ColumnIndex: 1, Header: "Item Description"},
	}
	resolved := map[string]FieldKey{"Item Description": FieldName} // conflita com a coluna 0

	newMapped, stillAmbiguous := applyAIResolution(mapped, ambiguous, resolved)

	if len(stillAmbiguous) != 1 || stillAmbiguous[0].Header != "Item Description" {
		t.Fatalf("stillAmbiguous = %+v, want 'Item Description' permanecer ambíguo (conflito com mapeamento determinístico)", stillAmbiguous)
	}
	if len(newMapped) != 1 {
		t.Fatalf("newMapped = %+v, want continuar com só o mapeamento determinístico original", newMapped)
	}
}

func TestApplyAIResolution_UnresolvedHeaderStaysAmbiguous(t *testing.T) {
	mapped := []ColumnMapping{}
	ambiguous := []AmbiguousHeader{
		{ColumnIndex: 0, Header: "Fornecedor Preferido"},
	}
	resolved := map[string]FieldKey{} // IA não conseguiu resolver nada

	newMapped, stillAmbiguous := applyAIResolution(mapped, ambiguous, resolved)
	if len(newMapped) != 0 {
		t.Errorf("newMapped = %+v, want vazio", newMapped)
	}
	if len(stillAmbiguous) != 1 {
		t.Errorf("stillAmbiguous = %+v, want 1 (cabeçalho continua sem mapear)", stillAmbiguous)
	}
}

func TestSummarize(t *testing.T) {
	items := []ImportItem{
		{Quantity: 25, Ready: true},
		{Quantity: 40, Ready: true},
		{Quantity: 30, Ready: false},
		{Quantity: 50, Ready: false},
	}
	s := summarize(items)
	if s.TotalItems != 4 {
		t.Errorf("TotalItems = %d, want 4", s.TotalItems)
	}
	if s.TotalUnits != 145 {
		t.Errorf("TotalUnits = %d, want 145", s.TotalUnits)
	}
	if s.ReadyCount != 2 {
		t.Errorf("ReadyCount = %d, want 2", s.ReadyCount)
	}
	if s.AttentionCount != 2 {
		t.Errorf("AttentionCount = %d, want 2", s.AttentionCount)
	}
}

func TestSummarize_SpecExampleNumbers(t *testing.T) {
	// Item 15 da especificação usa como exemplo: 32 produtos, 780
	// unidades, 28 prontos, 4 precisam de atenção. Testamos a fórmula
	// com números equivalentes, para garantir que a soma bate.
	items := make([]ImportItem, 0, 32)
	for i := 0; i < 28; i++ {
		items = append(items, ImportItem{Quantity: 20, Ready: true}) // 28*20 = 560
	}
	for i := 0; i < 4; i++ {
		items = append(items, ImportItem{Quantity: 55, Ready: false}) // 4*55 = 220
	}
	// total = 560 + 220 = 780, batendo com o exemplo da especificação
	s := summarize(items)
	if s.TotalItems != 32 || s.ReadyCount != 28 || s.AttentionCount != 4 || s.TotalUnits != 780 {
		t.Errorf("s = %+v, want {32, 780, 28, 4}", s)
	}
}

func TestResolveCategoryIDs_ExactMatch(t *testing.T) {
	items := []ImportItem{
		{Name: "Item A", CategoryRaw: "Alimentos"},
		{Name: "Item B", CategoryRaw: "alimentos"}, // case-insensitive
		{Name: "Item C", CategoryRaw: "Higiene"},   // sem correspondência
		{Name: "Item D", CategoryRaw: ""},          // sem categoria informada
	}
	lookup := map[string]string{
		"alimentos": "cat-001",
	}
	resolved := resolveCategoryIDs(items, lookup)

	if resolved[0].CategoryID == nil || *resolved[0].CategoryID != "cat-001" {
		t.Errorf("item A: CategoryID = %v, want cat-001", resolved[0].CategoryID)
	}
	if resolved[1].CategoryID == nil || *resolved[1].CategoryID != "cat-001" {
		t.Errorf("item B (case diferente): CategoryID = %v, want cat-001", resolved[1].CategoryID)
	}
	if resolved[2].CategoryID != nil {
		t.Errorf("item C (sem correspondência): CategoryID = %v, want nil", resolved[2].CategoryID)
	}
	if resolved[3].CategoryID != nil {
		t.Errorf("item D (sem categoria informada): CategoryID = %v, want nil", resolved[3].CategoryID)
	}
}

func TestResolveCategoryIDs_AccentInsensitive(t *testing.T) {
	items := []ImportItem{{Name: "Item", CategoryRaw: "Higiene Pessoal"}}
	lookup := map[string]string{
		normalizeForComparison("Higiêne Péssoal"): "cat-002", // grafia com acento diferente no cadastro
	}
	resolved := resolveCategoryIDs(items, lookup)
	if resolved[0].CategoryID == nil || *resolved[0].CategoryID != "cat-002" {
		t.Errorf("CategoryID = %v, want cat-002 (comparação deve ignorar acentuação)", resolved[0].CategoryID)
	}
}
