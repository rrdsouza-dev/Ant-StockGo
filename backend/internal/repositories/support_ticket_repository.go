package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

// SupportTicketRepository isola todo acesso SQL à tabela `support_tickets`.
type SupportTicketRepository struct {
	db *sql.DB
}

func NewSupportTicketRepository(db *sql.DB) *SupportTicketRepository {
	return &SupportTicketRepository{db: db}
}

func (r *SupportTicketRepository) Create(t domain.SupportTicket) (domain.SupportTicket, error) {
	query := `
		INSERT INTO support_tickets (professor_id, nome, email, categoria, tipo, descricao)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, professor_id, nome, email, categoria, tipo, descricao, created_at`
	row := r.db.QueryRow(query, t.ProfessorID, t.Nome, t.Email, t.Categoria, t.Tipo, t.Descricao)

	var created domain.SupportTicket
	err := row.Scan(&created.ID, &created.ProfessorID, &created.Nome, &created.Email,
		&created.Categoria, &created.Tipo, &created.Descricao, &created.CreatedAt)
	return created, err
}

// List retorna todos os chamados, mais recentes primeiro (painel da gestão).
func (r *SupportTicketRepository) List() ([]domain.SupportTicket, error) {
	query := `
		SELECT id, professor_id, nome, email, categoria, tipo, descricao, created_at
		FROM support_tickets ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SupportTicket
	for rows.Next() {
		var t domain.SupportTicket
		if err := rows.Scan(&t.ID, &t.ProfessorID, &t.Nome, &t.Email, &t.Categoria, &t.Tipo, &t.Descricao, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// DeleteAll apaga todo o histórico de chamados (somente gestão, com
// código administrativo validado antes de chegar aqui — ver SupportService).
func (r *SupportTicketRepository) DeleteAll() error {
	_, err := r.db.Exec(`DELETE FROM support_tickets`)
	return err
}
