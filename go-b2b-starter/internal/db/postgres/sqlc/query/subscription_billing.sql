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
-- Create or update quota tracking.
-- Non-destructive: invoice_count/max_seats of -1 mean "no new value"
-- (metadata/status-only updates, e.g. customer.updated without a count). On
-- conflict the stored values are preserved atomically, so a concurrent
-- decrement can never be clobbered back; on insert -1 normalizes to 0.
-- ai_credits_max is intentionally NOT touched here (UpdateAiCreditsMax owns it).
INSERT INTO subscription_billing.quota_tracking (
    organization_id,
    invoice_count,
    max_seats,
    period_start,
    period_end,
    last_synced_at,
    updated_at
) VALUES (
    sqlc.arg(organization_id),
    COALESCE(NULLIF(sqlc.arg(invoice_count)::int, -1), 0),
    COALESCE(NULLIF(sqlc.arg(max_seats)::int, -1), 0),
    sqlc.arg(period_start),
    sqlc.arg(period_end),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (organization_id)
DO UPDATE SET
    invoice_count = COALESCE(NULLIF(sqlc.arg(invoice_count)::int, -1), quota_tracking.invoice_count),
    max_seats = COALESCE(NULLIF(sqlc.arg(max_seats)::int, -1), quota_tracking.max_seats),
    period_start = EXCLUDED.period_start,
    period_end = EXCLUDED.period_end,
    last_synced_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DecrementInvoiceCount :one
-- Decrement invoice count by 1 (called after successful invoice processing).
-- Bounded: guarded by invoice_count > 0 so concurrent consumption can never
-- drive the count negative. When the count is already 0 no row is updated and
-- sqlc reports ErrNoRows, which callers map to a quota-exhausted error.
UPDATE subscription_billing.quota_tracking
SET
    invoice_count = invoice_count - 1,
    updated_at = CURRENT_TIMESTAMP
WHERE organization_id = $1 AND invoice_count > 0
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
-- Get combined subscription and quota status for fast quota checks.
-- LEFT JOIN: the subscription and quota rows are written in separate
-- non-transactional upserts, so a valid subscription may temporarily have no
-- quota row. The LEFT JOIN with zeroed quota defaults (invoice_count 0) keeps
-- the subscription visible with its real status instead of masking it as "none".
SELECT
    s.subscription_status,
    s.current_period_start,
    s.current_period_end,
    s.cancel_at_period_end,
    COALESCE(q.invoice_count, 0) AS invoice_count,
    q.max_seats,
    CASE
        WHEN s.subscription_status IN ('active','trialing') AND COALESCE(q.invoice_count, 0) > 0
        THEN TRUE
        ELSE FALSE
    END AS can_process_invoice
FROM subscription_billing.subscriptions s
LEFT JOIN subscription_billing.quota_tracking q ON s.organization_id = q.organization_id
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

-- name: InsertLocalTrial :one
-- Idempotent local trial subscription seed. ON CONFLICT DO NOTHING ensures a
-- provider-backed row is never overwritten and a bootstrap retry is a no-op.
-- Synthetic NOT NULL placeholders: external_customer_id='local-trial',
-- subscription_id='local-trial-'||orgID (unique per org), product_id='trial'.
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
    metadata
) VALUES (
    $1,
    'local-trial',
    'local-trial-' || $1::text,
    'trialing',
    'trial',
    'Trial',
    'Free Trial',
    NOW(),
    $2,
    '{}'
)
ON CONFLICT (organization_id) DO NOTHING
RETURNING *;

-- name: InsertLocalTrialQuota :one
-- Idempotent trial quota seed with zero grants. Real columns only:
-- invoice_count=0 (count-down: can_process_invoice=false while paywall passes).
INSERT INTO subscription_billing.quota_tracking (
    organization_id,
    invoice_count,
    period_start,
    period_end
) VALUES (
    $1,
    0,
    NOW(),
    $2
)
ON CONFLICT (organization_id) DO NOTHING
RETURNING *;

-- name: ListExpiredTrials :many
-- Global monitoring: trial rows past current_period_end, ordered by org.
SELECT * FROM subscription_billing.subscriptions
WHERE subscription_status = 'trialing'
  AND current_period_end < NOW()
ORDER BY organization_id;

-- name: ListExpiredTrialByOrg :one
-- Tenant-safe org-scoped variant (at most one row — organization_id is UNIQUE).
SELECT * FROM subscription_billing.subscriptions
WHERE organization_id = $1
  AND subscription_status = 'trialing'
  AND current_period_end < NOW()
LIMIT 1;
