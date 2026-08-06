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

// Delete remove permanentemente a categoria do banco. Só é chamado pelo
// service depois de confirmar (via CountUsage) que nenhum item de
// estoque ou Pré-Produto ainda a referencia — na prática o
// `ON DELETE SET NULL` das migrações 002/004 nunca chega a disparar por
// este caminho, mas permanece como rede de segurança.
func (r *CategoryRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM categories WHERE id = $1`, id)
	return err
}

// CountUsage retorna quantos itens de estoque e Pré-Produtos ainda
// referenciam esta categoria. Usado pelo service para bloquear a
// exclusão com uma mensagem clara em vez de deixar a categoria sumir
// silenciosamente dos registros que a usavam (ver complemento de
// exclusão de categorias — item 2).
func (r *CategoryRepository) CountUsage(id string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM inventory WHERE category_id = $1) +
			(SELECT COUNT(*) FROM pre_products WHERE category_id = $1)
	`, id).Scan(&count)
	return count, err
}

func scanCategory(row *sql.Row) (domain.Category, error) {
	var c domain.Category
	err := row.Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.Category{}, ErrNotFound
	}
	return c, err
}
