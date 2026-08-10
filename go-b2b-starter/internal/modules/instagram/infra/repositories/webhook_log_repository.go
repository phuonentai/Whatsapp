package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
)

type webhookLogRepository struct {
	store sqlc.Store
}

// transactioner is implemented by *sqlc.SQLStore (see gen/exec.go) and lets
// repositories compose queries atomically. Non-transactional stores run the
// function directly (test fakes).
type transactioner interface {
	Transaction(ctx context.Context, fn func(sqlc.Store) error) error
}

func NewWebhookLogRepository(store sqlc.Store) domain.WebhookLogRepository {
	return &webhookLogRepository{store: store}
}

func (r *webhookLogRepository) inTx(ctx context.Context, fn func(sqlc.Store) error) error {
	if t, ok := r.store.(transactioner); ok {
		return t.Transaction(ctx, fn)
	}
	return fn(r.store)
}

func (r *webhookLogRepository) Insert(ctx context.Context, log *domain.WebhookLog) (*domain.WebhookLog, error) {
	result, err := r.store.InsertInstagramWebhookLog(ctx, r.params(log))
	if err != nil {
		return nil, fmt.Errorf("failed to insert instagram webhook log: %w", err)
	}
	return r.mapToDomain(&result), nil
}

// InsertWithOutbox persists the webhook log and its outbox events atomically.
// A duplicate delivery (unique violation on delivery_key) maps to
// domain.ErrDuplicateDelivery; the transaction is rolled back entirely.
func (r *webhookLogRepository) InsertWithOutbox(ctx context.Context, log *domain.WebhookLog, events []domain.OutboxEventInput) (*domain.WebhookLog, error) {
	var result *domain.WebhookLog

	err := r.inTx(ctx, func(s sqlc.Store) error {
		inserted, err := s.InsertInstagramWebhookLog(ctx, r.params(log))
		if err != nil {
			return err
		}
		mapped := r.mapToDomain(&inserted)
		result = mapped

		for _, ev := range events {
			if _, err := s.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
				EventType:      ev.EventType,
				Payload:        ev.Payload,
				OrganizationID: helpers.ToPgInt4Ptr(log.OrganizationID),
			}); err != nil {
				return fmt.Errorf("failed to insert outbox event %q: %w", ev.EventType, err)
			}
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" && pgErr.ConstraintName == "idx_instagram_webhook_logs_delivery_key" {
			return nil, domain.ErrDuplicateDelivery
		}
		return nil, fmt.Errorf("failed to insert instagram webhook log with outbox events: %w", err)
	}

	return result, nil
}

func (r *webhookLogRepository) GetByID(ctx context.Context, id int32) (*domain.WebhookLog, error) {
	result, err := r.store.GetInstagramWebhookLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlc.ErrRecordNotFound) {
			return nil, domain.ErrWebhookLogNotFound
		}
		return nil, fmt.Errorf("failed to get instagram webhook log %d: %w", id, err)
	}
	return r.mapToDomain(&result), nil
}

func (r *webhookLogRepository) params(log *domain.WebhookLog) sqlc.InsertInstagramWebhookLogParams {
	return sqlc.InsertInstagramWebhookLogParams{
		OrganizationID: helpers.ToPgInt4Ptr(log.OrganizationID),
		Status:         log.Status,
		EventType:      helpers.ToPgText(log.EventType),
		IgUserID:       helpers.ToPgText(log.IGUserID),
		RawHeaders:     log.RawHeaders,
		RawBody:        log.RawBody,
		ErrorMessage:   helpers.ToPgText(log.ErrorMessage),
		ProcessedAt:    helpers.ToPgTimestampPtr(log.ProcessedAt),
		DeliveryKey:    helpers.ToPgText(log.DeliveryKey),
	}
}

func (r *webhookLogRepository) GetStatsByOrganization(ctx context.Context, orgID int32) (*domain.WebhookLogStats, error) {
	now := time.Now()
	last24hRows, err := r.store.GetInstagramWebhookLogStatsByOrganization(ctx, sqlc.GetInstagramWebhookLogStatsByOrganizationParams{
		OrganizationID: helpers.ToPgInt4(orgID),
		CreatedAt:      helpers.ToPgTimestamp(now.Add(-24 * time.Hour)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get 24h stats: %w", err)
	}

	last7dRows, err := r.store.GetInstagramWebhookLogStatsByOrganization(ctx, sqlc.GetInstagramWebhookLogStatsByOrganizationParams{
		OrganizationID: helpers.ToPgInt4(orgID),
		CreatedAt:      helpers.ToPgTimestamp(now.Add(-7 * 24 * time.Hour)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get 7d stats: %w", err)
	}

	totalRows, err := r.store.GetInstagramWebhookLogStatsByOrganization(ctx, sqlc.GetInstagramWebhookLogStatsByOrganizationParams{
		OrganizationID: helpers.ToPgInt4(orgID),
		CreatedAt:      helpers.ToPgTimestamp(time.Time{}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	stats := &domain.WebhookLogStats{ByStatus: make(map[string]int)}

	for _, row := range last24hRows {
		stats.Last24h += int(row.Count)
		stats.ByStatus[row.Status] += int(row.Count)
	}
	for _, row := range last7dRows {
		stats.Last7d += int(row.Count)
	}
	for _, row := range totalRows {
		stats.Total += int(row.Count)
	}

	lastError, err := r.store.GetLastInstagramWebhookErrorByOrganization(ctx, helpers.ToPgInt4(orgID))
	if err == nil && lastError.ErrorMessage.Valid {
		stats.LastError = lastError.ErrorMessage.String
		if lastError.CreatedAt.Valid {
			stats.LastErrorAt = &lastError.CreatedAt.Time
		}
	}

	return stats, nil
}

func (r *webhookLogRepository) mapToDomain(l *sqlc.WhatsappInstagramWebhookLog) *domain.WebhookLog {
	return &domain.WebhookLog{
		ID:             l.ID,
		OrganizationID: helpers.FromPgInt4Ptr(l.OrganizationID),
		Status:         l.Status,
		EventType:      helpers.FromPgText(l.EventType),
		IGUserID:       helpers.FromPgText(l.IgUserID),
		RawHeaders:     l.RawHeaders,
		RawBody:        l.RawBody,
		ErrorMessage:   helpers.FromPgText(l.ErrorMessage),
		ProcessedAt:    helpers.FromPgTimestampPtr(l.ProcessedAt),
		DeliveryKey:    helpers.FromPgText(l.DeliveryKey),
		CreatedAt:      l.CreatedAt.Time,
	}
}
