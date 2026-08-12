-- 000040_add_conversation_assignee.down.sql
-- conversation-row-scoping — reversión del paso 1.

DROP INDEX IF EXISTS idx_conversations_assignee;

ALTER TABLE crm.conversations
  DROP COLUMN IF EXISTS assignee_stytch_member_id;
