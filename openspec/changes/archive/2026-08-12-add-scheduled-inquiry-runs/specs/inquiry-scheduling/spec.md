# Inquiry Scheduling

## Purpose

Defines recurring supplier-inquiry schedules that durably create inquiry runs through the `supplier-inquiries` capability, and automated per-recipient follow-ups with a configurable deadline, one nudge, then human escalation. Fully additive: it consumes the `supplier-inquiries` capability by reference and modifies no existing capability.

## ADDED Requirements

### Requirement: Schedule CRUD with validation and next-run computation

The system SHALL allow organizations to create, read, update, pause, resume, and delete org-scoped inquiry schedules under `/api/procurement/schedules`. A schedule SHALL store `name`, `run_time` (TIME), `days_of_week` (smallint array, 0=Sunday..6=Saturday), a product list and a supplier list (join tables `procurement.schedule_products` / `procurement.schedule_suppliers`), an optional `note`, and `is_active`. Creation SHALL require `run_time`, a non-empty `days_of_week` with 1–7 distinct values in range, at least one product and one supplier, and every referenced product/supplier SHALL belong to the schedule's organization. The system SHALL compute `next_run_at` (timestamptz) as the next occurrence of `run_time` on a day in `days_of_week` strictly after the current instant, interpreted in the organization's timezone (`agent.agent_settings.timezone`, default `America/Bogota`). Validation failures SHALL return HTTP 400 with a Spanish error message. Writes SHALL require Stytch B2B RBAC permission `org:manage`; reads SHALL require `org:view`.

#### Scenario: Schedule created with computed next run

- **WHEN** a user with `org:manage` permission creates a schedule with `run_time` `"08:00"`, `days_of_week` `[1,2,3,4,5]`, two org-scoped products, two org-scoped suppliers, and a note
- **THEN** the system SHALL persist the schedule with `is_active = true` under the user's organization
- **AND** SHALL set `next_run_at` to the next Monday–Friday 08:00 in the org timezone strictly after now
- **AND** SHALL return HTTP 201 with the schedule including `next_run_at`

#### Scenario: Invalid schedule rejected in Spanish

- **WHEN** a user creates a schedule missing `run_time`, or with an empty `days_of_week`, or with no products or no suppliers
- **THEN** the system SHALL return HTTP 400 with a Spanish error message identifying the missing field
- **AND** SHALL NOT persist any schedule or join row

#### Scenario: Out-of-organization references rejected

- **WHEN** a user creates a schedule referencing a product or supplier that belongs to another organization
- **THEN** the system SHALL return HTTP 400 (or 404) with a Spanish error message
- **AND** SHALL NOT persist the schedule

#### Scenario: Update recomputes next run

- **WHEN** a user updates a schedule's `run_time`, `days_of_week`, product list, supplier list, or note
- **THEN** the system SHALL recompute and persist `next_run_at` as the next occurrence strictly after now matching the new configuration
- **AND** SHALL return the updated schedule

#### Scenario: Pause stops firing and resume recomputes

- **WHEN** a user pauses a schedule via `POST /api/procurement/schedules/:id/pause`
- **THEN** the system SHALL set `is_active = false`
- **AND** the scheduler SHALL NOT claim or fire that schedule
- **WHEN** the user resumes it via `POST /api/procurement/schedules/:id/resume`
- **THEN** the system SHALL set `is_active = true`
- **AND** SHALL recompute `next_run_at` as the next occurrence strictly after now (occurrences missed while paused SHALL NOT be backfilled)

#### Scenario: Delete removes future runs

- **WHEN** a user deletes a schedule via `DELETE /api/procurement/schedules/:id`
- **THEN** the system SHALL delete the schedule and its `schedule_products` / `schedule_suppliers` join rows
- **AND** SHALL NOT fire any further runs for it
- **AND** existing runs already created from it SHALL remain untouched

#### Scenario: Write denied without manage permission

- **WHEN** a user without `org:manage` permission attempts to create, update, pause, resume, or delete a schedule
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

#### Scenario: Schedule scoped to its organization

- **WHEN** a user with `org:view` permission lists schedules or requests a schedule by id
- **THEN** the system SHALL return only schedules of the user's organization
- **AND** a schedule id of another organization SHALL yield HTTP 404

### Requirement: Scheduler ticker claims due schedules and enqueues durably

The system SHALL run an in-process ticker (interval between 30 and 60 seconds) that SHALL select schedules where `is_active = true AND next_run_at <= now()`, claim each due schedule atomically with `FOR UPDATE SKIP LOCKED`, and in the SAME transaction: compute and persist the next `next_run_at`, and insert a durable outbox event of type `inquiry_run.scheduled` with payload `{schedule_id, organization_id, product_ids, supplier_ids, note, occurrence_at}`. The claim SHALL advance `next_run_at` strictly after the fired occurrence before the transaction commits. Concurrent tickers or replicas SHALL NOT double-claim a schedule. If the process crashes after claiming a schedule but before the transaction commits, the claim SHALL roll back and a later tick SHALL re-fire the occurrence (at-least-once delivery at the tick level).

#### Scenario: Due schedule fires exactly once per occurrence

- **WHEN** a tick runs and a schedule has `is_active = true` and `next_run_at <= now()`
- **THEN** the system SHALL commit exactly one outbox event `inquiry_run.scheduled` for that schedule with the fired `occurrence_at`
- **AND** SHALL persist the schedule's next `next_run_at` strictly after the fired occurrence

#### Scenario: Concurrent tickers do not double-fire

- **WHEN** two ticker instances claim the same due schedule concurrently
- **THEN** exactly one SHALL commit the claim and enqueue the outbox event (transaction-isolated claim via `FOR UPDATE SKIP LOCKED`)
- **AND** the other SHALL skip it

#### Scenario: Crash between claim and commit re-fires

- **WHEN** a ticker claims a due schedule and the process crashes before the transaction commits (no outbox event was enqueued)
- **THEN** the claim SHALL roll back
- **AND** a subsequent tick SHALL select the schedule again and fire the occurrence (at-least-once)

#### Scenario: Inactive or not-yet-due schedules are skipped

- **WHEN** a tick runs
- **THEN** the system SHALL NOT claim schedules with `is_active = false` or with `next_run_at > now()`

### Requirement: Scheduled run creation is idempotent and governed

The system SHALL dispatch `inquiry_run.scheduled` outbox events to a handler that SHALL call the `supplier-inquiries` run-creation service (the same entrypoint used for manual runs), passing `source = 'scheduled'`, the `schedule_id`, and the occurrence timestamp. The handler SHALL be idempotent per scheduled occurrence: in one transaction it SHALL lock the schedule row (`FOR UPDATE`) and, when a run for `(schedule_id, occurrence_at)` was already recorded (schedule `last_run_occurrence_at` equals the occurrence), SHALL skip creation and treat the event as handled. When creation proceeds, the system SHALL record the run's creation on the schedule (`last_run_at`, `last_run_occurrence_at`) and audit `inquiry_run_scheduled`. The handler SHALL run with `agent.agent_settings.kill_switch` checked first: with the kill switch enabled the handler SHALL NOT create the run and SHALL audit `skip` with reason `kill_switch`. Scheduling operations SHALL NOT invoke any LLM and SHALL NOT be metered against AI credits; if the organization's AI credits are exhausted at draft time, the run SHALL be created as `escalated` per the `supplier-inquiries` contract.

#### Scenario: Occurrence creates a scheduled run

- **WHEN** the handler processes `inquiry_run.scheduled` for a schedule with no run recorded for that occurrence
- **THEN** the system SHALL call the `supplier-inquiries` run-creation service with `source = 'scheduled'`, the `schedule_id`, and the occurrence timestamp
- **AND** SHALL set the schedule's `last_run_at` and `last_run_occurrence_at` to the occurrence
- **AND** SHALL audit `inquiry_run_scheduled`

#### Scenario: Duplicate firing does not create a second run

- **WHEN** the dispatcher redelivers `inquiry_run.scheduled` for an occurrence whose run was already created (e.g., the handler succeeded but the dispatch-completion mark failed)
- **THEN** the handler SHALL detect the existing `(schedule_id, occurrence_at)` record under a transaction-isolated lock
- **AND** SHALL skip run creation
- **AND** SHALL audit `skip` with reason `duplicate_occurrence`
- **AND** SHALL NOT call the run-creation service

#### Scenario: Kill switch skips creation

- **WHEN** the handler processes `inquiry_run.scheduled` for an organization with `agent_settings.kill_switch = true`
- **THEN** the system SHALL NOT create the run
- **AND** SHALL audit `skip` with reason `kill_switch`

#### Scenario: Exhausted credits escalate instead of burning unmetered tokens

- **WHEN** the organization's AI credits are exhausted at draft time
- **THEN** the run SHALL be created as `escalated` per the `supplier-inquiries` contract
- **AND** scheduling itself SHALL NOT consume AI credits or invoke an LLM

### Requirement: Follow-up settings are org-level configuration

The system SHALL expose `GET /api/procurement/followup-settings` (`org:view`) and `PUT /api/procurement/followup-settings` (`org:manage`) for one row per organization in `procurement.schedule_followups`: `enabled` (boolean), `deadline_hours` (integer, default 4), `max_nudges` (integer, default 1), and `message_template` (text, Spanish default "Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos."). Validation SHALL reject `deadline_hours` outside 1–168 and `max_nudges` outside 0–5 with a Spanish error message. When no row exists, the system SHALL behave as if the defaults applied.

#### Scenario: Settings are read and updated

- **WHEN** a user with `org:view` permission calls `GET /api/procurement/followup-settings`
- **THEN** the system SHALL return the organization's settings, or the defaults when no row exists
- **WHEN** a user with `org:manage` permission updates them
- **THEN** the system SHALL persist and return the updated settings

#### Scenario: Invalid settings rejected

- **WHEN** a user submits `deadline_hours = 0` or `max_nudges = 9`
- **THEN** the system SHALL return HTTP 400 with a Spanish error message
- **AND** SHALL NOT modify the stored settings

#### Scenario: Settings write denied without manage permission

- **WHEN** a user without `org:manage` permission updates follow-up settings
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Follow-up automation with deadline, one nudge, then escalation

For every inquiry run of the organization (scheduled and manual), the system SHALL track each recipient's `sent_at` and `nudge_count`. When a recipient is in status `sent`, `sent_at + deadline_hours` has passed, follow-ups are enabled for the organization, `nudge_count < max_nudges`, the recipient's contact consent (Ley 1581) is `granted`, and the kill switch is off, the system SHALL enqueue a durable outbox event `inquiry.followup_send` carrying `{run_id, organization_id, supplier_id, contact_phone, message_template, nudge_index}` and SHALL increment `nudge_count` atomically (`UPDATE ... SET nudge_count = nudge_count + 1 WHERE nudge_count < max_nudges`) in the same transaction, then audit `inquiry_followup`. The follow-up send SHALL reuse the outbox and the circuit-breakered, rate-limited WhatsApp outbound client. Candidates SHALL be detected by BOTH a check on reply arrival and a periodic sweep ticker (at most every 15 minutes); both paths SHALL be idempotent (the atomic nudge-count increment is the guard). After `max_nudges` nudges without an answer, the recipient and its run SHALL be flagged for human escalation and the aggregation board SHALL show an overdue badge.

#### Scenario: Overdue unanswered recipient receives one nudge

- **WHEN** an org has follow-ups enabled, `deadline_hours = 4`, `max_nudges = 1`, and a run recipient is `sent` with `sent_at` more than 4 hours ago, consent `granted`, and `nudge_count = 0`
- **THEN** the system SHALL enqueue exactly one `inquiry.followup_send` event for that recipient with the configured Spanish template
- **AND** SHALL set `nudge_count = 1` in the same transaction
- **AND** SHALL audit `inquiry_followup`

#### Scenario: Double nudge is prevented

- **WHEN** the reply-arrival check and the sweep ticker both detect the same overdue recipient concurrently, or the dispatcher redelivers a `inquiry.followup_send` event
- **THEN** the atomic increment SHALL allow only one nudge per index (the second attempt sees `nudge_count` at the limit and SHALL NOT enqueue another send)

#### Scenario: Max nudges reached escalates to a human

- **WHEN** a recipient has received `max_nudges` nudges and still has no answer
- **THEN** the system SHALL flag the recipient and its run for human escalation
- **AND** the aggregation board SHALL show an overdue badge for that recipient
- **AND** SHALL NOT enqueue further automatic nudges

#### Scenario: Withdrawn consent escalates instead of nudging

- **WHEN** a recipient's contact consent is `withdrawn` (Ley 1581)
- **THEN** the system SHALL NOT enqueue a follow-up send
- **AND** SHALL flag the recipient for human escalation
- **AND** SHALL audit `skip` with reason `consent_withdrawn`

#### Scenario: Kill switch cancels follow-up sends

- **WHEN** the organization's kill switch is enabled
- **THEN** the system SHALL NOT enqueue or execute follow-up sends
- **AND** SHALL audit `skip` with reason `kill_switch`

#### Scenario: Reply before deadline suppresses the nudge

- **WHEN** a recipient answers before `sent_at + deadline_hours` elapses
- **THEN** the system SHALL NOT enqueue a follow-up for that recipient

#### Scenario: Follow-up send respects the rate-limited send path

- **WHEN** a follow-up send is dispatched
- **THEN** the send SHALL go through the circuit-breakered WhatsApp outbound client respecting the existing rate limit (10 messages / 10 seconds)
- **AND** a send failure SHALL be retried per the durable pipeline with backoff and dead-letter after max attempts

### Requirement: Convergence and status surfaces

The system SHALL expose run/schedule status: `GET /api/procurement/schedules` SHALL return the organization's schedules with their `next_run_at` and the status of the last run created from each; `GET /api/procurement/schedules/:id` SHALL return the schedule detail including its recent runs (limited list) and follow-up state. A run SHALL be considered complete when all of its recipients are answered, escalated, or timed out. Late answers arriving after a recipient was flagged for escalation SHALL still be extracted by the `supplier-inquiries` reply pipeline and reflected on the board.

#### Scenario: Schedule list shows next run and last status

- **WHEN** a user with `org:view` permission calls `GET /api/procurement/schedules`
- **THEN** the response SHALL include each schedule with `next_run_at`, `is_active`, and the status of its last run (or `never_run`)

#### Scenario: Schedule detail shows recent runs

- **WHEN** a user with `org:view` permission calls `GET /api/procurement/schedules/:id`
- **THEN** the response SHALL include the schedule, its joined products and suppliers, its follow-up settings, and its recent runs (limited, newest first)

#### Scenario: Run completes when all recipients are terminal

- **WHEN** every recipient of a run is answered, escalated, or timed out
- **THEN** the run SHALL be marked complete
- **AND** the board SHALL stop showing it as in progress

#### Scenario: Late answer still surfaces

- **WHEN** a supplier answers after their recipient was flagged for escalation
- **THEN** the answer SHALL be extracted and shown on the board against that run
- **AND** the run SHALL reflect the late answer without a new run being created

### Requirement: Append-only audit of scheduling events

The system SHALL append one immutable row to the procurement audit ledger for every lifecycle event: `schedule_created`, `schedule_updated`, `schedule_paused`, `schedule_resumed`, `schedule_deleted`, `inquiry_run_scheduled`, `inquiry_followup`, and `skip` (with reason `kill_switch`, `consent_withdrawn`, or `duplicate_occurrence`). Audit rows SHALL include the organization id, the event type, the affected schedule/run/supplier ids, the payload snapshot, and a timestamp, and SHALL NOT be modified after insertion.

#### Scenario: Lifecycle events are audited

- **WHEN** a schedule is created, updated, paused, resumed, deleted, or an occurrence fires or a follow-up is sent
- **THEN** the corresponding audit event SHALL be appended with the affected ids and a payload snapshot

#### Scenario: Governed skips are audited

- **WHEN** the handler skips run creation (`kill_switch`, `duplicate_occurrence`) or follow-up send (`kill_switch`, `consent_withdrawn`)
- **THEN** an audit row SHALL be appended with event `skip` and the exact reason

#### Scenario: Audit rows are immutable

- **WHEN** an audit row has been written
- **THEN** it SHALL NOT be updated or deleted by any application path
