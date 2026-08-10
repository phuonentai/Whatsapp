-- WhatsApp queries

-- name: GetWhatsAppConfigByPhoneNumberID :one
SELECT * FROM whatsapp.whatsapp_configs
WHERE phone_number_id = $1 AND is_active = true;

-- name: GetWhatsAppConfigByOrganizationID :one
SELECT * FROM whatsapp.whatsapp_configs
WHERE organization_id = $1;

-- name: CreateWhatsAppConfig :one
INSERT INTO whatsapp.whatsapp_configs (
    organization_id,
    phone_number_id,
    business_phone,
    webhook_secret,
    verify_token,
    app_id,
    waba_id,
    access_token,
    api_version,
    graph_api_url,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateWhatsAppConfig :one
UPDATE whatsapp.whatsapp_configs
SET
    phone_number_id = COALESCE($2, phone_number_id),
    business_phone = COALESCE($3, business_phone),
    webhook_secret = COALESCE($4, webhook_secret),
    verify_token = COALESCE($5, verify_token),
    app_id = COALESCE($6, app_id),
    waba_id = COALESCE($7, waba_id),
    access_token = COALESCE($8, access_token),
    api_version = COALESCE($9, api_version),
    graph_api_url = COALESCE($10, graph_api_url),
    is_active = COALESCE($11, is_active),
    metadata = COALESCE($12, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetWhatsAppConfigByVerifyToken :one
SELECT * FROM whatsapp.whatsapp_configs
WHERE verify_token = $1 AND is_active = true;

-- name: InsertWebhookLog :one
INSERT INTO whatsapp.webhook_logs (
    organization_id,
    status,
    event_type,
    phone_number_id,
    raw_headers,
    raw_body,
    error_message,
    processed_at,
    delivery_key
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetWebhookLogByID :one
SELECT * FROM whatsapp.webhook_logs
WHERE id = $1;

-- name: UpdateWebhookLogStatus :one
UPDATE whatsapp.webhook_logs
SET
    status = $2,
    error_message = COALESCE($3, error_message),
    processed_at = COALESCE($4, processed_at)
WHERE id = $1
RETURNING *;

-- name: GetWebhookLogStatsByOrganization :many
SELECT
    status,
    COUNT(*) AS count
FROM whatsapp.webhook_logs
WHERE organization_id = $1
  AND created_at > $2
GROUP BY status;

-- name: GetLastWebhookErrorByOrganization :one
SELECT error_message, created_at
FROM whatsapp.webhook_logs
WHERE organization_id = $1
  AND status = 'failed'
ORDER BY created_at DESC
LIMIT 1;

-- name: UpsertWhatsAppSignupFlow :one
INSERT INTO whatsapp.signup_flows (
    organization_id, status, step, error_code, retry_count, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (organization_id) DO UPDATE SET
    status = EXCLUDED.status,
    step = EXCLUDED.step,
    error_code = COALESCE(EXCLUDED.error_code, whatsapp.signup_flows.error_code),
    retry_count = EXCLUDED.retry_count,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: GetWhatsAppSignupFlowByOrganization :one
SELECT * FROM whatsapp.signup_flows
WHERE organization_id = $1;

-- name: UpdateWhatsAppSignupFlowStatus :one
UPDATE whatsapp.signup_flows
SET
    status = COALESCE($2, status),
    step = COALESCE($3, step),
    error_code = COALESCE($4, error_code),
    retry_count = COALESCE($5, retry_count),
    metadata = COALESCE($6, metadata),
    updated_at = NOW()
WHERE organization_id = $1
RETURNING *;
