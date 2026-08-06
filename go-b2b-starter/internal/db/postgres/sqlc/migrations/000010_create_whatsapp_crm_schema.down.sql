DROP TRIGGER IF EXISTS trigger_whatsapp_configs_updated_at ON whatsapp.whatsapp_configs;
DROP TRIGGER IF EXISTS trigger_contacts_updated_at ON crm.contacts;
DROP TRIGGER IF EXISTS trigger_conversations_updated_at ON crm.conversations;
DROP TRIGGER IF EXISTS trigger_messages_updated_at ON crm.messages;

DROP TABLE IF EXISTS crm.messages CASCADE;
DROP TABLE IF EXISTS crm.conversations CASCADE;
DROP TABLE IF EXISTS crm.contacts CASCADE;
DROP TABLE IF EXISTS whatsapp.webhook_logs CASCADE;
DROP TABLE IF EXISTS whatsapp.whatsapp_configs CASCADE;

DROP SCHEMA IF EXISTS crm CASCADE;
DROP SCHEMA IF EXISTS whatsapp CASCADE;
