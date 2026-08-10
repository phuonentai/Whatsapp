ALTER TABLE whatsapp.webhook_logs
    ALTER COLUMN organization_id DROP NOT NULL;

COMMENT ON COLUMN whatsapp.webhook_logs.organization_id IS 'Resolved organization for the webhook; NULL when the failure occurs before organization resolution (e.g., unknown phone_number_id)';
