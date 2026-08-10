ALTER TABLE whatsapp.webhook_logs
    ALTER COLUMN organization_id SET NOT NULL;

COMMENT ON COLUMN whatsapp.webhook_logs.organization_id IS NULL;
