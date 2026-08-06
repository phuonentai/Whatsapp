ALTER TABLE crm.contacts
  ADD COLUMN email VARCHAR(255),
  ADD COLUMN company_id INTEGER,
  ADD COLUMN source VARCHAR(50) NOT NULL DEFAULT 'whatsapp',
  ADD COLUMN lead_status VARCHAR(50) NOT NULL DEFAULT 'nuevo',
  ADD COLUMN job_title VARCHAR(255),
  ADD COLUMN assigned_to INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
  ADD COLUMN tipo_documento VARCHAR(3),
  ADD COLUMN numero_documento VARCHAR(50);

ALTER TABLE crm.contacts
  ADD CONSTRAINT valid_tipo_documento CHECK (tipo_documento IN ('CC', 'NIT', 'CE', 'TI', 'PP'));

ALTER TABLE crm.contacts
  ADD CONSTRAINT valid_lead_status CHECK (lead_status IN ('nuevo', 'contactado', 'calificado', 'descalificado', 'cliente'));

ALTER TABLE crm.contacts
  ADD CONSTRAINT valid_source CHECK (source IN ('whatsapp', 'manual', 'import', 'api'));

CREATE UNIQUE INDEX idx_contacts_org_email ON crm.contacts(organization_id, email) WHERE email IS NOT NULL;

CREATE INDEX idx_contacts_company ON crm.contacts(company_id);
CREATE INDEX idx_contacts_source ON crm.contacts(organization_id, source);
CREATE INDEX idx_contacts_lead_status ON crm.contacts(organization_id, lead_status);
CREATE INDEX idx_contacts_assigned ON crm.contacts(assigned_to);

COMMENT ON COLUMN crm.contacts.email IS 'Email del contacto (unique per org when non-null)';
COMMENT ON COLUMN crm.contacts.company_id IS 'Empresa asociada (FK to crm.companies)';
COMMENT ON COLUMN crm.contacts.source IS 'Origen del contacto: whatsapp, manual, import, api';
COMMENT ON COLUMN crm.contacts.lead_status IS 'Estado comercial: nuevo, contactado, calificado, descalificado, cliente';
COMMENT ON COLUMN crm.contacts.job_title IS 'Cargo del contacto';
COMMENT ON COLUMN crm.contacts.assigned_to IS 'Responsable asignado (FK to accounts)';
COMMENT ON COLUMN crm.contacts.tipo_documento IS 'Tipo de documento: CC, NIT, CE, TI, PP';
COMMENT ON COLUMN crm.contacts.numero_documento IS 'Número de documento de identidad';
