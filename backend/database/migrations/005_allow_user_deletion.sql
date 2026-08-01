-- ============================================================================
-- WMS Backend — Migração 005 (permitir exclusão de conta preservando histórico)
-- Execute após 004_pre_products.sql. Compatível com Supabase (PostgreSQL).
-- Não altera nenhuma migração anterior — apenas ajusta constraints existentes.
-- ============================================================================

-- Contexto: a Gestão passou a poder excluir contas de professores (ver
-- UserService.DeleteProfessor). Antes desta migração, `created_by` em
-- deposits/stock_movements/pre_products e `professor_id` em
-- support_tickets referenciavam users(id) sem ON DELETE (= RESTRICT no
-- Postgres) — excluir um usuário que já tivesse qualquer histórico
-- (uma movimentação registrada, por exemplo) falharia com violação de
-- chave estrangeira.
--
-- A correção segue o mesmo princípio já usado no resto do sistema: o
-- HISTÓRICO NUNCA É APAGADO. Em vez de bloquear a exclusão da conta ou
-- apagar em cascata os registros que ela criou, a referência ao autor
-- vira NULL — o depósito, a movimentação, o pré-produto ou o chamado de
-- suporte continuam existindo normalmente, só "sem autor" (ou, no caso
-- de support_tickets, o nome/e-mail gravados no momento do envio
-- continuam presentes de qualquer forma, então nenhuma informação é
-- perdida).

-- A busca dinâmica abaixo (em vez de assumir o nome padrão gerado pelo
-- Postgres) torna a migração segura mesmo que a constraint tenha sido
-- criada ou renomeada de forma diferente do esperado.
DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'deposits'::regclass
      AND confrelid = 'users'::regclass
      AND contype = 'f';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE deposits DROP CONSTRAINT %I', constraint_name);
    END IF;
    ALTER TABLE deposits ALTER COLUMN created_by DROP NOT NULL;
    ALTER TABLE deposits ADD CONSTRAINT deposits_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'stock_movements'::regclass
      AND confrelid = 'users'::regclass
      AND contype = 'f';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE stock_movements DROP CONSTRAINT %I', constraint_name);
    END IF;
    ALTER TABLE stock_movements ALTER COLUMN created_by DROP NOT NULL;
    ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'pre_products'::regclass
      AND confrelid = 'users'::regclass
      AND contype = 'f';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE pre_products DROP CONSTRAINT %I', constraint_name);
    END IF;
    ALTER TABLE pre_products ALTER COLUMN created_by DROP NOT NULL;
    ALTER TABLE pre_products ADD CONSTRAINT pre_products_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'support_tickets'::regclass
      AND confrelid = 'users'::regclass
      AND contype = 'f';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE support_tickets DROP CONSTRAINT %I', constraint_name);
    END IF;
    ALTER TABLE support_tickets ALTER COLUMN professor_id DROP NOT NULL;
    ALTER TABLE support_tickets ADD CONSTRAINT support_tickets_professor_id_fkey
        FOREIGN KEY (professor_id) REFERENCES users(id) ON DELETE SET NULL;
END $$;
