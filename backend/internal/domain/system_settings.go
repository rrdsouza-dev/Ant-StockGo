package domain

import "time"

// SystemSettings agrupa configurações globais do sistema. Existe uma
// única linha na tabela `system_settings` (id sempre 1) — não há
// necessidade de mais de uma configuração simultânea.
type SystemSettings struct {
	MovementRetentionDays int       `json:"movement_retention_days"`
	UpdatedAt             time.Time `json:"updated_at"`
}
