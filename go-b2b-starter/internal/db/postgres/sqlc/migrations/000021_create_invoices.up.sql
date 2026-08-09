-- 000021_create_invoices.up.sql
-- Electronic invoicing (DIAN via Siigo rail): local system of record for
-- invoices created from won deals. One invoice per (org, deal).

CREATE SCHEMA IF NOT EXISTS invoicing;

CREATE TABLE invoicing.invoices (
    id BIGSERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    deal_id INTEGER NOT NULL REFERENCES crm.deals(id) ON DELETE CASCADE,
    external_id VARCHAR(200),
    cufe TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    pdf_url TEXT,
    amount NUMERIC(14,2),
    currency VARCHAR(3) NOT NULL DEFAULT 'COP',
    notified_status VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_invoice_status CHECK (status IN ('pending', 'valid', 'invalid', 'errored')),
    CONSTRAINT uq_invoices_org_deal UNIQUE (organization_id, deal_id)
);

CREATE INDEX idx_invoices_org ON invoicing.invoices(organization_id);
CREATE INDEX idx_invoices_status ON invoicing.invoices(status);

COMMENT ON TABLE invoicing.invoices IS 'Facturas electrónicas locales (rail: Siigo/DIAN)';
COMMENT ON COLUMN invoicing.invoices.external_id IS 'ID de la factura en el proveedor (Siigo)';
COMMENT ON COLUMN invoicing.invoices.cufe IS 'CUFE emitido por DIAN (opaco; se reenvía tal cual del proveedor)';
COMMENT ON COLUMN invoicing.invoices.status IS 'pending | valid | invalid | errored';
COMMENT ON COLUMN invoicing.invoices.notified_status IS 'Último estado por el que se notificó al contacto por WhatsApp (una notificación por transición)';
