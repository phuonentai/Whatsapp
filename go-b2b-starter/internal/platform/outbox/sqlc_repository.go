package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

type sqlcRepository struct {
	store sqlc.Store
}

// NewSQLCRepository builds the outbox repository over the SQLC store.
func NewSQLCRepository(store sqlc.Store) Repository {
	return &sqlcRepository{store: store}
}

func (r *sqlcRepository) Insert(ctx context.Context, eventType string, payload json.RawMessage, organizationID *int32) (*OutboxEvent, error) {
	row, err := r.store.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
		EventType:      eventType,
		Payload:        payload,
		OrganizationID: helpers.ToPgInt4Ptr(organizationID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert outbox event: %w", err)
	}
	return r.mapToDomain(&row), nil
}

func (r *sqlcRepository) ClaimPending(ctx context.Context, limit int32) ([]*OutboxEvent, error) {
	rows, err := r.store.ClaimPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim pending outbox events: %w", err)
	}
	events := make([]*OutboxEvent, 0, len(rows))
	for i := range rows {
		events = append(events, r.mapToDomain(&rows[i]))
	}
	return events, nil
}

func (r *sqlcRepository) MarkDispatched(ctx context.Context, id int64) error {
	if _, err := r.store.MarkOutboxEventDispatched(ctx, id); err != nil {
		return fmt.Errorf("failed to mark outbox event %d dispatched: %w", id, err)
	}
	return nil
}

func (r *sqlcRepository) Retry(ctx context.Context, id int64, nextAttemptAt time.Time, lastError string) error {
	if _, err := r.store.RetryOutboxEvent(ctx, sqlc.RetryOutboxEventParams{
		ID:            id,
		NextAttemptAt: helpers.ToPgTimestamptz(nextAttemptAt),
		LastError:     helpers.ToPgText(lastError),
	}); err != nil {
		return fmt.Errorf("failed to schedule retry for outbox event %d: %w", id, err)
	}
	return nil
}

func (r *sqlcRepository) DeadLetter(ctx context.Context, id int64, lastError string) error {
	if _, err := r.store.MarkOutboxEventDeadLetter(ctx, sqlc.MarkOutboxEventDeadLetterParams{
		ID:        id,
		LastError: helpers.ToPgText(lastError),
	}); err != nil {
		return fmt.Errorf("failed to dead-letter outbox event %d: %w", id, err)
	}
	return nil
}

func (r *sqlcRepository) ListDeadLetter(ctx context.Context, limit int32) ([]*OutboxEvent, error) {
	rows, err := r.store.ListDeadLetterOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list dead-letter outbox events: %w", err)
	}
	events := make([]*OutboxEvent, 0, len(rows))
	for i := range rows {
		events = append(events, r.mapToDomain(&rows[i]))
	}
	return events, nil
}

func (r *sqlcRepository) Requeue(ctx context.Context, id int64) error {
	if _, err := r.store.RequeueOutboxEvent(ctx, id); err != nil {
		return fmt.Errorf("failed to requeue outbox event %d: %w", id, err)
	}
	return nil
}

func (r *sqlcRepository) Prune(ctx context.Context, before time.Time) error {
	if err := r.store.PruneOutboxEvents(ctx, helpers.ToPgTimestamptz(before)); err != nil {
		return fmt.Errorf("failed to prune outbox events: %w", err)
	}
	return nil
}

func (r *sqlcRepository) mapToDomain(row *sqlc.OutboxEvent) *OutboxEvent {
	return &OutboxEvent{
		ID:             row.ID,
		EventType:      row.EventType,
		Payload:        json.RawMessage(row.Payload),
		OrganizationID: helpers.FromPgInt4Ptr(row.OrganizationID),
		Status:         row.Status,
		Attempts:       int(row.Attempts),
		NextAttemptAt:  row.NextAttemptAt.Time,
		LastError:      helpers.FromPgText(row.LastError),
		CreatedAt:      row.CreatedAt.Time,
		DispatchedAt:   helpers.FromPgTimestamptzPtr(row.DispatchedAt),
	}
}
