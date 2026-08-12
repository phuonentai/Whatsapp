-- CRM queries

-- Contacts

-- name: UpsertContact :one
INSERT INTO crm.contacts (
    organization_id,
    phone_number,
    display_name,
    avatar_url,
    metadata,
    last_message_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (organization_id, phone_number) WHERE phone_number IS NOT NULL
DO UPDATE SET
    display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), contacts.display_name),
    avatar_url = COALESCE(NULLIF(EXCLUDED.avatar_url, ''), contacts.avatar_url),
    last_message_at = GREATEST(contacts.last_message_at, EXCLUDED.last_message_at),
    metadata = contacts.metadata || EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: UpsertContactByIGUser :one
INSERT INTO crm.contacts (
    organization_id,
    instagram_user_id,
    instagram_username,
    display_name,
    avatar_url,
    source,
    metadata,
    last_message_at
) VALUES (
    $1, $2, $3, $4, $5, 'instagram', $6, $7
) ON CONFLICT (organization_id, instagram_user_id) WHERE instagram_user_id IS NOT NULL
DO UPDATE SET
    instagram_username = COALESCE(NULLIF(EXCLUDED.instagram_username, ''), contacts.instagram_username),
    display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), contacts.display_name),
    avatar_url = COALESCE(NULLIF(EXCLUDED.avatar_url, ''), contacts.avatar_url),
    last_message_at = GREATEST(contacts.last_message_at, EXCLUDED.last_message_at),
    metadata = contacts.metadata || EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: GetContactByIGUser :one
SELECT * FROM crm.contacts
WHERE organization_id = $1 AND instagram_user_id = $2;

-- name: UpdateContactInstagramProfile :one
UPDATE crm.contacts
SET
    instagram_username = COALESCE($3, instagram_username),
    avatar_url = COALESCE($4, avatar_url),
    display_name = COALESCE($5, display_name),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: GetContactByID :one
SELECT * FROM crm.contacts
WHERE id = $1 AND organization_id = $2;

-- name: GetContactByPhone :one
SELECT * FROM crm.contacts
WHERE organization_id = $1 AND phone_number = $2;

-- name: ListContactsByOrganization :many
SELECT * FROM crm.contacts
WHERE organization_id = $1
ORDER BY COALESCE(last_message_at, created_at) DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- Conversations

-- Predicado de scope de conversaciones (conversation-row-scoping). Regla de
-- unión: assignee = miembro | owner de empresa = miembro | view_all |
-- (sin asignar AND view_unassigned). Cuando el flag `conversation_row_scoping`
-- es false (free tier), el predicado es org-wide (comportamiento pre-cambio).
-- Params de scope (convención en todas las queries scoped):
--   $scope_enabled    boolean  entitlement conversation_row_scoping
--   $scope_view_all   boolean  inbox:view_all u org:manage
--   $scope_member     text     stytch_member_id del miembro
--   $scope_unassigned boolean  inbox:view_unassigned

-- name: GetConversationByID :one
SELECT c.*
FROM crm.conversations c
LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
LEFT JOIN crm.companies co ON co.id = ct.company_id AND co.organization_id = ct.organization_id
LEFT JOIN organizations.accounts a ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
WHERE c.id = @id AND c.organization_id = @organization_id
  AND (
    NOT @scope_enabled::boolean
    OR @scope_view_all::boolean
    OR c.assignee_stytch_member_id = @scope_member::text
    OR a.stytch_member_id = @scope_member::text
    OR (c.assignee_stytch_member_id IS NULL AND @scope_unassigned::boolean)
  );

-- name: GetActiveConversationByContact :one
SELECT * FROM crm.conversations
WHERE contact_id = $1
  AND organization_id = $2
  AND channel = $3
  AND status = 'active'
ORDER BY last_message_at DESC NULLS LAST
LIMIT 1;

-- name: CreateConversation :one
INSERT INTO crm.conversations (
    organization_id,
    contact_id,
    channel,
    status,
    last_message_at,
    metadata,
    assignee_stytch_member_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: InsertActiveConversationIdempotent :one
INSERT INTO crm.conversations (
    organization_id,
    contact_id,
    channel,
    status,
    last_message_at,
    metadata,
    assignee_stytch_member_id
) VALUES (
    $1, $2, $3, 'active', $4, $5, $6
) ON CONFLICT (organization_id, contact_id, channel) WHERE status = 'active' DO NOTHING
RETURNING *;

-- name: UpdateConversationLastMessageAt :one
UPDATE crm.conversations
SET
    last_message_at = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateConversationStatus :one
UPDATE crm.conversations c
SET
    status = @status,
    updated_at = NOW()
WHERE c.id = @id AND c.organization_id = @organization_id
  AND (
    NOT @scope_enabled::boolean
    OR @scope_view_all::boolean
    OR c.assignee_stytch_member_id = @scope_member::text
    OR EXISTS (
      SELECT 1
      FROM crm.contacts ct
      JOIN crm.companies co ON co.id = ct.company_id AND co.organization_id = ct.organization_id
      JOIN organizations.accounts a ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
      WHERE ct.id = c.contact_id AND ct.organization_id = c.organization_id
        AND a.stytch_member_id = @scope_member::text
    )
    OR (c.assignee_stytch_member_id IS NULL AND @scope_unassigned::boolean)
  )
RETURNING *;

-- name: ListConversationsByOrganization :many
SELECT c.*, ct.phone_number AS contact_phone, ct.display_name AS contact_display_name,
       ct.instagram_username AS contact_instagram_username, ct.avatar_url AS contact_avatar_url
FROM crm.conversations c
LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
LEFT JOIN crm.companies co ON co.id = ct.company_id AND co.organization_id = ct.organization_id
LEFT JOIN organizations.accounts a ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
WHERE c.organization_id = @organization_id
  AND (CASE WHEN @status_filter::text = '' THEN TRUE ELSE c.status = @status_filter::text END)
  AND (CASE WHEN @channel_filter::text = '' THEN TRUE ELSE c.channel = @channel_filter::text END)
  AND (
    NOT @scope_enabled::boolean
    OR @scope_view_all::boolean
    OR c.assignee_stytch_member_id = @scope_member::text
    OR a.stytch_member_id = @scope_member::text
    OR (c.assignee_stytch_member_id IS NULL AND @scope_unassigned::boolean)
  )
  AND (
    @scope_view::text = ''
    OR (@scope_view::text = 'mine' AND (c.assignee_stytch_member_id = @scope_member::text OR a.stytch_member_id = @scope_member::text))
    OR (@scope_view::text = 'queue' AND c.assignee_stytch_member_id IS NULL AND @scope_unassigned::boolean)
    OR (@scope_view::text = 'all' AND @scope_view_all::boolean)
  )
ORDER BY c.last_message_at DESC NULLS LAST
LIMIT @page_limit OFFSET @page_offset;

-- name: UpdateConversationAssignee :one
UPDATE crm.conversations
SET
    assignee_stytch_member_id = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: InsertConversationEvent :one
INSERT INTO crm.conversation_events (
    organization_id,
    conversation_id,
    event_type,
    actor_stytch_member_id,
    payload
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ResolveContactAssignee :one
SELECT a.stytch_member_id AS stytch_member_id
FROM crm.contacts ct
LEFT JOIN crm.companies co
  ON co.id = ct.company_id AND co.organization_id = ct.organization_id
LEFT JOIN organizations.accounts a
  ON a.id = COALESCE(ct.assigned_to, co.owner_account_id)
 AND a.organization_id = ct.organization_id
WHERE ct.id = $1 AND ct.organization_id = $2
LIMIT 1;

-- name: GetCompanyOwnerMemberByPhone :one
SELECT a.stytch_member_id AS stytch_member_id
FROM crm.companies co
JOIN crm.contacts ct
  ON ct.company_id = co.id AND ct.organization_id = co.organization_id
JOIN organizations.accounts a
  ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
WHERE co.organization_id = $1
  AND ct.phone_number = $2
  AND a.stytch_member_id IS NOT NULL
LIMIT 1;

-- name: GetCompanyOwnerMemberByNit :one
SELECT a.stytch_member_id AS stytch_member_id
FROM crm.companies co
JOIN organizations.accounts a
  ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
WHERE co.organization_id = $1
  AND co.nit = $2
  AND a.stytch_member_id IS NOT NULL
LIMIT 1;

-- Messages

-- name: CreateMessage :one
INSERT INTO crm.messages (
    organization_id,
    conversation_id,
    contact_id,
    channel,
    provider_message_id,
    direction,
    message_type,
    content,
    status,
    message_data,
    chat_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: InsertMessageIdempotent :one
INSERT INTO crm.messages (
    organization_id,
    conversation_id,
    contact_id,
    channel,
    provider_message_id,
    direction,
    message_type,
    content,
    status,
    message_data,
    chat_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) ON CONFLICT (organization_id, channel, provider_message_id) WHERE provider_message_id IS NOT NULL DO NOTHING
RETURNING *;

-- name: GetMessageByProviderID :one
SELECT * FROM crm.messages
WHERE organization_id = $1 AND channel = $2 AND provider_message_id = $3;

-- name: UpdateMessageStatus :one
UPDATE crm.messages
SET
    status = $2,
    provider_message_id = COALESCE($3, provider_message_id),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListMessagesByConversation :many
SELECT * FROM crm.messages
WHERE conversation_id = $1 AND organization_id = $2
ORDER BY created_at ASC
LIMIT $3 OFFSET $4;

-- name: GetCompanyByNit :one
SELECT * FROM crm.companies
WHERE organization_id = $1 AND nit = $2
LIMIT 1;
