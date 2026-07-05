package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

// PendingUserRepository isola todo acesso SQL à tabela `pending_users`.
type PendingUserRepository struct {
	db *sql.DB
}

func NewPendingUserRepository(db *sql.DB) *PendingUserRepository {
	return &PendingUserRepository{db: db}
}

// Create registra uma nova solicitação de conta com status "pending".
func (r *PendingUserRepository) Create(p domain.PendingUser) (domain.PendingUser, error) {
	query := `
		INSERT INTO pending_users (name, email, password_hash, role, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, name, email, password_hash, role, status, requested_at, reviewed_at`
	row := r.db.QueryRow(query, p.Name, p.Email, p.PasswordHash, p.Role)
	return scanPendingUser(row)
}

// EmailExists confirma duplicidade entre solicitações ainda pendentes.
func (r *PendingUserRepository) EmailPending(email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM pending_users WHERE email = $1 AND status = 'pending')`, email,
	).Scan(&exists)
	return exists, err
}

// ListPending retorna todas as solicitações aguardando decisão da gestão.
func (r *PendingUserRepository) ListPending() ([]domain.PendingUser, error) {
	query := `
		SELECT id, name, email, password_hash, role, status, requested_at, reviewed_at
		FROM pending_users WHERE status = 'pending' ORDER BY requested_at ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PendingUser
	for rows.Next() {
		p, err := scanPendingUserRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// FindByID busca uma solicitação específica (usada ao aprovar/rejeitar).
func (r *PendingUserRepository) FindByID(id string) (domain.PendingUser, error) {
	query := `
		SELECT id, name, email, password_hash, role, status, requested_at, reviewed_at
		FROM pending_users WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanPendingUser(row)
}

// UpdateStatus marca a solicitação como aprovada ou rejeitada.
func (r *PendingUserRepository) UpdateStatus(id string, status domain.PendingStatus) error {
	_, err := r.db.Exec(
		`UPDATE pending_users SET status = $1, reviewed_at = now() WHERE id = $2`,
		status, id,
	)
	return err
}

func scanPendingUser(row *sql.Row) (domain.PendingUser, error) {
	var p domain.PendingUser
	err := row.Scan(&p.ID, &p.Name, &p.Email, &p.PasswordHash, &p.Role, &p.Status, &p.RequestedAt, &p.ReviewedAt)
	if err == sql.ErrNoRows {
		return domain.PendingUser{}, ErrNotFound
	}
	return p, err
}

func scanPendingUserRows(rows *sql.Rows) (domain.PendingUser, error) {
	var p domain.PendingUser
	err := rows.Scan(&p.ID, &p.Name, &p.Email, &p.PasswordHash, &p.Role, &p.Status, &p.RequestedAt, &p.ReviewedAt)
	return p, err
}
