-- 000017_create_modules_tickets.up.sql
-- Sellable module registry + per-org module state + tickets (helpdesk) module.

CREATE SCHEMA IF NOT EXISTS modules;

-- ============================================================
-- Module registry (catalog of sellable modules)
-- ============================================================

CREATE TABLE modules.modules (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    granted_features JSONB NOT NULL DEFAULT '[]',
    requires JSONB NOT NULL DEFAULT '[]',
    config_schema JSONB NOT NULL DEFAULT '{}',
    is_internal BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE modules.modules IS 'Catálogo de módulos vendibles del producto';
COMMENT ON COLUMN modules.modules.granted_features IS 'Feature keys concedidos por el módulo (ej: ["tickets_module"])';
COMMENT ON COLUMN modules.modules.requires IS 'Keys de otros módulos requeridos';
COMMENT ON COLUMN modules.modules.config_schema IS 'Esquema JSON del config por organización';
COMMENT ON COLUMN modules.modules.is_internal IS 'Módulo solo para el vendor (org #0); oculto a tenants';

-- ============================================================
-- Per-org module state and configuration
-- ============================================================

CREATE TABLE modules.organization_modules (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    module_key VARCHAR(100) NOT NULL REFERENCES modules.modules(key),
    config JSONB NOT NULL DEFAULT '{}',
    enabled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, module_key)
);

CREATE INDEX idx_organization_modules_org ON modules.organization_modules(organization_id);

COMMENT ON TABLE modules.organization_modules IS 'Estado y configuración de módulos por organización';

-- ============================================================
-- Seed: the tickets module (first sellable module)
-- ============================================================

INSERT INTO modules.modules (key, name, description, granted_features, requires, config_schema, is_internal)
VALUES (
    'tickets',
    'Tickets (Helpdesk)',
    'Cola de tickets de soporte: ciclo de vida, asignación, prioridades, SLA y notas internas.',
    '["tickets_module"]',
    '[]',
    '{
        "type": "object",
        "properties": {
            "sla_hours": {"type": "object", "additionalProperties": {"type": "integer"}, "default": {"low": 48, "normal": 24, "high": 8}},
            "priorities": {"type": "array", "items": {"type": "string"}, "default": ["low", "normal", "high"]},
            "tags": {"type": "array", "items": {"type": "string"}, "default": []}
        }
    }',
    false
);

-- ============================================================
-- Tickets (helpdesk) — org-scoped, optional contact/conversation links
-- ============================================================

CREATE TABLE crm.tickets (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    contact_id INTEGER REFERENCES crm.contacts(id) ON DELETE SET NULL,
    conversation_id INTEGER REFERENCES crm.conversations(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    tags JSONB NOT NULL DEFAULT '[]',
    assignee_stytch_member_id TEXT,
    sla_due_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_ticket_status CHECK (status IN ('open', 'in_progress', 'waiting_customer', 'resolved', 'cancelled')),
    CONSTRAINT valid_ticket_priority CHECK (priority IN ('low', 'normal', 'high'))
);

CREATE INDEX idx_tickets_org ON crm.tickets(organization_id);
CREATE INDEX idx_tickets_status ON crm.tickets(organization_id, status);
CREATE INDEX idx_tickets_assignee ON crm.tickets(organization_id, assignee_stytch_member_id);

COMMENT ON TABLE crm.tickets IS 'Tickets de soporte (módulo vendible)';
COMMENT ON COLUMN crm.tickets.assignee_stytch_member_id IS 'Miembro asignado vía stytch_member_id (sin tabla local de miembros)';
COMMENT ON COLUMN crm.tickets.sla_due_at IS 'Fecha límite SLA calculada según config del módulo (por prioridad)';

-- ============================================================
-- Ticket events — append-only history
-- ============================================================

CREATE TABLE crm.ticket_events (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL REFERENCES crm.tickets(id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL,
    actor_stytch_member_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_ticket_event_type CHECK (event_type IN ('created', 'status_changed', 'assigned', 'unassigned', 'priority_changed', 'note_internal', 'tags_changed'))
);

CREATE INDEX idx_ticket_events_ticket ON crm.ticket_events(ticket_id, created_at);

COMMENT ON TABLE crm.ticket_events IS 'Historial append-only de eventos del ticket';
