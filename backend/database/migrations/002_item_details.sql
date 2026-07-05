-- ============================================================================
-- WMS Backend — Migração 002 (cadastro estendido de itens de estoque)
-- Execute após 001_init.sql. Compatível com Supabase (PostgreSQL).
-- Segura para rodar em bancos já em uso: usa IF NOT EXISTS / ADD COLUMN
-- e não força NOT NULL em colunas novas de uma tabela que já pode ter dados.
-- ============================================================================

-- ── categories ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Categorias iniciais de exemplo (idempotente: não duplica se já existir).
INSERT INTO categories (name) VALUES
    ('Arroz'), ('Feijão'), ('Macarrão'), ('Açúcar'), ('Sal'),
    ('Óleo'), ('Leite'), ('Café'), ('Outros')
ON CONFLICT (name) DO NOTHING;

-- ── inventory: novos campos do cadastro estendido ─────────────────────
-- expiry_date e category_id ficam NULLABLE no banco (itens cadastrados
-- antes desta migração não têm valor); a obrigatoriedade de "data de
-- validade" pedida na especificação é aplicada na camada de validação
-- (internal/validation), não como constraint destrutiva de schema —
-- assim a migração não quebra em cima de dados já existentes.
ALTER TABLE inventory
    ADD COLUMN IF NOT EXISTS expiry_date DATE,
    ADD COLUMN IF NOT EXISTS lot_number  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS notes       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS location    JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_inventory_category_id ON inventory (category_id);
