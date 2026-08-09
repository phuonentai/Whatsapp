-- 000019_create_agent_schema.up.sql
-- Agentic WhatsApp Assistant: linear pipeline + per-org settings + guardrails
-- + compliance consent fields. Replaces the earlier DAG design (flow_nodes,
-- flow_edges, autonomy_level) with the confirmed copilot/autopilot scope.

CREATE SCHEMA IF NOT EXISTS agent;

-- ============================================================
-- Conversation flows (one per agent run; linear pipeline state)
-- ============================================================

CREATE TABLE agent.conversation_flows (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    conversation_id INTEGER NOT NULL REFERENCES crm.conversations(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES crm.contacts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_flow_status CHECK (status IN ('running', 'awaiting_human', 'succeeded', 'failed', 'cancelled'))
);

CREATE INDEX idx_agent_flows_conv_status ON agent.conversation_flows(conversation_id, status);

-- ============================================================
-- Per-organization agent settings
-- ============================================================

CREATE TABLE agent.agent_settings (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    mode VARCHAR(10) NOT NULL DEFAULT 'copilot',
    tone VARCHAR(10) NOT NULL DEFAULT 'formal',
    brand_voice TEXT,
    autopilot_start TIME,
    autopilot_end TIME,
    timezone VARCHAR(64) NOT NULL DEFAULT 'America/Bogota',
    kill_switch BOOLEAN NOT NULL DEFAULT FALSE,
    max_daily_messages INTEGER NOT NULL DEFAULT 0,
    consent_required BOOLEAN NOT NULL DEFAULT TRUE,
    consent_template TEXT,
    guardrails JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id),
    CONSTRAINT valid_mode CHECK (mode IN ('copilot', 'autopilot')),
    CONSTRAINT valid_tone CHECK (tone IN ('formal', 'casual'))
);

-- ============================================================
-- Agent suggestions (pending/approved/rejected drafts and escalations)
-- ============================================================

CREATE TABLE agent.agent_suggestions (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    conversation_id INTEGER NOT NULL REFERENCES crm.conversations(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES crm.contacts(id) ON DELETE CASCADE,
    flow_id INTEGER REFERENCES agent.conversation_flows(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'reply',
    body TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    source VARCHAR(30) NOT NULL DEFAULT 'copilot',
    approved_by_member_id VARCHAR(255),
    whatsapp_message_id VARCHAR(100),
    request_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_suggestion_type CHECK (type IN ('reply', 'escalation')),
    CONSTRAINT valid_suggestion_status CHECK (status IN ('pending', 'approved', 'rejected', 'superseded')),
    CONSTRAINT valid_suggestion_source CHECK (source IN ('copilot', 'autopilot_fallback', 'escalation'))
);

CREATE INDEX idx_agent_suggestions_org_status ON agent.agent_suggestions(organization_id, status, created_at DESC);

-- ============================================================
-- Append-only agent action audit
-- ============================================================

CREATE TABLE agent.agent_actions (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    flow_id INTEGER,
    action VARCHAR(50) NOT NULL,
    decision VARCHAR(10) NOT NULL,
    policy_input JSONB NOT NULL DEFAULT '{}',
    reasons JSONB NOT NULL DEFAULT '[]',
    approved_by_member_id VARCHAR(255),
    whatsapp_message_id VARCHAR(100),
    request_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_decision CHECK (decision IN ('allow', 'deny', 'skip'))
);

CREATE INDEX idx_agent_actions_org_created ON agent.agent_actions(organization_id, created_at DESC);

-- ============================================================
-- Compliance (Ley 1581): consent state on contacts
-- ============================================================

ALTER TABLE crm.contacts
    ADD COLUMN consent_status VARCHAR(10) NOT NULL DEFAULT 'none',
    ADD COLUMN consented_at TIMESTAMP;

ALTER TABLE crm.contacts
    ADD CONSTRAINT valid_consent_status CHECK (consent_status IN ('none', 'requested', 'granted', 'withdrawn'));

-- ============================================================
-- updated_at triggers (shared function defined in 000010)
-- ============================================================

CREATE TRIGGER trigger_conversation_flows_updated_at
    BEFORE UPDATE ON agent.conversation_flows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_agent_settings_updated_at
    BEFORE UPDATE ON agent.agent_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_agent_suggestions_updated_at
    BEFORE UPDATE ON agent.agent_suggestions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE agent.agent_actions IS 'Auditoría append-only de decisiones de gobernanza del agente';
COMMENT ON COLUMN crm.contacts.consent_status IS 'Estado de consentimiento Ley 1581: none, requested, granted, withdrawn';
