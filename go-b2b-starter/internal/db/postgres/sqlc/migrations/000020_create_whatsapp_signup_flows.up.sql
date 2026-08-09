CREATE TABLE whatsapp.signup_flows (
    id BIGSERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL UNIQUE REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'exchanging',
    step VARCHAR(40),
    error_code VARCHAR(100),
    retry_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_signup_flow_status CHECK (status IN ('exchanging', 'registering', 'verifying', 'connected', 'failed'))
);

COMMENT ON TABLE whatsapp.signup_flows IS 'Embedded signup provisioning state per organization (one row per org)';
COMMENT ON COLUMN whatsapp.signup_flows.status IS 'exchanging | registering | verifying | connected | failed';
COMMENT ON COLUMN whatsapp.signup_flows.step IS 'Current sub-step for mid-flow recovery after token code consumption';
COMMENT ON COLUMN whatsapp.signup_flows.error_code IS 'Meta/backend error code on terminal failure';
COMMENT ON COLUMN whatsapp.signup_flows.metadata IS 'Provisioning metadata: waba_id, phone_number_id, coexistence, ticket_id';
