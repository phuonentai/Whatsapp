-- 000016_create_crm_integrity_constraints.up.sql
-- Phase A: CRM integrity baseline.
-- Composite tenant-safe FKs, stage<->pipeline consistency, idempotent message/conversation
-- invariants. Preserves pre-existing delete semantics (SET NULL (column_list) / CASCADE).
-- Prereq: run 000016_pre_migration_audit.sql and verify zero violations.

BEGIN;

-- ============================================================
-- 1. Tenant-scoped unique parent keys (prerequisite for composite FKs)
-- ============================================================

ALTER TABLE organizations.accounts       ADD CONSTRAINT accounts_organization_id_id_key       UNIQUE (organization_id, id);
ALTER TABLE crm.contacts                 ADD CONSTRAINT contacts_organization_id_id_key       UNIQUE (organization_id, id);
ALTER TABLE crm.companies                ADD CONSTRAINT companies_organization_id_id_key      UNIQUE (organization_id, id);
ALTER TABLE crm.deals                    ADD CONSTRAINT deals_organization_id_id_key          UNIQUE (organization_id, id);
ALTER TABLE crm.pipelines                ADD CONSTRAINT pipelines_organization_id_id_key      UNIQUE (organization_id, id);
-- NOTE: pipeline_stages gets its (organization_id, id) unique in section 2 after the column is added.
ALTER TABLE crm.conversations            ADD CONSTRAINT conversations_organization_id_id_key  UNIQUE (organization_id, id);
ALTER TABLE crm.tags                     ADD CONSTRAINT tags_organization_id_id_key           UNIQUE (organization_id, id);

-- ============================================================
-- 2. pipeline_stages: add organization_id (derived from pipeline)
-- ============================================================

ALTER TABLE crm.pipeline_stages ADD COLUMN organization_id INTEGER;

UPDATE crm.pipeline_stages ps
SET organization_id = p.organization_id
FROM crm.pipelines p
WHERE p.id = ps.pipeline_id;

ALTER TABLE crm.pipeline_stages
  ALTER COLUMN organization_id SET NOT NULL;

ALTER TABLE crm.pipeline_stages
  ADD CONSTRAINT pipeline_stages_organization_id_fkey
  FOREIGN KEY (organization_id) REFERENCES organizations.organizations(id) ON DELETE CASCADE;

CREATE OR REPLACE FUNCTION crm_pipeline_stages_sync_org()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  v_organization_id INTEGER;
BEGIN
  SELECT organization_id INTO v_organization_id
  FROM crm.pipelines
  WHERE id = NEW.pipeline_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'pipeline % does not exist', NEW.pipeline_id;
  END IF;

  NEW.organization_id := v_organization_id;
  RETURN NEW;
END;
$$;

CREATE TRIGGER pipeline_stages_sync_org
  BEFORE INSERT OR UPDATE OF pipeline_id ON crm.pipeline_stages
  FOR EACH ROW EXECUTE FUNCTION crm_pipeline_stages_sync_org();

-- Tenant-scoped unique parent key for pipeline_stages (column now exists)
ALTER TABLE crm.pipeline_stages
  ADD CONSTRAINT pipeline_stages_organization_id_id_key UNIQUE (organization_id, id);

-- ============================================================
-- 3. Replace assignment/owner FKs with composite tenant-safe FKs
--    (ON DELETE SET NULL (column): nulls only the assignment column)
-- ============================================================

ALTER TABLE crm.contacts    DROP CONSTRAINT contacts_assigned_to_fkey;
ALTER TABLE crm.deals       DROP CONSTRAINT deals_assigned_to_fkey;
ALTER TABLE crm.companies   DROP CONSTRAINT companies_owner_account_id_fkey;
ALTER TABLE crm.activities  DROP CONSTRAINT activities_realizada_por_fkey;

ALTER TABLE crm.contacts
  ADD CONSTRAINT contacts_assigned_to_org_fkey
  FOREIGN KEY (organization_id, assigned_to)
  REFERENCES organizations.accounts (organization_id, id)
  ON DELETE SET NULL (assigned_to) NOT VALID;
ALTER TABLE crm.contacts VALIDATE CONSTRAINT contacts_assigned_to_org_fkey;

ALTER TABLE crm.deals
  ADD CONSTRAINT deals_assigned_to_org_fkey
  FOREIGN KEY (organization_id, assigned_to)
  REFERENCES organizations.accounts (organization_id, id)
  ON DELETE SET NULL (assigned_to) NOT VALID;
ALTER TABLE crm.deals VALIDATE CONSTRAINT deals_assigned_to_org_fkey;

ALTER TABLE crm.companies
  ADD CONSTRAINT companies_owner_account_org_fkey
  FOREIGN KEY (organization_id, owner_account_id)
  REFERENCES organizations.accounts (organization_id, id)
  ON DELETE SET NULL (owner_account_id) NOT VALID;
ALTER TABLE crm.companies VALIDATE CONSTRAINT companies_owner_account_org_fkey;

ALTER TABLE crm.activities
  ADD CONSTRAINT activities_realizada_por_org_fkey
  FOREIGN KEY (organization_id, realizada_por)
  REFERENCES organizations.accounts (organization_id, id)
  ON DELETE SET NULL (realizada_por) NOT VALID;
ALTER TABLE crm.activities VALIDATE CONSTRAINT activities_realizada_por_org_fkey;

-- ============================================================
-- 4. Replace parent/child FKs with composite tenant-safe FKs
--    (preserving current delete semantics)
-- ============================================================

-- contacts.company_id: SET NULL -> SET NULL (company_id)
ALTER TABLE crm.contacts DROP CONSTRAINT fk_contacts_company;
ALTER TABLE crm.contacts
  ADD CONSTRAINT contacts_company_org_fkey
  FOREIGN KEY (organization_id, company_id)
  REFERENCES crm.companies (organization_id, id)
  ON DELETE SET NULL (company_id) NOT VALID;
ALTER TABLE crm.contacts VALIDATE CONSTRAINT contacts_company_org_fkey;

-- deals.contact_id / company_id: SET NULL -> SET NULL (column)
ALTER TABLE crm.deals DROP CONSTRAINT deals_contact_id_fkey;
ALTER TABLE crm.deals
  ADD CONSTRAINT deals_contact_org_fkey
  FOREIGN KEY (organization_id, contact_id)
  REFERENCES crm.contacts (organization_id, id)
  ON DELETE SET NULL (contact_id) NOT VALID;
ALTER TABLE crm.deals VALIDATE CONSTRAINT deals_contact_org_fkey;

ALTER TABLE crm.deals DROP CONSTRAINT deals_company_id_fkey;
ALTER TABLE crm.deals
  ADD CONSTRAINT deals_company_org_fkey
  FOREIGN KEY (organization_id, company_id)
  REFERENCES crm.companies (organization_id, id)
  ON DELETE SET NULL (company_id) NOT VALID;
ALTER TABLE crm.deals VALIDATE CONSTRAINT deals_company_org_fkey;

-- conversations.contact_id: CASCADE (column NOT NULL) -> composite CASCADE
ALTER TABLE crm.conversations DROP CONSTRAINT conversations_contact_id_fkey;
ALTER TABLE crm.conversations
  ADD CONSTRAINT conversations_contact_org_fkey
  FOREIGN KEY (organization_id, contact_id)
  REFERENCES crm.contacts (organization_id, id)
  ON DELETE CASCADE NOT VALID;
ALTER TABLE crm.conversations VALIDATE CONSTRAINT conversations_contact_org_fkey;

-- messages.conversation_id: CASCADE (column NOT NULL) -> composite CASCADE
ALTER TABLE crm.messages DROP CONSTRAINT messages_conversation_id_fkey;
ALTER TABLE crm.messages
  ADD CONSTRAINT messages_conversation_org_fkey
  FOREIGN KEY (organization_id, conversation_id)
  REFERENCES crm.conversations (organization_id, id)
  ON DELETE CASCADE NOT VALID;
ALTER TABLE crm.messages VALIDATE CONSTRAINT messages_conversation_org_fkey;

-- ============================================================
-- 5. Stage <-> pipeline consistency
-- ============================================================

ALTER TABLE crm.pipeline_stages
  ADD CONSTRAINT pipeline_stages_organization_id_id_pipeline_id_key
  UNIQUE (organization_id, id, pipeline_id);

ALTER TABLE crm.deals
  ADD CONSTRAINT deals_stage_pipeline_org_fkey
  FOREIGN KEY (organization_id, stage_id, pipeline_id)
  REFERENCES crm.pipeline_stages (organization_id, id, pipeline_id)
  ON DELETE SET NULL (stage_id) NOT VALID;
ALTER TABLE crm.deals VALIDATE CONSTRAINT deals_stage_pipeline_org_fkey;

CREATE OR REPLACE FUNCTION crm_deals_sync_pipeline_from_stage()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  v_pipeline_id INTEGER;
BEGIN
  IF NEW.stage_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT pipeline_id INTO v_pipeline_id
  FROM crm.pipeline_stages
  WHERE organization_id = NEW.organization_id
    AND id = NEW.stage_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'stage % does not exist in organization %', NEW.stage_id, NEW.organization_id;
  END IF;

  NEW.pipeline_id := v_pipeline_id;
  RETURN NEW;
END;
$$;

CREATE TRIGGER deals_sync_pipeline_from_stage
  BEFORE INSERT OR UPDATE OF stage_id ON crm.deals
  FOR EACH ROW EXECUTE FUNCTION crm_deals_sync_pipeline_from_stage();

-- ============================================================
-- 6. One active conversation per (organization, contact) [isolated step]
-- ============================================================

CREATE UNIQUE INDEX conversations_one_active_per_contact
  ON crm.conversations (organization_id, contact_id)
  WHERE status = 'active';

COMMIT;
