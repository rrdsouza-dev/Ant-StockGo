-- ============================================================================
-- WMS Backend — Migração inicial (001_init.sql)
-- Compatível com Supabase (PostgreSQL). Execute no SQL Editor do Supabase
-- ou via `psql "$DATABASE_URL" -f database/migrations/001_init.sql`.
-- ============================================================================

-- Necessário para gen_random_uuid(); no Supabase já vem habilitado por padrão.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── users ───────────────────────────────────────────────────────────────
-- Contas já aprovadas e ativas no sistema. Nunca recebe INSERT direto de
-- um cadastro público — apenas via aprovação de um pending_users.
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('professor', 'gestao')),
    active        BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── pending_users ───────────────────────────────────────────────────────
-- Solicitações de cadastro aguardando decisão da gestão.
CREATE TABLE IF NOT EXISTS pending_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('professor', 'gestao')),
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at   TIMESTAMPTZ
);

-- Impede duas solicitações pendentes simultâneas para o mesmo e-mail.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_users_email_pending
    ON pending_users (email) WHERE status = 'pending';

-- ── deposits ────────────────────────────────────────────────────────────
-- Depósitos/estoques. Entidade central do sistema.
CREATE TABLE IF NOT EXISTS deposits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT true,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── inventory ───────────────────────────────────────────────────────────
-- Itens de estoque, sempre pertencentes a um depósito.
CREATE TABLE IF NOT EXISTS inventory (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_id   UUID NOT NULL REFERENCES deposits(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    sku          TEXT NOT NULL DEFAULT '',
    quantity     INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    min_quantity INTEGER NOT NULL DEFAULT 0,
    active       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_inventory_deposit_id ON inventory (deposit_id);

-- ── stock_movements ─────────────────────────────────────────────────────
-- Auditoria imutável de toda entrada/saída de estoque.
CREATE TABLE IF NOT EXISTS stock_movements (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_item_id UUID NOT NULL REFERENCES inventory(id) ON DELETE CASCADE,
    deposit_id        UUID NOT NULL REFERENCES deposits(id) ON DELETE CASCADE,
    type              TEXT NOT NULL CHECK (type IN ('in', 'out')),
    quantity          INTEGER NOT NULL CHECK (quantity > 0),
    note              TEXT NOT NULL DEFAULT '',
    created_by        UUID NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_deposit_id ON stock_movements (deposit_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_item_id ON stock_movements (inventory_item_id);

-- ── classes ─────────────────────────────────────────────────────────────
-- Turmas criadas pela gestão.
CREATE TABLE IF NOT EXISTS classes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── class_teachers ──────────────────────────────────────────────────────
-- Vínculo N:N entre turmas e professores.
CREATE TABLE IF NOT EXISTS class_teachers (
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (class_id, user_id)
);

-- ── class_deposits ──────────────────────────────────────────────────────
-- Vínculo N:N entre turmas e depósitos: define quais estoques cada
-- turma (e, por consequência, cada professor vinculado) pode acessar.
CREATE TABLE IF NOT EXISTS class_deposits (
    class_id   UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    deposit_id UUID NOT NULL REFERENCES deposits(id) ON DELETE CASCADE,
    PRIMARY KEY (class_id, deposit_id)
);

-- ============================================================================
-- Seed opcional: primeira conta de gestão para conseguir logar no sistema
-- assim que ele sobe. Gere o hash com bcrypt (custo 10) e troque os valores
-- abaixo antes de rodar em produção. Senha de exemplo: "TrocarSenha123!"
-- Hash gerado com bcrypt.GenerateFromPassword (custo padrão = 10).
-- ============================================================================
-- INSERT INTO users (name, email, password_hash, role, active)
-- VALUES (
--     'Administrador',
--     'admin@escola.com',
--     '$2a$10$8K1p/a0dURXAm7QiTRqluOfCr5cy0S8CIH5aERAj4wVQzZg3hDsK.',
--     'gestao',
--     true
-- );
