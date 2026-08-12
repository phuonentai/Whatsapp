-- 000040_add_conversation_assignee.up.sql
-- conversation-row-scoping — paso 1: ADD COLUMN catalog-only.
--
-- assignee_stytch_member_id sigue el patrón de crm.tickets (000017): FK
-- lógico a Stytch (stytch_member_id) sin tabla local de miembros; NULL =
-- conversación no asignada (cola). Sin default volátil, expand-safe.

ALTER TABLE crm.conversations
  ADD COLUMN assignee_stytch_member_id TEXT;

COMMENT ON COLUMN crm.conversations.assignee_stytch_member_id IS
  'Miembro asignado vía stytch_member_id (FK lógico a Stytch, patrón crm.tickets; sin tabla local de miembros). NULL = conversación no asignada (cola de leads)';

CREATE INDEX idx_conversations_assignee
  ON crm.conversations(organization_id, assignee_stytch_member_id);
