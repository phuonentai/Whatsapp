-- 000031_add_instagram_schema.down.sql

DROP TRIGGER IF EXISTS trigger_instagram_configs_updated_at ON whatsapp.instagram_configs;

DROP TABLE IF EXISTS whatsapp.instagram_webhook_logs;
DROP TABLE IF EXISTS whatsapp.instagram_configs;

ALTER TABLE crm.activities
    DROP CONSTRAINT IF EXISTS valid_tipo;
ALTER TABLE crm.activities
    ADD CONSTRAINT valid_tipo CHECK (tipo IN ('nota', 'llamada', 'correo', 'reunion', 'tarea', 'whatsapp_message', 'sistema'));

ALTER TABLE crm.contacts
    DROP CONSTRAINT IF EXISTS valid_source;
ALTER TABLE crm.contacts
    ADD CONSTRAINT valid_source CHECK (source IN ('whatsapp', 'manual', 'import', 'api'));

DROP INDEX IF EXISTS crm.idx_contacts_org_ig_user;
DROP INDEX IF EXISTS crm.idx_contacts_org_phone;

ALTER TABLE crm.contacts
    DROP COLUMN IF EXISTS instagram_username,
    DROP COLUMN IF EXISTS instagram_user_id;
ALTER TABLE crm.contacts
    ALTER COLUMN phone_number SET NOT NULL;

CREATE UNIQUE INDEX idx_contacts_org_phone ON crm.contacts(organization_id, phone_number);

DROP INDEX IF EXISTS crm.conversations_one_active_per_contact;
CREATE UNIQUE INDEX conversations_one_active_per_contact
    ON crm.conversations (organization_id, contact_id)
    WHERE status = 'active';

ALTER TABLE crm.conversations
    DROP CONSTRAINT IF EXISTS valid_conv_channel;
ALTER TABLE crm.conversations
    DROP COLUMN IF EXISTS channel;

DROP INDEX IF EXISTS crm.idx_messages_provider_id;
ALTER TABLE crm.messages
    RENAME COLUMN provider_message_id TO whatsapp_message_id;
CREATE UNIQUE INDEX idx_messages_whatsapp_id ON crm.messages(organization_id, whatsapp_message_id);

ALTER TABLE crm.messages
    DROP CONSTRAINT IF EXISTS valid_msg_channel;
ALTER TABLE crm.messages
    DROP COLUMN IF EXISTS channel;
