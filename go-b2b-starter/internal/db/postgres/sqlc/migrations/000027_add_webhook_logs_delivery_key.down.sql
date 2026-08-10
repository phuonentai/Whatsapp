DROP INDEX IF EXISTS idx_webhook_logs_delivery_key;
ALTER TABLE whatsapp.webhook_logs
    DROP COLUMN IF EXISTS delivery_key;
