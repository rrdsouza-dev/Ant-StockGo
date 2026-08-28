package importing

import "testing"

func TestMapColumnsDeterministic_SpecExample(t *testing.T) {
	// Exatamente o exemplo do item 2 da especificação.
	table := RawTable{
		Headers: []string{"Produto", "Quantidade", "Lote", "Validade"},
		Rows: [][]string{
			{"Toddy 370g", "25", "TOD-001", "30/06/2027"},
		},
	}
	mapped, ambiguous := MapColumnsDeterministic(table)
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none (cabeçalhos padrão devem resolver sem IA)", ambiguous)
	}
	want := map[FieldKey]int{FieldName: 0, FieldQuantity: 1, FieldLotNumber: 2, FieldExpiryDate: 3}
	if len(mapped) != len(want) {
		t.Fatalf("got %d mappings, want %d: %+v", len(mapped), len(want), mapped)
	}
	for _, m := range mapped {
		if want[m.Field] != m.ColumnIndex {
			t.Errorf("field %s mapped to column %d, want %d", m.Field, m.ColumnIndex, want[m.Field])
		}
	}
}

func TestMapColumnsDeterministic_AlternativeHeaders(t *testing.T) {
	// "Nome do Produto", "Descrição", "Item" etc. citados no item 3 da
	// especificação como equivalentes de "Produto".
	cases := []string{"Nome do Produto", "Descrição", "Item", "Produto/Descrição", "Mercadoria"}
	for _, header := range cases {
		table := RawTable{Headers: []string{header}, Rows: [][]string{{"Algo"}}}
		mapped, ambiguous := MapColumnsDeterministic(table)
		if len(ambiguous) != 0 {
			t.Errorf("header %q ficou ambíguo, esperava mapear para FieldName", header)
			continue
		}
		if len(mapped) != 1 || mapped[0].Field != FieldName {
			t.Errorf("header %q mapeado para %+v, want FieldName", header, mapped)
		}
	}
}

func TestMapColumnsDeterministic_TrulyAmbiguousHeader(t *testing.T) {
	table := RawTable{
		Headers: []string{"Produto", "Fornecedor Preferido"},
		Rows:    [][]string{{"Item A", "Distribuidora XYZ"}},
	}
	mapped, ambiguous := MapColumnsDeterministic(table)
	if len(mapped) != 1 {
		t.Fatalf("mapped = %+v, want 1 (só Produto)", mapped)
	}
	if len(ambiguous) != 1 || ambiguous[0].Header != "Fornecedor Preferido" {
		t.Fatalf("ambiguous = %+v, want 1 item 'Fornecedor Preferido'", ambiguous)
	}
	if len(ambiguous[0].SampleValues) != 1 || ambiguous[0].SampleValues[0] != "Distribuidora XYZ" {
		t.Errorf("sample values = %v, want ['Distribuidora XYZ']", ambiguous[0].SampleValues)
	}
}

func TestMapColumnsDeterministic_EmptyAmbiguousColumnIgnored(t *testing.T) {
	// Coluna sem nenhum dado preenchido não deve virar "ambiguidade" —
	// não há nada a resolver nem custo de IA a evitar.
	table := RawTable{
		Headers: []string{"Produto", "Coluna Vazia"},
		Rows:    [][]string{{"Item A", ""}, {"Item B", ""}},
	}
	_, ambiguous := MapColumnsDeterministic(table)
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none (coluna sem dados não deve gerar ambiguidade)", ambiguous)
	}
}

func TestNormalizeQuantity(t *testing.T) {
	cases := []struct {
		raw    string
		want   int
		wantOk bool
	}{
		{"25", 25, true},
		{" 40 ", 40, true},
		{"1.234", 1234, true}, // separador de milhar (3 dígitos após o ponto)
		{"1,234", 1234, true}, // idem, com vírgula
		{"25 unidades", 25, true},
		{"", 0, false},
		{"abc", 0, false},
		{"-5", -5, true},
	}
	for _, c := range cases {
		got, ok := NormalizeQuantity(c.raw)
		if ok != c.wantOk {
			t.Errorf("NormalizeQuantity(%q) ok = %v, want %v", c.raw, ok, c.wantOk)
			continue
		}
		if ok && got != c.want {
			t.Errorf("NormalizeQuantity(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestNormalizeQuantity_DecimalWithFewerDigitsIsTruncated(t *testing.T) {
	// Com 1 ou 2 dígitos após o separador (não 3), a leitura mais
	// plausível é decimal, não milhar — "25,5" é 25 unidades e meia,
	// truncado para 25 (estoque é sempre inteiro).
	got, ok := NormalizeQuantity("25,5")
	if !ok || got != 25 {
		t.Errorf("NormalizeQuantity(\"25,5\") = (%d, %v), want (25, true)", got, ok)
	}
	got2, ok2 := NormalizeQuantity("3.14")
	if !ok2 || got2 != 3 {
		t.Errorf("NormalizeQuantity(\"3.14\") = (%d, %v), want (3, true)", got2, ok2)
	}
}



func TestNormalizeExpiryDate_BRFormat(t *testing.T) {
	got, ok := NormalizeExpiryDate("30/06/2027")
	if !ok || got != "30/06/2027" {
		t.Errorf("got (%q, %v), want (\"30/06/2027\", true)", got, ok)
	}
}

func TestNormalizeExpiryDate_SingleDigitDayMonth(t *testing.T) {
	got, ok := NormalizeExpiryDate("5/6/2027")
	if !ok || got != "05/06/2027" {
		t.Errorf("got (%q, %v), want (\"05/06/2027\", true)", got, ok)
	}
}

func TestNormalizeExpiryDate_ISOFormat(t *testing.T) {
	got, ok := NormalizeExpiryDate("2027-06-30")
	if !ok || got != "30/06/2027" {
		t.Errorf("got (%q, %v), want (\"30/06/2027\", true)", got, ok)
	}
}

func TestNormalizeExpiryDate_ExcelSerial(t *testing.T) {
	// 46568 confirmado (via openpyxl/Python) como 2027-06-30.
	got, ok := NormalizeExpiryDate("46568")
	if !ok || got != "30/06/2027" {
		t.Errorf("got (%q, %v), want (\"30/06/2027\", true)", got, ok)
	}
}

func TestNormalizeExpiryDate_TwoDigitYear(t *testing.T) {
	got, ok := NormalizeExpiryDate("30/06/27")
	if !ok || got != "30/06/2027" {
		t.Errorf("got (%q, %v), want (\"30/06/2027\", true)", got, ok)
	}
}

func TestNormalizeExpiryDate_Empty(t *testing.T) {
	got, ok := NormalizeExpiryDate("")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestNormalizeExpiryDate_InvalidDateStillFormatted(t *testing.T) {
	// 31/02 não existe, mas a FORMA é reconhecível como data — o
	// normalizer devolve o candidato formatado, e é o validator quem
	// decide reportar "data inválida" (não "ausente").
	got, ok := NormalizeExpiryDate("31/02/2027")
	if !ok {
		t.Fatal("esperava ok=true (forma reconhecida como data, mesmo que inválida)")
	}
	if got != "31/02/2027" {
		t.Errorf("got %q, want \"31/02/2027\"", got)
	}
	if IsValidBRDate(got) {
		t.Errorf("IsValidBRDate(%q) = true, want false (31 de fevereiro não existe)", got)
	}
}

func TestNormalizeExpiryDate_Garbage(t *testing.T) {
	got, ok := NormalizeExpiryDate("não é uma data")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestIsValidBRDate_ReusesSharedValidation(t *testing.T) {
	if !IsValidBRDate("30/06/2027") {
		t.Error("30/06/2027 deveria ser válida")
	}
	if IsValidBRDate("31/02/2027") {
		t.Error("31/02/2027 não deveria ser válida (fevereiro não tem 31 dias)")
	}
	if IsValidBRDate("29/02/2028") == false {
		t.Error("29/02/2028 deveria ser válida (2028 é bissexto)")
	}
	if IsValidBRDate("29/02/2027") {
		t.Error("29/02/2027 não deveria ser válida (2027 não é bissexto)")
	}
}

func TestNormalizeText(t *testing.T) {
	got := NormalizeText("  Toddy   Chocolate  em   Pó   370g  ")
	want := "Toddy Chocolate em Pó 370g"
	if got != want {
		t.Errorf("NormalizeText() = %q, want %q", got, want)
	}
}

func TestRequiredFieldsFor(t *testing.T) {
	invFields := RequiredFieldsFor(TargetInventoryItem)
	wantInv := map[FieldKey]bool{FieldName: true, FieldQuantity: true, FieldLotNumber: true, FieldExpiryDate: true}
	if len(invFields) != len(wantInv) {
		t.Fatalf("inventory required fields = %v, want %v", invFields, wantInv)
	}
	for _, f := range invFields {
		if !wantInv[f] {
			t.Errorf("campo inesperado nos obrigatórios de InventoryItem: %s", f)
		}
	}

	preFields := RequiredFieldsFor(TargetPreProduct)
	wantPre := map[FieldKey]bool{FieldName: true, FieldUnit: true}
	if len(preFields) != len(wantPre) {
		t.Fatalf("pre-product required fields = %v, want name+unit (PreProductService.Create exige unidade de medida)", preFields)
	}
	for _, f := range preFields {
		if !wantPre[f] {
			t.Errorf("campo inesperado nos obrigatórios de PreProduct: %s", f)
		}
	}
}
