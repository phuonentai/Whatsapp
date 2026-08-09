-- name: GetSubscriptionByOrgID :one
-- Get subscription details for an organization
SELECT * FROM subscription_billing.subscriptions
WHERE organization_id = $1
LIMIT 1;

-- name: GetSubscriptionBySubscriptionID :one
-- Get subscription by Polar subscription ID
SELECT * FROM subscription_billing.subscriptions
WHERE subscription_id = $1
LIMIT 1;

-- name: UpsertSubscription :one
-- Create or update subscription from Polar webhook
INSERT INTO subscription_billing.subscriptions (
    organization_id,
    external_customer_id,
    subscription_id,
    subscription_status,
    product_id,
    product_name,
    plan_name,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    canceled_at,
    metadata,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP
)
ON CONFLICT (organization_id)
DO UPDATE SET
    external_customer_id = EXCLUDED.external_customer_id,
    subscription_id = EXCLUDED.subscription_id,
    subscription_status = EXCLUDED.subscription_status,
    product_id = EXCLUDED.product_id,
    product_name = EXCLUDED.product_name,
    plan_name = EXCLUDED.plan_name,
    current_period_start = EXCLUDED.current_period_start,
    current_period_end = EXCLUDED.current_period_end,
    cancel_at_period_end = EXCLUDED.cancel_at_period_end,
    canceled_at = EXCLUDED.canceled_at,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteSubscription :exec
-- Delete subscription (when subscription is permanently deleted)
DELETE FROM subscription_billing.subscriptions
WHERE organization_id = $1;

-- name: GetQuotaByOrgID :one
-- Get quota tracking for an organization
SELECT * FROM subscription_billing.quota_tracking
WHERE organization_id = $1
LIMIT 1;

-- name: UpsertQuota :one
-- Create or update quota tracking
INSERT INTO subscription_billing.quota_tracking (
    organization_id,
    invoice_count,
    max_seats,
    period_start,
    period_end,
    last_synced_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT (organization_id)
DO UPDATE SET
    invoice_count = EXCLUDED.invoice_count,
    max_seats = EXCLUDED.max_seats,
    period_start = EXCLUDED.period_start,
    period_end = EXCLUDED.period_end,
    last_synced_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DecrementInvoiceCount :one
-- Decrement invoice count by 1 (called after successful invoice processing)
UPDATE subscription_billing.quota_tracking
SET
    invoice_count = invoice_count - 1,
    updated_at = CURRENT_TIMESTAMP
WHERE organization_id = $1
RETURNING *;

-- name: ResetQuotaForPeriod :one
-- Reset quota counters for a new billing period
UPDATE subscription_billing.quota_tracking
SET
    invoice_count = $2,
    period_start = $3,
    period_end = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE organization_id = $1
RETURNING *;

-- name: GetQuotaStatus :one
-- Get combined subscription and quota status for fast quota checks
SELECT
    s.subscription_status,
    s.current_period_start,
    s.current_period_end,
    s.cancel_at_period_end,
    q.invoice_count,
    q.max_seats,
    CASE
        WHEN s.subscription_status = 'active' AND q.invoice_count > 0
        THEN TRUE
        ELSE FALSE
    END AS can_process_invoice
FROM subscription_billing.subscriptions s
INNER JOIN subscription_billing.quota_tracking q ON s.organization_id = q.organization_id
WHERE s.organization_id = $1
LIMIT 1;

-- name: ListActiveSubscriptions :many
-- List all active subscriptions for monitoring/admin purposes
SELECT * FROM subscription_billing.subscriptions
WHERE subscription_status = 'active'
ORDER BY created_at DESC;

-- name: ListQuotasNearLimit :many
-- List organizations approaching their quota limit (for alerting)
SELECT
    q.*,
    s.subscription_status,
    s.product_name
FROM subscription_billing.quota_tracking q
INNER JOIN subscription_billing.subscriptions s ON q.organization_id = s.organization_id
WHERE
    s.subscription_status = 'active'
    AND q.invoice_count <= $1
ORDER BY q.invoice_count ASC;

-- name: UpdateAiCreditsMax :one
-- Set the period AI credit allowance (meter grant ai.tokens / metadata sync)
-- Only touches ai_credits_max so invoice counters are never clobbered.
INSERT INTO subscription_billing.quota_tracking (
    organization_id,
    ai_credits_max,
    period_start,
    period_end,
    updated_at
) VALUES (
    $1, $2, $3, $4, CURRENT_TIMESTAMP
)
ON CONFLICT (organization_id)
DO UPDATE SET
    ai_credits_max = EXCLUDED.ai_credits_max,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertAiUsage :one
-- Increment AI token/credit totals for an org's billing period (idempotent at the
-- event layer: callers only invoke this after a successful event insert)
INSERT INTO subscription_billing.ai_usage (
    organization_id,
    period_start,
    period_end,
    tokens_input,
    tokens_output,
    tokens_embedding,
    credits_used,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP
)
ON CONFLICT (organization_id, period_start)
DO UPDATE SET
    tokens_input = ai_usage.tokens_input + EXCLUDED.tokens_input,
    tokens_output = ai_usage.tokens_output + EXCLUDED.tokens_output,
    tokens_embedding = ai_usage.tokens_embedding + EXCLUDED.tokens_embedding,
    credits_used = ai_usage.credits_used + EXCLUDED.credits_used,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: InsertAiUsageEvent :execrows
-- Append an immutable AI usage event. ON CONFLICT DO NOTHING keeps recording
-- idempotent per (organization_id, request_id). Returns rows affected (0 = duplicate).
INSERT INTO subscription_billing.ai_usage_events (
    organization_id,
    feature,
    model,
    tokens_input,
    tokens_output,
    tokens_embedding,
    credits_consumed,
    request_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (organization_id, request_id) DO NOTHING;

-- name: GetAiUsageByOrgAndPeriod :one
-- Get AI usage totals for an org's billing period
SELECT * FROM subscription_billing.ai_usage
WHERE organization_id = $1 AND period_start = $2
LIMIT 1;

-- name: GetAiUsageEventsByOrg :many
-- Recent AI usage events for an org (paginated audit trail)
SELECT * FROM subscription_billing.ai_usage_events
WHERE organization_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;
