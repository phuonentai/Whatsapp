-- 000026_create_outbox_events.up.sql
-- Durable outbox for asynchronous event delivery (WhatsApp message events,
-- outbound Graph API send jobs). Guarantees at-least-once delivery after
-- commit: events are persisted in the same transaction as their trigger.

CREATE TABLE public.outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    organization_id INTEGER REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dispatched_at TIMESTAMPTZ,
    CONSTRAINT outbox_events_status_check CHECK (status IN ('pending', 'dispatched', 'dead_letter'))
);

CREATE INDEX idx_outbox_events_dispatch ON public.outbox_events(status, next_attempt_at);
CREATE INDEX idx_outbox_events_org_type ON public.outbox_events(organization_id, event_type);

COMMENT ON TABLE public.outbox_events IS 'Durable event outbox: events committed with their source transaction and dispatched asynchronously with retry and dead-letter';
COMMENT ON COLUMN public.outbox_events.status IS 'pending | dispatched | dead_letter';
COMMENT ON COLUMN public.outbox_events.next_attempt_at IS 'Backoff timestamp for retry scheduling; dispatcher claims rows where next_attempt_at <= now()';
