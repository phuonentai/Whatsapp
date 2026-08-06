DROP TRIGGER IF EXISTS trigger_tags_updated_at ON crm.tags;
DROP TRIGGER IF EXISTS trigger_activities_updated_at ON crm.activities;

DROP INDEX IF EXISTS idx_entity_tags_tag;
DROP INDEX IF EXISTS idx_entity_tags_entity;
DROP INDEX IF EXISTS idx_tags_org;
DROP INDEX IF EXISTS idx_activities_realizada_en;
DROP INDEX IF EXISTS idx_activities_tipo;
DROP INDEX IF EXISTS idx_activities_deal;
DROP INDEX IF EXISTS idx_activities_company;
DROP INDEX IF EXISTS idx_activities_contact;
DROP INDEX IF EXISTS idx_activities_org;

DROP TABLE IF EXISTS crm.entity_tags CASCADE;
DROP TABLE IF EXISTS crm.tags CASCADE;
DROP TABLE IF EXISTS crm.activities CASCADE;
