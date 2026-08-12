-- Inquiry scheduling queries (add-scheduled-inquiry-runs).
-- Consumes the sibling procurement tables by reference (inquiry_runs,
-- inquiry_recipients, suppliers, products, audit_log) plus agent_settings
-- (timezone/kill_switch) and crm.contacts (consent, display name, phone).

-- ============================================================
-- Schedules (procurement.schedules)
-- ============================================================

-- name: InsertSchedule :one
INSERT INTO procurement.schedules (
    organization_id, name, run_time, days_of_week, note, is_active, next_run_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSchedule :one
SELECT * FROM procurement.schedules
WHERE id = $1 AND organization_id = $2;

-- Handler-side dedupe lock: locks the schedule row FOR UPDATE inside the
-- run-creation transaction (exactly-once effect per occurrence).
-- name: GetScheduleForUpdate :one
SELECT * FROM procurement.schedules
WHERE id = $1 AND organization_id = $2
FOR UPDATE;

-- name: ListSchedulesByOrg :many
SELECT * FROM procurement.schedules
WHERE organization_id = $1
ORDER BY id ASC;

-- name: UpdateSchedule :one
UPDATE procurement.schedules
SET
    name = $3,
    run_time = $4,
    days_of_week = $5,
    note = $6,
    next_run_at = $7,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteSchedule :exec
DELETE FROM procurement.schedules
WHERE id = $1 AND organization_id = $2;

-- name: PauseSchedule :one
UPDATE procurement.schedules
SET is_active = FALSE, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- Resume recomputes next_run_at server-side (occurrences missed while paused
-- are not backfilled; next_run_at is strictly after now).
-- name: ResumeSchedule :one
UPDATE procurement.schedules
SET is_active = TRUE, next_run_at = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: SetNextRunAt :one
UPDATE procurement.schedules
SET next_run_at = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- Handler records the fired occurrence on the schedule in the same
-- transaction as run creation (idempotency marker).
-- name: UpdateScheduleLastRun :one
UPDATE procurement.schedules
SET
    last_run_at = $3,
    last_run_occurrence_at = $4,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- Ticker claim: due schedules of active orgs. FOR UPDATE SKIP LOCKED makes
-- concurrent tickers/replicas claim disjoint rows; a crash before commit
-- rolls back so a later tick re-fires (at-least-once at the tick level).
-- name: ClaimDueSchedules :many
SELECT * FROM procurement.schedules
WHERE is_active = TRUE AND next_run_at <= NOW()
ORDER BY id ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- Schedule list with the status of the last run created from it (LEFT JOIN
-- LATERAL: never_run when no run exists yet).
-- name: ListSchedulesWithStatus :many
SELECT
    s.*,
    run.id AS last_run_id,
    run.status AS last_run_status,
    run.created_at AS last_run_created_at
FROM procurement.schedules s
LEFT JOIN LATERAL (
    SELECT id, status, created_at
    FROM procurement.inquiry_runs
    WHERE organization_id = s.organization_id AND schedule_ref = s.id
    ORDER BY id DESC
    LIMIT 1
) run ON TRUE
WHERE s.organization_id = $1
ORDER BY s.id ASC;

-- Recent runs of a schedule (schedule detail surface, newest first).
-- name: ListScheduleRuns :many
SELECT * FROM procurement.inquiry_runs
WHERE organization_id = $1 AND schedule_ref = $2
ORDER BY id DESC
LIMIT $3;

-- Overdue recipients of a schedule's active runs (followup_count at the
-- cap and still unanswered): the overdue badge (human escalation).
-- name: CountOverdueRecipientsForSchedule :one
SELECT COUNT(*) AS overdue_count
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id AND run.organization_id = r.organization_id
LEFT JOIN procurement.schedule_followups f ON f.organization_id = r.organization_id
WHERE run.organization_id = $1
  AND run.schedule_ref = $2
  AND r.status = 'sent'
  AND r.followup_count >= COALESCE(f.max_nudges, 1)::int
  AND run.status IN ('sending', 'awaiting_responses');

-- ============================================================
-- Join tables (schedule_products / schedule_suppliers)
-- ============================================================

-- name: InsertScheduleProduct :exec
INSERT INTO procurement.schedule_products (organization_id, schedule_id, product_id)
VALUES ($1, $2, $3)
ON CONFLICT (schedule_id, product_id) DO NOTHING;

-- name: InsertScheduleSupplier :exec
INSERT INTO procurement.schedule_suppliers (organization_id, schedule_id, supplier_id)
VALUES ($1, $2, $3)
ON CONFLICT (schedule_id, supplier_id) DO NOTHING;

-- name: DeleteScheduleProducts :exec
DELETE FROM procurement.schedule_products
WHERE schedule_id = $1 AND organization_id = $2;

-- name: DeleteScheduleSuppliers :exec
DELETE FROM procurement.schedule_suppliers
WHERE schedule_id = $1 AND organization_id = $2;

-- name: ListScheduleProducts :many
SELECT * FROM procurement.schedule_products
WHERE schedule_id = $1 AND organization_id = $2
ORDER BY id ASC;

-- name: ListScheduleSuppliers :many
SELECT * FROM procurement.schedule_suppliers
WHERE schedule_id = $1 AND organization_id = $2
ORDER BY id ASC;

-- ============================================================
-- Follow-up settings (procurement.schedule_followups)
-- ============================================================

-- name: GetFollowUpSettingsByOrg :one
SELECT * FROM procurement.schedule_followups
WHERE organization_id = $1;

-- name: UpsertFollowUpSettings :one
INSERT INTO procurement.schedule_followups (
    organization_id, enabled, deadline_hours, max_nudges, message_template
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (organization_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    deadline_hours = EXCLUDED.deadline_hours,
    max_nudges = EXCLUDED.max_nudges,
    message_template = EXCLUDED.message_template,
    updated_at = NOW()
RETURNING *;

-- Sweep ticker: orgs with follow-ups enabled.
-- name: ListFollowUpEnabledOrgs :many
SELECT organization_id FROM procurement.schedule_followups
WHERE enabled = TRUE
ORDER BY organization_id ASC;

-- ============================================================
-- Scheduled run creation (sibling procurement.inquiry_runs; the sibling
-- reserved source='scheduled' + schedule_ref for this change)
-- ============================================================

-- name: InsertScheduledRun :one
INSERT INTO procurement.inquiry_runs (
    organization_id, status, source, schedule_ref, nota
) VALUES (
    $1, 'draft', 'scheduled', $2, $3
) RETURNING *;

-- ============================================================
-- Org timezone (agent.agent_settings.timezone, default America/Bogota)
-- ============================================================

-- name: GetOrgTimezone :one
SELECT COALESCE(
    (SELECT timezone FROM agent.agent_settings WHERE organization_id = $1),
    'America/Bogota'
)::text AS timezone;

-- ============================================================
-- Follow-up candidates and the atomic nudge guard
-- ============================================================

-- Org-wide sweep candidates: recipients sent before the deadline, org
-- follow-ups enabled, nudge budget left, consent granted, kill switch off.
-- name: ListFollowUpCandidates :many
SELECT
    r.id AS recipient_id,
    r.organization_id,
    r.run_id,
    r.supplier_id,
    r.contact_id,
    r.status AS recipient_status,
    r.sent_at,
    r.followup_count,
    run.status AS run_status,
    c.phone_number AS contact_phone,
    c.consent_status AS consent_status,
    c.display_name AS supplier_display_name,
    sup.nit AS supplier_nit,
    f.deadline_hours,
    f.max_nudges,
    f.message_template
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id AND run.organization_id = r.organization_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
JOIN procurement.suppliers sup ON sup.id = r.supplier_id AND sup.organization_id = r.organization_id
JOIN procurement.schedule_followups f ON f.organization_id = r.organization_id AND f.enabled = TRUE
WHERE r.organization_id = $1
  AND run.status IN ('sending', 'awaiting_responses')
  AND r.status = 'sent'
  AND r.sent_at <= NOW() - (f.deadline_hours::int * INTERVAL '1 hour')
  AND r.followup_count < f.max_nudges
  AND c.consent_status = 'granted'
  AND NOT EXISTS (
      SELECT 1 FROM agent.agent_settings ks
      WHERE ks.organization_id = r.organization_id AND ks.kill_switch = TRUE
  )
ORDER BY r.id ASC
LIMIT $2
FOR UPDATE OF r SKIP LOCKED;

-- Cheap reply-arrival check for one run's remaining recipients (same guard).
-- name: ListOverdueRecipientsForRun :many
SELECT
    r.id AS recipient_id,
    r.organization_id,
    r.run_id,
    r.supplier_id,
    r.contact_id,
    r.status AS recipient_status,
    r.sent_at,
    r.followup_count,
    run.status AS run_status,
    c.phone_number AS contact_phone,
    c.consent_status AS consent_status,
    c.display_name AS supplier_display_name,
    sup.nit AS supplier_nit,
    f.deadline_hours,
    f.max_nudges,
    f.message_template
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id AND run.organization_id = r.organization_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
JOIN procurement.suppliers sup ON sup.id = r.supplier_id AND sup.organization_id = r.organization_id
JOIN procurement.schedule_followups f ON f.organization_id = r.organization_id AND f.enabled = TRUE
WHERE r.organization_id = $2
  AND r.run_id = $1
  AND run.status IN ('sending', 'awaiting_responses')
  AND r.status = 'sent'
  AND r.sent_at <= NOW() - (f.deadline_hours::int * INTERVAL '1 hour')
  AND r.followup_count < f.max_nudges
  AND c.consent_status = 'granted'
  AND NOT EXISTS (
      SELECT 1 FROM agent.agent_settings ks
      WHERE ks.organization_id = r.organization_id AND ks.kill_switch = TRUE
  )
ORDER BY r.id ASC;

-- Reply-arrival trigger: the whatsapp.message.received event carries
-- (org, contact); find the contact's other overdue recipients (same guard).
-- name: ListOverdueRecipientsForContact :many
SELECT
    r.id AS recipient_id,
    r.organization_id,
    r.run_id,
    r.supplier_id,
    r.contact_id,
    r.status AS recipient_status,
    r.sent_at,
    r.followup_count,
    run.status AS run_status,
    c.phone_number AS contact_phone,
    c.consent_status AS consent_status,
    c.display_name AS supplier_display_name,
    sup.nit AS supplier_nit,
    f.deadline_hours,
    f.max_nudges,
    f.message_template
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id AND run.organization_id = r.organization_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
JOIN procurement.suppliers sup ON sup.id = r.supplier_id AND sup.organization_id = r.organization_id
JOIN procurement.schedule_followups f ON f.organization_id = r.organization_id AND f.enabled = TRUE
WHERE r.organization_id = $2
  AND r.contact_id = $1
  AND run.status IN ('sending', 'awaiting_responses')
  AND r.status = 'sent'
  AND r.sent_at <= NOW() - (f.deadline_hours::int * INTERVAL '1 hour')
  AND r.followup_count < f.max_nudges
  AND c.consent_status = 'granted'
  AND NOT EXISTS (
      SELECT 1 FROM agent.agent_settings ks
      WHERE ks.organization_id = r.organization_id AND ks.kill_switch = TRUE
  )
ORDER BY r.id ASC;

-- Dispatch-time re-validation for the followup_send handler.
-- name: GetFollowUpTarget :one
SELECT
    r.id AS recipient_id,
    r.organization_id,
    r.run_id,
    r.supplier_id,
    r.contact_id,
    r.status AS recipient_status,
    r.sent_at,
    r.followup_count,
    run.status AS run_status,
    c.phone_number AS contact_phone,
    c.consent_status AS consent_status,
    c.display_name AS supplier_display_name,
    sup.nit AS supplier_nit,
    f.deadline_hours,
    f.max_nudges,
    f.message_template
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id AND run.organization_id = r.organization_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
JOIN procurement.suppliers sup ON sup.id = r.supplier_id AND sup.organization_id = r.organization_id
LEFT JOIN procurement.schedule_followups f ON f.organization_id = r.organization_id
WHERE r.id = $1 AND r.organization_id = $2;

-- Atomic nudge guard: increments only while below the cap; no row when the
-- cap was reached (the double-nudge guard for sweep/reply races and
-- dispatcher redelivery).
-- name: IncrementFollowupCount :one
UPDATE procurement.inquiry_recipients
SET followup_count = followup_count + 1, updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND followup_count < $3
RETURNING *;
