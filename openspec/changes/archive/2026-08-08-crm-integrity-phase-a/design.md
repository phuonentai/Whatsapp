## Context

The CRM schema (migrations `000010`–`000013`) models contacts, conversations, messages, companies, pipelines, stages, deals, activities, and tags in the `crm` schema. Two integrity gaps exist:

1. **Tenant isolation is app-convention only.** `contacts.assigned_to`, `deals.assigned_to`, `companies.owner_account_id`, and `activities.realizada_por` are single-column FKs to `organizations.accounts(id)` with no constraint that the account belongs to the same `organization_id`. A buggy or malicious query can assign cross-tenant references.
2. **Check-then-insert races.** `ProcessInboundMessage` (crm_service.go:75-89) does `GetByWhatsAppID` then `CreateMessage`. Under concurrent/retried webhooks the second insert hits the existing `idx_messages_whatsapp_id` unique index and surfaces as a 500. The same race exists for conversations (`GetActiveConversationByContact` then `CreateConversation`).
3. **Stage/pipeline corruption.** `deals.stage_id` and `deals.pipeline_id` are independently writable; nothing prevents a stage from another pipeline.

Environment: PostgreSQL 16 (dev) / 17 (prod) via `pgvector/pgvector` images — both support `ON DELETE SET NULL (column_list)` (PG 15+). The app uses SQLC-generated queries (`internal/db/postgres/sqlc/gen/`), Clean Architecture with repositories in `internal/modules/crm/infra/repositories/`.

## Goals / Non-Goals

**Goals:**
- Make cross-tenant assignment/parenting impossible at the DB layer via composite tenant-scoped FKs.
- Guarantee `deals.pipeline_id` always matches `deals.stage_id`'s pipeline.
- Make message (and active-conversation) insertion idempotent, eliminating the retry race.
- Preserve **exactly** the current delete semantics (SET NULL / CASCADE) — zero behavior change for existing delete flows.
- Provide a Go integration test harness proving the new failure modes are blocked.

**Non-Goals:**
- Contact identity redesign, merge workflows, `whatsapp_configs` 1:N, `message_events`, notes, custom fields, search/vector, RLS, English renaming, blanket soft-delete, timestamp type migration, `entity_tags` replacement.
- Any change to Stytch B2B contracts or local credential/session storage.

## Decisions

### D1: Composite tenant-scoped FKs with `ON DELETE SET NULL (column_list)`, not RESTRICT

The original review recommended `RESTRICT`. Rejected: the app hard-deletes contacts, companies, pipeline stages, deals (crm_extended.sql:44/106/186/251), and `DeleteOrganization` (organizations.sql:52) cascades into accounts; RESTRICT would break all of those flows. PG 15+ `ON DELETE SET NULL (col)` nulls only the listed column, preserving current semantics while adding tenant safety.

**Alternatives considered:** plain `SET NULL` (nulls `organization_id` too → NOT NULL violation; rejected), RESTRICT (behavior regression; rejected), RLS (deferred — all SQLC reads already scope by `organization_id`; composite FKs close the write hole).

Delete-action matrix (from existing migrations, verified nullable/action):

| FK (existing, `migration`) | Nullable | Composite replacement |
|---|---|---|
| `contacts.assigned_to` (000011, SET NULL) | yes | `SET NULL (assigned_to)` |
| `companies.owner_account_id` (000012, SET NULL) | yes | `SET NULL (owner_account_id)` |
| `deals.assigned_to` (000012, SET NULL) | yes | `SET NULL (assigned_to)` |
| `activities.realizada_por` (000013, SET NULL) | yes | `SET NULL (realizada_por)` |
| `contacts.company_id` (000012, SET NULL) | yes | `SET NULL (company_id)` |
| `deals.contact_id` (000012, SET NULL) | yes | `SET NULL (contact_id)` |
| `deals.company_id` (000012, SET NULL) | yes | `SET NULL (company_id)` |
| `conversations.contact_id` (000010, CASCADE, **NOT NULL**) | no | composite `CASCADE` |
| `messages.conversation_id` (000010, CASCADE, **NOT NULL**) | no | composite `CASCADE` |
| `deals.stage_id` (000012, SET NULL) | yes | `SET NULL (stage_id)` via triple-key FK |

`deals.pipeline_id` keeps its existing FK to `pipelines(id)` unchanged.

Prerequisite: `UNIQUE (organization_id, id)` on `accounts`, `contacts`, `companies`, `deals`, `pipelines`, `pipeline_stages`, `conversations`, `tags`.

### D2: Stage↔pipeline enforced by triple-key unique + FK + trigger

**Correction discovered during implementation:** `crm.pipeline_stages` currently has NO `organization_id` column (it links to pipelines only via `pipeline_id`). The triple-key unique and the composite FK both require it. The migration therefore MUST:

1. `ALTER TABLE crm.pipeline_stages ADD COLUMN organization_id INTEGER` (nullable).
2. Backfill from the owning pipeline: `UPDATE pipeline_stages ps SET organization_id = p.organization_id FROM pipelines p WHERE p.id = ps.pipeline_id`.
3. `SET NOT NULL` + add FK to `organizations(id)`.
4. Add a BEFORE trigger (`crm_pipeline_stages_sync_org`) on `INSERT OR UPDATE OF pipeline_id` that derives `organization_id` from the pipeline, so app code that creates stages via `pipeline_id` keeps the column correct.

- `UNIQUE (organization_id, id, pipeline_id)` on `pipeline_stages` (valid: id is PK, so the triple is inherently unique — cheap to add).
- `FOREIGN KEY (organization_id, stage_id, pipeline_id) REFERENCES pipeline_stages (organization_id, id, pipeline_id) ON DELETE SET NULL (stage_id)`.
- BEFORE trigger `crm_deals_sync_pipeline_from_stage` on `INSERT OR UPDATE OF stage_id` that looks up the stage's pipeline for the same org, raises if absent, and sets `NEW.pipeline_id`.

The trigger makes `stage_id` the source of truth while `pipeline_id` remains writable/readable by existing SQLC queries (`CreateDeal` writes both; `UpdateDealStage` writes only `stage_id` — both normalize before the FK validates).

**Alternatives considered:** drop `pipeline_id` and derive via view (churn on existing filters `ListDealsByOrganization` uses `d.pipeline_id`; rejected for this phase); app-layer validation only (rejected — the DB must not accept invalid states).

### D3: Idempotent message insert — the constraint already exists, only the flow changes

`idx_messages_whatsapp_id UNIQUE (organization_id, whatsapp_message_id)` already exists (000010). Duplicates are already impossible at the DB level; the defect is the check-then-insert in `ProcessInboundMessage`. New SQLC query:

```sql
-- name: InsertMessageIdempotent :one
INSERT INTO crm.messages (organization_id, conversation_id, contact_id, whatsapp_message_id, direction, message_type, content, status, message_data, chat_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (organization_id, whatsapp_message_id) DO NOTHING
RETURNING *;
```

Service flow: attempt idempotent insert; on `pgx.ErrNoRows` (no row returned because conflict) fetch via `GetMessageByWhatsAppID` and use the existing message. The pre-insert `GetByWhatsAppID` check is removed from the write path.

Note: the conflict target works because the existing unique index is non-partial; NULL `whatsapp_message_id` values never conflict (unique indexes ignore NULLs), so outbound messages with no provider ID are unaffected.

### D4: One-active-conversation invariant (isolated, reversible step)

Partial unique index `(organization_id, contact_id) WHERE status = 'active'` + `InsertActiveConversationIdempotent` (`ON CONFLICT (organization_id, contact_id) WHERE status = 'active' DO NOTHING RETURNING *` + fallback `GetActiveConversationByContact`). Matches the existing product model (`GetActiveConversationByContact` returns the single active conversation). Isolated in its own sub-step so it can be reverted independently if the multi-number product model emerges.

**Alternatives considered:** defer (rejected — same race class as messages, cheap to close); per-config uniqueness `(org, config_id, external_chat_id)` (premature — `whatsapp_configs` is 1:1 per org today).

### D5: Go integration test harness with testcontainers

No DB integration harness exists (only 4 unit test files). Add testcontainers-go with `pgvector/pgvector:pg16`, run migrations, expose a store handle. Tests cover: cross-tenant FK rejections (6), delete-behavior preservation (7), stage normalization (3), message idempotency incl. concurrent inserts (4), conversation idempotency (4). `make test-integration` target.

**Alternatives considered:** compose-DB harness (`make test` against docker-compose postgres — works but pollutes the dev DB; rejected), API-level Playwright assertions of 500s (weak signal; rejected).

## Risks / Trade-offs

- **Trigger + FK ordering race** — a stage deleted concurrently with a deal insert could leave the FK check unhappy. → Mitigation: FK validation happens under row locks after the trigger; the trigger lookup and FK check are within the same statement's snapshot; residual window is negligible for this app's write volume. Documented; covered by integration tests.
- **`ON DELETE SET NULL (column_list)` is PG15+** — dev/prod are 16/17 (verified in docker-compose). → Mitigation: assert `version() >= 15` guard is unnecessary; migration will fail loudly if run elsewhere.
- **NOT NULL columns cannot use SET NULL** — `conversations.contact_id`, `messages.conversation_id` must stay CASCADE. Verified nullable via `information_schema` before writing migrations (task 1).
- **Data repairs are not reversible** — nulled cross-tenant assignments and normalized `pipeline_id` values are recorded in audit output before migration. → Mitigation: snapshot/export audit table; repair policies are explicit (nullable → NULL; required → block migration).
- **`add-crm-e2e-tests` in flight** — Playwright specs exercise contact/company deletion. → Mitigation: delete semantics preserved exactly (D1 matrix), so no regression expected; sequencing coordinated.
- **Down migration restores schema, not data** — documented in rollback section; assignments nulled after Phase A is live are not restored by down migration.

## Migration Plan

1. Inventory current FK delete actions (`pg_constraint.confdeltype`) and column nullability (`information_schema.columns`).
2. Run audit queries (cross-tenant assignments ×4, stage/pipeline mismatches, orphan stages, message dupes → expected zero, active-conversation dupes).
3. Repair per policy (nullable → NULL; stage mismatch → set `pipeline_id = s.pipeline_id`; conversation dupes → keep newest, close older).
4. Add `UNIQUE (organization_id, id)` to the 8 parent tables.
5. Add composite FKs as `NOT VALID`, then `VALIDATE CONSTRAINT` (avoids long table locks).
6. Add stage triple-key unique + FK + trigger.
7. Add conversation partial unique index (isolated step).
8. Regenerate SQLC; deploy `InsertMessageIdempotent` / `InsertActiveConversationIdempotent` + repository/service changes.
9. Add integration tests; run full suite (`make test`, `make test-integration`, `pnpm build`/`pnpm lint` as needed).

**Rollback (Git + Stytch):** Down migration drops composite FKs, the trigger, the triple-key unique, and the conversation partial index, and recreates the original single-column FKs with original actions. Git state rolls back via revert of the migration + SQLC changes. Stytch tenant policy state requires no rollback — this change never mutates Stytch state and stores no credentials/sessions (constitution-compliant).

## Open Questions

- None blocking. (Harness: testcontainers-go confirmed available; no CI constraints documented.)
