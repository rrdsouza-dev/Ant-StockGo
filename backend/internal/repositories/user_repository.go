package repositories

import (
	"database/sql"
	"errors"

	"wms-backend/internal/domain"
)

var ErrNotFound = errors.New("registro não encontrado")

const userColumns = `id, name, email, password_hash, role, support_code, active, created_at, updated_at`

// UserRepository isola todo acesso SQL à tabela `users`.
// Nenhuma outra camada do sistema executa SQL diretamente sobre esta tabela.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create insere um usuário já aprovado (chamado apenas pelo fluxo de
// aprovação). SupportCode é gerado pelo AuthService antes de chamar aqui.
func (r *UserRepository) Create(u domain.User) (domain.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash, role, support_code, active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING ` + userColumns
	row := r.db.QueryRow(query, u.Name, u.Email, u.PasswordHash, u.Role, u.SupportCode)
	return scanUser(row)
}

// FindByEmail busca um usuário ativo pelo e-mail (usado no login).
func (r *UserRepository) FindByEmail(email string) (domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	row := r.db.QueryRow(query, email)
	return scanUser(row)
}

// FindByID busca um usuário pelo id (usado por /users/me e middlewares).
func (r *UserRepository) FindByID(id string) (domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanUser(row)
}

// EmailExists confirma duplicidade de e-mail entre contas já ativas.
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

// List retorna todos os usuários ativos do sistema (usado no painel de gestão).
func (r *UserRepository) List() ([]domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// ClassIDsForProfessor retorna as turmas às quais o professor está vinculado.
func (r *UserRepository) ClassIDsForProfessor(userID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT class_id FROM class_teachers WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanUser(row *sql.Row) (domain.User, error) {
	var u domain.User
	var supportCode sql.NullString
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &supportCode, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	u.SupportCode = supportCode.String
	return u, nil
}

func scanUserRows(rows *sql.Rows) (domain.User, error) {
	var u domain.User
	var supportCode sql.NullString
	err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &supportCode, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return domain.User{}, err
	}
	u.SupportCode = supportCode.String
	return u, nil
}
