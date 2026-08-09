-- 000019_create_agent_schema.down.sql

DROP TABLE IF EXISTS agent.agent_actions;
DROP TABLE IF EXISTS agent.agent_suggestions;
DROP TABLE IF EXISTS agent.agent_settings;
DROP TABLE IF EXISTS agent.conversation_flows;
DROP SCHEMA IF EXISTS agent;

ALTER TABLE crm.contacts DROP CONSTRAINT IF EXISTS valid_consent_status;
ALTER TABLE crm.contacts DROP COLUMN IF EXISTS consent_status;
ALTER TABLE crm.contacts DROP COLUMN IF EXISTS consented_at;
