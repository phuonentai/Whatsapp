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
) ON CONFLICT (organization_id, phone_number)
DO UPDATE SET
    display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), contacts.display_name),
    avatar_url = COALESCE(NULLIF(EXCLUDED.avatar_url, ''), contacts.avatar_url),
    last_message_at = GREATEST(contacts.last_message_at, EXCLUDED.last_message_at),
    metadata = contacts.metadata || EXCLUDED.metadata,
    updated_at = NOW()
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
ORDER BY last_message_at DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- Conversations

-- name: GetConversationByID :one
SELECT * FROM crm.conversations
WHERE id = $1 AND organization_id = $2;

-- name: GetActiveConversationByContact :one
SELECT * FROM crm.conversations
WHERE contact_id = $1
  AND organization_id = $2
  AND status = 'active'
ORDER BY last_message_at DESC NULLS LAST
LIMIT 1;

-- name: CreateConversation :one
INSERT INTO crm.conversations (
    organization_id,
    contact_id,
    status,
    last_message_at,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: InsertActiveConversationIdempotent :one
INSERT INTO crm.conversations (
    organization_id,
    contact_id,
    status,
    last_message_at,
    metadata
) VALUES (
    $1, $2, 'active', $3, $4
) ON CONFLICT (organization_id, contact_id) WHERE status = 'active' DO NOTHING
RETURNING *;

-- name: UpdateConversationLastMessageAt :one
UPDATE crm.conversations
SET
    last_message_at = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateConversationStatus :one
UPDATE crm.conversations
SET
    status = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: ListConversationsByOrganization :many
SELECT c.*, ct.phone_number AS contact_phone, ct.display_name AS contact_display_name
FROM crm.conversations c
LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
WHERE c.organization_id = $1
  AND (CASE WHEN $4::text = '' THEN TRUE ELSE c.status = $4::text END)
ORDER BY c.last_message_at DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- Messages

-- name: CreateMessage :one
INSERT INTO crm.messages (
    organization_id,
    conversation_id,
    contact_id,
    whatsapp_message_id,
    direction,
    message_type,
    content,
    status,
    message_data,
    chat_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: InsertMessageIdempotent :one
INSERT INTO crm.messages (
    organization_id,
    conversation_id,
    contact_id,
    whatsapp_message_id,
    direction,
    message_type,
    content,
    status,
    message_data,
    chat_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) ON CONFLICT (organization_id, whatsapp_message_id) DO NOTHING
RETURNING *;

-- name: GetMessageByWhatsAppID :one
SELECT * FROM crm.messages
WHERE organization_id = $1 AND whatsapp_message_id = $2;

-- name: ListMessagesByConversation :many
SELECT * FROM crm.messages
WHERE conversation_id = $1 AND organization_id = $2
ORDER BY created_at ASC
LIMIT $3 OFFSET $4;
