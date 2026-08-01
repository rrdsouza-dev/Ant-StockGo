package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

// SystemSettingsRepository isola todo acesso SQL à tabela
// `system_settings` (configuração única do sistema) e à exclusão de
// movimentações antigas conforme o período de retenção configurado.
type SystemSettingsRepository struct {
	db *sql.DB
}

func NewSystemSettingsRepository(db *sql.DB) *SystemSettingsRepository {
	return &SystemSettingsRepository{db: db}
}

// Get retorna a configuração atual. A migração 006 já garante que a
// linha singleton (id = 1) sempre existe.
func (r *SystemSettingsRepository) Get() (domain.SystemSettings, error) {
	var s domain.SystemSettings
	err := r.db.QueryRow(`SELECT movement_retention_days, updated_at FROM system_settings WHERE id = 1`).
		Scan(&s.MovementRetentionDays, &s.UpdatedAt)
	return s, err
}

// UpdateRetentionDays altera o período de retenção de movimentações.
func (r *SystemSettingsRepository) UpdateRetentionDays(days int) (domain.SystemSettings, error) {
	var s domain.SystemSettings
	err := r.db.QueryRow(`
		UPDATE system_settings SET movement_retention_days = $1, updated_at = now()
		WHERE id = 1
		RETURNING movement_retention_days, updated_at`, days,
	).Scan(&s.MovementRetentionDays, &s.UpdatedAt)
	return s, err
}

// DeleteMovementsOlderThan apaga permanentemente as movimentações mais
// antigas que `days` dias (contados a partir de `created_at`). Retorna
// quantas linhas foram removidas, só para fins de log/observabilidade —
// esta é a única rotina do sistema que efetivamente apaga histórico de
// movimentações (ver MaintenanceService.CleanupOldMovements, chamado
// periodicamente por uma goroutine em main.go).
func (r *SystemSettingsRepository) DeleteMovementsOlderThan(days int) (int64, error) {
	result, err := r.db.Exec(
		`DELETE FROM stock_movements WHERE created_at < now() - make_interval(days => $1)`, days,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
