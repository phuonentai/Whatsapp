package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

// Outbox event statuses.
const (
	StatusPending    = "pending"
	StatusDispatched = "dispatched"
	StatusDeadLetter = "dead_letter"
)

// OutboxEvent is a durable, dispatcher-managed event row.
type OutboxEvent struct {
	ID             int64           `json:"id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	OrganizationID *int32          `json:"organization_id,omitempty"`
	Status         string          `json:"status"`
	Attempts       int             `json:"attempts"`
	NextAttemptAt  time.Time       `json:"next_attempt_at"`
	LastError      string          `json:"last_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	DispatchedAt   *time.Time      `json:"dispatched_at,omitempty"`
}

// EventCodec reconstructs a typed domain event from its stored payload.
type EventCodec func(payload json.RawMessage) (eventbus.Event, error)

// Repository abstracts outbox persistence.
type Repository interface {
	Insert(ctx context.Context, eventType string, payload json.RawMessage, organizationID *int32) (*OutboxEvent, error)
	ClaimPending(ctx context.Context, limit int32) ([]*OutboxEvent, error)
	MarkDispatched(ctx context.Context, id int64) error
	Retry(ctx context.Context, id int64, nextAttemptAt time.Time, lastError string) error
	DeadLetter(ctx context.Context, id int64, lastError string) error
	ListDeadLetter(ctx context.Context, limit int32) ([]*OutboxEvent, error)
	Requeue(ctx context.Context, id int64) error
	Prune(ctx context.Context, before time.Time) error
}
