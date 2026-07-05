package domain

import "time"

// Category é uma categoria de item de estoque (ex.: "Arroz", "Feijão").
// É uma entidade própria (não um enum fixo no código) justamente para
// que a gestão possa cadastrar novas categorias em tempo de uso, pelo
// botão "+" no formulário de item — e elas persistam nos próximos acessos.
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
