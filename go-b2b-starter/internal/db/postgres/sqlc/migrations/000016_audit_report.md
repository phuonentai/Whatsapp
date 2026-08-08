# Migration Audit Report — crm-integrity-phase-a (000016)

Date: 2026-08-08
Audit DB: scratch database `crm_integrity_scratch` (PostgreSQL 17.10 + pgvector), schema at migration 000015.

## Findings

| Audit | Result |
|---|---|
| FK inventory (`pg_constraint.confdeltype`) | 24 FKs inventoried; matches design matrix. `conversations.contact_id` and `messages.conversation_id` are NOT NULL + CASCADE; all assignment/owner columns nullable + SET NULL; `deals.pipeline_id` NOT NULL + RESTRICT. |
| Nullability inventory | `conversations.contact_id`, `messages.conversation_id`, `messages.contact_id`, `deals.pipeline_id`, `pipeline_stages.pipeline_id` are NOT NULL. All composite-FK candidate child columns else nullable. |
| Cross-tenant assignments | 4 violations found (seeded): 1 `contacts.assigned_to`, 1 `deals.assigned_to`, 1 `companies.owner_account_id`, 1 `activities.realizada_por` — all referenced account from org 2 while row belonged to org 1. |
| Stage/pipeline mismatch | 1 violation found (seeded): deal with `pipeline_id=1` but `stage_id` of pipeline 2. |
| Orphan stages | 0 |
| Message duplicates | 0 (unique index from 000010 prevents) |
| Active-conversation duplicates | 1 duplicate pair found (seeded): contact 1 had 2 active conversations. |

## Repairs applied (scratch DB)

- R1: 4 cross-tenant assignment columns set to NULL.
- R2: 1 `pipeline_id` normalized from stage's pipeline.
- R3: 1 duplicate active conversation closed (kept newest).

Re-run of all audits after repair: 0 rows. Safe to apply `000016_create_crm_integrity_constraints.up.sql`.

## Structural discovery

`crm.pipeline_stages` has no `organization_id` column. The triple-key unique `(organization_id, id, pipeline_id)` and the composite deals FK require it; migration 000016 adds the column, backfills from `pipelines`, and adds a sync trigger (`crm_pipeline_stages_sync_org`) on `INSERT OR UPDATE OF pipeline_id`. Recorded in design.md (D2).

## Pre-existing migration landmine (out of scope)

`000002_add_tenant_isolation.up.sql` references `public.organizations`, which does not exist (orgs live in schema `organizations`). It was skipped when applying migrations to the audit DB; the live `file_manager.file_assets` table has no `organization_id` column, consistent with this migration never having applied. Both `000002_*` files share migration version `000002`, so a `migrate` CLI run silently resolves one of them. Recommend a separate cleanup change.
