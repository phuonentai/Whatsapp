-- 000028_create_org_connections.up.sql
-- Per-organization Siigo invoicing connection: onboarding state machine +
-- encrypted credentials. One row per organization. Credentials are stored
-- AES-256-GCM encrypted at rest (envelope key from env SIIGO_MASTER_KEY);
-- never plaintext, never in logs, never returned by any API.

CREATE TABLE invoicing.org_connections (
    organization_id INTEGER PRIMARY KEY REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL DEFAULT 'none',
    status VARCHAR(30) NOT NULL DEFAULT 'none',
    client_id_enc BYTEA,
    client_secret_enc BYTEA,
    nit VARCHAR(30),
    siigo_company_name VARCHAR(255),
    last_error TEXT,
    paused_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_connection_provider CHECK (provider IN ('siigo', 'none')),
    CONSTRAINT valid_connection_status CHECK (status IN (
        'none', 'awaiting_setup', 'connected', 'numeracion_ok',
        'sandbox_ok', 'live', 'paused', 'invoicing_disabled'
    ))
);

CREATE INDEX idx_org_connections_status ON invoicing.org_connections(status);

COMMENT ON TABLE invoicing.org_connections IS 'Conexión de facturación por organización (rail: Siigo)';
COMMENT ON COLUMN invoicing.org_connections.client_id_enc IS 'client_id cifrado (AES-256-GCM, nonce+tag incluidos)';
COMMENT ON COLUMN invoicing.org_connections.client_secret_enc IS 'client_secret cifrado (AES-256-GCM, nonce+tag incluidos)';
COMMENT ON COLUMN invoicing.org_connections.status IS 'none | awaiting_setup | connected | numeracion_ok | sandbox_ok | live | paused | invoicing_disabled';
