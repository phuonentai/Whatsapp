package repositories

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type activityRepository struct {
	store sqlc.Store
}

func NewActivityRepository(store sqlc.Store) domain.ActivityRepository {
	return &activityRepository{store: store}
}

func (r *activityRepository) Create(ctx context.Context, a *domain.Activity) (*domain.Activity, error) {
	result, err := r.store.CreateActivity(ctx, sqlc.CreateActivityParams{
		OrganizationID:   a.OrganizationID,
		ContactID:        helpers.ToPgInt4Ptr(a.ContactID),
		CompanyID:        helpers.ToPgInt4Ptr(a.CompanyID),
		DealID:           helpers.ToPgInt4Ptr(a.DealID),
		ConversationID:   helpers.ToPgInt4Ptr(a.ConversationID),
		Tipo:             string(a.Tipo),
		Asunto:           helpers.ToPgText(a.Asunto),
		Contenido:        helpers.ToPgText(a.Contenido),
		Estado:           helpers.ToPgText(a.Estado),
		FechaVencimiento: helpers.ToPgTimestamptzPtr(a.FechaVencimiento),
		RealizadaPor:     helpers.ToPgInt4Ptr(a.RealizadaPor),
		RealizadaEn:      helpers.ToPgTimestamptz(a.RealizadaEn),
		Metadata:         helpers.ToJSONB(a.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}
	return mapActivityFromDomain(&result), nil
}

func (r *activityRepository) GetByID(ctx context.Context, orgID, activityID int32) (*domain.Activity, error) {
	result, err := r.store.GetActivityByID(ctx, sqlc.GetActivityByIDParams{ID: activityID, OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	return mapActivityFromDomain(&result), nil
}

func (r *activityRepository) ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	results, err := r.store.ListActivitiesByOrganization(ctx, sqlc.ListActivitiesByOrganizationParams{
		OrganizationID: orgID,
		Column2:        helpers.ToPgText(tipo),
		Column3:        helpers.ToPgText(entityType),
		Column4:        entityID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}
	return mapActivityRows(results), nil
}

func (r *activityRepository) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	results, err := r.store.ListActivitiesByContact(ctx, sqlc.ListActivitiesByContactParams{
		ContactID:      helpers.ToPgInt4(contactID),
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities by contact: %w", err)
	}
	return mapActivityRowsFromContact(results), nil
}

func (r *activityRepository) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	results, err := r.store.ListActivitiesByDeal(ctx, sqlc.ListActivitiesByDealParams{
		DealID:         helpers.ToPgInt4(dealID),
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities by deal: %w", err)
	}
	return mapActivityRowsFromDeal(results), nil
}

func (r *activityRepository) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	results, err := r.store.ListActivitiesByCompany(ctx, sqlc.ListActivitiesByCompanyParams{
		CompanyID:      helpers.ToPgInt4(companyID),
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities by company: %w", err)
	}
	return mapActivityRowsFromCompany(results), nil
}

func (r *activityRepository) CountByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID int32) (int32, error) {
	count, err := r.store.CountActivitiesByOrganization(ctx, sqlc.CountActivitiesByOrganizationParams{
		OrganizationID: orgID,
		Column2:        helpers.ToPgText(tipo),
		Column3:        helpers.ToPgText(entityType),
		Column4:        entityID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count activities: %w", err)
	}
	return int32(count), nil
}

func (r *activityRepository) CountByContact(ctx context.Context, contactID, orgID int32) (int32, error) {
	count, err := r.store.CountActivitiesByContact(ctx, sqlc.CountActivitiesByContactParams{ContactID: helpers.ToPgInt4(contactID), OrganizationID: orgID})
	if err != nil {
		return 0, fmt.Errorf("failed to count activities by contact: %w", err)
	}
	return int32(count), nil
}

func (r *activityRepository) CountByDeal(ctx context.Context, dealID, orgID int32) (int32, error) {
	count, err := r.store.CountActivitiesByDeal(ctx, sqlc.CountActivitiesByDealParams{DealID: helpers.ToPgInt4(dealID), OrganizationID: orgID})
	if err != nil {
		return 0, fmt.Errorf("failed to count activities by deal: %w", err)
	}
	return int32(count), nil
}

func (r *activityRepository) CountByCompany(ctx context.Context, companyID, orgID int32) (int32, error) {
	count, err := r.store.CountActivitiesByCompany(ctx, sqlc.CountActivitiesByCompanyParams{CompanyID: helpers.ToPgInt4(companyID), OrganizationID: orgID})
	if err != nil {
		return 0, fmt.Errorf("failed to count activities by company: %w", err)
	}
	return int32(count), nil
}

func mapActivityFromDomain(c *sqlc.CrmActivity) *domain.Activity {
	return &domain.Activity{
		ID:               c.ID,
		OrganizationID:   c.OrganizationID,
		ContactID:        helpers.FromPgInt4Ptr(c.ContactID),
		CompanyID:        helpers.FromPgInt4Ptr(c.CompanyID),
		DealID:           helpers.FromPgInt4Ptr(c.DealID),
		ConversationID:   helpers.FromPgInt4Ptr(c.ConversationID),
		Tipo:             domain.ActivityType(c.Tipo),
		Asunto:           helpers.FromPgText(c.Asunto),
		Contenido:        helpers.FromPgText(c.Contenido),
		Estado:           helpers.FromPgText(c.Estado),
		FechaVencimiento: helpers.FromPgTimestamptzPtr(c.FechaVencimiento),
		RealizadaPor:     helpers.FromPgInt4Ptr(c.RealizadaPor),
		RealizadaEn:      c.RealizadaEn.Time,
		Metadata:         helpers.FromJSONB(c.Metadata),
		CreatedAt:        c.CreatedAt.Time,
		UpdatedAt:        c.UpdatedAt.Time,
	}
}

func mapActivityRows(rows []sqlc.ListActivitiesByOrganizationRow) []*domain.ActivityWithActor {
	result := make([]*domain.ActivityWithActor, len(rows))
	for i := range rows {
		r := &rows[i]
		result[i] = &domain.ActivityWithActor{
			Activity: domain.Activity{
				ID:               r.ID,
				OrganizationID:   r.OrganizationID,
				ContactID:        helpers.FromPgInt4Ptr(r.ContactID),
				CompanyID:        helpers.FromPgInt4Ptr(r.CompanyID),
				DealID:           helpers.FromPgInt4Ptr(r.DealID),
				ConversationID:   helpers.FromPgInt4Ptr(r.ConversationID),
				Tipo:             domain.ActivityType(r.Tipo),
				Asunto:           helpers.FromPgText(r.Asunto),
				Contenido:        helpers.FromPgText(r.Contenido),
				Estado:           helpers.FromPgText(r.Estado),
				FechaVencimiento: helpers.FromPgTimestamptzPtr(r.FechaVencimiento),
				RealizadaPor:     helpers.FromPgInt4Ptr(r.RealizadaPor),
				RealizadaEn:      r.RealizadaEn.Time,
				Metadata:         helpers.FromJSONB(r.Metadata),
				CreatedAt:        r.CreatedAt.Time,
				UpdatedAt:        r.UpdatedAt.Time,
			},
			RealizadaPorNombre: helpers.FromPgText(r.RealizadaPorNombre),
		}
	}
	return result
}

func mapActivityRowsFromContact(rows []sqlc.ListActivitiesByContactRow) []*domain.ActivityWithActor {
	result := make([]*domain.ActivityWithActor, len(rows))
	for i := range rows {
		r := &rows[i]
		result[i] = &domain.ActivityWithActor{
			Activity: domain.Activity{
				ID: r.ID, OrganizationID: r.OrganizationID,
				ContactID: helpers.FromPgInt4Ptr(r.ContactID), CompanyID: helpers.FromPgInt4Ptr(r.CompanyID),
				DealID: helpers.FromPgInt4Ptr(r.DealID), ConversationID: helpers.FromPgInt4Ptr(r.ConversationID),
				Tipo: domain.ActivityType(r.Tipo), Asunto: helpers.FromPgText(r.Asunto),
				Contenido: helpers.FromPgText(r.Contenido), Estado: helpers.FromPgText(r.Estado),
				FechaVencimiento: helpers.FromPgTimestamptzPtr(r.FechaVencimiento),
				RealizadaPor: helpers.FromPgInt4Ptr(r.RealizadaPor), RealizadaEn: r.RealizadaEn.Time,
				Metadata: helpers.FromJSONB(r.Metadata), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
			},
			RealizadaPorNombre: helpers.FromPgText(r.RealizadaPorNombre),
		}
	}
	return result
}

func mapActivityRowsFromDeal(rows []sqlc.ListActivitiesByDealRow) []*domain.ActivityWithActor {
	result := make([]*domain.ActivityWithActor, len(rows))
	for i := range rows {
		r := &rows[i]
		result[i] = &domain.ActivityWithActor{
			Activity: domain.Activity{
				ID: r.ID, OrganizationID: r.OrganizationID,
				ContactID: helpers.FromPgInt4Ptr(r.ContactID), CompanyID: helpers.FromPgInt4Ptr(r.CompanyID),
				DealID: helpers.FromPgInt4Ptr(r.DealID), ConversationID: helpers.FromPgInt4Ptr(r.ConversationID),
				Tipo: domain.ActivityType(r.Tipo), Asunto: helpers.FromPgText(r.Asunto),
				Contenido: helpers.FromPgText(r.Contenido), Estado: helpers.FromPgText(r.Estado),
				FechaVencimiento: helpers.FromPgTimestamptzPtr(r.FechaVencimiento),
				RealizadaPor: helpers.FromPgInt4Ptr(r.RealizadaPor), RealizadaEn: r.RealizadaEn.Time,
				Metadata: helpers.FromJSONB(r.Metadata), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
			},
			RealizadaPorNombre: helpers.FromPgText(r.RealizadaPorNombre),
		}
	}
	return result
}

func mapActivityRowsFromCompany(rows []sqlc.ListActivitiesByCompanyRow) []*domain.ActivityWithActor {
	result := make([]*domain.ActivityWithActor, len(rows))
	for i := range rows {
		r := &rows[i]
		result[i] = &domain.ActivityWithActor{
			Activity: domain.Activity{
				ID: r.ID, OrganizationID: r.OrganizationID,
				ContactID: helpers.FromPgInt4Ptr(r.ContactID), CompanyID: helpers.FromPgInt4Ptr(r.CompanyID),
				DealID: helpers.FromPgInt4Ptr(r.DealID), ConversationID: helpers.FromPgInt4Ptr(r.ConversationID),
				Tipo: domain.ActivityType(r.Tipo), Asunto: helpers.FromPgText(r.Asunto),
				Contenido: helpers.FromPgText(r.Contenido), Estado: helpers.FromPgText(r.Estado),
				FechaVencimiento: helpers.FromPgTimestamptzPtr(r.FechaVencimiento),
				RealizadaPor: helpers.FromPgInt4Ptr(r.RealizadaPor), RealizadaEn: r.RealizadaEn.Time,
				Metadata: helpers.FromJSONB(r.Metadata), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
			},
			RealizadaPorNombre: helpers.FromPgText(r.RealizadaPorNombre),
		}
	}
	return result
}
