# Add Scheduled Inquiry Runs

## Why

The stockless-retail procurement loop (capability `supplier-inquiries`, sibling change `add-supplier-inquiry-agent`) is a daily ritual: check supplier availability and price every morning before quoting customers. Manual re-triggering of inquiry runs defeats the purpose — the business should wake up to today's price/availability board already assembled. Today every inquiry run is started by hand and every unanswered supplier requires a human to chase. This change adds org-scoped recurring schedules (time-of-day + weekdays) that durably create inquiry runs, plus automated per-supplier follow-ups with a configurable deadline, so daily runs converge without babysitting.

## What Changes

- **New capability `inquiry-scheduling`** (pure addition; no BREAKING changes):
  - **Schedule CRUD** — org-scoped schedules (name, `run_time`, `days_of_week`, product list, supplier list, optional note, `is_active`) under `/api/procurement/schedules...`; create/update validate and compute `next_run_at` in the org's timezone; pause/resume flips `is_active`; delete removes future runs.
  - **Scheduler ticker** — an in-process ticker (30–60s, same pattern as the durable outbox dispatcher) that durably claims due schedules (`FOR UPDATE SKIP LOCKED`), advances `next_run_at`, and enqueues a durable outbox event `inquiry_run.scheduled` in the same transaction as the claim.
  - **Scheduled run creation** — an outbox handler that calls the `supplier-inquiries` run-creation service (same entrypoint as manual runs; run records `source = 'scheduled'` and `schedule_id`), idempotently deduped per scheduled occurrence, honoring the agent kill switch and credit exhaustion (run created as `escalated`).
  - **Follow-up automation** — org-level follow-up settings (enabled, `deadline_hours`, `max_nudges`, Spanish message template) with per-recipient nudges for unanswered `sent` recipients past the deadline, triggered on reply arrival plus a periodic sweep; one automatic nudge, then human escalation with an overdue badge.
  - **Status surfaces** — schedule list with `next_run_at` and last-run status; schedule detail with recent runs.
- **Data**: new `procurement` schema tables only for scheduling concerns (`procurement.schedules`, `procurement.schedule_products`, `procurement.schedule_suppliers`, `procurement.schedule_followups`); no tables belonging to the sibling `supplier-inquiries` capability; no changes to existing specs.
- **Backend**: new migration (next number after `000035`), SQLC queries, module `internal/modules/inquiryschedule/` (domain / app / infra / cmd per repo convention).
- **Frontend**: schedules page under the procurement section (create/edit/pause, time + weekday pickers, supplier/product multi-select, next-run display) and follow-up settings; Spanish-first copy; TanStack Query; shadcn/ui.
- **RBAC**: `org:manage` for schedule writes and follow-up settings, `org:view` for reads; routes behind the existing `auth` + `org_context` + `subscription` middleware. No Stytch B2B policy changes — the existing Stytch B2B RBAC roles (`org:manage`/`org:view`) and Stytch organization identity (`stytch_organization_id`, per the repo constitution's storage invariant) are reused as-is; local tables store only application business data keyed to the local organization row that references `stytch_organization_id`.
- **Compliance**: follow-up sends respect the Ley 1581 consent state machine (`granted` only; `withdrawn` → escalate instead of nudge) and the kill switch.

## Capabilities

### New Capabilities

- `inquiry-scheduling`: org-scoped recurring supplier-inquiry schedules (CRUD + pause/resume, `next_run_at` computation in the org timezone), a durable scheduler ticker that claims due schedules and enqueues `inquiry_run.scheduled` outbox events exactly-once per occurrence (at-least-once delivery, idempotent handler), scheduled run creation through the `supplier-inquiries` run-creation service (kill-switch and credit-exhaustion aware), per-recipient follow-up automation with configurable deadline and one nudge before human escalation, and schedule/run status surfaces.

### Modified Capabilities

- **None.** This change is fully additive. The sibling capability `supplier-inquiries` is consumed **by reference** (its run-creation entrypoint and run/recipient state) and is **NOT modified** — no delta specs are created for it, and it may not exist yet in `openspec/specs/` (it is being generated in parallel). Cross-change contracts (the run-creation call signature, recipient follow-up state, and the append-only procurement audit ledger) are defined in this proposal's design and must be reconciled at merge time before archiving.

## Impact

- **Code**: new Go module `internal/modules/inquiryschedule/` (domain models, app services, SQLC-backed repository, ticker/infrastructure adapters, API handlers); new SQLC queries; new migration `000037+` (next available after the sibling change's migration; final number resolved at merge). Existing modules are untouched.
- **API**: new endpoints `GET/POST /api/procurement/schedules`, `GET/PUT /api/procurement/schedules/:id`, `POST /api/procurement/schedules/:id/pause`, `POST /api/procurement/schedules/:id/resume`, `DELETE /api/procurement/schedules/:id`, `GET/PUT /api/procurement/followup-settings`. No existing endpoint changes.
- **Dependencies**: consumes the `supplier-inquiries` run-creation service (domain interface, no direct table coupling) and the existing durable outbox, WhatsApp outbound send client (circuit-breakered, rate limited), agent kill-switch/settings state, and consent state. No new external services; no Stytch B2B API changes.
- **Systems**: two new in-process loops (schedule claim ticker ~30–60s, follow-up sweep ~15 min) that reuse the existing outbox dispatcher pattern; no new infrastructure.
- **Deployment order**: `add-whatsapp-template-messages` → `add-supplier-inquiry-agent` → `add-scheduled-inquiry-runs`. End-to-end value requires the sibling change to land first; this change is independently reviewable and archiveable.

## Non-Goals

- **Local credential storage**: explicitly rejected — no credentials, tokens, or secrets are stored in the new tables or anywhere in this change; authentication and RBAC remain exclusively on the Stytch B2B side (per the repo constitution's storage invariant).
- No external cron / scheduled-job service dependency — scheduling uses an in-process ticker only.
- No timezone-aware DST complexity beyond storing/reading the org's IANA timezone (reused from `agent.agent_settings.timezone`); `next_run_at` is computed server-side at save/claim time.
- No reconciliation or backfill of missed runs beyond next-occurrence computation — a paused-then-resumed schedule skips occurrences that fell while paused.
- No WhatsApp template-message scheduling — that is the sibling change `add-whatsapp-template-messages`.
- No generalization to campaign scheduling — `whatsapp-campaigns` explicitly keeps "scheduler consumes recipients later" as future work.
- No changes to the `supplier-inquiries` capability or any other existing spec; no changes to Stytch B2B tenant policy.

## Rollback

- **Git state**: revert the change commit(s) and apply the migration `down` file (drops `procurement.schedules`, `procurement.schedule_products`, `procurement.schedule_suppliers`, `procurement.schedule_followups`); scheduled runs already created remain as ordinary inquiry runs (their `source = 'scheduled'` flag is inert without the scheduler). Both the `.up.sql` and `.down.sql` migrations ship with the change.
- **Stytch tenant policy state**: no Stytch B2B API calls or policy changes are introduced (no new roles, permissions, or org configuration), so **no Stytch policy rollback is required**; the rollback is complete at the Git/DB layer.
- Runtime loops stop on deploy revert; any pending `inquiry_run.scheduled` / `inquiry.followup_send` outbox events are handled by the existing dispatcher with no scheduler present (they fail-safe or are pruned per the durable pipeline's replay tooling).

## Assumptions

- The org timezone is read from `agent.agent_settings.timezone` (default `America/Bogota`), verified present in the repo (`000019_create_agent_schema.up.sql`); if a more authoritative org timezone exists at implementation time, it is used instead (single source of truth).
- The sibling change `add-supplier-inquiry-agent` lands first and exposes: (a) a run-creation service callable with `source = 'scheduled'`, `schedule_id`, and an occurrence timestamp, (b) per-recipient run state (`status`/`sent_at` and a `nudge_count` counter used as the follow-up idempotency guard), and (c) an append-only procurement audit ledger that scheduling audit events are written to. If (b)/(c) are not present in the sibling's spec at merge time, this change's migration adds them additively (a column and/or the audit table) — flagged in `design.md` Open Questions.
- Meta/WhatsApp delivery latency is outside our control; follow-up deadlines are measured from our recorded `sent_at`, and late supplier answers after escalation are still extracted by the `supplier-inquiries` reply pipeline and reflected on the board.
- Scheduling operations themselves are NOT LLM calls and are not metered against AI credits; only the sibling's drafting step is metered (per the sibling change).
