-- 000042_conversation_row_scoping_rls.down.sql
-- conversation-row-scoping — reversión del paso 4 (RLS opt-in).

DROP POLICY IF EXISTS conversations_scope_insert ON crm.conversations;
DROP POLICY IF EXISTS conversations_scope_delete ON crm.conversations;
DROP POLICY IF EXISTS conversations_scope_update ON crm.conversations;
DROP POLICY IF EXISTS conversations_scope_select ON crm.conversations;

ALTER TABLE crm.conversations DISABLE ROW LEVEL SECURITY;

REVOKE USAGE, SELECT ON SEQUENCE crm.conversations_id_seq FROM app_session;
REVOKE SELECT, INSERT, UPDATE, DELETE ON crm.conversations FROM app_session;
REVOKE USAGE ON SCHEMA crm FROM app_session;

-- El rol app_session se conserva (puede ser reutilizado por el patrón RLS);
-- para eliminarlo por completo:
--   DROP ROLE IF EXISTS app_session;
