-- 000031_add_instagram_schema.up.sql
-- Instagram DM integration: generalize CRM to multi-channel and add
-- Instagram config + webhook log tables.

-- ============================================================
-- 1. crm.messages: multi-channel
-- ============================================================

ALTER TABLE crm.messages
    ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'whatsapp';

ALTER TABLE crm.messages
    ADD CONSTRAINT valid_msg_channel CHECK (channel IN ('whatsapp', 'instagram'));

ALTER TABLE crm.messages
    RENAME COLUMN whatsapp_message_id TO provider_message_id;

DROP INDEX IF EXISTS crm.idx_messages_whatsapp_id;

CREATE UNIQUE INDEX idx_messages_provider_id
    ON crm.messages(organization_id, channel, provider_message_id)
    WHERE provider_message_id IS NOT NULL;

-- ============================================================
-- 2. crm.conversations: multi-channel
-- ============================================================

ALTER TABLE crm.conversations
    ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'whatsapp';

ALTER TABLE crm.conversations
    ADD CONSTRAINT valid_conv_channel CHECK (channel IN ('whatsapp', 'instagram'));

DROP INDEX IF EXISTS crm.conversations_one_active_per_contact;

CREATE UNIQUE INDEX conversations_one_active_per_contact
    ON crm.conversations (organization_id, contact_id, channel)
    WHERE status = 'active';

-- ============================================================
-- 3. crm.contacts: nullable phone + Instagram identity
-- ============================================================

ALTER TABLE crm.contacts
    ALTER COLUMN phone_number DROP NOT NULL;

ALTER TABLE crm.contacts
    ADD COLUMN instagram_user_id VARCHAR(50),
    ADD COLUMN instagram_username VARCHAR(255);

ALTER TABLE crm.contacts
    DROP CONSTRAINT IF EXISTS contacts_organization_id_phone_number_key;

DROP INDEX IF EXISTS crm.idx_contacts_org_phone;

CREATE UNIQUE INDEX idx_contacts_org_phone
    ON crm.contacts(organization_id, phone_number)
    WHERE phone_number IS NOT NULL;

CREATE UNIQUE INDEX idx_contacts_org_ig_user
    ON crm.contacts(organization_id, instagram_user_id)
    WHERE instagram_user_id IS NOT NULL;

ALTER TABLE crm.contacts
    DROP CONSTRAINT IF EXISTS valid_source;

ALTER TABLE crm.contacts
    ADD CONSTRAINT valid_source CHECK (source IN ('whatsapp', 'instagram', 'manual', 'import', 'api'));

-- ============================================================
-- 3b. crm.activities: instagram_message type
-- ============================================================

ALTER TABLE crm.activities
    DROP CONSTRAINT IF EXISTS valid_tipo;

ALTER TABLE crm.activities
    ADD CONSTRAINT valid_tipo CHECK (tipo IN ('nota', 'llamada', 'correo', 'reunion', 'tarea', 'whatsapp_message', 'instagram_message', 'sistema'));

-- ============================================================
-- 4. whatsapp.instagram_configs
-- ============================================================

CREATE TABLE whatsapp.instagram_configs (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL UNIQUE REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    ig_user_id VARCHAR(50) NOT NULL UNIQUE,
    ig_username VARCHAR(255),
    fb_page_id VARCHAR(100),
    access_token VARCHAR(500) NOT NULL,
    token_expires_at TIMESTAMPTZ,
    webhook_secret VARCHAR(255) NOT NULL,
    verify_token VARCHAR(255) NOT NULL,
    api_version VARCHAR(20) NOT NULL DEFAULT 'v21.0',
    graph_api_url VARCHAR(255) NOT NULL DEFAULT 'https://graph.facebook.com',
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instagram_configs_org ON whatsapp.instagram_configs(organization_id);
CREATE INDEX idx_instagram_configs_ig_user ON whatsapp.instagram_configs(ig_user_id) WHERE is_active = true;

-- ============================================================
-- 5. whatsapp.instagram_webhook_logs
-- ============================================================

CREATE TABLE whatsapp.instagram_webhook_logs (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'received',
    event_type VARCHAR(50),
    ig_user_id VARCHAR(50),
    raw_headers JSONB DEFAULT '{}',
    raw_body JSONB DEFAULT '{}',
    error_message TEXT,
    processed_at TIMESTAMP,
    delivery_key VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instagram_webhook_logs_org ON whatsapp.instagram_webhook_logs(organization_id);
CREATE INDEX idx_instagram_webhook_logs_created ON whatsapp.instagram_webhook_logs(created_at DESC);

CREATE UNIQUE INDEX idx_instagram_webhook_logs_delivery_key
    ON whatsapp.instagram_webhook_logs(ig_user_id, delivery_key)
    WHERE delivery_key IS NOT NULL;

-- ============================================================
-- Triggers
-- ============================================================

CREATE TRIGGER trigger_instagram_configs_updated_at
    BEFORE UPDATE ON whatsapp.instagram_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE whatsapp.instagram_configs IS 'Maps Instagram business ig_user_id to organizations with webhook secrets and messaging tokens';
COMMENT ON TABLE whatsapp.instagram_webhook_logs IS 'Raw Instagram webhook payloads for audit and replay';
COMMENT ON COLUMN crm.messages.channel IS 'Messaging channel: whatsapp or instagram';
COMMENT ON COLUMN crm.conversations.channel IS 'Messaging channel: whatsapp or instagram';
COMMENT ON COLUMN crm.contacts.instagram_user_id IS 'Instagram scoped user id (webhook sender/recipient id)';
COMMENT ON COLUMN crm.contacts.instagram_username IS 'Instagram username, backfilled from the Graph API';
