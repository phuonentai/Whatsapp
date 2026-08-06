CREATE TABLE crm.activities (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    contact_id INTEGER REFERENCES crm.contacts(id) ON DELETE SET NULL,
    company_id INTEGER REFERENCES crm.companies(id) ON DELETE SET NULL,
    deal_id INTEGER REFERENCES crm.deals(id) ON DELETE SET NULL,
    conversation_id INTEGER REFERENCES crm.conversations(id) ON DELETE SET NULL,
    tipo VARCHAR(30) NOT NULL,
    asunto VARCHAR(500),
    contenido TEXT,
    estado VARCHAR(20),
    fecha_vencimiento TIMESTAMPTZ,
    realizada_por INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    realizada_en TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_tipo CHECK (tipo IN ('nota', 'llamada', 'correo', 'reunion', 'tarea', 'whatsapp_message', 'sistema'))
);

COMMENT ON TABLE crm.activities IS 'Línea de tiempo de actividades CRM (notas, llamadas, correos, etc.)';
COMMENT ON COLUMN crm.activities.tipo IS 'Tipo de actividad: nota, llamada, correo, reunion, tarea, whatsapp_message, sistema';
COMMENT ON COLUMN crm.activities.asunto IS 'Asunto o título de la actividad';
COMMENT ON COLUMN crm.activities.contenido IS 'Contenido detallado de la actividad';
COMMENT ON COLUMN crm.activities.estado IS 'Estado (ej: pendiente, completada para tareas)';
COMMENT ON COLUMN crm.activities.fecha_vencimiento IS 'Fecha de vencimiento (para tareas)';
COMMENT ON COLUMN crm.activities.realizada_por IS 'Usuario que realizó la actividad';
COMMENT ON COLUMN crm.activities.realizada_en IS 'Cuándo ocurrió la actividad';
COMMENT ON COLUMN crm.activities.metadata IS 'Metadatos adicionales (ej: message_id para WhatsApp)';

CREATE TABLE crm.tags (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    nombre VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, nombre)
);

COMMENT ON TABLE crm.tags IS 'Etiquetas para segmentar contactos, empresas y negocios';
COMMENT ON COLUMN crm.tags.nombre IS 'Nombre de la etiqueta';
COMMENT ON COLUMN crm.tags.color IS 'Color hex (ej: #F59E0B)';

CREATE TABLE crm.entity_tags (
    id SERIAL PRIMARY KEY,
    tag_id INTEGER NOT NULL REFERENCES crm.tags(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL,
    entity_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tag_id, entity_type, entity_id),
    CONSTRAINT valid_entity_type CHECK (entity_type IN ('contact', 'company', 'deal'))
);

COMMENT ON TABLE crm.entity_tags IS 'Relación muchos-a-muchos entre etiquetas y entidades CRM';

-- Indexes
CREATE INDEX idx_activities_org ON crm.activities(organization_id);
CREATE INDEX idx_activities_contact ON crm.activities(contact_id);
CREATE INDEX idx_activities_company ON crm.activities(company_id);
CREATE INDEX idx_activities_deal ON crm.activities(deal_id);
CREATE INDEX idx_activities_tipo ON crm.activities(organization_id, tipo);
CREATE INDEX idx_activities_realizada_en ON crm.activities(organization_id, realizada_en DESC);

CREATE INDEX idx_tags_org ON crm.tags(organization_id);

CREATE INDEX idx_entity_tags_entity ON crm.entity_tags(entity_type, entity_id);
CREATE INDEX idx_entity_tags_tag ON crm.entity_tags(tag_id);

-- Triggers
CREATE TRIGGER trigger_activities_updated_at
    BEFORE UPDATE ON crm.activities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tags_updated_at
    BEFORE UPDATE ON crm.tags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
