-- ============================================================================
-- WMS Backend — Migração 007 (marca no item de estoque + lote obrigatório)
-- Execute após 006_system_settings.sql. Compatível com Supabase (PostgreSQL).
-- Não altera nenhuma migração anterior — apenas adiciona/ajusta.
-- ============================================================================

-- ── inventory: marca como campo próprio ────────────────────────────────
-- Até aqui, a marca só existia em pre_products. Um item de estoque real
-- (inventory) representa, na prática, um LOTE de um produto: já possui
-- lot_number, expiry_date e quantity individuais. Faltava apenas o campo
-- de marca para permitir diferenciar produtos com nome/código parecidos
-- mas marcas diferentes (ex.: "Pão Puma — Panco" vs "Pão Puma — Bauducco"),
-- sem depender de `notes` (ver item 2/3/7 da especificação de melhorias).
--
-- NOT NULL DEFAULT '' para não quebrar itens já cadastrados: eles passam
-- a ter marca vazia, que a interface trata como "sem marca definida" (a
-- mesma convenção já usada em pre_products.brand).
ALTER TABLE inventory
    ADD COLUMN IF NOT EXISTS brand TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_inventory_brand ON inventory (brand);

-- ── inventory: lote passa a ser obrigatório em novos registros ────────
-- Regra explícita da especificação: ao entrar no estoque, lote deixa de
-- ser opcional. Isso é aplicado de verdade na camada de validação do
-- backend (internal/services/inventory_service.go), não apenas no
-- frontend — requisições diretas à API sem lote são recusadas.
--
-- Não adicionamos uma CHECK (lot_number <> '') aqui de propósito: itens
-- cadastrados ANTES desta migração podem ter lot_number = '' (era
-- opcional até então) e uma constraint destrutiva quebraria a leitura/
-- atualização desses registros existentes. A obrigatoriedade de fato é
-- garantida na validação do service, que é a fonte da verdade para
-- INSERTs e UPDATEs novos — mesmo princípio já usado para expiry_date
-- na migração 002.
