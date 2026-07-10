package domain

import "time"

// SupportCategory e SupportType restringem os valores aceitos no
// formulário de suporte às opções definidas na especificação.
type SupportCategory string
type SupportIssueType string

const (
	SupportCategoryEstoque SupportCategory = "estoque"
	SupportCategorySistema SupportCategory = "sistema"

	SupportTypeOperacional SupportIssueType = "erro_operacional"
	SupportTypeSistematico SupportIssueType = "erro_sistematico"
)

func (c SupportCategory) IsValid() bool {
	return c == SupportCategoryEstoque || c == SupportCategorySistema
}

func (t SupportIssueType) IsValid() bool {
	return t == SupportTypeOperacional || t == SupportTypeSistematico
}

// SupportTicket é uma reclamação/chamado de suporte aberto por um
// professor. Nome e e-mail são gravados no momento do envio (não via
// JOIN com users) para que o histórico permaneça correto mesmo que a
// conta do professor mude de nome ou seja desativada depois.
type SupportTicket struct {
	ID          string           `json:"id"`
	ProfessorID string           `json:"professor_id"`
	Nome        string           `json:"nome"`
	Email       string           `json:"email"`
	Categoria   SupportCategory  `json:"categoria"`
	Tipo        SupportIssueType `json:"tipo"`
	Descricao   string           `json:"descricao"`
	CreatedAt   time.Time        `json:"created_at"`
}
