-- 000035_add_campaign_message.up.sql
-- Optional message body for campaign drafts: the AI audience builder drafts
-- the copy, the user edits it, and it is stored with the draft. Nothing sends
-- at create/launch; the scheduler consumes mensaje when the send path lands.

ALTER TABLE crm.campaigns ADD COLUMN mensaje TEXT NULL;

COMMENT ON COLUMN crm.campaigns.mensaje IS 'Cuerpo del mensaje de la campaña (borrador opcional; nada se envía al crear o lanzar)';
