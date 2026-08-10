-- 000030_onboarding_data.up.sql
-- Onboarding data for Siigo connections: numeration confirmation snapshot
-- and customer-import run log. Also relaxes invoicing.invoices.deal_id to
-- allow sandbox test invoices (no deal), keeping the one-invoice-per-deal
-- uniqueness as a partial unique index.

CREATE TABLE invoicing.org_numerations (
    organization_id INTEGER PRIMARY KEY REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    mode VARCHAR(10) NOT NULL DEFAULT 'auto' CHECK (mode IN ('auto', 'manual')),
    resolution_id VARCHAR(50),
    prefijo VARCHAR(10),
    next_number VARCHAR(20),
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE invoicing.org_numerations IS 'Confirmación de numeración (resolución DIAN) por organización; snapshot en el momento de confirmación';
COMMENT ON COLUMN invoicing.org_numerations.mode IS 'auto: Siigo asigna consecutivo; manual: la plataforma obtiene el próximo número (reservado)';

CREATE TABLE invoicing.import_runs (
    id BIGSERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    kind VARCHAR(10) NOT NULL CHECK (kind IN ('preview', 'confirm', 'delta')),
    counts JSONB NOT NULL DEFAULT '{}',
    error TEXT,
    pulled_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_runs_org ON invoicing.import_runs(organization_id, id DESC);

COMMENT ON TABLE invoicing.import_runs IS 'Registro de corridas de importación de clientes (preview/confirm/delta)';

-- Sandbox test invoices: allow rows without a deal.
ALTER TABLE invoicing.invoices ALTER COLUMN deal_id DROP NOT NULL;
ALTER TABLE invoicing.invoices DROP CONSTRAINT IF EXISTS uq_invoices_org_deal;
CREATE UNIQUE INDEX uq_invoices_org_deal ON invoicing.invoices (organization_id, deal_id) WHERE deal_id IS NOT NULL;
