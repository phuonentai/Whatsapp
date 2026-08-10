-- Instagram queries

-- name: GetInstagramConfigByIGUserID :one
SELECT * FROM whatsapp.instagram_configs
WHERE ig_user_id = $1 AND is_active = true;

-- name: GetInstagramConfigByOrganizationID :one
SELECT * FROM whatsapp.instagram_configs
WHERE organization_id = $1;

-- name: GetInstagramConfigByVerifyToken :one
SELECT * FROM whatsapp.instagram_configs
WHERE verify_token = $1 AND is_active = true;

-- name: CreateInstagramConfig :one
INSERT INTO whatsapp.instagram_configs (
    organization_id,
    ig_user_id,
    ig_username,
    fb_page_id,
    access_token,
    token_expires_at,
    webhook_secret,
    verify_token,
    api_version,
    graph_api_url,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateInstagramConfig :one
UPDATE whatsapp.instagram_configs
SET
    ig_user_id = COALESCE($2, ig_user_id),
    ig_username = COALESCE($3, ig_username),
    fb_page_id = COALESCE($4, fb_page_id),
    access_token = COALESCE($5, access_token),
    token_expires_at = COALESCE($6, token_expires_at),
    webhook_secret = COALESCE($7, webhook_secret),
    verify_token = COALESCE($8, verify_token),
    api_version = COALESCE($9, api_version),
    graph_api_url = COALESCE($10, graph_api_url),
    is_active = COALESCE($11, is_active),
    metadata = COALESCE($12, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: InsertInstagramWebhookLog :one
INSERT INTO whatsapp.instagram_webhook_logs (
    organization_id,
    status,
    event_type,
    ig_user_id,
    raw_headers,
    raw_body,
    error_message,
    processed_at,
    delivery_key
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetInstagramWebhookLogByID :one
SELECT * FROM whatsapp.instagram_webhook_logs
WHERE id = $1;

-- name: GetInstagramWebhookLogStatsByOrganization :many
SELECT
    status,
    COUNT(*) AS count
FROM whatsapp.instagram_webhook_logs
WHERE organization_id = $1
  AND created_at > $2
GROUP BY status;

-- name: GetLastInstagramWebhookErrorByOrganization :one
SELECT error_message, created_at
FROM whatsapp.instagram_webhook_logs
WHERE organization_id = $1
  AND status = 'failed'
ORDER BY created_at DESC
LIMIT 1;
