package repositories

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type importRunRepository struct{ store sqlc.Store }

func NewImportRunRepository(store sqlc.Store) domain.ImportRunRepository {
	return &importRunRepository{store: store}
}

func (r *importRunRepository) Record(ctx context.Context, run *domain.ImportRun) (*domain.ImportRun, error) {
	row, err := r.store.InsertImportRun(ctx, sqlc.InsertImportRunParams{
		OrganizationID: run.OrganizationID,
		Kind:           string(run.Kind),
		Counts:         helpers.ToJSONB(countsToAny(run.Counts)),
		Error:          helpers.ToPgText(run.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to record import run: %w", err)
	}
	return mapImportRun(&row), nil
}

func (r *importRunRepository) ListByOrg(ctx context.Context, orgID int32, limit int32) ([]*domain.ImportRun, error) {
	rows, err := r.store.ListImportRunsByOrg(ctx, sqlc.ListImportRunsByOrgParams{OrganizationID: orgID, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("failed to list import runs: %w", err)
	}
	runs := make([]*domain.ImportRun, len(rows))
	for i := range rows {
		runs[i] = mapImportRun(&rows[i])
	}
	return runs, nil
}

func countsToAny(counts map[string]int32) map[string]any {
	out := make(map[string]any, len(counts))
	for k, v := range counts {
		out[k] = v
	}
	return out
}

func mapImportRun(row *sqlc.InvoicingImportRun) *domain.ImportRun {
	raw := helpers.FromJSONB(row.Counts)
	counts := map[string]int32{}
	for k, v := range raw {
		if f, ok := v.(float64); ok {
			counts[k] = int32(f)
		}
	}
	return &domain.ImportRun{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Kind:           domain.ImportRunKind(row.Kind),
		Counts:         counts,
		Error:          helpers.FromPgText(row.Error),
		PulledAt:       row.PulledAt.Time,
	}
}
