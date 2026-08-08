## Why

The CRM schema captures the right objects but is under-constrained at the database layer. The database currently accepts invalid states that application-level scoping is expected to prevent on its own: cross-tenant assignment (`contacts.assigned_to`, `deals.assigned_to`, `companies.owner_account_id`, `activities.realizada_por` can reference an account from another organization), deals whose `stage_id` belongs to a different `pipeline_id`, and a check-then-insert message flow that surfaces unique violations (HTTP 500) on concurrent or retried WhatsApp webhooks.

## What Changes

- Add `UNIQUE (organization_id, id)` tenant-scoped keys on parent tables (`accounts`, `contacts`, `companies`, `deals`, `pipelines`, `pipeline_stages`, `conversations`, `tags`) so composite FKs are possible.
- Replace 10 single-column FKs with composite tenant-safe FKs using `ON DELETE SET NULL (column_list)` (PostgreSQL 16/17) or `CASCADE` **exactly matching current delete semantics** — no behavior change for existing delete flows.
- Enforce stage↔pipeline consistency: `UNIQUE (organization_id, id, pipeline_id)` on `pipeline_stages`, a triple-key composite FK from `deals`, and a BEFORE trigger (`crm_deals_sync_pipeline_from_stage`) that derives `pipeline_id` from `stage_id` on insert/update.
- Replace the message check-then-insert with `InsertMessageIdempotent` (`INSERT ... ON CONFLICT (organization_id, whatsapp_message_id) DO NOTHING RETURNING *` with fallback fetch), closing the webhook-retry race in `ProcessInboundMessage`.
- Add an isolated one-active-conversation partial unique index (`(organization_id, contact_id) WHERE status = 'active'`) plus idempotent conversation creation, closing the same race class for conversations.
- Add a Go DB integration test harness (testcontainers-go, pgvector/pg16) covering FK violations, delete semantics, stage normalization, and idempotency.
- Pre-migration audit queries with repair policies; down migrations restoring the prior FK shape.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- `crm-core-data`: message insertion becomes idempotent at the DB layer; assignment and parent/child references become composite tenant-safe FKs; one-active-conversation invariant per contact.
- `pipeline-management`: deals can no longer hold a stage from another pipeline; `pipeline_id` is derived from `stage_id` by the database.
- `lean-data-isolation`: tenant isolation strengthened from query-layer convention to database referential constraints (composite FKs); RLS remains optional (MAY), unchanged.
- `whatsapp-webhook-ingress`: duplicate or retried webhooks must not create duplicate messages (idempotent insert replaces check-then-insert).

## Impact

- **DB**: new migration `000016_create_crm_integrity_constraints` (up/down) in `go-b2b-starter/internal/db/postgres/sqlc/migrations/`.
- **SQLC**: new queries `InsertMessageIdempotent` and `InsertActiveConversationIdempotent` in `internal/db/postgres/sqlc/query/crm.sql`; regenerated `gen/` models; other query shapes unchanged.
- **App**: `internal/modules/crm/app/services/crm_service.go` (`ProcessInboundMessage` drops the pre-insert `GetByWhatsAppID` check), `internal/modules/crm/infra/repositories/message_repository.go` (new/updated repository methods).
- **Tests**: new integration test package (testcontainers) + `make test-integration` target; no impact on the in-flight `add-crm-e2e-tests` Playwright specs (delete semantics preserved).
- **Auth boundary**: no change. This change does not touch the Stytch B2B runtime SSOT — no credentials, sessions, or identity data are added locally; `stytch_member_id`/`stytch_organization_id` linkage and all Stytch B2B API contracts (JWKS verification, webhook signatures, circuit breaker) are unaffected.
- **Rollback**: down migration drops the composite FKs and trigger and restores the original single-column FKs (Git state reversible). Stytch tenant policy state requires no rollback because this change never mutates Stytch state. Data repairs (nulled assignments, normalized `pipeline_id`) are recorded in audit output before migration and are not automatically reversible.

## Non-Goals

- Contact identity redesign (`contact_identifiers`, merge workflows, alias routing).
- `whatsapp_configs` 1:N multi-number support.
- `message_events` history, `notes`, custom fields, search/vector tables.
- English renaming of Spanish columns, RLS enablement, blanket soft-delete, timestamp type migration.
- Replacing `entity_tags` with explicit tag tables (tracked separately).
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
