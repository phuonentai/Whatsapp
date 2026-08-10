-- 000029_create_campaign_segments.up.sql
-- WhatsApp campaign audience layer: saved contact segments, campaign drafts,
-- and the audience snapshot (bill of materials for the future scheduler).

-- ============================================================
-- Segments: saved, whitelisted contact filter specs
-- ============================================================

CREATE TABLE crm.segments (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    nombre VARCHAR(200) NOT NULL,
    filter_spec JSONB NOT NULL DEFAULT '[]',
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE crm.segments IS 'Segmentos guardados de contactos (audiencia de campañas)';
COMMENT ON COLUMN crm.segments.filter_spec IS 'JSON array de filtros validados: [{"field","op","value"}] con semántica AND';
COMMENT ON COLUMN crm.segments.created_by IS 'Stytch member_id que creó el segmento';

-- Org-scoped uniqueness for composite FKs (000016 pattern)
ALTER TABLE crm.segments ADD CONSTRAINT segments_organization_id_id_key UNIQUE (organization_id, id);

CREATE INDEX idx_segments_org ON crm.segments(organization_id);

CREATE TRIGGER trigger_segments_updated_at
    BEFORE UPDATE ON crm.segments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Campaigns: draft state machine, one segment each
-- ============================================================

CREATE TABLE crm.campaigns (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    nombre VARCHAR(200) NOT NULL,
    segment_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    recipient_count INTEGER NOT NULL DEFAULT 0,
    launched_at TIMESTAMPTZ,
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_campaign_status CHECK (status IN ('draft', 'ready')),
    CONSTRAINT campaigns_organization_id_segment_id_fkey
        FOREIGN KEY (organization_id, segment_id)
        REFERENCES crm.segments(organization_id, id)
        ON DELETE RESTRICT
);

COMMENT ON TABLE crm.campaigns IS 'Campañas de WhatsApp (v1: draft + snapshot de audiencia)';
COMMENT ON COLUMN crm.campaigns.status IS 'Estado: draft (no lanzada) o ready (audiencia capturada)';
COMMENT ON COLUMN crm.campaigns.created_by IS 'Stytch member_id que creó/lanzó la campaña';

ALTER TABLE crm.campaigns ADD CONSTRAINT campaigns_organization_id_id_key UNIQUE (organization_id, id);

CREATE INDEX idx_campaigns_org ON crm.campaigns(organization_id);
CREATE INDEX idx_campaigns_segment ON crm.campaigns(segment_id);

CREATE TRIGGER trigger_campaigns_updated_at
    BEFORE UPDATE ON crm.campaigns FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Campaign recipients: audience snapshot (one row per contact)
-- ============================================================

CREATE TABLE crm.campaign_recipients (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES crm.campaigns(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES crm.contacts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    whatsapp_message_id VARCHAR(100),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_recipient_status CHECK (status IN ('pending', 'sent', 'failed', 'skipped')),
    CONSTRAINT campaign_recipients_campaign_id_contact_id_key UNIQUE (campaign_id, contact_id)
);

COMMENT ON TABLE crm.campaign_recipients IS 'Instantánea de audiencia de campaña (bill of materials del envío)';
COMMENT ON COLUMN crm.campaign_recipients.status IS 'Estado por destinatario: pending, sent, failed, skipped';

CREATE INDEX idx_campaign_recipients_campaign ON crm.campaign_recipients(campaign_id);
CREATE INDEX idx_campaign_recipients_contact ON crm.campaign_recipients(contact_id);

CREATE TRIGGER trigger_campaign_recipients_updated_at
    BEFORE UPDATE ON crm.campaign_recipients FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Seed: sellable "campaigns" module (grants the crm_campaigns feature)
-- ============================================================

INSERT INTO modules.modules (key, name, description, granted_features, requires, config_schema, is_internal)
VALUES (
    'campaigns',
    'Campañas de WhatsApp',
    'Segmentos de audiencia y campañas: filtros guardados, vista previa y captura de audiencia para envíos masivos.',
    '["crm_campaigns"]',
    '["crm"]',
    '{}',
    false
)
ON CONFLICT (key) DO NOTHING;
