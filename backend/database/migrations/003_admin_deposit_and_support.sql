-- ============================================================================
-- WMS Backend — Migração 003 (permissões turma/depósito administrativo + suporte)
-- Execute após 002_item_details.sql. Compatível com Supabase (PostgreSQL).
-- Não altera nenhuma migração anterior — apenas adiciona.
-- ============================================================================

-- ── deposits: depósito administrativo ──────────────────────────────────
-- Novo modelo de acesso: a gestão deixa de enxergar o estoque de todos os
-- depósitos e passa a acessar apenas o(s) depósito(s) marcado(s) como
-- administrativo. A gestão continua podendo criar/editar/excluir QUALQUER
-- depósito e turma (gestão de entidades) — a restrição é só sobre acesso
-- ao ESTOQUE (itens/movimentações), aplicada na camada de services.
ALTER TABLE deposits
    ADD COLUMN IF NOT EXISTS is_administrative BOOLEAN NOT NULL DEFAULT false;

-- Garante no banco, como rede de segurança final, que só existe UM
-- depósito administrativo por vez (a troca é feita de forma atômica pelo
-- DepositRepository, que desmarca o antigo antes de marcar o novo).
CREATE UNIQUE INDEX IF NOT EXISTS idx_deposits_single_administrative
    ON deposits (is_administrative) WHERE is_administrative = true;

-- ── users: código de suporte do professor ──────────────────────────────
-- Gerado automaticamente na aprovação da conta (ver AuthService.ApproveOrReject).
-- Formato SKU-XXX-XXX. Visível ao próprio professor em /users/me (usado
-- para validar a abertura de chamados de suporte).
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS support_code TEXT;

-- ── support_tickets: reclamações/chamados de suporte ───────────────────
-- nome/email são gravados no momento do envio (não via JOIN) para que o
-- histórico permaneça correto mesmo que o professor mude de nome depois.
CREATE TABLE IF NOT EXISTS support_tickets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nome         TEXT NOT NULL,
    email        TEXT NOT NULL,
    categoria    TEXT NOT NULL CHECK (categoria IN ('estoque', 'sistema')),
    tipo         TEXT NOT NULL CHECK (tipo IN ('erro_operacional', 'erro_sistematico')),
    descricao    TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_professor_id ON support_tickets (professor_id);
