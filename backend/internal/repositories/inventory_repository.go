package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"wms-backend/internal/domain"
)

// InventoryRepository isola todo acesso SQL às tabelas `inventory` e
// `stock_movements`. É a única camada que sabe que uma movimentação de
// estoque é, na verdade, duas escritas atômicas (update + insert).
type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

const insertColumns = `id, deposit_id, name, sku, quantity, min_quantity,
		expiry_date, lot_number, category_id, notes, location, active, created_at, updated_at`

func (r *InventoryRepository) Create(item domain.InventoryItem) (domain.InventoryItem, error) {
	query := `
		INSERT INTO inventory (deposit_id, name, sku, quantity, min_quantity,
			expiry_date, lot_number, category_id, notes, location, active)
		VALUES ($1, $2, $3, 0, $4, $5, $6, $7, $8, $9, true)
		RETURNING ` + insertColumns
	row := r.db.QueryRow(query,
		item.DepositID, item.Name, item.SKU, item.MinQuantity,
		item.ExpiryDate, item.LotNumber, item.CategoryID, item.Notes, item.Location,
	)
	return scanInventoryItem(row)
}

// ListByDeposits retorna os itens de inventário pertencentes a um conjunto
// de depósitos, já com a categoria hidratada via LEFT JOIN (é o caminho de
// leitura usado pela tela de Estoque, que precisa exibir o nome da
// categoria, não só o id). Para a gestão (acesso total), o service passa
// todos os ids existentes; para o professor, apenas os depósitos de suas
// turmas. Cast ::uuid[] explícito — ver nota em DepositRepository.ListByIDs.
func (r *InventoryRepository) ListByDeposits(depositIDs []string) ([]domain.InventoryItem, error) {
	if len(depositIDs) == 0 {
		return []domain.InventoryItem{}, nil
	}
	query := `
		SELECT i.id, i.deposit_id, i.name, i.sku, i.quantity, i.min_quantity,
			i.expiry_date, i.lot_number, i.category_id, i.notes, i.location,
			i.active, i.created_at, i.updated_at, c.name
		FROM inventory i
		LEFT JOIN categories c ON c.id = i.category_id
		WHERE i.active = true AND i.deposit_id = ANY($1::uuid[])
		ORDER BY i.name ASC`
	rows, err := r.db.Query(query, pq.Array(depositIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.InventoryItem
	for rows.Next() {
		item, err := scanInventoryItemWithCategoryRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *InventoryRepository) FindByID(id string) (domain.InventoryItem, error) {
	query := `SELECT ` + insertColumns + ` FROM inventory WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanInventoryItem(row)
}

func (r *InventoryRepository) Update(id, name, sku string, minQuantity int, expiryDate *time.Time, lotNumber string, categoryID *string, notes string, location domain.Location) (domain.InventoryItem, error) {
	query := `
		UPDATE inventory SET
			name = $1, sku = $2, min_quantity = $3, expiry_date = $4,
			lot_number = $5, category_id = $6, notes = $7, location = $8,
			updated_at = now()
		WHERE id = $9
		RETURNING ` + insertColumns
	row := r.db.QueryRow(query, name, sku, minQuantity, expiryDate, lotNumber, categoryID, notes, location, id)
	return scanInventoryItem(row)
}

func (r *InventoryRepository) Deactivate(id string) error {
	_, err := r.db.Exec(`UPDATE inventory SET active = false, updated_at = now() WHERE id = $1`, id)
	return err
}

// Move executa, em uma única transação, a atualização da quantidade do
// item e a criação do registro de auditoria em stock_movements. Se a
// movimentação for de saída ("out") e resultar em quantidade negativa,
// a transação é revertida e um erro é retornado — o sistema nunca
// permite estoque negativo.
//
// Fluxo no sistema: chamado exclusivamente por InventoryService.MoveStock,
// nunca diretamente por um handler.
func (r *InventoryRepository) Move(item domain.InventoryItem, movementType domain.MovementType, quantity int, note, createdBy string) (domain.InventoryItem, domain.StockMovement, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}
	defer tx.Rollback()

	delta := quantity
	if movementType == domain.MovementOut {
		delta = -quantity
	}

	// Lock a linha e confirma saldo suficiente ANTES de tentar o UPDATE,
	// para devolver uma mensagem amigável em vez do erro bruto do banco.
	// A constraint `quantity >= 0` na tabela permanece como rede de
	// segurança final contra corridas concorrentes.
	var currentQuantity int
	if err := tx.QueryRow(`SELECT quantity FROM inventory WHERE id = $1 FOR UPDATE`, item.ID).Scan(&currentQuantity); err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}
	if currentQuantity+delta < 0 {
		return domain.InventoryItem{}, domain.StockMovement{}, fmt.Errorf("quantidade insuficiente em estoque para esta saída (disponível: %d)", currentQuantity)
	}

	row := tx.QueryRow(`
		UPDATE inventory SET quantity = quantity + $1, updated_at = now()
		WHERE id = $2
		RETURNING `+insertColumns,
		delta, item.ID,
	)
	updated, err := scanInventoryItem(row)
	if err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}

	var movement domain.StockMovement
	row = tx.QueryRow(`
		INSERT INTO stock_movements (inventory_item_id, deposit_id, type, quantity, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, inventory_item_id, deposit_id, type, quantity, note, created_by, created_at`,
		item.ID, item.DepositID, movementType, quantity, note, createdBy,
	)
	if err := row.Scan(&movement.ID, &movement.InventoryItemID, &movement.DepositID, &movement.Type,
		&movement.Quantity, &movement.Note, &movement.CreatedBy, &movement.CreatedAt); err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}

	return updated, movement, nil
}

// ListMovements retorna o histórico de movimentações dos depósitos informados,
// mais recentes primeiro. Usado pelas páginas de Relatórios e Exportações.
// Cast ::uuid[] explícito — ver nota em DepositRepository.ListByIDs.
func (r *InventoryRepository) ListMovements(depositIDs []string, limit int) ([]domain.StockMovement, error) {
	if len(depositIDs) == 0 {
		return []domain.StockMovement{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `
		SELECT id, inventory_item_id, deposit_id, type, quantity, note, created_by, created_at
		FROM stock_movements WHERE deposit_id = ANY($1::uuid[])
		ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.Query(query, pq.Array(depositIDs), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.StockMovement
	for rows.Next() {
		var m domain.StockMovement
		if err := rows.Scan(&m.ID, &m.InventoryItemID, &m.DepositID, &m.Type, &m.Quantity, &m.Note, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// scanInventoryItem/scanInventoryItemRows leem as colunas "cruas" da
// tabela inventory (sem hidratar o nome da categoria — usado pelos
// caminhos de escrita e pelo FindByID interno, onde só o category_id
// importa). scanInventoryItemWithCategoryRows é usado nos caminhos de
// LEITURA para exibição (ListByDeposits), que fazem o LEFT JOIN e
// preenchem item.Category com o nome real.

func scanInventoryItem(row *sql.Row) (domain.InventoryItem, error) {
	var i domain.InventoryItem
	var expiry sql.NullTime
	var categoryID sql.NullString
	err := row.Scan(&i.ID, &i.DepositID, &i.Name, &i.SKU, &i.Quantity, &i.MinQuantity,
		&expiry, &i.LotNumber, &categoryID, &i.Notes, &i.Location,
		&i.Active, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.InventoryItem{}, ErrNotFound
	}
	if err != nil {
		return domain.InventoryItem{}, err
	}
	applyNullableItemFields(&i, expiry, categoryID)
	return i, nil
}

func scanInventoryItemWithCategoryRows(rows *sql.Rows) (domain.InventoryItem, error) {
	var i domain.InventoryItem
	var expiry sql.NullTime
	var categoryID sql.NullString
	var categoryName sql.NullString
	err := rows.Scan(&i.ID, &i.DepositID, &i.Name, &i.SKU, &i.Quantity, &i.MinQuantity,
		&expiry, &i.LotNumber, &categoryID, &i.Notes, &i.Location,
		&i.Active, &i.CreatedAt, &i.UpdatedAt, &categoryName)
	if err != nil {
		return domain.InventoryItem{}, err
	}
	applyNullableItemFields(&i, expiry, categoryID)
	if categoryID.Valid && categoryName.Valid {
		i.Category = &domain.Category{ID: categoryID.String, Name: categoryName.String}
	}
	return i, nil
}

func applyNullableItemFields(i *domain.InventoryItem, expiry sql.NullTime, categoryID sql.NullString) {
	if expiry.Valid {
		t := expiry.Time
		i.ExpiryDate = &t
	}
	if categoryID.Valid {
		id := categoryID.String
		i.CategoryID = &id
	}
}
