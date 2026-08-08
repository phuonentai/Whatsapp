-- 000016_create_crm_integrity_constraints.down.sql
-- Restores the pre-Phase-A FK shape (single-column FKs with original delete actions).
-- NOTE: restores schema, not data (assignments nulled by SET NULL after this migration
-- was live are not restored).

BEGIN;

-- ============================================================
-- 1. Drop stage <-> pipeline machinery
-- ============================================================

DROP TRIGGER IF EXISTS deals_sync_pipeline_from_stage ON crm.deals;
DROP FUNCTION IF EXISTS crm_deals_sync_pipeline_from_stage();

ALTER TABLE crm.deals DROP CONSTRAINT IF EXISTS deals_stage_pipeline_org_fkey;
ALTER TABLE crm.pipeline_stages DROP CONSTRAINT IF EXISTS pipeline_stages_organization_id_id_pipeline_id_key;

-- ============================================================
-- 2. Drop one-active-conversation index (isolated step)
-- ============================================================

DROP INDEX IF EXISTS crm.conversations_one_active_per_contact;

-- ============================================================
-- 3. Drop composite FKs, restore original single-column FKs
-- ============================================================

ALTER TABLE crm.contacts DROP CONSTRAINT IF EXISTS contacts_assigned_to_org_fkey;
ALTER TABLE crm.contacts
  ADD CONSTRAINT contacts_assigned_to_fkey
  FOREIGN KEY (assigned_to) REFERENCES organizations.accounts(id) ON DELETE SET NULL;

ALTER TABLE crm.deals DROP CONSTRAINT IF EXISTS deals_assigned_to_org_fkey;
ALTER TABLE crm.deals
  ADD CONSTRAINT deals_assigned_to_fkey
  FOREIGN KEY (assigned_to) REFERENCES organizations.accounts(id) ON DELETE SET NULL;

ALTER TABLE crm.companies DROP CONSTRAINT IF EXISTS companies_owner_account_org_fkey;
ALTER TABLE crm.companies
  ADD CONSTRAINT companies_owner_account_id_fkey
  FOREIGN KEY (owner_account_id) REFERENCES organizations.accounts(id) ON DELETE SET NULL;

ALTER TABLE crm.activities DROP CONSTRAINT IF EXISTS activities_realizada_por_org_fkey;
ALTER TABLE crm.activities
  ADD CONSTRAINT activities_realizada_por_fkey
  FOREIGN KEY (realizada_por) REFERENCES organizations.accounts(id) ON DELETE SET NULL;

ALTER TABLE crm.contacts DROP CONSTRAINT IF EXISTS contacts_company_org_fkey;
ALTER TABLE crm.contacts
  ADD CONSTRAINT fk_contacts_company
  FOREIGN KEY (company_id) REFERENCES crm.companies(id) ON DELETE SET NULL;

ALTER TABLE crm.deals DROP CONSTRAINT IF EXISTS deals_contact_org_fkey;
ALTER TABLE crm.deals
  ADD CONSTRAINT deals_contact_id_fkey
  FOREIGN KEY (contact_id) REFERENCES crm.contacts(id) ON DELETE SET NULL;

ALTER TABLE crm.deals DROP CONSTRAINT IF EXISTS deals_company_org_fkey;
ALTER TABLE crm.deals
  ADD CONSTRAINT deals_company_id_fkey
  FOREIGN KEY (company_id) REFERENCES crm.companies(id) ON DELETE SET NULL;

-- NOTE: original single-column deals_stage_id_fkey and deals_pipeline_id_fkey were
-- retained in the up migration (the composite FK adds the cross-pipeline guarantee);
-- only the composite deals_stage_pipeline_org_fkey is dropped here (see section 1).

ALTER TABLE crm.conversations DROP CONSTRAINT IF EXISTS conversations_contact_org_fkey;
ALTER TABLE crm.conversations
  ADD CONSTRAINT conversations_contact_id_fkey
  FOREIGN KEY (contact_id) REFERENCES crm.contacts(id) ON DELETE CASCADE;

ALTER TABLE crm.messages DROP CONSTRAINT IF EXISTS messages_conversation_org_fkey;
ALTER TABLE crm.messages
  ADD CONSTRAINT messages_conversation_id_fkey
  FOREIGN KEY (conversation_id) REFERENCES crm.conversations(id) ON DELETE CASCADE;

-- ============================================================
-- 4. pipeline_stages: drop organization_id column and its machinery
-- ============================================================

DROP TRIGGER IF EXISTS pipeline_stages_sync_org ON crm.pipeline_stages;
DROP FUNCTION IF EXISTS crm_pipeline_stages_sync_org();

ALTER TABLE crm.pipeline_stages DROP CONSTRAINT IF EXISTS pipeline_stages_organization_id_fkey;
ALTER TABLE crm.pipeline_stages DROP COLUMN IF EXISTS organization_id;

-- ============================================================
-- 5. Drop tenant-scoped unique parent keys
-- ============================================================

ALTER TABLE organizations.accounts       DROP CONSTRAINT IF EXISTS accounts_organization_id_id_key;
ALTER TABLE crm.contacts                 DROP CONSTRAINT IF EXISTS contacts_organization_id_id_key;
ALTER TABLE crm.companies                DROP CONSTRAINT IF EXISTS companies_organization_id_id_key;
ALTER TABLE crm.deals                    DROP CONSTRAINT IF EXISTS deals_organization_id_id_key;
ALTER TABLE crm.pipelines                DROP CONSTRAINT IF EXISTS pipelines_organization_id_id_key;
ALTER TABLE crm.pipeline_stages          DROP CONSTRAINT IF EXISTS pipeline_stages_organization_id_id_key;
ALTER TABLE crm.conversations            DROP CONSTRAINT IF EXISTS conversations_organization_id_id_key;
ALTER TABLE crm.tags                     DROP CONSTRAINT IF EXISTS tags_organization_id_id_key;

COMMIT;
