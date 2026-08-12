-- 000036_create_whatsapp_templates.up.sql
-- Org-scoped WhatsApp message template registry. Local-first source of truth
-- for authoring; Meta is the runtime authority for sendability (approval
-- status synced via message_template_status_update webhooks + manual sync).
-- Stores NO credentials: access tokens live only in whatsapp.whatsapp_configs.

CREATE TABLE whatsapp.templates (
    id BIGSERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    language VARCHAR NOT NULL,
    body TEXT NOT NULL,
    param_count INT NOT NULL DEFAULT 0,
    status VARCHAR NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'submitted', 'approved', 'rejected', 'paused')),
    meta_template_id VARCHAR NULL,
    rejection_reason TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_whatsapp_templates_org_name_language UNIQUE (organization_id, name, language)
);

CREATE INDEX idx_whatsapp_templates_org ON whatsapp.templates(organization_id);
CREATE INDEX idx_whatsapp_templates_meta ON whatsapp.templates(meta_template_id);

CREATE TRIGGER trigger_whatsapp_templates_updated_at
    BEFORE UPDATE ON whatsapp.templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE whatsapp.templates IS 'Plantillas de mensaje de WhatsApp por organización (autoría local; Meta aprueba el envío)';
COMMENT ON COLUMN whatsapp.templates.param_count IS 'Número de parámetros {{N}} en body (calculado al crear/editar)';
COMMENT ON COLUMN whatsapp.templates.status IS 'draft | submitted | approved | rejected | paused';
