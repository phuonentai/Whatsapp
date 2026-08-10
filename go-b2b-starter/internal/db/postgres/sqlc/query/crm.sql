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

-- name: GetConversationByID :one
SELECT * FROM crm.conversations
WHERE id = $1 AND organization_id = $2;

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
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: InsertActiveConversationIdempotent :one
INSERT INTO crm.conversations (
    organization_id,
    contact_id,
    channel,
    status,
    last_message_at,
    metadata
) VALUES (
    $1, $2, $3, 'active', $4, $5
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
UPDATE crm.conversations
SET
    status = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: ListConversationsByOrganization :many
SELECT c.*, ct.phone_number AS contact_phone, ct.display_name AS contact_display_name,
       ct.instagram_username AS contact_instagram_username, ct.avatar_url AS contact_avatar_url
FROM crm.conversations c
LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
WHERE c.organization_id = $1
  AND (CASE WHEN $4::text = '' THEN TRUE ELSE c.status = $4::text END)
  AND (CASE WHEN $5::text = '' THEN TRUE ELSE c.channel = $5::text END)
ORDER BY c.last_message_at DESC NULLS LAST
LIMIT $2 OFFSET $3;

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
