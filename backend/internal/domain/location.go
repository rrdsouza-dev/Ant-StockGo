package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// Location descreve a posição física de um item dentro do depósito.
// É armazenada como JSONB (ver Value/Scan abaixo) exatamente para ser
// genérica: adicionar uma nova dimensão no futuro (ex.: "zona", "andar")
// é só acrescentar um campo aqui — nenhuma migração de coluna é necessária.
type Location struct {
	Aisle    int    `json:"aisle,omitempty"`    // Corredor: 1-10
	Tower    int    `json:"tower,omitempty"`    // Torre: 1-10
	Shelf    int    `json:"shelf,omitempty"`    // Prateleira: 1-10
	Position string `json:"position,omitempty"` // Posição: A1-A10
}

// IsEmpty confirma se nenhuma dimensão foi preenchida (localização opcional).
func (l Location) IsEmpty() bool {
	return l.Aisle == 0 && l.Tower == 0 && l.Shelf == 0 && l.Position == ""
}

// Validate garante que cada dimensão informada está dentro do intervalo
// suportado pela interface. Todas as dimensões são opcionais
// individualmente, mas se vierem preenchidas precisam ser válidas.
func (l Location) Validate() error {
	if l.Aisle != 0 && (l.Aisle < 1 || l.Aisle > 10) {
		return errors.New("corredor deve estar entre 1 e 10")
	}
	if l.Tower != 0 && (l.Tower < 1 || l.Tower > 10) {
		return errors.New("torre deve estar entre 1 e 10")
	}
	if l.Shelf != 0 && (l.Shelf < 1 || l.Shelf > 10) {
		return errors.New("prateleira deve estar entre 1 e 10")
	}
	if l.Position != "" && !isValidPosition(l.Position) {
		return errors.New("posição deve estar entre A1 e A10")
	}
	return nil
}

func isValidPosition(p string) bool {
	for i := 1; i <= 10; i++ {
		if p == fmt.Sprintf("A%d", i) {
			return true
		}
	}
	return false
}

// Value implementa driver.Valuer para que Location possa ser gravada
// diretamente como parâmetro de query (coluna `location JSONB`).
func (l Location) Value() (driver.Value, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implementa sql.Scanner para que Location possa ser lida
// diretamente de uma coluna JSONB via row.Scan(&item.Location).
func (l *Location) Scan(value any) error {
	if value == nil {
		*l = Location{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("tipo inesperado para Location: %T", value)
	}
	if len(b) == 0 {
		*l = Location{}
		return nil
	}
	return json.Unmarshal(b, l)
}
