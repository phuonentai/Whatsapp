DROP INDEX IF EXISTS idx_contacts_assigned;
DROP INDEX IF EXISTS idx_contacts_lead_status;
DROP INDEX IF EXISTS idx_contacts_source;
DROP INDEX IF EXISTS idx_contacts_company;
DROP INDEX IF EXISTS idx_contacts_org_email;

ALTER TABLE crm.contacts
  DROP CONSTRAINT IF EXISTS valid_source,
  DROP CONSTRAINT IF EXISTS valid_lead_status,
  DROP CONSTRAINT IF EXISTS valid_tipo_documento;

ALTER TABLE crm.contacts
  DROP COLUMN IF EXISTS numero_documento,
  DROP COLUMN IF EXISTS tipo_documento,
  DROP COLUMN IF EXISTS assigned_to,
  DROP COLUMN IF EXISTS job_title,
  DROP COLUMN IF EXISTS lead_status,
  DROP COLUMN IF EXISTS source,
  DROP COLUMN IF EXISTS company_id,
  DROP COLUMN IF EXISTS email;
