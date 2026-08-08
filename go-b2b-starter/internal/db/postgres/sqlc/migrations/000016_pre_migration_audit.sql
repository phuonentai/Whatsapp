-- Pre-migration audit & repair for 000016_create_crm_integrity_constraints
-- Run against a snapshot of the target database BEFORE applying the up migration.
-- Record all rows matched before repair (export to CSV); repairs are NOT reversible.

-- 1. FK inventory (delete actions)
SELECT con.conrelid::regclass AS child_table, con.confrelid::regclass AS parent_table,
       con.conname, pg_get_constraintdef(con.oid) AS definition, con.confdeltype AS del_action
FROM pg_constraint con
WHERE con.contype = 'f'
ORDER BY con.conrelid::regclass::text, con.conname;

-- 2. Column nullability inventory
SELECT table_schema||'.'||table_name AS tbl, column_name, is_nullable
FROM information_schema.columns
WHERE column_name IN ('assigned_to','owner_account_id','realizada_por','company_id',
                      'contact_id','conversation_id','stage_id','pipeline_id')
ORDER BY 1,2;

-- 3. Cross-tenant assignment violations (audit)
SELECT 'contacts.assigned_to' AS violation, c.id, c.organization_id, c.assigned_to
FROM crm.contacts c
WHERE c.assigned_to IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM organizations.accounts a WHERE a.id = c.assigned_to AND a.organization_id = c.organization_id)
UNION ALL
SELECT 'deals.assigned_to', d.id, d.organization_id, d.assigned_to
FROM crm.deals d
WHERE d.assigned_to IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM organizations.accounts a WHERE a.id = d.assigned_to AND a.organization_id = d.organization_id)
UNION ALL
SELECT 'companies.owner_account_id', co.id, co.organization_id, co.owner_account_id
FROM crm.companies co
WHERE co.owner_account_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM organizations.accounts a WHERE a.id = co.owner_account_id AND a.organization_id = co.organization_id)
UNION ALL
SELECT 'activities.realizada_por', a.id, a.organization_id, a.realizada_por
FROM crm.activities a
WHERE a.realizada_por IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM organizations.accounts acc WHERE acc.id = a.realizada_por AND acc.organization_id = a.organization_id);

-- 4. Stage/pipeline mismatches (audit)
SELECT d.id, d.organization_id, d.pipeline_id, d.stage_id, s.pipeline_id AS actual_pipeline_id
FROM crm.deals d
JOIN crm.pipeline_stages s ON s.id = d.stage_id
WHERE d.pipeline_id IS DISTINCT FROM s.pipeline_id;

-- 5. Orphan stages (audit)
SELECT d.id, d.organization_id, d.stage_id
FROM crm.deals d
WHERE d.stage_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM crm.pipeline_stages s WHERE s.id = d.stage_id);

-- 6. Message duplicates (audit; expected zero rows - unique index exists since 000010)
SELECT organization_id, whatsapp_message_id, count(*)
FROM crm.messages
WHERE whatsapp_message_id IS NOT NULL
GROUP BY 1, 2 HAVING count(*) > 1;

-- 7. Active-conversation duplicates (audit)
SELECT organization_id, contact_id, count(*)
FROM crm.conversations
WHERE status = 'active'
GROUP BY 1, 2 HAVING count(*) > 1;

-- ============ REPAIR (run only after reviewing audit output) ============

-- R1: null cross-tenant nullable assignments (policy: nullable -> NULL)
UPDATE crm.contacts   SET assigned_to = NULL     WHERE assigned_to IS NOT NULL AND NOT EXISTS (SELECT 1 FROM organizations.accounts a WHERE a.id = assigned_to AND a.organization_id = crm.contacts.organization_id);
UPDATE crm.deals      SET assigned_to = NULL     WHERE assigned_to IS NOT NULL AND NOT EXISTS (SELECT 1 FROM organizations.accounts a WHERE a.id = assigned_to AND a.organization_id = crm.deals.organization_id);
UPDATE crm.companies  SET owner_account_id = NULL WHERE owner_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM organizations.accounts a WHERE a.id = owner_account_id AND a.organization_id = crm.companies.organization_id);
UPDATE crm.activities SET realizada_por = NULL  WHERE realizada_por IS NOT NULL AND NOT EXISTS (SELECT 1 FROM organizations.accounts a WHERE a.id = realizada_por AND a.organization_id = crm.activities.organization_id);

-- R2: normalize deals.pipeline_id from stage (stage is source of truth)
UPDATE crm.deals d SET pipeline_id = s.pipeline_id
FROM crm.pipeline_stages s
WHERE s.id = d.stage_id AND d.pipeline_id IS DISTINCT FROM s.pipeline_id;

-- R3: active-conversation duplicates - keep newest, close older
UPDATE crm.conversations c SET status = 'closed'
WHERE c.status = 'active'
  AND c.id NOT IN (SELECT DISTINCT ON (organization_id, contact_id) id
                   FROM crm.conversations WHERE status = 'active'
                   ORDER BY organization_id, contact_id, last_message_at DESC NULLS LAST, id DESC);

-- Verify: all four re-run audits must return 0 rows before migrating up.
