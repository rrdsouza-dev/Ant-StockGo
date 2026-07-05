package repositories

import (
	"database/sql"
	"fmt"

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

func (r *InventoryRepository) Create(item domain.InventoryItem) (domain.InventoryItem, error) {
	query := `
		INSERT INTO inventory (deposit_id, name, sku, quantity, min_quantity, active)
		VALUES ($1, $2, $3, 0, $4, true)
		RETURNING id, deposit_id, name, sku, quantity, min_quantity, active, created_at, updated_at`
	row := r.db.QueryRow(query, item.DepositID, item.Name, item.SKU, item.MinQuantity)
	return scanInventoryItem(row)
}

// ListByDeposits retorna os itens de inventário pertencentes a um conjunto
// de depósitos. Para a gestão (acesso total), o service passa todos os
// ids existentes; para o professor, apenas os depósitos de suas turmas.
func (r *InventoryRepository) ListByDeposits(depositIDs []string) ([]domain.InventoryItem, error) {
	if len(depositIDs) == 0 {
		return []domain.InventoryItem{}, nil
	}
	query := `
		SELECT id, deposit_id, name, sku, quantity, min_quantity, active, created_at, updated_at
		FROM inventory WHERE active = true AND deposit_id = ANY($1)
		ORDER BY name ASC`
	rows, err := r.db.Query(query, pq.Array(depositIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.InventoryItem
	for rows.Next() {
		item, err := scanInventoryItemRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *InventoryRepository) FindByID(id string) (domain.InventoryItem, error) {
	query := `
		SELECT id, deposit_id, name, sku, quantity, min_quantity, active, created_at, updated_at
		FROM inventory WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanInventoryItem(row)
}

func (r *InventoryRepository) Update(id, name, sku string, minQuantity int) (domain.InventoryItem, error) {
	query := `
		UPDATE inventory SET name = $1, sku = $2, min_quantity = $3, updated_at = now()
		WHERE id = $4
		RETURNING id, deposit_id, name, sku, quantity, min_quantity, active, created_at, updated_at`
	row := r.db.QueryRow(query, name, sku, minQuantity, id)
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

	var updated domain.InventoryItem
	row := tx.QueryRow(`
		UPDATE inventory SET quantity = quantity + $1, updated_at = now()
		WHERE id = $2
		RETURNING id, deposit_id, name, sku, quantity, min_quantity, active, created_at, updated_at`,
		delta, item.ID,
	)
	if err := row.Scan(&updated.ID, &updated.DepositID, &updated.Name, &updated.SKU,
		&updated.Quantity, &updated.MinQuantity, &updated.Active, &updated.CreatedAt, &updated.UpdatedAt); err != nil {
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
func (r *InventoryRepository) ListMovements(depositIDs []string, limit int) ([]domain.StockMovement, error) {
	if len(depositIDs) == 0 {
		return []domain.StockMovement{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `
		SELECT id, inventory_item_id, deposit_id, type, quantity, note, created_by, created_at
		FROM stock_movements WHERE deposit_id = ANY($1)
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

func scanInventoryItem(row *sql.Row) (domain.InventoryItem, error) {
	var i domain.InventoryItem
	err := row.Scan(&i.ID, &i.DepositID, &i.Name, &i.SKU, &i.Quantity, &i.MinQuantity, &i.Active, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.InventoryItem{}, ErrNotFound
	}
	return i, err
}

func scanInventoryItemRows(rows *sql.Rows) (domain.InventoryItem, error) {
	var i domain.InventoryItem
	err := rows.Scan(&i.ID, &i.DepositID, &i.Name, &i.SKU, &i.Quantity, &i.MinQuantity, &i.Active, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}
