-- 000030_onboarding_data.down.sql
-- Restores the pre-change invoices schema. Fails if sandbox test rows exist
-- (deal_id NULL); remove test rows first.
DROP TABLE IF EXISTS invoicing.import_runs;
DROP TABLE IF EXISTS invoicing.org_numerations;

DROP INDEX IF EXISTS uq_invoices_org_deal;
ALTER TABLE invoicing.invoices ADD CONSTRAINT uq_invoices_org_deal UNIQUE (organization_id, deal_id);
DELETE FROM invoicing.invoices WHERE deal_id IS NULL;
ALTER TABLE invoicing.invoices ALTER COLUMN deal_id SET NOT NULL;
