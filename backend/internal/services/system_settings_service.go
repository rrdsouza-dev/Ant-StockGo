package services

import (
	"errors"
	"log"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

// Limites defensivos para o período de retenção: evita que a Gestão
// configure, por engano, um valor absurdo (ex.: 0 dias, apagando tudo
// imediatamente, ou um número tão grande que não faz sentido prático).
const (
	MinRetentionDays = 7
	MaxRetentionDays = 365
)

// SystemSettingsService concentra a regra de negócio da configuração de
// retenção de movimentações e a rotina de limpeza automática.
//
// Nota de arquitetura: até uma versão anterior deste sistema, o
// histórico de `stock_movements` era tratado como permanente e nunca
// apagado. A limpeza automática implementada aqui é uma reversão
// deliberada dessa política, solicitada explicitamente para evitar
// acúmulo excessivo de registros — ver RELATORIO.md.
type SystemSettingsService struct {
	settings *repositories.SystemSettingsRepository
}

func NewSystemSettingsService(settings *repositories.SystemSettingsRepository) *SystemSettingsService {
	return &SystemSettingsService{settings: settings}
}

func (s *SystemSettingsService) Get() (domain.SystemSettings, error) {
	return s.settings.Get()
}

// UpdateRetentionDays altera o período de retenção (somente gestão,
// restrição aplicada pelo middleware de rota). Valida o intervalo antes
// de gravar.
func (s *SystemSettingsService) UpdateRetentionDays(days int) (domain.SystemSettings, error) {
	if days < MinRetentionDays || days > MaxRetentionDays {
		return domain.SystemSettings{}, errors.New("período de retenção deve estar entre 7 e 365 dias")
	}
	return s.settings.UpdateRetentionDays(days)
}

// CleanupOldMovements lê o período de retenção configurado e apaga
// permanentemente as movimentações mais antigas que esse período.
// Chamado periodicamente por uma goroutine iniciada em main.go — nunca
// diretamente por uma requisição HTTP.
func (s *SystemSettingsService) CleanupOldMovements() {
	settings, err := s.settings.Get()
	if err != nil {
		log.Printf("limpeza de movimentações: erro ao ler configuração: %v", err)
		return
	}
	removed, err := s.settings.DeleteMovementsOlderThan(settings.MovementRetentionDays)
	if err != nil {
		log.Printf("limpeza de movimentações: erro ao apagar registros antigos: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("limpeza de movimentações: %d registro(s) com mais de %d dia(s) removido(s)", removed, settings.MovementRetentionDays)
	}
}
