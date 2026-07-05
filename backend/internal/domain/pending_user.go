package domain

import "time"

// PendingStatus representa o estado de uma solicitação de conta.
type PendingStatus string

const (
	PendingStatusPending  PendingStatus = "pending"
	PendingStatusApproved PendingStatus = "approved"
	PendingStatusRejected PendingStatus = "rejected"
)

// PendingUser representa uma conta que se cadastrou mas ainda não foi
// aprovada pela gestão.
//
// Fluxo no sistema:
//  1. POST /auth/register cria um registro aqui com status "pending".
//  2. GET  /auth/pending lista os registros pendentes para a gestão revisar.
//  3. POST /auth/approve aprova (cria o User real e remove/atualiza este
//     registro) ou rejeita (marca como "rejected").
//
// Nenhum PendingUser pode autenticar-se: /auth/login só reconhece
// contas que já existem na tabela users.
type PendingUser struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Email        string        `json:"email"`
	PasswordHash string        `json:"-"`
	Role         Role          `json:"role"`
	Status       PendingStatus `json:"status"`
	RequestedAt  time.Time     `json:"requested_at"`
	ReviewedAt   *time.Time    `json:"reviewed_at,omitempty"`
}
