-- name: CreateTicket :one
INSERT INTO crm.tickets (
    organization_id, contact_id, conversation_id, title, description, status, priority, tags, assignee_stytch_member_id, sla_due_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetTicketByID :one
SELECT * FROM crm.tickets
WHERE id = $1 AND organization_id = $2
LIMIT 1;

-- name: ListTicketsByOrg :many
SELECT * FROM crm.tickets
WHERE organization_id = $1
  AND ($2::text = '' OR status = $2)
  AND ($3::text = '' OR assignee_stytch_member_id = $3)
ORDER BY (status = 'open') DESC, updated_at DESC
LIMIT $4 OFFSET $5;

-- name: UpdateTicketStatus :one
UPDATE crm.tickets
SET status = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateTicketAssignee :one
UPDATE crm.tickets
SET assignee_stytch_member_id = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateTicketPriority :one
UPDATE crm.tickets
SET priority = $3, sla_due_at = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateTicketTags :one
UPDATE crm.tickets
SET tags = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: InsertTicketEvent :one
INSERT INTO crm.ticket_events (ticket_id, event_type, actor_stytch_member_id, payload)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTicketEvents :many
SELECT * FROM crm.ticket_events
WHERE ticket_id = $1
ORDER BY created_at ASC;
