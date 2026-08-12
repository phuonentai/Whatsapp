-- 000037_create_procurement_schema.up.sql
-- Procurement capability (add-supplier-inquiry-agent):
-- suppliers registry (contact-linked, NIT persona jurídica), product catalog,
-- inquiry runs with durable fan-out recipients/responses, human-approved
-- orders, and an append-only audit log.
--
-- Composite tenant FKs (organization_id, ...) mirror crm-core-data; the
-- UNIQUE (organization_id, id) constraints back them. suppliers.contact_id is
-- ON DELETE RESTRICT so procurement history (responses, orders) survives
-- contact edits; deactivation (is_active = false) is the supported lifecycle.

CREATE SCHEMA IF NOT EXISTS procurement;

-- ============================================================
-- Suppliers: a supplier IS a CRM contact (NIT legal entity)
-- ============================================================

CREATE TABLE procurement.suppliers (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL,
    nit VARCHAR(50) NOT NULL,
    delivery_days INTEGER,
    min_order_amount NUMERIC(14,2),
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT suppliers_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT suppliers_org_nit_unique UNIQUE (organization_id, nit),
    CONSTRAINT suppliers_contact_org_fkey
        FOREIGN KEY (organization_id, contact_id)
        REFERENCES crm.contacts (organization_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_suppliers_org ON procurement.suppliers(organization_id);

-- ============================================================
-- Products: org-scoped SKU catalog. Deactivation preserves history.
-- ============================================================

CREATE TABLE procurement.products (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT 'und',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT products_org_sku_unique UNIQUE (organization_id, sku)
);

CREATE INDEX idx_products_org ON procurement.products(organization_id);

-- ============================================================
-- Inquiry runs: flow row mirroring agent.conversation_flows.
-- source = 'manual' in this change; schedule_ref reserved for the
-- future scheduled-runs change.
-- ============================================================

CREATE TABLE procurement.inquiry_runs (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    source VARCHAR(20) NOT NULL DEFAULT 'manual',
    schedule_ref BIGINT,
    nota TEXT,
    created_by_member_id VARCHAR(255),
    sent_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT runs_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT valid_run_status CHECK (status IN (
        'draft', 'sending', 'awaiting_responses', 'completed',
        'partially_answered', 'failed', 'escalated', 'cancelled'
    )),
    CONSTRAINT valid_run_source CHECK (source IN ('manual', 'scheduled'))
);

CREATE INDEX idx_runs_org_status ON procurement.inquiry_runs(organization_id, status);

-- ============================================================
-- Recipients: one row per supplier per run; durable send state machine.
-- ============================================================

CREATE TABLE procurement.inquiry_recipients (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    run_id INTEGER NOT NULL,
    supplier_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    drafted_message TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    provider_message_id VARCHAR(100),
    sent_at TIMESTAMPTZ,
    answered_at TIMESTAMPTZ,
    followup_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recipients_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT recipients_run_org_fkey
        FOREIGN KEY (organization_id, run_id)
        REFERENCES procurement.inquiry_runs (organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT recipients_supplier_org_fkey
        FOREIGN KEY (organization_id, supplier_id)
        REFERENCES procurement.suppliers (organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT recipients_contact_org_fkey
        FOREIGN KEY (organization_id, contact_id)
        REFERENCES crm.contacts (organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT valid_recipient_status CHECK (status IN ('pending', 'sent', 'answered', 'timed_out', 'failed'))
);

-- Hot inbound lookup: every whatsapp.message.received event resolves whether
-- the sender is an active run recipient (procurement subscriber + agent skip).
CREATE INDEX idx_recipients_org_contact ON procurement.inquiry_recipients(organization_id, contact_id);
CREATE INDEX idx_recipients_run ON procurement.inquiry_recipients(run_id);

-- ============================================================
-- Responses: structured extraction per replied message. Unique per
-- (recipient_id, raw_message_id) makes webhook redelivery idempotent.
-- ============================================================

CREATE TABLE procurement.inquiry_responses (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    recipient_id INTEGER NOT NULL,
    raw_message_id VARCHAR(100) NOT NULL,
    extracted JSONB NOT NULL DEFAULT '{}',
    resumen TEXT,
    confidence DOUBLE PRECISION,
    requiere_humano BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT responses_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT responses_recipient_org_fkey
        FOREIGN KEY (organization_id, recipient_id)
        REFERENCES procurement.inquiry_recipients (organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT responses_recipient_message_unique UNIQUE (recipient_id, raw_message_id)
);

CREATE INDEX idx_responses_recipient ON procurement.inquiry_responses(recipient_id);

-- ============================================================
-- Orders: human-approved placements. UNIQUE (run_id, supplier_id) makes
-- retried approval POSTs idempotent (D13).
-- ============================================================

CREATE TABLE procurement.orders (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    run_id INTEGER NOT NULL,
    supplier_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    negocio_id INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'placed',
    items JSONB NOT NULL DEFAULT '[]',
    notes TEXT,
    confirm_message TEXT,
    blocked_reason VARCHAR(50),
    created_by_member_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT orders_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT orders_run_supplier_unique UNIQUE (run_id, supplier_id),
    CONSTRAINT orders_run_org_fkey
        FOREIGN KEY (organization_id, run_id)
        REFERENCES procurement.inquiry_runs (organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT orders_supplier_org_fkey
        FOREIGN KEY (organization_id, supplier_id)
        REFERENCES procurement.suppliers (organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT orders_contact_org_fkey
        FOREIGN KEY (organization_id, contact_id)
        REFERENCES crm.contacts (organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT valid_order_status CHECK (status IN ('placed', 'confirm_sent', 'send_blocked', 'confirm_failed'))
);

CREATE INDEX idx_orders_org ON procurement.orders(organization_id);
CREATE INDEX idx_orders_run ON procurement.orders(run_id);

-- ============================================================
-- Append-only audit log: supplier_created, consent_grant, order_placed,
-- and skip blocks (kill_switch / consent_withdrawn / run_cancelled).
-- ============================================================

CREATE TABLE procurement.audit_log (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INTEGER,
    action VARCHAR(50) NOT NULL,
    decision VARCHAR(10) NOT NULL DEFAULT 'allow',
    reason VARCHAR(50),
    member_id VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_audit_decision CHECK (decision IN ('allow', 'deny', 'skip'))
);

CREATE INDEX idx_audit_org_created ON procurement.audit_log(organization_id, created_at DESC);

-- ============================================================
-- updated_at triggers (shared function defined in 000010)
-- ============================================================

CREATE TRIGGER trigger_suppliers_updated_at
    BEFORE UPDATE ON procurement.suppliers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_products_updated_at
    BEFORE UPDATE ON procurement.products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_runs_updated_at
    BEFORE UPDATE ON procurement.inquiry_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_recipients_updated_at
    BEFORE UPDATE ON procurement.inquiry_recipients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON procurement.orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE procurement.suppliers IS 'Proveedores: un proveedor ES un contacto CRM (NIT persona jurídica)';
COMMENT ON TABLE procurement.inquiry_runs IS 'Ejecuciones de cotización (flujo de procurement)';
COMMENT ON TABLE procurement.audit_log IS 'Auditoría append-only de procurement (alta, consentimiento, órdenes, bloqueos)';
