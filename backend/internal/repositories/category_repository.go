package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

// CategoryRepository isola todo acesso SQL à tabela `categories`.
type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create insere uma nova categoria. Chamado pelo botão "+" no formulário
// de item de estoque.
func (r *CategoryRepository) Create(name string) (domain.Category, error) {
	query := `
		INSERT INTO categories (name)
		VALUES ($1)
		RETURNING id, name, created_at`
	row := r.db.QueryRow(query, name)
	return scanCategory(row)
}

// List retorna todas as categorias, em ordem alfabética (usado para
// popular o <select> de categoria no formulário de item).
func (r *CategoryRepository) List() ([]domain.Category, error) {
	query := `SELECT id, name, created_at FROM categories ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// NameExists confirma duplicidade de nome (case-sensitive; a coluna já
// tem UNIQUE no banco como rede de segurança final).
func (r *CategoryRepository) NameExists(name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM categories WHERE name = $1)`, name).Scan(&exists)
	return exists, err
}

func scanCategory(row *sql.Row) (domain.Category, error) {
	var c domain.Category
	err := row.Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.Category{}, ErrNotFound
	}
	return c, err
}
