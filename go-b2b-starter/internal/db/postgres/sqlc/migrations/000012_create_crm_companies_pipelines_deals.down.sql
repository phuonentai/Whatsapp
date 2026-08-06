ALTER TABLE crm.contacts DROP CONSTRAINT IF EXISTS fk_contacts_company;

DROP TRIGGER IF EXISTS trigger_deals_updated_at ON crm.deals;
DROP TRIGGER IF EXISTS trigger_pipeline_stages_updated_at ON crm.pipeline_stages;
DROP TRIGGER IF EXISTS trigger_pipelines_updated_at ON crm.pipelines;
DROP TRIGGER IF EXISTS trigger_companies_updated_at ON crm.companies;

DROP INDEX IF EXISTS idx_deals_asignado;
DROP INDEX IF EXISTS idx_deals_monto;
DROP INDEX IF EXISTS idx_deals_estado;
DROP INDEX IF EXISTS idx_deals_company;
DROP INDEX IF EXISTS idx_deals_contact;
DROP INDEX IF EXISTS idx_deals_stage;
DROP INDEX IF EXISTS idx_deals_pipeline;
DROP INDEX IF EXISTS idx_deals_org;

DROP INDEX IF EXISTS idx_pipeline_stages_orden;
DROP INDEX IF EXISTS idx_pipeline_stages_pipeline;
DROP INDEX IF EXISTS idx_pipelines_org;

DROP INDEX IF EXISTS idx_companies_ciudad;
DROP INDEX IF EXISTS idx_companies_sector;
DROP INDEX IF EXISTS idx_companies_nit;
DROP INDEX IF EXISTS idx_companies_org;

DROP TABLE IF EXISTS crm.deals CASCADE;
DROP TABLE IF EXISTS crm.pipeline_stages CASCADE;
DROP TABLE IF EXISTS crm.pipelines CASCADE;
DROP TABLE IF EXISTS crm.companies CASCADE;
