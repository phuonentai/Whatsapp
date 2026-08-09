package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// aiUsageRepository implements domain.AiUsageRepository using SQLC internally.
type aiUsageRepository struct {
	store sqlc.Store
}

// NewAiUsageRepository creates a new AiUsageRepository implementation.
func NewAiUsageRepository(store sqlc.Store) domain.AiUsageRepository {
	return &aiUsageRepository{store: store}
}

// RecordUsage appends the event (idempotent per request_id) and then increments
// the period totals. The event insert is the idempotency gate: when the
// request_id was already recorded, totals are NOT incremented again.
func (r *aiUsageRepository) RecordUsage(ctx context.Context, event *domain.AiUsageEvent, periodStart, periodEnd time.Time) (bool, error) {
	affected, err := r.store.InsertAiUsageEvent(ctx, sqlc.InsertAiUsageEventParams{
		OrganizationID:  event.OrganizationID,
		Feature:         event.Feature,
		Model:           event.Model,
		TokensInput:     event.TokensInput,
		TokensOutput:    event.TokensOutput,
		TokensEmbedding: event.TokensEmbedding,
		CreditsConsumed: event.CreditsConsumed,
		RequestID:       event.RequestID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to insert ai usage event: %w", err)
	}
	if affected == 0 {
		// Duplicate request_id — already recorded.
		return false, nil
	}

	_, err = r.store.UpsertAiUsage(ctx, sqlc.UpsertAiUsageParams{
		OrganizationID:  event.OrganizationID,
		PeriodStart:     pgtype.Timestamptz{Time: periodStart, Valid: true},
		PeriodEnd:       pgtype.Timestamptz{Time: periodEnd, Valid: true},
		TokensInput:     event.TokensInput,
		TokensOutput:    event.TokensOutput,
		TokensEmbedding: event.TokensEmbedding,
		CreditsUsed:     event.CreditsConsumed,
	})
	if err != nil {
		return false, fmt.Errorf("failed to upsert ai usage totals: %w", err)
	}

	return true, nil
}

func (r *aiUsageRepository) GetAiUsageByOrgAndPeriod(ctx context.Context, organizationID int32, periodStart time.Time) (*domain.AiUsage, error) {
	result, err := r.store.GetAiUsageByOrgAndPeriod(ctx, sqlc.GetAiUsageByOrgAndPeriodParams{
		OrganizationID: organizationID,
		PeriodStart:    pgtype.Timestamptz{Time: periodStart, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAiUsageNotFound
		}
		return nil, fmt.Errorf("failed to get ai usage: %w", err)
	}

	return &domain.AiUsage{
		OrganizationID:  result.OrganizationID,
		PeriodStart:     result.PeriodStart.Time,
		PeriodEnd:       result.PeriodEnd.Time,
		TokensInput:     result.TokensInput,
		TokensOutput:    result.TokensOutput,
		TokensEmbedding: result.TokensEmbedding,
		CreditsUsed:     result.CreditsUsed,
	}, nil
}

func (r *aiUsageRepository) UpdateAiCreditsMax(ctx context.Context, organizationID int32, creditsMax int32, periodStart, periodEnd time.Time) error {
	_, err := r.store.UpdateAiCreditsMax(ctx, sqlc.UpdateAiCreditsMaxParams{
		OrganizationID: organizationID,
		AiCreditsMax:   creditsMax,
		PeriodStart:    toPgTimestamp(periodStart),
		PeriodEnd:      toPgTimestamp(periodEnd),
	})
	if err != nil {
		return fmt.Errorf("failed to update ai credits max: %w", err)
	}
	return nil
}
