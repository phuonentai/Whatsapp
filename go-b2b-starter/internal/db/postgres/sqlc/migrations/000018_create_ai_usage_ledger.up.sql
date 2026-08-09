-- 000018_create_ai_usage_ledger.up.sql
-- AI Credit / Token Consumption Ledger: per-org period totals + append-only event audit trail.

-- ============================================================
-- AI usage running totals (one row per org per billing period)
-- ============================================================

CREATE TABLE subscription_billing.ai_usage (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    tokens_input BIGINT NOT NULL DEFAULT 0,
    tokens_output BIGINT NOT NULL DEFAULT 0,
    tokens_embedding BIGINT NOT NULL DEFAULT 0,
    credits_used INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, period_start)
);

CREATE INDEX idx_ai_usage_org_period ON subscription_billing.ai_usage(organization_id, period_start);

COMMENT ON TABLE subscription_billing.ai_usage IS 'Totales corrientes de consumo de tokens/credits IA por organización y periodo';
COMMENT ON COLUMN subscription_billing.ai_usage.credits_used IS 'Créditos consumidos en el periodo (conversión token->crédito)';

-- ============================================================
-- Append-only AI usage event audit trail
-- ============================================================

CREATE TABLE subscription_billing.ai_usage_events (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    feature VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    tokens_input BIGINT NOT NULL DEFAULT 0,
    tokens_output BIGINT NOT NULL DEFAULT 0,
    tokens_embedding BIGINT NOT NULL DEFAULT 0,
    credits_consumed INT NOT NULL DEFAULT 0,
    request_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, request_id)
);

CREATE INDEX idx_ai_usage_events_org_created ON subscription_billing.ai_usage_events(organization_id, created_at);

COMMENT ON TABLE subscription_billing.ai_usage_events IS 'Auditoría append-only de consumo IA por organización';
COMMENT ON COLUMN subscription_billing.ai_usage_events.request_id IS 'Identificador idempotente (ON CONFLICT DO NOTHING evita dobles registros)';

-- ============================================================
-- Period credit allowance on quota tracking
-- ============================================================

ALTER TABLE subscription_billing.quota_tracking
    ADD COLUMN ai_credits_max INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN subscription_billing.quota_tracking.ai_credits_max IS 'Créditos IA disponibles en el periodo (metadata de producto / meter grant ai.tokens)';
