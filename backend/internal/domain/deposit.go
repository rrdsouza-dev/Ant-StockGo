package domain

import "time"

// Deposit representa um depósito/estoque físico ou lógico da escola
// (ex.: "Almoxarifado Central", "Material Didático 2A").
//
// Regra de negócio central do sistema: TUDO gira em torno do Deposit.
// Não existe "produto isolado" — apenas itens de inventário pertencentes
// a um depósito, movimentados dentro dele.
type Deposit struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
