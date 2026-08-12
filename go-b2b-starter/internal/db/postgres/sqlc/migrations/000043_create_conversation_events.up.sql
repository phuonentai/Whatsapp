-- 000043_create_conversation_events.up.sql
-- conversation-row-scoping — audit ledger append-only de eventos de
-- conversación (re-asignaciones). Patrón crm.ticket_events (000017);
-- alternativas evaluadas: procurement.audit_log (000037) y crm.actividades
-- (000013) — se elige tabla dedicada por simetría con tickets y para no
-- contaminar el timeline de actividades con eventos de asignación.

CREATE TABLE crm.conversation_events (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    conversation_id INTEGER NOT NULL REFERENCES crm.conversations(id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL,
    actor_stytch_member_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_conversation_event_type CHECK (event_type IN ('assigned', 'unassigned', 'status_changed'))
);

CREATE INDEX idx_conversation_events_conversation ON crm.conversation_events(conversation_id, created_at);
CREATE INDEX idx_conversation_events_org ON crm.conversation_events(organization_id, created_at);

COMMENT ON TABLE crm.conversation_events IS 'Historial append-only de eventos de conversación (re-asignaciones y cambios de estado)';
COMMENT ON COLUMN crm.conversation_events.actor_stytch_member_id IS 'stytch_member_id del actor (FK lógico a Stytch; sin tabla local de miembros)';
COMMENT ON COLUMN crm.conversation_events.payload IS 'Detalle del evento, p. ej. {"from": <member_id>, "to": <member_id>}';
