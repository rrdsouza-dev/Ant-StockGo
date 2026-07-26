-- ============================================================================
-- WMS Backend — Migração 004 (módulo Pré-Produto)
-- Execute após 003_admin_deposit_and_support.sql. Compatível com Supabase.
-- Não altera nenhuma migração anterior — apenas adiciona.
-- ============================================================================

-- ── pre_products: catálogo permanente de produtos ──────────────────────
-- Um Pré-Produto NÃO representa estoque: é apenas um "molde" reutilizável
-- (nome, categoria, marca, unidade de medida, observações) usado para
-- agilizar o cadastro de um item de estoque quando o material chega
-- fisicamente. Ele nunca é apagado automaticamente por uma entrada de
-- estoque — permanece disponível para reutilização em entradas futuras
-- (ver InventoryService / fluxo "Entrada utilizando Pré-Produto").
--
-- barcode é opcional e usado exclusivamente pelo fluxo de associação de
-- código de barras (ver item 6 da especificação): quando um código
-- escaneado não corresponde a nenhum item de estoque existente, ele pode
-- ser associado a um Pré-Produto aqui, e passa a ser reconhecido
-- automaticamente nas próximas leituras.
CREATE TABLE IF NOT EXISTS pre_products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    brand       TEXT NOT NULL DEFAULT '',
    unit        TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    barcode     TEXT,
    active      BOOLEAN NOT NULL DEFAULT true,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Um código de barras só pode estar associado a um único Pré-Produto por
-- vez. Índice único parcial (ignora NULL e strings vazias) para permitir
-- múltiplos Pré-Produtos sem código de barras associado ainda.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pre_products_barcode
    ON pre_products (barcode) WHERE barcode IS NOT NULL AND barcode <> '';

CREATE INDEX IF NOT EXISTS idx_pre_products_category_id ON pre_products (category_id);
