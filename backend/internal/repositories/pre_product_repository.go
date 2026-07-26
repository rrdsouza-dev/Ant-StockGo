package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

const preProductColumns = `pp.id, pp.name, pp.category_id, pp.brand, pp.unit, pp.notes, pp.barcode,
		pp.active, pp.created_by, pp.created_at, pp.updated_at`

// PreProductRepository isola todo acesso SQL à tabela `pre_products`.
type PreProductRepository struct {
	db *sql.DB
}

func NewPreProductRepository(db *sql.DB) *PreProductRepository {
	return &PreProductRepository{db: db}
}

func (r *PreProductRepository) Create(p domain.PreProduct) (domain.PreProduct, error) {
	query := `
		INSERT INTO pre_products (name, category_id, brand, unit, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, category_id, brand, unit, notes, barcode, active, created_by, created_at, updated_at`
	row := r.db.QueryRow(query, p.Name, p.CategoryID, p.Brand, p.Unit, p.Notes, p.CreatedBy)
	return scanPreProduct(row)
}

// List retorna todos os Pré-Produtos ativos, com a categoria já
// hidratada via LEFT JOIN — o catálogo é exibido com o nome da
// categoria, não só o id.
func (r *PreProductRepository) List() ([]domain.PreProduct, error) {
	query := `
		SELECT ` + preProductColumns + `, c.name
		FROM pre_products pp
		LEFT JOIN categories c ON c.id = pp.category_id
		WHERE pp.active = true
		ORDER BY pp.name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PreProduct
	for rows.Next() {
		p, err := scanPreProductWithCategoryRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PreProductRepository) FindByID(id string) (domain.PreProduct, error) {
	query := `SELECT id, name, category_id, brand, unit, notes, barcode, active, created_by, created_at, updated_at
		FROM pre_products WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanPreProduct(row)
}

// FindByBarcode busca o Pré-Produto associado a um código de barras —
// usado pelo fluxo de leitura do scanner (ver item 6 da especificação):
// quando o código não corresponde a nenhum item de estoque, o sistema
// procura aqui antes de oferecer a associação manual.
func (r *PreProductRepository) FindByBarcode(barcode string) (domain.PreProduct, error) {
	query := `SELECT id, name, category_id, brand, unit, notes, barcode, active, created_by, created_at, updated_at
		FROM pre_products WHERE active = true AND barcode = $1`
	row := r.db.QueryRow(query, barcode)
	return scanPreProduct(row)
}

func (r *PreProductRepository) Update(id, name string, categoryID *string, brand, unit, notes string) (domain.PreProduct, error) {
	query := `
		UPDATE pre_products SET name = $1, category_id = $2, brand = $3, unit = $4, notes = $5, updated_at = now()
		WHERE id = $6
		RETURNING id, name, category_id, brand, unit, notes, barcode, active, created_by, created_at, updated_at`
	row := r.db.QueryRow(query, name, categoryID, brand, unit, notes, id)
	return scanPreProduct(row)
}

// SetBarcode associa (ou remove, se barcode == "") um código de barras a
// este Pré-Produto. Separado de Update para deixar explícito que essa é
// uma operação do fluxo de scanner, não uma edição de cadastro comum.
func (r *PreProductRepository) SetBarcode(id, barcode string) (domain.PreProduct, error) {
	var barcodeValue any
	if barcode != "" {
		barcodeValue = barcode
	}
	query := `
		UPDATE pre_products SET barcode = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, name, category_id, brand, unit, notes, barcode, active, created_by, created_at, updated_at`
	row := r.db.QueryRow(query, barcodeValue, id)
	return scanPreProduct(row)
}

func (r *PreProductRepository) Deactivate(id string) error {
	_, err := r.db.Exec(`UPDATE pre_products SET active = false, updated_at = now() WHERE id = $1`, id)
	return err
}

func scanPreProduct(row *sql.Row) (domain.PreProduct, error) {
	var p domain.PreProduct
	var categoryID sql.NullString
	var barcode sql.NullString
	err := row.Scan(&p.ID, &p.Name, &categoryID, &p.Brand, &p.Unit, &p.Notes, &barcode,
		&p.Active, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.PreProduct{}, ErrNotFound
	}
	if err != nil {
		return domain.PreProduct{}, err
	}
	applyNullablePreProductFields(&p, categoryID, barcode)
	return p, nil
}

func scanPreProductWithCategoryRows(rows *sql.Rows) (domain.PreProduct, error) {
	var p domain.PreProduct
	var categoryID sql.NullString
	var barcode sql.NullString
	var categoryName sql.NullString
	err := rows.Scan(&p.ID, &p.Name, &categoryID, &p.Brand, &p.Unit, &p.Notes, &barcode,
		&p.Active, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &categoryName)
	if err != nil {
		return domain.PreProduct{}, err
	}
	applyNullablePreProductFields(&p, categoryID, barcode)
	if categoryID.Valid && categoryName.Valid {
		p.Category = &domain.Category{ID: categoryID.String, Name: categoryName.String}
	}
	return p, nil
}

func applyNullablePreProductFields(p *domain.PreProduct, categoryID, barcode sql.NullString) {
	if categoryID.Valid {
		id := categoryID.String
		p.CategoryID = &id
	}
	if barcode.Valid {
		p.Barcode = barcode.String
	}
}
