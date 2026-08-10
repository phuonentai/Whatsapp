-- 000029_create_campaign_segments.down.sql

DROP TABLE IF EXISTS crm.campaign_recipients CASCADE;
DROP TABLE IF EXISTS crm.campaigns CASCADE;
DROP TABLE IF EXISTS crm.segments CASCADE;

DELETE FROM modules.modules WHERE key = 'campaigns';
