package domain

import "time"

// InventoryItem é um item de estoque pertencente a um depósito.
// A quantidade é sempre o resultado acumulado dos StockMovements
// (mantida de forma desnormalizada no campo Quantity para leitura rápida,
// mas nunca alterada diretamente por um handler — apenas pelo
// InventoryService.Move, dentro de uma transação junto com o movimento).
type InventoryItem struct {
	ID          string    `json:"id"`
	DepositID   string    `json:"deposit_id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku,omitempty"`
	Quantity    int       `json:"quantity"`
	MinQuantity int       `json:"min_quantity"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MovementType define o sentido de uma movimentação de estoque.
type MovementType string

const (
	MovementIn  MovementType = "in"
	MovementOut MovementType = "out"
)

// IsValid confirma se o tipo de movimentação é suportado.
func (m MovementType) IsValid() bool {
	return m == MovementIn || m == MovementOut
}

// StockMovement é o histórico imutável de toda entrada/saída de estoque.
// Toda movimentação de InventoryItem.Quantity DEVE gerar um registro aqui —
// é a auditoria obrigatória exigida pelas regras de negócio do sistema.
type StockMovement struct {
	ID              string       `json:"id"`
	InventoryItemID string       `json:"inventory_item_id"`
	DepositID       string       `json:"deposit_id"`
	Type            MovementType `json:"type"`
	Quantity        int          `json:"quantity"`
	Note            string       `json:"note,omitempty"`
	CreatedBy       string       `json:"created_by"`
	CreatedAt       time.Time    `json:"created_at"`
}
