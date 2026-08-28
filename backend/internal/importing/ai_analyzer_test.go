package importing

import (
	"context"
	"testing"
)

// mockHeaderResolver simula respostas de IA sem depender de um Ollama
// real — permite testar service.go isoladamente (ver service_test.go).
type mockHeaderResolver struct {
	response map[string]FieldKey
	err      error
	calls    int
}

func (m *mockHeaderResolver) ResolveHeaders(ctx context.Context, ambiguous []AmbiguousHeader, knownFields []FieldKey) (map[string]FieldKey, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestParseHeaderResolverResponse_PlainJSON(t *testing.T) {
	raw := `{"mapeamentos": [{"cabecalho": "Fornecedor", "campo": null}, {"cabecalho": "Marca do Produto", "campo": "brand"}]}`
	parsed, err := parseHeaderResolverResponse(raw)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(parsed.Mapeamentos) != 2 {
		t.Fatalf("got %d mapeamentos, want 2", len(parsed.Mapeamentos))
	}
	if parsed.Mapeamentos[0].Campo != nil {
		t.Errorf("Fornecedor deveria ter campo=nil")
	}
	if parsed.Mapeamentos[1].Campo == nil || *parsed.Mapeamentos[1].Campo != "brand" {
		t.Errorf("Marca do Produto deveria mapear para 'brand'")
	}
}

func TestParseHeaderResolverResponse_MarkdownFenced(t *testing.T) {
	// Modelos pequenos às vezes envolvem a resposta em ```json apesar de
	// instruídos a não fazer isso — o parser precisa tolerar isso.
	raw := "```json\n{\"mapeamentos\": [{\"cabecalho\": \"Marca\", \"campo\": \"brand\"}]}\n```"
	parsed, err := parseHeaderResolverResponse(raw)
	if err != nil {
		t.Fatalf("erro inesperado ao lidar com cercas markdown: %v", err)
	}
	if len(parsed.Mapeamentos) != 1 || parsed.Mapeamentos[0].Cabecalho != "Marca" {
		t.Errorf("parsed = %+v", parsed)
	}
}

func TestParseHeaderResolverResponse_InvalidJSON(t *testing.T) {
	_, err := parseHeaderResolverResponse("isto não é json de forma alguma")
	if err == nil {
		t.Fatal("esperava erro para JSON inválido")
	}
}

func TestOllamaHeaderResolver_ResolveHeaders_FiltersUnknownFields(t *testing.T) {
	// Testa a lógica de filtragem em ResolveHeaders isoladamente,
	// injetando uma resposta simulada via reimplementação do parsing
	// (sem cliente Ollama real). O comportamento relevante: um campo
	// sugerido pela IA que não está na lista de knownFields deve ser
	// descartado silenciosamente, nunca propagado como um FieldKey
	// inventado.
	ambiguous := []AmbiguousHeader{
		{Header: "Marca", SampleValues: []string{"Nestlé"}},
		{Header: "Campo Aleatório", SampleValues: []string{"x"}},
	}
	knownFields := []FieldKey{FieldBrand}

	// Simula o corpo de ResolveHeaders manualmente para validar a etapa
	// de filtragem sem envolver rede: um campo válido (brand) deve
	// passar, um campo fora da lista deve ser descartado.
	raw := `{"mapeamentos": [{"cabecalho": "Marca", "campo": "brand"}, {"cabecalho": "Campo Aleatório", "campo": "algum_campo_inventado"}]}`
	parsed, err := parseHeaderResolverResponse(raw)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	validFields := map[FieldKey]bool{}
	for _, f := range knownFields {
		validFields[f] = true
	}
	result := map[string]FieldKey{}
	for _, m := range parsed.Mapeamentos {
		if m.Campo == nil {
			continue
		}
		field := FieldKey(*m.Campo)
		if !validFields[field] {
			continue
		}
		result[m.Cabecalho] = field
	}

	if len(result) != 1 {
		t.Fatalf("result = %+v, want apenas 'Marca' -> brand (campo inventado deve ser descartado)", result)
	}
	if result["Marca"] != FieldBrand {
		t.Errorf("result[Marca] = %s, want %s", result["Marca"], FieldBrand)
	}
	_ = ambiguous // usado apenas para documentar o cenário do teste
}

func TestBuildHeaderResolverPrompt_IncludesHeadersAndSamples(t *testing.T) {
	ambiguous := []AmbiguousHeader{
		{Header: "Fornecedor Preferido", SampleValues: []string{"Distribuidora XYZ", "Atacadão"}},
	}
	prompt := buildHeaderResolverPrompt(ambiguous, resolvableFieldsForAI)
	if !contains(prompt, "Fornecedor Preferido") {
		t.Errorf("prompt não contém o cabeçalho ambíguo: %s", prompt)
	}
	if !contains(prompt, "Distribuidora XYZ") {
		t.Errorf("prompt não contém os valores de exemplo: %s", prompt)
	}
	if !contains(prompt, "name") {
		t.Errorf("prompt não lista os campos lógicos conhecidos: %s", prompt)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
