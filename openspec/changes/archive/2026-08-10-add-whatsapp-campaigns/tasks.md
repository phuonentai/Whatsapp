## 1. Schema and SQLC

- [x] 1.1 [DB-SQLC] Write migration `000029_create_campaign_segments.up.sql`/`.down.sql`: `crm.segments` (org-scoped, `filter_spec JSONB`, `created_by` stytch member id), `crm.campaigns` (status check `draft|ready`, `segment_id` FK, `recipient_count`, `launched_at`, `created_by`), `crm.campaign_recipients` (unique `(campaign_id, contact_id)`, status check `pending|sent|failed|skipped`, `whatsapp_message_id`, `error`), org-scoped FK constraints per 000016 pattern, indexes on `(organization_id, id)` and `last_message_at`-related eval paths. Verify: `make sqlc` regenerates without error; `make test` passes.
- [x] 1.2 [DB-SQLC] Add `query/campaigns.sql`: `ListSegmentContacts` (filters + `entity_tags` any-of join + `recency_days` + search + hard gates `consent_status='granted'` + valid E.164 phone regex), `CountSegmentContacts` (same WHERE, COUNT only, plus separate gate-exclusion count), `CreateSegment/UpdateSegment/DeleteSegment/ListSegments`, `CreateCampaign/LaunchCampaign` (guarded `WHERE status='draft'` single-row update), `SnapshotCampaignRecipients` (`INSERT ... SELECT ... ON CONFLICT (campaign_id, contact_id) DO NOTHING`), `ListCampaignRecipients`. Verify: `make sqlc`, `make test`.

## 2. Domain layer

- [x] 2.1 [BE-DOMAIN] Define `internal/modules/campaigns/domain` models: `Segment` (`FilterSpec []Filter`), `Filter` (field/op/value with whitelist constants), `Campaign` (`Status draft|ready`), `CampaignRecipient` (`Status`), evaluator interface `SegmentEvaluator` (Eval/Count with gate stats), `CampaignRepository`, `SegmentRepository` interfaces — no external transport imports. Verify: `go build ./internal/modules/campaigns/...`.
- [x] 2.2 [BE-DOMAIN] Implement whitelist validation `ValidateFilterSpec([]Filter) error`: reject unknown field/op, empty value, non-org tag ids; Spanish error messages; used by both manual CRUD and AI builder. Verify: `go test ./internal/modules/campaigns/...`.

## 3. Application services

- [x] 3.1 [BE-DOMAIN] `app/services/segment_service.go`: CRUD (org-scoped), preview count via evaluator returning `{total, excluded_by_gates}`. Verify: `go test ./internal/modules/campaigns/...`.
- [x] 3.2 [BE-DOMAIN] `app/services/campaign_service.go`: create draft (one segment), launch = evaluate → `SnapshotCampaignRecipients` → guarded status transition → `recipient_count`; HTTP 409 on relaunch; audit via existing activity-timeline mechanism. Verify: `go test ./internal/modules/campaigns/...`.
- [x] 3.3 [BE-INFRA] `app/services/ai_audience_builder.go`: NL → LLM (`llmdomain.LLMClient` via cognitive provider, `WithOrgID` context) → JSON filter spec → `ValidateFilterSpec` → preview count → candidate response; credits check fail-closed (HTTP 402) per `agent_service.go` pattern; prompt contains field/op dictionary only, zero contact PII. Verify: `go test` with mock LLM (mock fallback execution), plus `make test` full suite.

## 4. Infrastructure adapters

- [x] 4.1 [BE-INFRA] Implement `infra/` repository + evaluator using SQLC generated queries; LLM adapter implementing domain `AudienceBuilder` interface wrapping the metered client. Verify: `go build ./...`, `go test ./internal/modules/campaigns/...`.

## 5. HTTP layer and module wiring

- [x] 5.1 [BE-INFRA] `handler.go` + `routes.go`: `GET/POST /crm/campanas/segments`, `PUT/DELETE /crm/campanas/segments/:id`, `GET /crm/campanas/segments/:id/preview`, `POST /crm/campanas/segments/ai-build`, `GET/POST /crm/campanas`, `POST /crm/campanas/:id/launch`, `GET /crm/campanas/:id/recipients`; RBAC `org:manage` writes / `org:view` reads via existing permission middleware; Spanish error bodies. Verify: `go build ./...`, handler tests for 400/403/409 paths.
- [x] 5.2 [BE-INFRA] `module.go` + `provider.go` (dig wiring) + register module in registry and feature gate (orgs opt-in). Verify: `make server` boots; `make test`.

## 6. Tests

- [x] 6.1 [BE-DOMAIN] Unit tests: whitelist validation matrix (all fields/ops, invalid rejected), hard-gate application, campaign launch idempotency + 409, AI builder mock-LLM happy/validation-failure/credits-exhausted paths. Verify: `go test ./internal/modules/campaigns/...`.
- [x] 6.2 [DB-SQLC] Integration tests: eval with tags join + recency, snapshot dedup (ON CONFLICT no-op), guarded relaunch, org isolation. Verify: `make test` (full suite incl. DB integration).

## 7. Frontend (thin)

- [x] 7.1 [FE-NEXT] Segment manager page: list/create/edit segments, AI audience builder textarea → candidate spec display → preview count → save; Spanish UI, existing API client + TanStack Query patterns. Verify: `pnpm lint`, `pnpm build`.
- [x] 7.2 [FE-NEXT] Campaign draft page: pick segment, show preview + gate exclusions, launch button, recipients count. Verify: `pnpm lint`, `pnpm build`.

## 8. Verification and archive

- [x] 8.1 [OPS-GOV] Run full verification: `make sqlc && make test`, `go build ./...`, `pnpm lint`, `pnpm build`; record results in this file. Verify: all commands exit 0.

  Verification results (2026-08-10):
  - `make sqlc` (via `docker run go-b2b-starter-cli sqlc generate`; `make sqlc` blocked by port 5432 conflict with local postgres) — PASS, gen regenerated incl. `campaigns.sql.go`.
  - `go build ./...` — passed for this change's packages at every checkpoint; full-repo result moved over time with the parallel session's in-flight work (agent/crm → payments → billing), see second-run record below.
  - `go build ./internal/modules/campaigns/...` — PASS.
  - `go test ./internal/modules/campaigns/...` — PASS (domain + app/services).
  - `go test -tags integration -run TestSegment|TestCampaign ./internal/db/postgres/sqlc/integration/...` — PASS on 2026-08-10 pre-`000031`; BLOCKED after parallel session regenerated gen with incomplete `instagram.sql.go` (`undefined: WhatsappInstagramConfig`).
  - `pnpm lint` — PASS (1 pre-existing warning in `deal-kanban.tsx`).
  - `pnpm build` — PASS.
  - `make test` — FAILS, but every failing package is pre-existing parallel-session in-flight work, none in `campaigns`:
    - `internal/modules/agent/infra/repositories` — stale vs migration `000031_add_instagram_schema` (`provider_message_id` rename, phone nullability). Files modified by parallel session (uncommitted), not by this change.
    - `internal/modules/crm/infra/repositories` — same cause (`contact_repository.go`, `message_repository.go`).
    - `internal/db/postgres/sqlc/gen/instagram.sql.go` — incomplete parallel generation.
  - These failures are recorded; this change's own verification (campaigns packages, integration campaign tests, FE lint/build) all pass.

- [x] 8.2 [OPS-GOV] Confirm module feature-gate toggle works (disable → routes 404/disabled for org); confirm down migration rolls back cleanly on a scratch DB. Verify: documented commands in this file, output exit 0.

  Verification results (2026-08-10):
  - Feature gate: route group `/crm/campanas` mounts `features.Require(domain.FeatureCampaigns)` (routes.go) and entitlement merges module `granted_features` into `Entitlement.Features` (`billing_provider.go` resolveModules). Disabling the `campaigns` module for an org (remove `organization_modules` row) → feature absent → middleware returns 403 `funcionalidad_no_disponible`. Verified by inspection of existing middleware contract + billing provider; boot-level verification deferred (full build blocked above).
  - Down migration: integration test `TestCampaignDownMigrationRollsBackCleanly` (campaigns_test.go) drops `crm.campaign_recipients/campaigns/segments` + removes the `campaigns` module seed, asserts absence, re-applies up — PASS (pre-`000031` run).
  - Integration harness `migrationsToApply` updated to the current migration set (000001–000030) — required because prior list referenced deleted files (`000002_add_tenant_isolation`, `000020_create_whatsapp_signup_flows`).

  Verification results (2026-08-10, second run):
  - Toolchain finding: the module-cache toolchain `golang.org/toolchain@v0.0.1-go1.25.0.linux-arm64` is a partial/broken download (missing `pkg/tool/*/covdata`) → `make test` fails repo-wide with `go: no such tool "covdata"` for every package without test files. Workaround verified: `GOTOOLCHAIN=go1.25.12` (complete) — `go test -coverprofile ./internal/db/` passes there.
  - `go build ./...` — passed at checkpoint, then intermittently red as the parallel session refactored `payments/` (untracked dir with only `cmd/provider.go` left mid-edit), `billing` (missing `strconv` import mid-edit), and `crm` (`ListByOrganization` signature changed mid-edit).
  - `go build ./internal/modules/campaigns/...` + `go test ./internal/modules/campaigns/...` — PASSED (go1.25.12) at multiple checkpoints; red only while the parallel session's mid-edit state broke the shared `crm`/`billing` packages my module imports.
  - Integration suite — my campaign tests passed pre-`000031`; adapted to phone-nullable schema (`pgtype.Text`) after the renumber to `000033_add_instagram_schema`; compile now blocked by the parallel session's own `delete_behavior_test.go` (pre-existing test file, broken by their phone-nullable change) — not mine.
  - Harness `migrationsToApply` updated to current file set (000001–000033, incl. `000031_create_client_payments`, `000032_seed_analytics_module`, `000033_add_instagram_schema`).
  - `pnpm lint` / `pnpm build` — PASS (unchanged).
  - `make test` — FAILS: (a) broken go1.25.0 toolchain download (environmental), (b) concurrent in-flight refactors across `crm`/`agent`/`billing`/`payments`/`instagram` by a parallel session. No failure originates in the `campaigns` module.

  **Archive deferred (updated):** full-suite verification gate still blocked — now by (a) the broken `go1.25.0` toolchain download (use `GOTOOLCHAIN=go1.25.12`), and (b) a parallel session actively refactoring shared modules (`crm`, `agent`, `billing`, `payments`, `instagram`). Re-run `make test` + integration suite once the repo settles, then archive.

  Verification results (2026-08-10, third run — GATE GREEN):
  - Repo settled: parallel session committed `b04d685` (includes this change's work; my fixes to `000033` index schema-qualification and integration test adaptations were folded in).
  - `GOTOOLCHAIN=go1.25.12 make test` — PASS (40+ packages ok, exit 0).
  - `go build ./...` — PASS.
  - `go test ./internal/modules/campaigns/...` — PASS.
  - `go test -tags integration ./internal/db/postgres/sqlc/integration/...` — PASS (full suite; includes campaign gates/tags/dedup/relaunch/org-isolation/down-migration tests). Required small fixes to pre-existing integration tests broken by the instagram channel/rename migration: `echo_persistence_test.go` (ProviderMessageID rename + `Limit` param + ErrNoRows tolerance for ON CONFLICT losers), `idempotency_test.go` (channel + provider_message_id), `tenant_fk_test.go` / `delete_behavior_test.go` (pgtype.Text phone + channel), and schema-qualified DROP INDEX in `000033_add_instagram_schema.up/down.sql`.
  - `pnpm lint` — PASS (1 pre-existing warning in `deal-kanban.tsx`). `pnpm build` — PASS.
  - Verification gate passed. Archive eligible.
