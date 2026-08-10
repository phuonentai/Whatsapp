-- Outbox queries

-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (
    event_type,
    payload,
    organization_id
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: ClaimPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'pending'
  AND next_attempt_at <= NOW()
ORDER BY id ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventDispatched :one
UPDATE outbox_events
SET
    status = 'dispatched',
    dispatched_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RetryOutboxEvent :one
UPDATE outbox_events
SET
    attempts = attempts + 1,
    next_attempt_at = $2,
    last_error = $3
WHERE id = $1
RETURNING *;

-- name: MarkOutboxEventDeadLetter :one
UPDATE outbox_events
SET
    status = 'dead_letter',
    attempts = attempts + 1,
    last_error = $2
WHERE id = $1
RETURNING *;

-- name: ListDeadLetterOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'dead_letter'
ORDER BY created_at DESC
LIMIT $1;

-- name: RequeueOutboxEvent :one
UPDATE outbox_events
SET
    status = 'pending',
    attempts = 0,
    next_attempt_at = NOW(),
    last_error = NULL
WHERE id = $1
RETURNING *;

-- name: PruneOutboxEvents :exec
DELETE FROM outbox_events
WHERE status = 'dispatched'
  AND created_at < $1;
