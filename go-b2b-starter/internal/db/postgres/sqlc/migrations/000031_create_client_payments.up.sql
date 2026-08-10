-- 000031_create_client_payments.up.sql
-- Client-facing one-shot MercadoPago payments (payment links sent inside
-- WhatsApp). Local system of record for customer payments made to the SME;
-- stores only MercadoPago preference/payment identifiers, never tokens,
-- card data, or wallet credentials.

CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE payments.client_payments (
    id BIGSERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    deal_id INTEGER NOT NULL REFERENCES crm.deals(id) ON DELETE CASCADE,
    invoice_id INTEGER REFERENCES invoicing.invoices(id) ON DELETE SET NULL,
    amount_cop BIGINT NOT NULL,
    commission_cop BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'COP',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    mp_preference_id VARCHAR(100),
    mp_payment_id VARCHAR(100),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_payment_status CHECK (status IN ('pending', 'paid', 'failed', 'expired')),
    CONSTRAINT uq_client_payments_preference UNIQUE (mp_preference_id),
    CONSTRAINT uq_client_payments_payment UNIQUE (mp_payment_id)
);

CREATE INDEX idx_client_payments_org ON payments.client_payments(organization_id);
CREATE INDEX idx_client_payments_deal ON payments.client_payments(deal_id);
CREATE INDEX idx_client_payments_status ON payments.client_payments(status);

COMMENT ON TABLE payments.client_payments IS 'Pagos de clientes del SMB (rail: MercadoPago, one-shot payment links)';
COMMENT ON COLUMN payments.client_payments.amount_cop IS 'Monto base en COP (por cobrar del SMB)';
COMMENT ON COLUMN payments.client_payments.commission_cop IS 'Comisión de plataforma en COP (marcado sobre amount_cop)';
COMMENT ON COLUMN payments.client_payments.status IS 'pending | paid | failed | expired';
