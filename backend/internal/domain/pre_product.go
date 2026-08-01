package domain

import "time"

// PreProduct é um catálogo permanente de produtos, reutilizável em
// futuras entradas de estoque. Diferente de InventoryItem, um
// PreProduct NÃO representa estoque: não tem quantidade, validade, lote
// ou código de barras fixo — apenas os dados "base" de um produto que a
// escola pretende (ou pode vir a) receber.
//
// Fluxo no sistema: cadastrado a qualquer momento pelo botão "Pré-Produto"
// na tela de Estoque. Quando o produto chega fisicamente, o Pré-Produto
// é usado para agilizar a criação do InventoryItem real (nome, categoria,
// marca e unidade já vêm preenchidos) — o Pré-Produto em si permanece
// intacto no catálogo, disponível para a próxima entrada.
type PreProduct struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CategoryID *string   `json:"category_id,omitempty"`
	Category   *Category `json:"category,omitempty"` // hidratado via LEFT JOIN, só para leitura
	Brand      string    `json:"brand,omitempty"`
	Unit       string    `json:"unit"`
	Notes      string    `json:"notes,omitempty"`
	Barcode    string    `json:"barcode,omitempty"`
	Active     bool      `json:"active"`
	CreatedBy  string    `json:"created_by"` // vazio se o usuário autor já foi excluído (ver migração 005)
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
