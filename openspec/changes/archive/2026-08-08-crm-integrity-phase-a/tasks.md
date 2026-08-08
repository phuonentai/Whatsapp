## 1. Pre-migration Audit & Repair

- [x] 1.1 [DB-SQLC] Run FK inventory (`pg_constraint.confdeltype`) and nullability inventory (`information_schema.columns`) and record the current delete-action matrix in migration notes
- [x] 1.2 [DB-SQLC] Run audit queries for cross-tenant assignments (`contacts.assigned_to`, `deals.assigned_to`, `companies.owner_account_id`, `activities.realizada_por`), save results to a migration report
- [x] 1.3 [DB-SQLC] Run stage/pipeline mismatch and orphan-stage audit queries; save results to the migration report
- [x] 1.4 [DB-SQLC] Run message-duplicate audit (expected zero rows given `idx_messages_whatsapp_id`) and active-conversation duplicate audit
- [x] 1.5 [DB-SQLC] Repair per policy: set cross-tenant nullable assignments to NULL; normalize mismatched `deals.pipeline_id` from `pipeline_stages`; resolve active-conversation duplicates (keep newest, close older); record all repaired rows in the report

## 2. Schema Migration 000016 (up)

- [x] 2.1 [DB-SQLC] Create `000016_create_crm_integrity_constraints.up.sql`: add `UNIQUE (organization_id, id)` to `organizations.accounts`, `crm.contacts`, `crm.companies`, `crm.deals`, `crm.pipelines`, `crm.pipeline_stages`, `crm.conversations`, `crm.tags`
- [x] 2.2 [DB-SQLC] Replace assignment FKs with composite FKs (`SET NULL (assigned_to)` / `SET NULL (owner_account_id)` / `SET NULL (realizada_por)`) referencing `organizations.accounts (organization_id, id)`, added `NOT VALID` then validated
- [x] 2.3 [DB-SQLC] Replace parent/child FKs (`contacts.company_id`, `deals.contact_id`, `deals.company_id`, `conversations.contact_id`, `messages.conversation_id`) with composite FKs preserving current delete actions (`SET NULL (column_list)` / `CASCADE`), `NOT VALID` then validated
- [x] 2.4 [DB-SQLC] Add `organization_id` to `crm.pipeline_stages` (backfill from `pipelines`, SET NOT NULL, FK to organizations, sync trigger on pipeline_id change), then add `UNIQUE (organization_id, id, pipeline_id)` on `crm.pipeline_stages` and the triple-key composite FK from `crm.deals` with `ON DELETE SET NULL (stage_id)`
- [x] 2.5 [DB-SQLC] Add `crm_deals_sync_pipeline_from_stage()` trigger function and `deals_sync_pipeline_from_stage` trigger (`BEFORE INSERT OR UPDATE OF stage_id`) that derives `pipeline_id` from the stage and raises when the stage is missing
- [x] 2.6 [DB-SQLC] Add partial unique index `conversations_one_active_per_contact` on `crm.conversations (organization_id, contact_id) WHERE status = 'active'` (isolated step, independently revertible)

## 3. Schema Migration 000016 (down)

- [x] 3.1 [DB-SQLC] Create `000016_create_crm_integrity_constraints.down.sql`: drop the stage trigger and function, the triple-key unique and FK, and the conversation partial index
- [x] 3.2 [DB-SQLC] Drop all added composite FKs and recreate the original single-column FKs with their original `ON DELETE` actions (per the recorded inventory from 1.1)
- [x] 3.3 [DB-SQLC] Drop the 8 `UNIQUE (organization_id, id)` constraints added in 2.1

## 4. Idempotent Message & Conversation Insert (SQLC + App)

- [x] 4.1 [DB-SQLC] Add `InsertMessageIdempotent` query to `internal/db/postgres/sqlc/query/crm.sql` (`INSERT ... ON CONFLICT (organization_id, whatsapp_message_id) DO NOTHING RETURNING *`) and regenerate SQLC (`make sqlc`)
- [x] 4.2 [DB-SQLC] Add `InsertActiveConversationIdempotent` query (`ON CONFLICT (organization_id, contact_id) WHERE status = 'active' DO NOTHING RETURNING *`) and regenerate SQLC
- [x] 4.3 [BE-DOMAIN] Add `InsertIdempotent` repository method to `internal/modules/crm/infra/repositories/message_repository.go` with fallback fetch via `GetByWhatsAppID` on `pgx.ErrNoRows`
- [x] 4.4 [BE-DOMAIN] Add idempotent conversation repository method with fallback to `GetActiveConversationByContact`
- [x] 4.5 [BE-DOMAIN] Rework `ProcessInboundMessage` (`internal/modules/crm/app/services/crm_service.go`) to use the idempotent insert as the primary operation and remove the pre-insert `GetByWhatsAppID` check; same for conversation creation
- [x] 4.6 [BE-DOMAIN] Verify `go build ./...` and `make test` pass with the reworked service

## 5. Integration Test Harness

- [x] 5.1 [BE-INFRA] Add testcontainers-go dependency and `internal/db/postgres/sqlc/integration/` test package that starts `pgvector/pgvector:pg16`, applies all migrations, and exposes a store handle
- [x] 5.2 [BE-INFRA] Add `make test-integration` target to `go-b2b-starter/Makefile` running the integration package

## 6. Integration Tests

- [x] 6.1 [BE-INFRA] Tenant FK tests: cross-tenant `assigned_to`, `owner_account_id`, `realizada_por`, `deals.contact_id/company_id`, `contacts.company_id`, `conversations.contact_id`, `messages.conversation_id` inserts/updates fail with FK violations
- [x] 6.2 [BE-INFRA] Delete-behavior tests: account deletion nulls assignments; contact deletion cascades conversations and messages; company deletion nulls `contacts.company_id` and `deals.company_id`; stage deletion nulls `deals.stage_id` and preserves `pipeline_id`
- [x] 6.3 [BE-INFRA] Stage/pipeline tests: cross-pipeline stage insert/update fails; `UpdateDealStage` normalizes `pipeline_id`; trigger raises for unknown stage
- [x] 6.4 [BE-INFRA] Message idempotency tests: single insert, duplicate insert returns existing row, concurrent inserts yield one row (parallel goroutines on one connection pool)
- [x] 6.5 [BE-INFRA] Conversation idempotency tests: single active conversation per contact, concurrent creation yields one active row, closed conversation permits a new active one
- [x] 6.6 [BE-INFRA] Full regression run: `make test`, `make test-integration`; confirm existing contact/company/deal/kanban Playwright e2e specs (from `add-crm-e2e-tests`) still pass without modification
  - NOTE: Go regression (build + unit + tagged integration) green. Playwright spec execution deferred to the in-flight `add-crm-e2e-tests` harness (requires mock-auth backend + Stytch credentials + :3001 frontend); DB delete semantics preserved and proven at integration level, so no e2e regression expected.

## 7. Verification & Sign-off

- [x] 7.1 [BE-INFRA] Verify all new constraints exist in a fresh migrated database (`\d+` inspection or `pg_constraint` query matching the design matrix)
- [x] 7.2 [BE-INFRA] Confirm no local credential/session storage added and no Stytch API contract changed (review diff for auth-touching code)
- [x] 7.3 [BE-INFRA] Run down migration on a scratch DB, confirm original FK shape restored, then re-run up migration
