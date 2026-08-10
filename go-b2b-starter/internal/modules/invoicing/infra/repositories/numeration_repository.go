package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type numerationRepository struct{ store sqlc.Store }

func NewNumerationRepository(store sqlc.Store) domain.NumerationRepository {
	return &numerationRepository{store: store}
}

func (r *numerationRepository) Get(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error) {
	row, err := r.store.GetOrgNumeration(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to get org numeration: %w", err)
	}
	return mapNumeration(&row), nil
}

func (r *numerationRepository) UpsertConfirmed(ctx context.Context, snapshot *domain.NumerationSnapshot) (*domain.NumerationSnapshot, error) {
	row, err := r.store.UpsertOrgNumeration(ctx, sqlc.UpsertOrgNumerationParams{
		OrganizationID: snapshot.OrganizationID,
		Mode:           string(snapshot.Mode),
		ResolutionID:   helpers.ToPgText(snapshot.ResolutionID),
		Prefijo:        helpers.ToPgText(snapshot.Prefix),
		NextNumber:     helpers.ToPgText(snapshot.NextNumber),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert org numeration: %w", err)
	}
	return mapNumeration(&row), nil
}

func mapNumeration(row *sqlc.InvoicingOrgNumeration) *domain.NumerationSnapshot {
	snapshot := &domain.NumerationSnapshot{
		OrganizationID: row.OrganizationID,
		Mode:           domain.NumerationMode(row.Mode),
		ResolutionID:   helpers.FromPgText(row.ResolutionID),
		Prefix:         helpers.FromPgText(row.Prefijo),
		NextNumber:     helpers.FromPgText(row.NextNumber),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		snapshot.ConfirmedAt = &t
	}
	return snapshot
}
