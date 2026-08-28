// Package importing implementa a Importação Inteligente: recebe uma
// planilha (XLSX/CSV), interpreta suas colunas e linhas de forma
// determinística, usa IA apenas para resolver ambiguidades de cabeçalho,
// e produz uma prévia que o usuário revisa antes de qualquer gravação
// real no banco (sempre através dos Services já existentes de
// InventoryItem/PreProduct — este pacote nunca fala com o banco).
package importing

// TargetKind define para qual entidade do sistema uma linha da planilha
// será convertida na hora da confirmação (ver item 13 da especificação:
// "Produto ou Pré-produto"). "Produto", no vocabulário desta
// funcionalidade, significa item de estoque real (domain.InventoryItem)
// — não existe uma entidade "Produto" separada no sistema.
type TargetKind string

const (
	TargetInventoryItem TargetKind = "inventory_item"
	TargetPreProduct    TargetKind = "pre_product"
)

// FieldKey identifica um campo lógico reconhecido pelo importador,
// independente de como a coluna foi nomeada na planilha original.
type FieldKey string

const (
	FieldName        FieldKey = "name"
	FieldQuantity    FieldKey = "quantity"
	FieldLotNumber   FieldKey = "lot_number"
	FieldExpiryDate  FieldKey = "expiry_date"
	FieldSKU         FieldKey = "sku"
	FieldBrand       FieldKey = "brand"
	FieldMinQuantity FieldKey = "min_quantity"
	FieldCategory    FieldKey = "category"
	FieldNotes       FieldKey = "notes"
	FieldUnit        FieldKey = "unit" // usado só quando target = pre_product
)

// RawTable é o resultado cru do parser: cabeçalhos e linhas de célula
// como texto, antes de qualquer normalização ou interpretação.
type RawTable struct {
	Headers []string
	Rows    [][]string
}

// ColumnMapping associa uma coluna da planilha (pelo índice) a um campo
// lógico do sistema. Source indica se a associação veio de correspondência
// determinística (nome de coluna já conhecido) ou de sugestão da IA —
// informação usada só para depuração/log, nunca muda o comportamento.
type ColumnMapping struct {
	ColumnIndex int
	Header      string
	Field       FieldKey
	Source      string // "deterministic" | "ai"
}

// FieldIssueCode identifica o tipo de problema encontrado em um campo
// específico de um item durante a validação. Usado pelo frontend para
// decidir ícone/cor/mensagem sem precisar interpretar texto livre.
type FieldIssueCode string

const (
	IssueMissing         FieldIssueCode = "missing"          // campo obrigatório ausente
	IssueInvalidFormat   FieldIssueCode = "invalid_format"    // presente mas em formato inválido
	IssueInvalidQuantity FieldIssueCode = "invalid_quantity"  // quantidade <= 0 ou não numérica
	IssueExpired         FieldIssueCode = "expired"           // validade já vencida
	IssueExpiringSoon    FieldIssueCode = "expiring_soon"     // validade nos próximos 30 dias
	IssuePossibleDup     FieldIssueCode = "possible_duplicate"
)

// FieldIssue descreve um problema específico de um campo de um item.
type FieldIssue struct {
	Field   FieldKey       `json:"field"`
	Code    FieldIssueCode `json:"code"`
	Message string         `json:"message"`
}

// ImportItem é uma linha da planilha já normalizada e validada,
// pronta para ser revisada pelo usuário e, se aprovada, convertida em
// InventoryItem ou PreProduct real na hora do commit.
type ImportItem struct {
	// RowIndex é 1-based e conta a partir da primeira linha de DADOS
	// (não a linha de cabeçalho) — é o número que faz sentido mostrar
	// ao usuário ("linha 3 da planilha"), já usado nas mensagens de erro.
	RowIndex int `json:"row_index"`

	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	LotNumber   string  `json:"lot_number"`
	ExpiryDate  string  `json:"expiry_date"` // DD/MM/AAAA, já normalizado — vazio se ausente/inválido
	SKU         string  `json:"sku"`
	Brand       string  `json:"brand"`
	MinQuantity int     `json:"min_quantity"`
	CategoryID  *string `json:"category_id,omitempty"` // resolvido por nome contra categorias existentes, se houver correspondência exata
	CategoryRaw string  `json:"category_raw,omitempty"`
	Notes       string  `json:"notes"`
	Unit        string  `json:"unit,omitempty"`

	// Ready indica se o item pode ser importado como está: nenhum campo
	// obrigatório ausente/inválido para o TargetKind escolhido. Itens não
	// "ready" ainda podem ser importados individualmente após edição, ou
	// simplesmente ficam de fora de "importar todos os válidos".
	Ready  bool         `json:"ready"`
	Issues []FieldIssue `json:"issues,omitempty"`

	// DuplicateOfRowIndexes lista outras linhas desta mesma planilha que
	// parecem representar o mesmo produto (nome muito similar). Apenas
	// informativo — nunca remove itens automaticamente (item 12 da
	// especificação).
	DuplicateOfRowIndexes []int `json:"duplicate_of_row_indexes,omitempty"`
}

// PreviewSummary agrega números da prévia inteira, usado pela tela de
// confirmação (item 15 da especificação).
type PreviewSummary struct {
	TotalItems    int `json:"total_items"`
	TotalUnits    int `json:"total_units"`
	ReadyCount    int `json:"ready_count"`
	AttentionCount int `json:"attention_count"`
}

// AmbiguousHeader é um cabeçalho de coluna que o normalizador
// determinístico não conseguiu associar a nenhum campo conhecido —
// candidato a ser resolvido pela IA (ver item 3 da especificação).
type AmbiguousHeader struct {
	ColumnIndex int      `json:"column_index"`
	Header      string   `json:"header"`
	SampleValues []string `json:"sample_values,omitempty"`
}

// Preview é o resultado completo devolvido por POST /imports/preview.
// Nada aqui foi persistido — é só a interpretação da planilha para o
// usuário revisar (ver item 8 da especificação: "Prévia dos dados").
type Preview struct {
	Items         []ImportItem    `json:"items"`
	Summary       PreviewSummary  `json:"summary"`
	ColumnMapping []ColumnMapping `json:"column_mapping"`
	// UnmappedHeaders são cabeçalhos que nem o normalizador nem a IA
	// conseguiram associar a um campo conhecido — ficam de fora da
	// interpretação, mas são reportados para transparência.
	UnmappedHeaders []string `json:"unmapped_headers,omitempty"`
	// AIUsed indica se a IA foi de fato chamada para resolver ambiguidade
	// de cabeçalhos nesta planilha (ver item 19/29: só usar IA quando
	// necessário). Puramente informativo para o frontend/logs.
	AIUsed bool `json:"ai_used"`
}

// CommitItemInput é a versão de um ImportItem já revisada/editada pelo
// usuário no frontend, enviada de volta ao backend em POST
// /imports/commit. Note que o backend NUNCA confia nos valores vindos
// daqui sem revalidar — são passados pelos mesmos Services/validações
// que qualquer cadastro manual usaria.
type CommitItemInput struct {
	RowIndex    int     `json:"row_index"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	LotNumber   string  `json:"lot_number"`
	ExpiryDate  string  `json:"expiry_date"`
	SKU         string  `json:"sku"`
	Brand       string  `json:"brand"`
	MinQuantity int     `json:"min_quantity"`
	CategoryID  *string `json:"category_id,omitempty"`
	Notes       string  `json:"notes"`
	Unit        string  `json:"unit,omitempty"`
}

// CommitRequest é o corpo de POST /imports/commit.
type CommitRequest struct {
	DepositID string            `json:"deposit_id"`
	Target    TargetKind        `json:"target"`
	Items     []CommitItemInput `json:"items"`
}

// ItemResult é o resultado da tentativa de importar UM item durante o
// commit (item 26 da especificação: "não deixar o sistema em estado
// inconsistente" + "o usuário precisa saber quais falharam").
type ItemResult struct {
	RowIndex int    `json:"row_index"`
	Name     string `json:"name"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	ItemID   string `json:"item_id,omitempty"`
}

// CommitResult é o resultado completo devolvido por POST /imports/commit.
type CommitResult struct {
	TotalAnalyzed int          `json:"total_analyzed"`
	TotalImported int          `json:"total_imported"`
	TotalFailed   int          `json:"total_failed"`
	Results       []ItemResult `json:"results"`
}
