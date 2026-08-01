-- ============================================================================
-- WMS Backend — Migração 006 (configurações do sistema + retenção de movimentações)
-- Execute após 005_allow_user_deletion.sql. Compatível com Supabase.
-- Não altera nenhuma migração anterior — apenas adiciona.
-- ============================================================================

-- ATENÇÃO — mudança de política em relação a versões anteriores:
-- até aqui, o histórico de `stock_movements` era tratado como permanente
-- e nunca apagado (essa era, inclusive, uma decisão de design registrada
-- em relatórios anteriores). A partir desta migração, a Gestão pode optar
-- por uma limpeza automática de movimentações antigas (ver
-- MaintenanceService.CleanupOldMovements, executado periodicamente pelo
-- backend). Isso é uma reversão deliberada dessa política anterior,
-- solicitada explicitamente para evitar acúmulo excessivo de registros —
-- ver RELATORIO.md para o registro completo dessa mudança.

-- ── system_settings: configuração única do sistema (singleton) ────────
-- Uma única linha (id sempre 1) guarda ajustes globais do sistema. Hoje
-- só existe o período de retenção de movimentações; novas configurações
-- futuras podem ser adicionadas como novas colunas aqui, via migração
-- própria, seguindo o mesmo padrão do restante do projeto.
CREATE TABLE IF NOT EXISTS system_settings (
    id                      INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    movement_retention_days INT NOT NULL DEFAULT 30 CHECK (movement_retention_days > 0),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO system_settings (id, movement_retention_days)
VALUES (1, 30)
ON CONFLICT (id) DO NOTHING;
