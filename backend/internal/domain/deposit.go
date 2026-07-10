package domain

import "time"

// Deposit representa um depósito/estoque físico ou lógico da escola
// (ex.: "Almoxarifado Central", "Material Didático 2A").
//
// Regra de negócio central do sistema: TUDO gira em torno do Deposit.
// Não existe "produto isolado" — apenas itens de inventário pertencentes
// a um depósito, movimentados dentro dele.
//
// IsAdministrative marca o depósito de uso exclusivo da gestão. Só pode
// existir um por vez (garantido por DepositRepository + constraint no
// banco) — é o único depósito cujo ESTOQUE a gestão pode acessar; os
// demais depósitos ela continua podendo criar/editar/excluir como
// entidade, mas não vê os itens/movimentações deles (ver
// ClassService.AccessibleDepositIDs).
type Deposit struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	IsAdministrative bool      `json:"is_administrative"`
	Active           bool      `json:"active"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
