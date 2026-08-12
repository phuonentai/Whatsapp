-- 000038_create_procurement_schedules.up.sql
-- Inquiry scheduling (add-scheduled-inquiry-runs):
-- org-scoped recurring schedules that durably create inquiry runs through the
-- sibling procurement capability, plus org-level follow-up settings.
--
-- Sibling tables consumed by reference (read/written through SQLC queries
-- only): procurement.suppliers, procurement.products,
-- procurement.inquiry_runs (source='scheduled', schedule_ref reserved by the
-- sibling for this change), procurement.inquiry_recipients (followup_count is
-- the nudge guard), procurement.audit_log (append-only audit ledger).
--
-- Composite tenant FKs (organization_id, ...) mirror crm-core-data and the
-- sibling procurement tables; the UNIQUE (organization_id, id) constraints
-- back them. schedule_products/schedule_suppliers cascade with their parent
-- schedule.

CREATE SCHEMA IF NOT EXISTS procurement;

-- ============================================================
-- Schedules: run_time on days_of_week, interpreted in the org timezone
-- (agent.agent_settings.timezone); next_run_at is computed server-side.
-- ============================================================

CREATE TABLE procurement.schedules (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    run_time TIME NOT NULL,
    days_of_week SMALLINT[] NOT NULL DEFAULT ARRAY[]::SMALLINT[],
    note TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    last_run_occurrence_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedules_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT schedules_valid_days_count CHECK (array_length(days_of_week, 1) BETWEEN 1 AND 7)
);

CREATE INDEX idx_schedules_org ON procurement.schedules(organization_id);
-- Ticker hot path: due schedules of active orgs.
CREATE INDEX idx_schedules_due ON procurement.schedules(is_active, next_run_at);

-- ============================================================
-- schedule_products: FK to the sibling's procurement.products (CASCADE).
-- ============================================================

CREATE TABLE procurement.schedule_products (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    schedule_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_products_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT schedule_products_schedule_unique UNIQUE (schedule_id, product_id),
    CONSTRAINT schedule_products_schedule_org_fkey
        FOREIGN KEY (organization_id, schedule_id)
        REFERENCES procurement.schedules (organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT schedule_products_product_org_fkey
        FOREIGN KEY (organization_id, product_id)
        REFERENCES procurement.products (organization_id, id)
        ON DELETE CASCADE
);

CREATE INDEX idx_schedule_products_schedule ON procurement.schedule_products(schedule_id);

-- ============================================================
-- schedule_suppliers: FK to the sibling's procurement.suppliers (CASCADE).
-- ============================================================

CREATE TABLE procurement.schedule_suppliers (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    schedule_id INTEGER NOT NULL,
    supplier_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_suppliers_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT schedule_suppliers_schedule_unique UNIQUE (schedule_id, supplier_id),
    CONSTRAINT schedule_suppliers_schedule_org_fkey
        FOREIGN KEY (organization_id, schedule_id)
        REFERENCES procurement.schedules (organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT schedule_suppliers_supplier_org_fkey
        FOREIGN KEY (organization_id, supplier_id)
        REFERENCES procurement.suppliers (organization_id, id)
        ON DELETE CASCADE
);

CREATE INDEX idx_schedule_suppliers_schedule ON procurement.schedule_suppliers(schedule_id);

-- ============================================================
-- Follow-up settings: one row per organization (org-wide policy, applies to
-- scheduled AND manual runs). Absent row => spec defaults apply.
-- ============================================================

CREATE TABLE procurement.schedule_followups (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL UNIQUE REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    deadline_hours INTEGER NOT NULL DEFAULT 4,
    max_nudges INTEGER NOT NULL DEFAULT 1,
    message_template TEXT NOT NULL DEFAULT 'Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos.',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_followups_organization_id_id_key UNIQUE (organization_id, id),
    CONSTRAINT valid_followup_deadline CHECK (deadline_hours BETWEEN 1 AND 168),
    CONSTRAINT valid_followup_max_nudges CHECK (max_nudges BETWEEN 0 AND 5)
);

-- ============================================================
-- updated_at triggers (shared function defined in 000010)
-- ============================================================

CREATE TRIGGER trigger_schedules_updated_at
    BEFORE UPDATE ON procurement.schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_schedule_followups_updated_at
    BEFORE UPDATE ON procurement.schedule_followups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE procurement.schedules IS 'Programaciones de cotizaciones (hora + días de la semana, zona horaria de la organización)';
COMMENT ON TABLE procurement.schedule_products IS 'Productos incluidos en una programación';
COMMENT ON TABLE procurement.schedule_suppliers IS 'Proveedores incluidos en una programación';
COMMENT ON TABLE procurement.schedule_followups IS 'Configuración de recordatorios por organización (plazo, recordatorios máximos, plantilla)';

-- Council SEV-5: the follow-up sweep query (ListFollowUpCandidates) joins on
-- r.organization_id, r.status='sent', r.sent_at; the sibling 000037 migration
-- created this table without a covering index. This additive index avoids full
-- scans on the hot recipients table during the periodic sweep (~15 min ticker).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_recipients_org_status_sent_at
    ON procurement.inquiry_recipients(organization_id, status, sent_at);
