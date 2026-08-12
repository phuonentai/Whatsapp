STATUS: APPROVED

# Council Verdict — add-scheduled-inquiry-runs

Reviewed `design.md` (12 decisions, risks, migration plan, open questions) and `proposal.md`
for intent. The design is additive, tenant-isolated by construction, reuses the durable
outbox / breaker / rate-limit primitives, and ships a complete rollback. The exactly-once
claim is sound: single-transaction claim→advance→enqueue, and the updated Decision 3
(explicitly joining the `InquiryRunCreator` to the caller's transaction — no independent
commit) closes the commit-ordering window that would otherwise allow a committed run without
its dedupe marker. No blocking defects found; findings below are non-blocking residuals or
minor recommendations.

## Staff Security Engineer

- **[MED] Kill-switch pre-check for run creation is check-then-create, not atomic.**
  The handler checks `agent.agent_settings.kill_switch` before opening the creation
  transaction; a kill switch flipped between the pre-check and commit still creates the run.
  The follow-up path is protected (Decision 4 re-validates inside the dispatch claim), but the
  run-creation path is not explicitly re-checked in-transaction. Impact is bounded: the
  sibling's own kill-switch handling cancels in-progress runs, so the window produces an
  audited run that is quickly cancelled. Required change (recommended, not blocking): re-read
  the kill switch inside the creation transaction (it already locks the schedule row) and skip
  with `skip`/`kill_switch` audit on the same transaction boundary as the dedupe check.
- **[LOW] Stale payload consistency on dispatch.** The handler creates the run with
  `product_ids`/`supplier_ids` from the outbox event payload; a schedule edit between claim
  and dispatch yields a run referencing the pre-edit catalog. Dedupe correctness is unaffected
  (keyed on `(schedule_id, occurrence_at)`). Recommended: re-read the locked schedule's join
  rows inside the creation transaction so the run always reflects the schedule at fire time.
- **[LOW] RBAC and tenant isolation are sound.** All API routes sit behind `auth` +
  `org_context` + `subscription` with `org:manage`/`org:view`; every repository query is
  org-scoped; join tables use composite tenant FKs `(organization_id, product_id|supplier_id)`
  (Decision 6), so cross-org references are rejected at the constraint level, not just in app
  code. No credentials/secrets stored; no Stytch policy surface. No OWASP-10 findings.
- **[LOW] Webhook path is idempotent.** The reply-arrival trigger flows from
  `whatsapp.message.received` through the event bus; the atomic nudge increment is the
  transaction-isolated guard, so redelivered messages cannot double-nudge (spec: at most
  `max_nudges` sends).

## Staff DBA

- **[MED] LLM drafting inside a locked transaction (bounded).** Per Decision 3, the run-creation
  transaction holds the schedule `FOR UPDATE` lock across the metered per-supplier drafting
  calls. This serializes occurrences per schedule and is acceptable for a daily cadence, but a
  slow or hung LLM call extends lock duration. The design explicitly rejects the compensating
  alternative (Decision 3a), so this is an accepted residual. Recommended: cap the drafting
  phase with a per-supplier timeout/context deadline and bound the supplier count per schedule.
- **[LOW] Missing FK-supporting indexes on the join tables.** The cascade FKs are
  `(organization_id, schedule_id)` and `(organization_id, product_id|supplier_id)`; the
  `UNIQUE(schedule_id, product_id|supplier_id)` constraints do not cover the
  `(organization_id, ...)` prefix, so `ON DELETE CASCADE` on `procurement.schedules` /
  `products` / `suppliers` will seq-scan the join tables. Fine at launch volumes (matches
  sibling convention); add `(organization_id, schedule_id)` / `(organization_id, product_id)`
  / `(organization_id, supplier_id)` indexes if these tables grow.
- **[LOW] Ticker transaction does 3N queries per batch** (per schedule: products + suppliers
  join rows + org timezone) while holding `FOR UPDATE SKIP LOCKED` locks. Bounded by batch
  size and a 45s cadence; keep the batch modest and prefer a single aggregate claim query if
  scale demands it.
- **[OK] Migration is pure-additive.** No ALTER on existing tables; new schema tables only;
  `idx_schedules_due (is_active, next_run_at)` covers the ticker hot path; `days_of_week`
  array-length CHECK plus domain-level distinct-value validation. No expand-contract concern.

## SRE

- **[MED] Mixed clocks for occurrence semantics.** `next_run_at` comparison uses the DB clock
  (`NOW()`), while `occurrence_at` and the next-occurrence computation use the app clock.
  Seconds-level skew is accepted for a daily cadence (documented), but a large app/DB skew
  could fire an occurrence "early" relative to the DB and affect the dedupe marker equality.
  Recommended: derive `occurrence_at` from the DB clock inside the claim transaction, or
  document an operational bound on clock skew.
- **[LOW] Escalation is derived, not stored.** The overdue/`awaiting_human` flag is computed at
  read time (`status='sent' AND followup_count >= max_nudges`); lowering `max_nudges` later
  retroactively flags recipients, and there is no durable "human handled" marker for the
  escalated state. Acceptable for the badge-only surface; residual risk if an audit trail of
  human resolution is later required (add an escalation-resolution audit event).
- **[LOW] Observability of the new loops.** Ticker/sweep log at info, but there is no
  heartbeat metric or dead-letter alert for the two new event types
  (`inquiry_run.scheduled`, `inquiry.followup_send`). A dead-lettered occurrence skips the run
  until operator replay (accepted, documented). Recommended: alert on `dead_letter` rows for
  these event types or surface ticker lag in metrics.
- **[OK] Distributed safety.** `FOR UPDATE SKIP LOCKED` makes concurrent tickers/replicas safe
  (no leader election needed); crash-before-commit rolls back cleanly and re-fires
  (at-least-once at the tick level, exactly-once effect at the handler). Circuit breaker and
  10 msgs/10s rate limit are reused for follow-up sends; consent and kill switch are
  re-validated at dispatch (Decision 4). Rollback is complete at the Git/DB layer — `.down.sql`
  ships, no Stytch policy state to revert.

## Summary

**STATUS: APPROVED.** The design satisfies the repo governance gates (no local credential
storage, Stytch RBAC reuse, tenant-isolated additive tables, Spanish-first, append-only
audits) and the three lifecycle gates' intent. Findings are non-blocking residuals (kill-switch
race on run creation, LLM-inside-lock bounded, FK index coverage, clock-source mixing, derived
escalation, loop observability). Recommended follow-ups before or shortly after merge, in
priority order: (1) in-transaction kill-switch re-check for run creation; (2) DB-clock
`occurrence_at`; (3) per-supplier drafting timeout; (4) dead-letter alerting for the new event
types; (5) FK-supporting join-table indexes at scale.
