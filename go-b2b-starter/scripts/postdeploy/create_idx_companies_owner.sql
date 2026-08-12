-- ============================================================================
-- postdeploy: conversation-row-scoping — índice de ownership (paso 3 del plan
-- de migración, task 2.4).
--
-- golang-migrate v4.17.1 envuelve CADA archivo de migración en una transacción
-- y NO soporta la anotación `-- +migrate NoTransaction` (verificado en el
-- código fuente de la v4.17.1), por lo que `CREATE INDEX CONCURRENTLY` NO puede
-- vivir dentro del secuenciador. Este archivo se ejecuta como paso POST-DEPLOY
-- del job (runner no-transaccional) — nunca dentro de la transacción envuelta.
--
-- Uso:
--   psql "$DATABASE_URL" -f scripts/postdeploy/create_idx_companies_owner.sql
--
-- Reversión:
--   psql "$DATABASE_URL" -c "DROP INDEX IF EXISTS idx_companies_owner;"
-- ============================================================================

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_companies_owner
  ON crm.companies(owner_account_id);
