package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
)

type analyticsRepository struct {
	store sqlc.Store
}

func NewAnalyticsRepository(store sqlc.Store) domain.AnalyticsRepository {
	return &analyticsRepository{store: store}
}

func (r *analyticsRepository) RevenueByPeriod(ctx context.Context, orgID int32, from, to time.Time, period string) ([]domain.RevenuePoint, error) {
	rows, err := r.store.RevenueByPeriod(ctx, sqlc.RevenueByPeriodParams{
		OrganizationID: orgID,
		Column2:        pgtype.Timestamptz{Time: from, Valid: true},
		Column3:        pgtype.Timestamptz{Time: to, Valid: true},
		Column4:        period,
	})
	if err != nil {
		return nil, fmt.Errorf("revenue by period: %w", err)
	}
	points := make([]domain.RevenuePoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.RevenuePoint{
			Periodo:    row.Periodo.Time.Format("2006-01-02"),
			MontoTotal: numericToFloat64(row.MontoTotal),
		})
	}
	return points, nil
}

func (r *analyticsRepository) TopCustomersByRevenue(ctx context.Context, orgID int32, limit int32) ([]domain.TopCustomer, error) {
	rows, err := r.store.TopCustomersByRevenue(ctx, sqlc.TopCustomersByRevenueParams{
		OrganizationID: orgID,
		Limit:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("top customers by revenue: %w", err)
	}
	customers := make([]domain.TopCustomer, 0, len(rows))
	for _, row := range rows {
		customers = append(customers, domain.TopCustomer{
			Nombre:     row.Nombre,
			MontoTotal: numericToFloat64(row.MontoTotal),
		})
	}
	return customers, nil
}

func (r *analyticsRepository) FunnelByStage(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	rows, err := r.store.FunnelByStageAggregates(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("funnel by stage: %w", err)
	}
	entries := make([]domain.FunnelEntry, 0, len(rows))
	for _, row := range rows {
		etapa := "otras_pipelines"
		if row.Etapa.Valid {
			etapa = row.Etapa.String
		}
		entries = append(entries, domain.FunnelEntry{
			Etapa:      etapa,
			Cantidad:   row.Cantidad,
			MontoTotal: numericToFloat64(row.MontoTotal),
		})
	}
	return entries, nil
}

func (r *analyticsRepository) DealStateCounts(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	rows, err := r.store.DealStateCounts(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("deal state counts: %w", err)
	}
	entries := make([]domain.FunnelEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, domain.FunnelEntry{
			Etapa:      row.Estado,
			Cantidad:   row.Cantidad,
			MontoTotal: numericToFloat64(row.MontoTotal),
		})
	}
	return entries, nil
}

func (r *analyticsRepository) InactiveContacts(ctx context.Context, orgID int32, since time.Time) ([]domain.InactiveContact, error) {
	rows, err := r.store.InactiveContacts(ctx, sqlc.InactiveContactsParams{
		OrganizationID: orgID,
		Column2:        pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("inactive contacts: %w", err)
	}
	contacts := make([]domain.InactiveContact, 0, len(rows))
	for _, row := range rows {
		var lastMessageAt *time.Time
		if row.UltimoMensajeAt.Valid {
			t := row.UltimoMensajeAt.Time
			lastMessageAt = &t
		}
		contacts = append(contacts, domain.InactiveContact{
			Telefono:        helpers.FromPgText(row.Telefono),
			Nombre:          row.Nombre,
			UltimoMensajeAt: lastMessageAt,
			Clasificacion:   row.Clasificacion,
		})
	}
	return contacts, nil
}

func (r *analyticsRepository) DefaultPipelineStages(ctx context.Context, orgID int32) ([]string, error) {
	pipeline, err := r.store.GetDefaultPipelineByOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("default pipeline: %w", err)
	}
	stages, err := r.store.ListStagesByPipeline(ctx, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	names := make([]string, 0, len(stages))
	for _, stage := range stages {
		names = append(names, stage.Nombre)
	}
	return names, nil
}

func numericToFloat64(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
