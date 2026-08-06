package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

type webhookLogRepository struct {
	store sqlc.Store
}

func NewWebhookLogRepository(store sqlc.Store) domain.WebhookLogRepository {
	return &webhookLogRepository{store: store}
}

func (r *webhookLogRepository) Insert(ctx context.Context, log *domain.WebhookLog) (*domain.WebhookLog, error) {
	params := sqlc.InsertWebhookLogParams{
		OrganizationID: log.OrganizationID,
		Status:         log.Status,
		EventType:      helpers.ToPgText(log.EventType),
		PhoneNumberID:  helpers.ToPgText(log.PhoneNumberID),
		RawHeaders:     log.RawHeaders,
		RawBody:        log.RawBody,
		ErrorMessage:   helpers.ToPgText(log.ErrorMessage),
		ProcessedAt:    helpers.ToPgTimestampPtr(log.ProcessedAt),
	}

	result, err := r.store.InsertWebhookLog(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert webhook log: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *webhookLogRepository) GetStatsByOrganization(ctx context.Context, orgID int32) (*domain.WebhookLogStats, error) {
	now := time.Now()
	last24hRows, err := r.store.GetWebhookLogStatsByOrganization(ctx, sqlc.GetWebhookLogStatsByOrganizationParams{
		OrganizationID: orgID,
		CreatedAt:      helpers.ToPgTimestamp(now.Add(-24 * time.Hour)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get 24h stats: %w", err)
	}

	last7dRows, err := r.store.GetWebhookLogStatsByOrganization(ctx, sqlc.GetWebhookLogStatsByOrganizationParams{
		OrganizationID: orgID,
		CreatedAt:      helpers.ToPgTimestamp(now.Add(-7 * 24 * time.Hour)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get 7d stats: %w", err)
	}

	totalRows, err := r.store.GetWebhookLogStatsByOrganization(ctx, sqlc.GetWebhookLogStatsByOrganizationParams{
		OrganizationID: orgID,
		CreatedAt:      helpers.ToPgTimestamp(time.Time{}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	stats := &domain.WebhookLogStats{
		ByStatus: make(map[string]int),
	}

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

	lastError, err := r.store.GetLastWebhookErrorByOrganization(ctx, orgID)
	if err == nil && lastError.ErrorMessage.Valid {
		stats.LastError = lastError.ErrorMessage.String
		if lastError.CreatedAt.Valid {
			stats.LastErrorAt = &lastError.CreatedAt.Time
		}
	}

	return stats, nil
}

func (r *webhookLogRepository) mapToDomain(l *sqlc.WhatsappWebhookLog) *domain.WebhookLog {
	return &domain.WebhookLog{
		ID:             l.ID,
		OrganizationID: l.OrganizationID,
		Status:         l.Status,
		EventType:      helpers.FromPgText(l.EventType),
		PhoneNumberID:  helpers.FromPgText(l.PhoneNumberID),
		RawHeaders:     l.RawHeaders,
		RawBody:        l.RawBody,
		ErrorMessage:   helpers.FromPgText(l.ErrorMessage),
		ProcessedAt:    helpers.FromPgTimestampPtr(l.ProcessedAt),
		CreatedAt:      l.CreatedAt.Time,
	}
}
