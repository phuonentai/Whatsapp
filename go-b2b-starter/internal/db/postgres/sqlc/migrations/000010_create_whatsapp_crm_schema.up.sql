CREATE SCHEMA IF NOT EXISTS whatsapp;

CREATE SCHEMA IF NOT EXISTS crm;

CREATE TABLE whatsapp.whatsapp_configs (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL UNIQUE REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    phone_number_id VARCHAR(100) NOT NULL UNIQUE,
    business_phone VARCHAR(20) NOT NULL,
    webhook_secret VARCHAR(255) NOT NULL,
    verify_token VARCHAR(255) NOT NULL,
    app_id VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE whatsapp.webhook_logs (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'received',
    event_type VARCHAR(50),
    phone_number_id VARCHAR(100),
    raw_headers JSONB DEFAULT '{}',
    raw_body JSONB DEFAULT '{}',
    error_message TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE crm.contacts (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    phone_number VARCHAR(20) NOT NULL,
    display_name VARCHAR(255),
    avatar_url VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    is_blocked BOOLEAN NOT NULL DEFAULT false,
    last_message_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, phone_number)
);

CREATE TABLE crm.conversations (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES crm.contacts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_message_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_conv_status CHECK (status IN ('active', 'closed', 'archived'))
);

CREATE TABLE crm.messages (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    conversation_id INTEGER NOT NULL REFERENCES crm.conversations(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES crm.contacts(id) ON DELETE CASCADE,
    whatsapp_message_id VARCHAR(100),
    direction VARCHAR(10) NOT NULL DEFAULT 'inbound',
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'received',
    message_data JSONB DEFAULT '{}',
    chat_timestamp TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_direction CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT valid_msg_type CHECK (message_type IN ('text', 'image', 'video', 'audio', 'document', 'location', 'interactive', 'button', 'sticker', 'order', 'system')),
    CONSTRAINT valid_msg_status CHECK (status IN ('received', 'delivered', 'read', 'failed', 'sent'))
);

CREATE UNIQUE INDEX idx_whatsapp_configs_org ON whatsapp.whatsapp_configs(organization_id);
CREATE UNIQUE INDEX idx_whatsapp_configs_phone ON whatsapp.whatsapp_configs(phone_number_id);
CREATE INDEX idx_webhook_logs_org ON whatsapp.webhook_logs(organization_id);
CREATE INDEX idx_webhook_logs_status ON whatsapp.webhook_logs(status);
CREATE INDEX idx_webhook_logs_created ON whatsapp.webhook_logs(created_at DESC);
CREATE INDEX idx_contacts_org ON crm.contacts(organization_id);
CREATE UNIQUE INDEX idx_contacts_org_phone ON crm.contacts(organization_id, phone_number);
CREATE INDEX idx_contacts_last_message ON crm.contacts(organization_id, last_message_at DESC);
CREATE INDEX idx_conversations_org ON crm.conversations(organization_id);
CREATE INDEX idx_conversations_contact ON crm.conversations(contact_id);
CREATE INDEX idx_conversations_status ON crm.conversations(organization_id, status);
CREATE INDEX idx_conversations_last_msg ON crm.conversations(organization_id, last_message_at DESC);
CREATE INDEX idx_messages_org ON crm.messages(organization_id);
CREATE INDEX idx_messages_conversation ON crm.messages(conversation_id);
CREATE INDEX idx_messages_contact ON crm.messages(contact_id);
CREATE UNIQUE INDEX idx_messages_whatsapp_id ON crm.messages(organization_id, whatsapp_message_id);
CREATE INDEX idx_messages_created ON crm.messages(organization_id, created_at DESC);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_whatsapp_configs_updated_at
    BEFORE UPDATE ON whatsapp.whatsapp_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_contacts_updated_at
    BEFORE UPDATE ON crm.contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_conversations_updated_at
    BEFORE UPDATE ON crm.conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON crm.messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON SCHEMA whatsapp IS 'WhatsApp Cloud API integration';
COMMENT ON TABLE whatsapp.whatsapp_configs IS 'Maps WhatsApp phone_number_id to organizations with webhook secrets';
COMMENT ON TABLE whatsapp.webhook_logs IS 'Raw webhook payloads for audit and replay';
COMMENT ON SCHEMA crm IS 'Customer relationship management data';
COMMENT ON TABLE crm.contacts IS 'Contacts (people who send messages) scoped by organization';
COMMENT ON TABLE crm.conversations IS 'Conversation threads with contact, supporting 24-hour window matching';
COMMENT ON TABLE crm.messages IS 'Individual messages within conversations with WhatsApp metadata';
