-- 000041_conversation_assignee_audit_backfill.down.sql
-- conversation-row-scoping — reversión del paso 3 (el backfill no es
-- reversible dato por dato; al revertir, la columna se elimina en 000040
-- down, lo que descarta el dato backfilled junto con todo el cambio).

SELECT 1;
