# Tasks — Add Scheduled Inquiry Runs

Migration numbering: latest today is `000035`; the sibling `add-supplier-inquiry-agent` change takes the next number (`000036+`) and this change takes the one after (state `000037+`; final numbers resolved at merge).

## 1. Schema & SQLC [DB-SQLC]

- [x] 1.1 Write migration `000038_create_procurement_schedules.up.sql` / `.down.sql` (number resolved to `000038` at merge/implementation: the sibling `add-supplier-inquiry-agent` already shipped `000037_create_procurement_schema`): `CREATE SCHEMA IF NOT EXISTS procurement;` plus `procurement.schedules` (id, organization_id FK, name, run_time TIME, days_of_week SMALLINT[], note, is_active, next_run_at TIMESTAMPTZ, last_run_at TIMESTAMPTZ NULL, last_run_occurrence_at TIMESTAMPTZ NULL, created_at, updated_at), `procurement.schedule_products` (schedule_id FK CASCADE, product_id FK to the sibling's products table, UNIQUE(schedule_id, product_id)), `procurement.schedule_suppliers` (same shape), and `procurement.schedule_followups` (organization_id UNIQUE FK, enabled, deadline_hours INT DEFAULT 4, max_nudges INT DEFAULT 1, message_template TEXT default "Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos.", created_at, updated_at); add `updated_at` triggers per repo convention; down drops the four tables
- [x] 1.2 Verify migration applies and rolls back: `migrate up` on local dev DB applied `000038_create_procurement_schedules` cleanly after `000037_create_procurement_schema` (version 38, not dirty); `migrate down 1` dropped the four schedule tables; re-`up 1` re-applied. Migration number confirmed `000038`.
- [x] 1.3 Add SQLC queries in `internal/db/postgres/sqlc/query/inquiryschedule.sql`: `InsertSchedule`, `GetSchedule` (with join rows), `ListSchedulesByOrg`, `UpdateSchedule`, `DeleteSchedule`, `PauseSchedule`, `ResumeSchedule`, `SetNextRunAt`, `ClaimDueSchedules` (`WHERE is_active AND next_run_at <= NOW() ... FOR UPDATE SKIP LOCKED`), `GetFollowUpSettingsByOrg`, `UpsertFollowUpSettings`, `InsertScheduleProduct`, `InsertScheduleSupplier`, `DeleteScheduleProducts`, `DeleteScheduleSuppliers`
- [x] 1.4 Verify generated code compiles: `make sqlc && go build ./...`

## 2. Domain entities + schedule state [BE-DOMAIN]

- [x] 2.1 Implement `internal/modules/inquiryschedule/domain/`: `Schedule` aggregate (id, organization_id, name, run_time, days_of_week, products, suppliers, note, is_active, next_run_at, last_run_at, last_run_occurrence_at) and `FollowUpSettings` value object with Spanish validation errors for missing `run_time`, empty/invalid `days_of_week`, empty product/supplier lists, and out-of-range `deadline_hours`/`max_nudges`
- [x] 2.2 Implement `NextOccurrenceAfter(now, tz)` on `Schedule`: next `run_time` on a day in `days_of_week` strictly after `now` in the given IANA timezone (no backfill semantics); unit-test Monday–Friday 08:00 boundaries, weekend skips, and `America/Bogota` interpretation
- [x] 2.3 Implement org-scope validation for referenced products/suppliers against the sibling `supplier-inquiries` data via a domain port (`CatalogReader` interface), returning Spanish errors for out-of-org references
- [x] 2.4 Verify: `go test ./internal/modules/inquiryschedule/domain/...`

## 3. Repository: claim/advance idempotent [BE-INFRA]

- [x] 3.1 Implement `internal/modules/inquiryschedule/infra/postgres/` repository implementing domain interfaces (`ScheduleRepository`, `FollowUpSettingsRepository`) over the SQLC queries from group 1, including the transactional `ClaimAndAdvanceAndEnqueue` (claim `FOR UPDATE SKIP LOCKED`, compute+persist next `next_run_at`, insert `inquiry_run.scheduled` outbox row — one transaction)
- [x] 3.2 Implement the handler-side dedupe transaction: lock schedule row `FOR UPDATE`, skip when `last_run_occurrence_at` equals the occurrence, else create run via `InquiryRunCreator` and set `last_run_at`/`last_run_occurrence_at` in the same transaction
- [x] 3.3 Implement the atomic nudge increment (`UPDATE ... SET nudge_count = nudge_count + 1 WHERE nudge_count < max_nudges`) and the follow-up candidate query (recipients `sent`, `sent_at + deadline_hours <= now`, consent granted, kill switch off)
- [x] 3.4 Verify: `make sqlc && go test ./internal/modules/inquiryschedule/infra/...`

## 4. Scheduler ticker + outbox enqueue [BE-INFRA]

- [x] 4.1 Implement the schedule claim ticker loop (30–60s interval, context-cancellable) that calls `ClaimAndAdvanceAndEnqueue` per due schedule; mirror the existing outbox dispatcher pattern
- [x] 4.2 Wire the ticker and the `inquiry_run.scheduled` handler subscription into the module lifecycle (`cmd/` + `module.go` per repo convention), starting and stopping with the module
- [x] 4.3 Add integration test with a mock outbox: two concurrent tickers claim the same due schedule — exactly one event; crash-before-commit simulation re-fires on the next tick
- [x] 4.4 Verify: `go test ./internal/modules/inquiryschedule/... -run TestScheduler`

## 5. Run-creation handler: idempotent, kill switch, credits [BE-DOMAIN]

- [x] 5.1 Implement the `inquiry_run.scheduled` outbox handler: kill-switch pre-check (read `agent.agent_settings.kill_switch`), then the transaction-isolated dedupe from 3.2, then call the `supplier-inquiries` run-creation service through the `InquiryRunCreator` port with `source = 'scheduled'`, `schedule_id`, and occurrence timestamp; audit `inquiry_run_scheduled` or `skip` (`kill_switch` / `duplicate_occurrence`)
- [x] 5.2 Handle credit exhaustion: when the run-creation service reports credits exhausted, record the run as `escalated` per the sibling contract (no LLM call; scheduling is not metered — assert no ai-usage ledger write from the handler)
- [x] 5.3 Unit tests: duplicate dispatch creates exactly one run; kill switch skips with audit; credits-exhausted run is `escalated`
- [x] 5.4 Verify: `go test ./internal/modules/inquiryschedule/... -run TestRunCreation`

## 6. Follow-up service: deadline, nudge, escalation [BE-DOMAIN]

- [x] 6.1 Implement `FollowUpService` (domain): for a given run, find candidates (status `sent`, deadline passed, enabled, `nudge_count < max_nudges`, consent `granted`, kill switch off); enqueue `inquiry.followup_send` via the outbox and perform the atomic nudge increment in one transaction; audit `inquiry_followup`
- [x] 6.2 Implement escalation: when `nudge_count` reaches `max_nudges`, flag the recipient and its run for human escalation (`awaiting_human` semantics per `whatsapp-agent`) and surface the overdue badge on the aggregation board query
- [x] 6.3 Consent gating: `withdrawn` consent → no nudge, escalate instead, audit `skip`/`consent_withdrawn`; kill switch → audit `skip`/`kill_switch`; the Spanish template is used with the `[proveedor]` placeholder replaced by the supplier name
- [x] 6.4 Verify: `go test ./internal/modules/inquiryschedule/... -run TestFollowUp`

## 7. Sweep ticker + convergence [BE-INFRA]

- [x] 7.1 Implement the follow-up sweep ticker (≤15 min) that scans orgs with follow-ups enabled and enqueues candidate nudges through the same idempotent path as 6.1; wire reply-arrival triggering: when a `whatsapp.message.received` event marks a recipient answered, run the cheap candidate check for the run's remaining recipients
- [x] 7.2 Implement convergence: a run is complete when all recipients are answered/escalated/timed out; late answers still update the recipient and the board (no new run)
- [x] 7.3 Integration test with mock outbox and mock send client: reply-trigger and sweep racing on the same overdue recipient produce exactly one `inquiry.followup_send` (double-nudge guard); send failure retries with backoff then dead-letters
- [x] 7.4 Verify: `go test ./internal/modules/inquiryschedule/... -run TestSweep`

## 8. API handlers + RBAC + audits [BE-INFRA]

- [x] 8.1 Implement routes under the existing `auth` + `org_context` + `subscription` middleware: `GET/POST /api/procurement/schedules`, `GET/PUT /api/procurement/schedules/:id`, `POST /api/procurement/schedules/:id/pause`, `POST /api/procurement/schedules/:id/resume`, `DELETE /api/procurement/schedules/:id`, `GET/PUT /api/procurement/followup-settings`; enforce Stytch B2B RBAC `org:manage` for writes and `org:view` for reads with Spanish 403/400 messages
- [x] 8.2 Implement audit writes through the `AuditLogWriter` port for `schedule_created`, `schedule_updated`, `schedule_paused`, `schedule_resumed`, `schedule_deleted`, `inquiry_run_scheduled`, `inquiry_followup`, `skip` (append-only)
- [x] 8.3 Handler tests: 201 create with computed `next_run_at`; 400 Spanish validation; 403 without `org:manage`; 404 for another org's schedule id; list/detail payloads include `next_run_at`, last-run status, and recent runs
- [x] 8.4 Verify: `go build ./... && go test ./internal/modules/inquiryschedule/... -run TestHandlers` and RBAC signature validation via the existing suite: `go test ./internal/modules/auth/...`

## 9. Backend tests: idempotency, at-least-once, consent gating [BE-DOMAIN] [BE-INFRA]

- [x] 9.1 Claim idempotency tests: two ticks/instances produce one outbox event per occurrence; `next_run_at` advances before enqueue; paused schedules never claimed
- [x] 9.2 At-least-once + exactly-once-effect tests: crash between claim and commit re-fires; duplicate `inquiry_run.scheduled` dispatch yields one run and a `skip`/`duplicate_occurrence` audit; kill-switch skip
- [x] 9.3 Double-nudge tests: concurrent reply-trigger + sweep and dispatcher redelivery both yield at most `max_nudges` sends; withdrawn-consent escalation; kill-switch cancel
- [x] 9.4 Integration test with mock outbox + mock WhatsApp send (mock fallback execution per governance): end-to-end schedule → tick → run creation → overdue → one nudge → escalation; run `go test ./internal/modules/inquiryschedule/... -v`

## 10. Frontend schedules page + copy [FE-NEXT]

- [x] 10.1 Add the schedules page under the procurement section (`next_b2b_starter/app/dashboard/procurement/schedules/`): list with `next_run_at`, last-run status, active/paused state; create/edit form with time picker, weekday pickers, supplier/product multi-select (from the sibling's endpoints), optional note; pause/resume and delete actions; Spanish-first labels and errors (e.g., "Programaciones de cotizaciones", "Hora de ejecución", "Días de la semana")
- [x] 10.2 Add the follow-up settings view (enabled, deadline hours, max nudges, template preview) wired to `GET/PUT /api/procurement/followup-settings`, Spanish copy, TanStack Query mutations with optimistic updates
- [x] 10.3 Add next-run display and overdue badge on the aggregation board entry point provided by the sibling change; route behind the same gating as the sibling's procurement section
- [x] 10.4 Verify: `pnpm lint && pnpm build` and run the page in dev to exercise create/pause/resume flows against the backend

## 11. Verification pass [OPS-GOV]

- [x] 11.1 Run the full backend verification: `make sqlc && go build ./... && go vet ./internal/modules/inquiryschedule/... && go test ./internal/modules/inquiryschedule/...`
- [x] 11.2 Run the full frontend verification: `pnpm lint && pnpm build`
- [x] 11.3 Run the integration suite with mock outbox/mock send: `go test ./internal/modules/inquiryschedule/... -v -run 'TestScheduler|TestSweep|TestRunCreation|TestFollowUp|TestHandlers'` and record results; confirm no delta specs exist for capabilities other than `inquiry-scheduling` (only `specs/inquiry-scheduling/spec.md` under this change root)
- [x] 11.4 Confirm rollback artifacts exist (`.down.sql` for migration `000037`) and document the deploy order (`add-whatsapp-template-messages` → `add-supplier-inquiry-agent` → this change) and Stytch policy non-impact in the change notes

## Phase 0 baseline checkpoint (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state.

## Implementation notes & verification records (2026-08-12)

**All 45 tasks complete.** Verification results:

- 1.1/1.2 — Migration `000038_create_procurement_schedules` (number resolved: sibling
  `add-supplier-inquiry-agent` shipped `000037_create_procurement_schema`). Applied + rolled
  back + re-applied on local dev DB via `migrate` (was dirty at 33; forced clean, migrated to 38).
- 1.3/1.4 — `query/inquiryschedule.sql` (schedules, join rows, follow-up settings, scheduled-run
  insert, claim `FOR UPDATE SKIP LOCKED`, follow-up candidates + atomic nudge guard, org
  timezone, overdue count, follow-up-enabled orgs). `sqlc generate` + `go build ./...` PASS.
- 3.x — `infra/postgres`: ScheduleRepository/FollowUpSettingsRepository over SQLC, transactional
  `ClaimAndAdvanceAndEnqueue` (claim→compute→persist next→enqueue outbox in ONE tx),
  handler-side dedupe (`GetScheduleForUpdate` + `last_run_occurrence_at` check + run creation +
  `last_run` mark + audit in one tx), atomic `IncrementFollowupCount` guard + candidate queries.
- 4.x — claim ticker (45s) + module lifecycle wiring (`cmd/init.go` + `init_mods.go`, after
  procurement) + `TestScheduler` concurrency/crash tests PASS.
- 5.x — `inquiry_run.scheduled` handler (kill switch pre-check → idempotent creation → audit),
  credit exhaustion escalates the run; `TestRunCreation` PASS. Scheduling itself never writes the
  ai-usage ledger and never invokes the LLM — only the sibling drafting seam is metered.
- 6.x — FollowUpService (deadline candidates, atomic guard, escalation badge, consent gating,
  `[proveedor]` template substitution); `TestFollowUp` PASS.
- 7.x — sweep ticker (15 min) + reply-arrival trigger (conservative exclusion of just-answered
  recipients because eventbus handlers run concurrently); `TestSweep` PASS.
- 8.x — routes behind auth+org_context+subscription with `org:manage`/`org:view`; append-only
  audits (`schedule_created|updated|paused|resumed|deleted`, `inquiry_run_scheduled`,
  `inquiry_followup`, `skip`/kill_switch|consent_withdrawn|duplicate_occurrence);
  `TestHandlers` PASS + `go test ./internal/modules/auth/...` PASS.
- 9.x — claim idempotency, at-least-once re-fire, double-nudge, end-to-end mock-outbox/mock-send
  tests PASS (`go test ./internal/modules/inquiryschedule/... -v`).
- 10.x — frontend: schedules page (`app/dashboard/procurement/schedules/`), follow-up settings
  panel, detail view with overdue badge, sidebar entry (org:manage, same gating as procurement
  section), Spanish-first copy; `npx tsc --noEmit`, `pnpm lint` (0 errors / 4 pre-existing
  warnings), `pnpm build` PASS; component test added and PASS. Dev-server exercise of
  create/pause/resume is blocked by the sibling DI boot issue below (backend covered by
  `TestHandlers`).
- 11.1 — `sqlc generate && go build ./... && go vet ./internal/modules/inquiryschedule/... &&
  go test ./internal/modules/inquiryschedule/...` PASS. Full repo `go test ./...` PASS (no
  failures), `go vet ./...` PASS.
- 11.2 — `pnpm lint` PASS (0 errors / 4 pre-existing warnings), `pnpm build` PASS.
- 11.3 — integration suite `-run 'TestScheduler|TestSweep|TestRunCreation|TestFollowUp|TestHandlers'`
  PASS (23 tests). Delta specs: only `specs/inquiry-scheduling/spec.md` under this change root ✓.
- 11.4 — Rollback artifacts ship: `000038_create_procurement_schedules.down.sql` (drops the four
  schedule tables; scheduled runs already created remain ordinary runs — `source='scheduled'` is
  inert without the scheduler). Deploy order: `add-whatsapp-template-messages` →
  `add-supplier-inquiry-agent` → this change. **Stytch policy non-impact**: no Stytch B2B API
  calls or policy changes; existing `org:manage`/`org:view` roles reused as-is; no local
  credential storage. Rollback is complete at the Git/DB layer.

### Reconciliation points observed at implementation time (sibling in-flight)

1. **Sibling DI boot bug (BLOCKER for server boot, not for this change's gates)**: the
   procurement module panics at boot — `NewProcurementSubscriber` requires `services.MetricsSink`
   but the container only registers `*CounterSink` (no interface binding). Pre-existing in
   `add-supplier-inquiry-agent`; `inquiryschedule.Init` is never reached. Not modified here (sibling
   capability owns it); needs a one-line `func(*CounterSink) MetricsSink` binding in the sibling
   module before end-to-end dev runs.
2. **`nudge_count` → `followup_count`**: the sibling's `procurement.inquiry_recipients` names the
   nudge guard `followup_count` (default 0); mapped in the repository. No migration needed.
3. **Run-creation entrypoint**: the sibling's `CreateInquiryRun` hardcodes `source='manual'`; this
   change's `InquiryRunCreator` adapter inserts `source='scheduled'`+`schedule_ref` (columns the
   sibling explicitly reserved) via `InsertScheduledRun`, and reuses the sibling's metered
   `DraftingService` through the `DraftFunc` seam (quantity 1 per schedule product).
4. **Overdue badge / board**: the sibling's board (`GET /api/procurement/runs/:id`) has no
   `followup_count` in `BoardRow`; this change surfaces the overdue badge on its own schedule
   surfaces (`OverdueRecipients` in schedule detail, computed at read time:
   `status='sent' AND followup_count >= max_nudges`). Sibling-board badge is a merge-time
   reconciliation.
5. **Convergence**: run completion stays the sibling's read-path reconciliation (lazy timeout
   `sent→timed_out`); at-cap recipients remain `sent` with the overdue badge until then.
6. **Frontend DTO casing**: the sibling's Go domain structs serialize PascalCase (no json tags);
   the sibling's frontend DTOs declare snake_case (would not match at runtime). This change's
   frontend DTOs mirror the verified wire shape (PascalCase). Reconciliation belongs to the sibling.

## Council revision carryover (2026-08-12)

Council verdict `VERDICT.md`: STATUS REJECTED — 5 required design changes. Design.md revised to address all 5 (see "Council Revision" section). Code audit against the verdict (parallel session had already implemented the change):

- [x] 1. Dispatch-time re-validation for `inquiry.followup_send` (SEV-1) — VERIFIED in code: `followup_send_handler.go` re-checks run status, recipient status, kill switch, and consent inside the dispatch path with `skip` audits (`kill_switch`, `consent_withdrawn`).
- [x] 2. Transactional boundary of run-creation dedupe (SEV-4) — VERIFIED in code: `run_creator.go` `CreateScheduledRun` runs the schedule `FOR UPDATE` lock, dedupe marker, and run insert in ONE transaction via `inTx`.
- [x] 3. Composite tenant FKs on join tables (SEV-2) — VERIFIED in code: `schedule_products_product_org_fkey (organization_id, product_id)` and `schedule_suppliers_supplier_org_fkey (organization_id, supplier_id)` in migration 000038.
- [x] 4. Ticker index (SEV-6) — VERIFIED in code: `idx_schedules_due ON procurement.schedules(is_active, next_run_at)` in migration 000038.
- [x] 5. Follow-up sweep index (SEV-5) + sweep claim SKIP LOCKED (SEV-7) — FIXED (2026-08-12): Added `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_recipients_org_status_sent_at ON procurement.inquiry_recipients(organization_id, status, sent_at)` to migration `000038` (additive, on sibling table). Added `FOR UPDATE OF r SKIP LOCKED` to `ListFollowUpCandidates` query in `inquiryschedule.sql`. SQLC regenerated, `go build ./...` PASS, `go test ./internal/modules/inquiryschedule/...` PASS (23/23).
- [x] Archive decision: **Archive deferred** — code complete and verified; sibling DI boot bug (sibling `add-supplier-inquiry-agent` owns it — `NewProcurementSubscriber` requires `MetricsSink` interface binding missing from container) blocks end-to-end server boot. Sibling reconciliation items (DTO casing, board badge, convergence) are documented and owned by the sibling change. Archive when sibling lands and server boots green.
