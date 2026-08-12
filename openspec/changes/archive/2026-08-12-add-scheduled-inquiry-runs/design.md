# Design — Add Scheduled Inquiry Runs

## Context

The stockless-retail procurement loop (sibling capability `supplier-inquiries`) is a daily ritual: every morning the business checks supplier availability/price before quoting customers. Runs are currently triggered manually and unanswered suppliers are chased by hand. This change makes the ritual automatic: org-scoped recurring schedules (time-of-day + weekdays) durably create inquiry runs through the existing durable outbox, and per-recipient follow-ups (configurable deadline, one nudge, then human escalation) make runs converge.

Building blocks reused as-is (read from `openspec/specs/`): `durable-message-pipeline` (outbox, dispatcher, `FOR UPDATE SKIP LOCKED` claims, retry/backoff, dead-letter, replay), `whatsapp-outbound-send` (circuit-breakered Cloud API client, 10 msgs/10s rate limit), `whatsapp-agent` / `agent-governance` (flow lifecycle, kill switch, consent state machine), `whatsapp-compliance` (Ley 1581 consent states), `whatsapp-campaigns` (read-only reference; its "scheduler consumes recipients later" note stays future work). The `supplier-inquiries` capability (sibling change) does not exist yet in `openspec/specs/` and is consumed **by name/reference only** — this change creates delta specs exclusively for its own capability `inquiry-scheduling`.

Repo facts verified: latest migration is `000035`; outbox claim query already uses `FOR UPDATE SKIP LOCKED` (`query/outbox.sql`); org timezone lives on `agent.agent_settings.timezone` (default `America/Bogota`, `000019`); module convention is `internal/modules/<name>/` with `domain/`, `app/`, `infra/`, `cmd/` plus top-level `handler.go`/`routes.go`/`module.go`/`provider.go`; dashboard routes live under `next_b2b_starter/app/dashboard/`.

## Goals / Non-Goals

**Goals:**
- Org-scoped recurring schedules that durably create inquiry runs without human action, with next-run visibility.
- Exactly-once effect per scheduled occurrence despite at-least-once delivery (crash-safe claim, idempotent handler).
- Automated per-recipient follow-ups: one nudge after a configurable deadline, then human escalation — no follow-up spam.
- Follow-ups gated by consent (Ley 1581), kill switch, and the existing rate-limited, circuit-breakered send path.
- Read/status surfaces: schedule list with `next_run_at` + last-run status; schedule detail with recent runs.
- Fully additive: no existing spec modified, no delta specs for any capability other than `inquiry-scheduling`.

**Non-Goals:**
- External cron/scheduler service (in-process ticker only); DST-correct schedule computation beyond storing the org IANA timezone; backfill/reconciliation of missed occurrences (next-run computation only); WhatsApp template-message scheduling (sibling change); campaign scheduling generalization; Stytch B2B policy changes; local credential storage (explicitly rejected per repo governance).

## Decisions

### 1. In-process ticker, not an external cron service

The scheduler is an in-process ticker loop (30–60s) that follows the existing outbox dispatcher pattern (`ClaimPendingOutboxEvents ... FOR UPDATE SKIP LOCKED`). **Rationale**: zero new infrastructure, deployable/revertable with the app, and the dispatch side already has battle-tested claim semantics. **Alternatives**: external cron (new infra + credential management, rejected), pg_cron (DB extension not present, adds ops surface, rejected).

### 2. Claim + advance + enqueue in ONE transaction

The tick, for each due schedule (`is_active AND next_run_at <= now`), runs a single transaction: select the schedule `FOR UPDATE SKIP LOCKED`, compute and persist the *next* `next_run_at`, insert the outbox event `inquiry_run.scheduled` carrying `{schedule_id, organization_id, product_ids, supplier_ids, note, occurrence_at}`, and commit. **Rationale**: schedules and `public.outbox_events` live in the same PostgreSQL database, so a single transaction gives atomic claim→advance→enqueue: every occurrence produces exactly one outbox row, and a crash before commit rolls back cleanly so the next tick re-fires (at-least-once at the tick level). **Alternatives**: claim-then-enqueue as separate steps (crash window between advance and enqueue either loses the occurrence or needs compensation; rejected), enqueue-then-advance (duplicate enqueue risk on retry; rejected).

### 3. Exactly-once effect via idempotent handler, with an explicit transaction boundary

Despite transactional enqueue, the outbox dispatcher can redeliver (handler succeeds, dispatch-completion mark fails → retry). The run-creation handler therefore dedupes per occurrence: in one transaction it locks the schedule row (`FOR UPDATE`) and skips creation when a run for `(schedule_id, occurrence_at)` was already recorded (`last_run_occurrence_at IS DISTINCT FROM $occurrence`), then creates the run via the `supplier-inquiries` run-creation service (`source = 'scheduled'`, `schedule_id`, occurrence timestamp), sets `last_run_at`/`last_run_occurrence_at`, and audits `inquiry_run_scheduled`. **The `InquiryRunCreator` port SHALL join the caller's transaction** — it accepts the same transaction handle (a transaction-scoped store) and performs no independent commit — so the dedupe marker (`last_run_occurrence_at`) and the run insert commit atomically with the schedule `FOR UPDATE` lock held across both. A crash before commit rolls back the marker, the run, and the lock together, so a redelivered event re-checks the marker and dedupes correctly; there is no window in which a committed run exists without its marker (or vice versa). **Rationale**: transaction-isolated state check (no TOCTOU between check and insert), self-contained in tables this change owns, testable. **Alternatives**: (a) run creation on a separate connection then a compensating marker update — REJECTED: creates exactly the commit-ordering window (run committed, marker lost on crash) that this change's exactly-once claim depends on; (b) unique partial index on the sibling's `inquiry_runs` table (`WHERE source='scheduled'`) — stronger, but modifies a table the sibling capability owns; recorded as a reconciliation point if the sibling spec already defines it.

### 4. Follow-up triggers: reply arrival + periodic sweep, with the nudge count as the guard — and dispatch-time re-validation (D14 parity)

A follow-up candidate is a run recipient with status `sent` whose `sent_at + deadline_hours` has passed, with org follow-ups enabled and `nudge_count < max_nudges`. Two triggers: (a) cheap check on reply arrival (the message event that marks a recipient answered — if other recipients are still `sent` and overdue, nudge them then), and (b) a periodic sweep ticker (~15 min) for robustness. Both paths perform the same conditional increment: `UPDATE recipient SET nudge_count = nudge_count + 1 WHERE nudge_count < max_nudges` inside the same transaction that enqueues the `inquiry.followup_send` outbox event and audits `inquiry_followup`; the UPDATE affects 0 rows when the limit is reached, making the send a no-op. **The enqueue-time checks (kill switch off, consent `granted`) are the first gate; the `inquiry.followup_send` dispatch handler SHALL re-validate kill switch and consent transaction-isolated INSIDE the dispatch claim, immediately before invoking the WhatsApp client, mirroring the sibling change's D14 exactly.** Because outbox redelivery and retry/backoff can hold an event for minutes after enqueue, consent may have been withdrawn (Ley 1581) or the kill switch flipped in that window; a blocked dispatch SHALL NOT invoke the client, SHALL complete the event, and SHALL audit `skip` with reason `kill_switch` / `consent_withdrawn`. **Rationale**: the nudge count is a single race-free guard — duplicate sweeps, a sweep racing a reply-trigger, or dispatcher redelivery all converge on "at most `max_nudges` sends"; dispatch-time re-validation closes the enqueue→dispatch race for consent/kill-switch, matching the sibling's standard. **Alternatives**: event-driven only (misses stuck recipients if no reply ever arrives; rejected — kept as a complement), sweep only (up to 15 min latency on the reply path; rejected — kept as complement), audit-count-derived guards (fragile; the atomic counter is the state of record).

### 5. Separate tables, not JSONB columns

Schedules and follow-up state are real tables (`procurement.schedules`, join tables, `procurement.schedule_followups`), not JSONB on the sibling's run rows. **Rationale**: queryable (`WHERE is_active AND next_run_at <= now`), FK-safe, indexable, migration-friendly. JSONB state would be unqueryable for the ticker and sweep (rejected).

### 6. Join tables for products and suppliers, with composite tenant FKs

`procurement.schedule_products` and `procurement.schedule_suppliers` use **composite tenant foreign keys** — `(organization_id, product_id)` referencing the sibling's `procurement.products(organization_id, id)` and `(organization_id, supplier_id)` referencing `procurement.suppliers(organization_id, id)`, both `ON DELETE CASCADE` — never bare single-column FKs. **Rationale**: a bare `product_id` FK would accept a product from another organization, breaking tenant isolation; the composite form enforces both referential integrity and org-scope in one constraint (mirroring `crm-core-data`). **Alternatives**: `int[]` arrays (no FK integrity, rejected), `text[]` (worse, rejected), single-column FKs with app-level org checks (rejected: app checks are bypassable and drift; the constraint is the invariant).

### 7. Follow-up config: separate org-level table; applies to scheduled AND manual runs

`procurement.schedule_followups` is one row per organization (`enabled`, `deadline_hours` default 4, `max_nudges` default 1, Spanish default template "Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos."). It is org-wide policy, not per-schedule: every inquiry run (scheduled **or** manual) benefits from the same convergence rule, and the board behaves uniformly. **Rationale**: avoids per-schedule config drift, one place to manage, and the sibling's manual runs get follow-ups for free. **Alternatives**: fold config into `schedules` columns (per-schedule duplication and drift; rejected), config per run (rejected). The per-recipient `nudge_count` lives on the run-recipient model (see Decision 10).

### 8. Timezone: reuse `agent.agent_settings.timezone`

`run_time` (TIME) is interpreted in the organization's timezone read from `agent.agent_settings.timezone` (default `America/Bogota` — already verified in the repo); `next_run_at` is always computed and stored as `timestamptz` server-side at save/claim time. Schedules do not store their own timezone. **Rationale**: single source of truth, zero duplication, DST handled by Go's IANA `time.LoadLocation` at computation time; per the non-goals, no deeper DST logic. **Alternatives**: per-schedule tz column (duplication and drift; rejected), store UTC-only wall-clock (wrong local firing times; rejected).

### 9. `days_of_week` as `smallint[]` (0=Sunday..6=Saturday)

PostgreSQL's `EXTRACT(DOW)` returns 0=Sunday..6=Saturday, so the array is compared naturally with `days_of_week @> ARRAY[EXTRACT(DOW FROM ts)::smallint]`. **Rationale**: one fixed 7-element enum needs no join; validation enforces 1–7 distinct values in range. **Alternatives**: `text[]` (`'lun'...`) — locale-dependent and awkward in SQL (rejected); a join table — overkill for 7 fixed values (rejected).

### 10. Cross-change contracts with `supplier-inquiries` (by reference, unmodified)

This change consumes three things from the sibling capability through domain interfaces, never raw SQL on its tables: (a) the run-creation service (called with `source = 'scheduled'`, `schedule_id`, occurrence timestamp — the sibling's spec is expected to accept these), (b) run-recipient state including `status`, `sent_at`, and `nudge_count` (the follow-up guard), and (c) an append-only procurement audit ledger receiving `schedule_created`, `schedule_updated`, `schedule_paused`, `schedule_resumed`, `schedule_deleted`, `inquiry_run_scheduled`, `inquiry_followup`, and `skip` (with reason `kill_switch` / `consent_withdrawn` / `duplicate_occurrence`) events. **Rationale**: keeps this change independently archiveable and additive; the interface seam (`InquiryRunCreator`, `RecipientStateReader`, `AuditLogWriter`) keeps the Go domain free of sibling internals. **Alternatives**: direct SQL against sibling tables (spec drift, rejected), duplicating the run model (rejected). If (b)/(c) are absent from the sibling spec at merge time, this change's migration adds them additively — recorded as an Open Question below.

### 11. Domain purity and governance constraints

Go domain models in `inquiryschedule/domain` are pure Go (no Stytch SDK, no transport packages); all persistence, ticking, and send-side effects live in `infra/` adapters implementing domain interfaces (`ScheduleRepository`, `Ticker`, `InquiryRunCreator`, `FollowUpSender`, `AuditLogWriter`, `Clock`). Organization identity is the local org row referencing `stytch_organization_id`; RBAC (`org:manage`/`org:view`) is enforced by the existing auth middleware at the API boundary. Audit writes triggered from the message-event path (reply-arrival follow-up check) are idempotent via the same transactional nudge guard — satisfying the webhook-idempotency rule (message events flow from the WhatsApp webhook ingress through the event bus).

### 12. Escalation and late answers

After `max_nudges` without an answer, the recipient is flagged for escalation (`awaiting_human` semantics — the run surfaces an overdue badge; a human can nudge/call from the board). The run stays open until every recipient is answered, escalated, or timed out; late answers arriving after escalation are still extracted by the sibling's reply pipeline and reflected on the board (the run is not deleted or sealed). **Rationale**: the procurement decision needs the latest price even if the supplier answered late.

## Risks / Trade-offs

- **[Risk] Duplicate run creation after crash/redelivery** → Mitigation: single-transaction claim+advance+enqueue plus the idempotent handler dedupe on `(schedule_id, occurrence_at)` with a `FOR UPDATE` lock; audited `skip`/`duplicate_occurrence` for observability.
- **[Risk] Clock skew / sleeping replicas** → Mitigation: `next_run_at` is computed and compared server-side (`timestamptz`, DB clock); tick drift of seconds is acceptable for a daily cadence; `FOR UPDATE SKIP LOCKED` prevents two replicas claiming the same schedule.
- **[Risk] Follow-up spam / double nudge** → Mitigation: `max_nudges` cap enforced by the atomic conditional increment (the guard), consent gating (`granted` only), and the existing 10 msgs/10s rate limit and circuit breaker on the send client.
- **[Risk] Kill switch / consent changes race the follow-up send** → Mitigation: TWO gates — the follow-up handler re-checks kill switch and consent transaction-isolated inside its enqueue transaction, AND the `inquiry.followup_send` dispatch handler re-validates both again inside the dispatch claim immediately before invoking the client (D14 parity); violations produce `skip` audits (fail-safe direction). Enqueue-time checks alone cannot cover the outbox retry/backoff window.
- **[Risk] Sibling `supplier-inquiries` spec differs from the assumed contract** → Mitigation: consumption is interface-only; the two reconciliation points (recipient `nudge_count`, audit ledger) are additive if missing and are tracked as an Open Question; the change remains independently archiveable.
- **[Risk] Dead-lettered `inquiry_run.scheduled` skips an occurrence** → Mitigation: the outbox already advances `next_run_at` at claim time, so a dead-lettered occurrence requires operator replay (existing `RequeueOutboxEvent`/replay tooling per `durable-message-pipeline`); accepted, documented.

## Migration Plan

1. Land `add-whatsapp-template-messages`, then `add-supplier-inquiry-agent` (deployment order per the consistency contract; the sibling defines the `procurement` schema and its tables).
2. This change ships migration `000037+` (next available after the sibling's; number resolved at merge — head is `000038_create_procurement_schedules` as of this revision): `CREATE SCHEMA IF NOT EXISTS procurement;` + `procurement.schedules`, `procurement.schedule_products`, `procurement.schedule_suppliers`, `procurement.schedule_followups` (with `updated_at` triggers per repo convention) — `.up.sql` and `.down.sql`. **Indexes and constraints (per council revision):** (a) ticker predicate `(is_active, next_run_at)` index on `schedules`; (b) composite tenant FKs `(organization_id, product_id)` / `(organization_id, supplier_id)` on the join tables (never bare FKs); (c) follow-up sweep index on the sibling's `inquiry_recipients(organization_id, status, sent_at)` — added by THIS change's migration as a reconciliation point with the sibling (D15 sibling parity; if the sibling spec already defines it, confirm shape at merge); (d) `updated_at` triggers on all new tables.
3. If the sibling's spec lacks the recipient `nudge_count` column or the procurement audit table, THIS change's migration adds them additively (open questions below); the `.down.sql` SHALL drop every object this migration creates, including any additively-added sibling objects.
3. SQLC queries + module `internal/modules/inquiryschedule/` (domain → app → infra → cmd wiring), API routes behind `auth` + `org_context` + `subscription` (`GET/POST /api/procurement/schedules`, `GET/PUT /api/procurement/schedules/:id`, `POST /api/procurement/schedules/:id/pause`, `POST /api/procurement/schedules/:id/resume`, `DELETE /api/procurement/schedules/:id`, `GET/PUT /api/procurement/followup-settings`), ticker and sweep started with the module.
4. Frontend: schedules page + follow-up settings under the procurement section (TanStack Query, shadcn/ui, Spanish copy).
5. Verification: `make sqlc`, `go build ./...`, `go vet ./internal/modules/inquiryschedule/...`, `go test ./internal/modules/inquiryschedule/...`, `pnpm lint`, `pnpm build`, plus integration tests with mock outbox/send (see `tasks.md`).
6. Rollback: revert commit + run the `.down.sql` migration; no Stytch policy state exists for this change, so no Stytch rollback is required.

## Open Questions

- **Recipient `nudge_count` home**: the sibling spec (in progress) may or may not define `nudge_count` on its run-recipient model. If absent, this change's migration adds it additively; confirm at merge before archiving either change.
- **Audit ledger home**: whether the sibling defines the append-only procurement audit table (expected) or this change must add it additively; confirm shapes (event-type CHECK vs free-form) at merge.
- **Run-creation entrypoint signature**: exact parameters/return of the sibling's run-creation service (expected to accept `source`, `schedule_id`, occurrence timestamp and return the created run id); adapt the `InquiryRunCreator` adapter at implementation time.
- **Org timezone source of truth**: `agent.agent_settings.timezone` is verified; if a separate org-level timezone exists in the sibling or platform code, prefer it (single source).
- **Late-reply semantics**: whether the sibling's reply extraction can attach an answer to an already-`escalated` recipient without a new run; confirm the flag semantics at implementation (spec here requires the board to reflect late answers).
- **Sweep candidate claim semantics**: the follow-up sweep SHALL claim candidate recipients with `FOR UPDATE SKIP LOCKED` (alongside the schedule ticker) so concurrent sweeps on multi-replica deploys do not double-enqueue; the atomic `nudge_count` increment remains the final idempotency guard. Confirm this is implemented in the sweep query.

## Council Revision (2026-08-12)

Addressed the council verdict (`VERDICT.md`, STATUS: REJECTED — 5 required changes):

1. **Dispatch-time re-validation for `inquiry.followup_send`** (SEV-1) → Decision 4 now mandates kill-switch + consent re-validation transaction-isolated INSIDE the dispatch claim, before invoking the WhatsApp client, with `skip` audits (`kill_switch` / `consent_withdrawn`) — D14 parity with the sibling change.
2. **Transactional boundary of the run-creation dedupe** (SEV-4) → Decision 3 now states the `InquiryRunCreator` port SHALL join the caller's transaction (transaction-scoped store, no independent commit), so `last_run_occurrence_at` and the run insert commit atomically under the schedule `FOR UPDATE` lock; the two-connection alternative is explicitly rejected.
3. **Composite tenant FKs on join tables** (SEV-2) → Decision 6 now requires `(organization_id, product_id)` / `(organization_id, supplier_id)` composite FKs, never bare FKs.
4. **Follow-up sweep index** (SEV-5) → Migration plan now ships `inquiry_recipients(organization_id, status, sent_at)` as a reconciliation point with the sibling.
5. **Ticker index + sweep claim semantics** (SEV-6/SEV-7) → Migration plan ships `(is_active, next_run_at)`; sweep claim uses `FOR UPDATE SKIP LOCKED` (recorded in Open Questions).
