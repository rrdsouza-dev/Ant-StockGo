package domain

import "time"

// Class representa uma turma criada pela gestão. Turmas vinculam
// professores a depósitos: um professor só enxerga/movimenta os
// depósitos ligados às turmas às quais está vinculado.
type Class struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Preenchidos pelo service ao montar a resposta (não persistidos
	// diretamente nesta struct — vêm das tabelas de junção
	// class_teachers e class_deposits).
	TeacherIDs []string  `json:"teacher_ids,omitempty"`
	Deposits   []Deposit `json:"deposits,omitempty"`
}
