CREATE TABLE crm.companies (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    nit VARCHAR(50),
    tipo_empresa VARCHAR(20),
    sector VARCHAR(100),
    ciudad VARCHAR(100),
    departamento VARCHAR(100),
    website VARCHAR(500),
    phone VARCHAR(20),
    address TEXT,
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    owner_account_id INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_tipo_empresa CHECK (tipo_empresa IN ('microempresa', 'pequena', 'mediana', 'grande')),
    UNIQUE(organization_id, name)
);

COMMENT ON TABLE crm.companies IS 'Empresas/clientes CRM (no confundir con organizations que son los tenants)';
COMMENT ON COLUMN crm.companies.nit IS 'NIT con dígito de verificación';
COMMENT ON COLUMN crm.companies.tipo_empresa IS 'Tamaño: microempresa, pequena, mediana, grande';
COMMENT ON COLUMN crm.companies.sector IS 'Sector industrial';
COMMENT ON COLUMN crm.companies.ciudad IS 'Ciudad colombiana';
COMMENT ON COLUMN crm.companies.departamento IS 'Departamento colombiano';
COMMENT ON COLUMN crm.companies.owner_account_id IS 'Responsable de la empresa';

CREATE TABLE crm.pipelines (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    nombre VARCHAR(255) NOT NULL,
    es_predeterminado BOOLEAN NOT NULL DEFAULT false,
    orden INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE crm.pipelines IS 'Pipelines de ventas configurables por tenant';
COMMENT ON COLUMN crm.pipelines.nombre IS 'Nombre del pipeline (ej: Pipeline de Ventas)';
COMMENT ON COLUMN crm.pipelines.es_predeterminado IS 'Indica si es el pipeline por defecto';
COMMENT ON COLUMN crm.pipelines.orden IS 'Orden de visualización';

CREATE TABLE crm.pipeline_stages (
    id SERIAL PRIMARY KEY,
    pipeline_id INTEGER NOT NULL REFERENCES crm.pipelines(id) ON DELETE CASCADE,
    nombre VARCHAR(255) NOT NULL,
    orden INTEGER NOT NULL DEFAULT 0,
    color VARCHAR(7),
    probabilidad INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE crm.pipeline_stages IS 'Etapas de cada pipeline de ventas';
COMMENT ON COLUMN crm.pipeline_stages.nombre IS 'Nombre en español (ej: Prospección, Calificado, etc.)';
COMMENT ON COLUMN crm.pipeline_stages.color IS 'Color hex para kanban (ej: #3B82F6)';
COMMENT ON COLUMN crm.pipeline_stages.probabilidad IS 'Probabilidad de cierre (0-100, NULL para etapas de salida)';

CREATE TABLE crm.deals (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    nombre VARCHAR(255) NOT NULL,
    contact_id INTEGER REFERENCES crm.contacts(id) ON DELETE SET NULL,
    company_id INTEGER REFERENCES crm.companies(id) ON DELETE SET NULL,
    pipeline_id INTEGER NOT NULL REFERENCES crm.pipelines(id) ON DELETE RESTRICT,
    stage_id INTEGER REFERENCES crm.pipeline_stages(id) ON DELETE SET NULL,
    monto DECIMAL(12,2),
    moneda VARCHAR(3) NOT NULL DEFAULT 'COP',
    fecha_cierre_esperada DATE,
    estado VARCHAR(20) NOT NULL DEFAULT 'abierto',
    probabilidad INTEGER,
    notas TEXT,
    metadata JSONB DEFAULT '{}',
    assigned_to INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_estado CHECK (estado IN ('abierto', 'ganado', 'perdido', 'abandonado'))
);

COMMENT ON TABLE crm.deals IS 'Negocios/oportunidades de venta';
COMMENT ON COLUMN crm.deals.nombre IS 'Nombre del negocio';
COMMENT ON COLUMN crm.deals.monto IS 'Valor en COP (pesos colombianos)';
COMMENT ON COLUMN crm.deals.moneda IS 'Moneda (default COP)';
COMMENT ON COLUMN crm.deals.estado IS 'Estado: abierto, ganado, perdido, abandonado';

-- Indexes
CREATE INDEX idx_companies_org ON crm.companies(organization_id);
CREATE INDEX idx_companies_nit ON crm.companies(nit);
CREATE INDEX idx_companies_sector ON crm.companies(organization_id, sector);
CREATE INDEX idx_companies_ciudad ON crm.companies(organization_id, ciudad);

CREATE INDEX idx_pipelines_org ON crm.pipelines(organization_id);

CREATE INDEX idx_pipeline_stages_pipeline ON crm.pipeline_stages(pipeline_id);
CREATE INDEX idx_pipeline_stages_orden ON crm.pipeline_stages(pipeline_id, orden);

CREATE INDEX idx_deals_org ON crm.deals(organization_id);
CREATE INDEX idx_deals_pipeline ON crm.deals(pipeline_id);
CREATE INDEX idx_deals_stage ON crm.deals(stage_id);
CREATE INDEX idx_deals_contact ON crm.deals(contact_id);
CREATE INDEX idx_deals_company ON crm.deals(company_id);
CREATE INDEX idx_deals_estado ON crm.deals(organization_id, estado);
CREATE INDEX idx_deals_monto ON crm.deals(organization_id, monto DESC);
CREATE INDEX idx_deals_asignado ON crm.deals(assigned_to);

-- Triggers
CREATE TRIGGER trigger_companies_updated_at
    BEFORE UPDATE ON crm.companies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_pipelines_updated_at
    BEFORE UPDATE ON crm.pipelines FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_pipeline_stages_updated_at
    BEFORE UPDATE ON crm.pipeline_stages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_deals_updated_at
    BEFORE UPDATE ON crm.deals FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- FK from contacts to companies (companies must exist first)
ALTER TABLE crm.contacts
    ADD CONSTRAINT fk_contacts_company
    FOREIGN KEY (company_id) REFERENCES crm.companies(id) ON DELETE SET NULL;
