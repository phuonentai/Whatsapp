-- 000027_add_webhook_logs_delivery_key.up.sql
-- Webhook delivery deduplication: stable key derived from the provider
-- delivery (entry[].id + message ids). Unique per phone_number_id so Meta
-- retries of the same delivery are acknowledged without re-dispatch.

ALTER TABLE whatsapp.webhook_logs
    ADD COLUMN delivery_key VARCHAR(255);

CREATE UNIQUE INDEX idx_webhook_logs_delivery_key ON whatsapp.webhook_logs(phone_number_id, delivery_key)
    WHERE delivery_key IS NOT NULL;

COMMENT ON COLUMN whatsapp.webhook_logs.delivery_key IS 'Stable provider delivery key (entry id + message ids) used for at-least-once dedup; NULL for legacy/failed-before-resolution rows';
