package repositories

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type dealRepository struct{ store sqlc.Store }

func NewDealRepository(store sqlc.Store) domain.DealRepository {
	return &dealRepository{store: store}
}

func (r *dealRepository) Create(ctx context.Context, deal *domain.Deal) (*domain.Deal, error) {
	result, err := r.store.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: deal.OrganizationID, Nombre: deal.Nombre,
		ContactID: helpers.ToPgInt4Ptr(deal.ContactID), CompanyID: helpers.ToPgInt4Ptr(deal.CompanyID),
		PipelineID: deal.PipelineID, StageID: helpers.ToPgInt4Ptr(deal.StageID),
		Monto: helpers.ToPgNumeric(deal.Monto), Moneda: deal.Moneda,
		FechaCierreEsperada: helpers.ToPgDate(deal.FechaCierreEsperada),
		Estado: string(deal.Estado), Probabilidad: helpers.ToPgInt4Ptr(deal.Probabilidad),
		Notas: helpers.ToPgText(deal.Notas), Metadata: helpers.ToJSONB(deal.Metadata),
		AssignedTo: helpers.ToPgInt4Ptr(deal.AssignedTo),
	})
	if err != nil { return nil, fmt.Errorf("failed to create deal: %w", err) }
	return mapDealCrm(&result), nil
}

func (r *dealRepository) GetByID(ctx context.Context, orgID, dealID int32) (*domain.DealWithRefs, error) {
	result, err := r.store.GetDealByID(ctx, sqlc.GetDealByIDParams{ID: dealID, OrganizationID: orgID})
	if err != nil { return nil, fmt.Errorf("failed to get deal: %w", err) }
	return &domain.DealWithRefs{Deal: *mapDealGetRow(&result), ContactName: helpers.FromPgText(result.ContactName),
		ContactPhone: helpers.FromPgText(result.ContactPhone), CompanyName: helpers.FromPgText(result.CompanyName)}, nil
}

func (r *dealRepository) List(ctx context.Context, orgID int32, pipelineID, stageID int32, status string, contactID, limit, offset int32) ([]*domain.DealWithRefs, error) {
	results, err := r.store.ListDealsByOrganization(ctx, sqlc.ListDealsByOrganizationParams{
		OrganizationID: orgID, Column2: pipelineID, Column3: stageID,
		Column4: helpers.ToPgText(status), Column5: contactID, Limit: limit, Offset: offset,
	})
	if err != nil { return nil, fmt.Errorf("failed to list deals: %w", err) }
	deals := make([]*domain.DealWithRefs, len(results))
	for i := range results {
		deals[i] = &domain.DealWithRefs{Deal: *mapDealListRow(&results[i]),
			ContactName: helpers.FromPgText(results[i].ContactName),
			ContactPhone: helpers.FromPgText(results[i].ContactPhone),
			CompanyName: helpers.FromPgText(results[i].CompanyName)}
	}
	return deals, nil
}

func (r *dealRepository) Update(ctx context.Context, deal *domain.Deal) (*domain.Deal, error) {
	result, err := r.store.UpdateDeal(ctx, sqlc.UpdateDealParams{
		ID: deal.ID, OrganizationID: deal.OrganizationID,
		Column3: helpers.ToPgText(deal.Nombre),
		ContactID: helpers.ToPgInt4Ptr(deal.ContactID), CompanyID: helpers.ToPgInt4Ptr(deal.CompanyID),
		Monto: helpers.ToPgNumeric(deal.Monto),
		Column7: helpers.ToPgText(deal.Moneda),
		FechaCierreEsperada: helpers.ToPgDate(deal.FechaCierreEsperada),
		Column9: helpers.ToPgText(string(deal.Estado)),
		Probabilidad: helpers.ToPgInt4Ptr(deal.Probabilidad),
		Column11: helpers.ToPgText(deal.Notas),
		Column12: helpers.ToJSONB(deal.Metadata),
		AssignedTo: helpers.ToPgInt4Ptr(deal.AssignedTo),
	})
	if err != nil { return nil, fmt.Errorf("failed to update deal: %w", err) }
	return mapDealCrm(&result), nil
}

func (r *dealRepository) UpdateStage(ctx context.Context, orgID, dealID, stageID int32) (*domain.Deal, error) {
	result, err := r.store.UpdateDealStage(ctx, sqlc.UpdateDealStageParams{
		ID: dealID, OrganizationID: orgID, StageID: helpers.ToPgInt4(stageID),
	})
	if err != nil { return nil, fmt.Errorf("failed to update deal stage: %w", err) }
	return mapDealCrm(&result), nil
}

func (r *dealRepository) Delete(ctx context.Context, orgID, dealID int32) error {
	return r.store.DeleteDeal(ctx, sqlc.DeleteDealParams{ID: dealID, OrganizationID: orgID})
}

func mapDealCrm(c *sqlc.CrmDeal) *domain.Deal {
	return &domain.Deal{
		ID: c.ID, OrganizationID: c.OrganizationID, Nombre: c.Nombre,
		ContactID: helpers.FromPgInt4Ptr(c.ContactID), CompanyID: helpers.FromPgInt4Ptr(c.CompanyID),
		PipelineID: c.PipelineID, StageID: helpers.FromPgInt4Ptr(c.StageID),
		Monto: helpers.FromPgNumeric(c.Monto), Moneda: c.Moneda,
		FechaCierreEsperada: helpers.FromPgDate(c.FechaCierreEsperada), Estado: domain.DealStatus(c.Estado),
		Probabilidad: helpers.FromPgInt4Ptr(c.Probabilidad), Notas: helpers.FromPgText(c.Notas),
		Metadata: helpers.FromJSONB(c.Metadata), AssignedTo: helpers.FromPgInt4Ptr(c.AssignedTo),
		CreatedAt: c.CreatedAt.Time, UpdatedAt: c.UpdatedAt.Time,
	}
}

func mapDealGetRow(r *sqlc.GetDealByIDRow) *domain.Deal {
	return &domain.Deal{
		ID: r.ID, OrganizationID: r.OrganizationID, Nombre: r.Nombre,
		ContactID: helpers.FromPgInt4Ptr(r.ContactID), CompanyID: helpers.FromPgInt4Ptr(r.CompanyID),
		PipelineID: r.PipelineID, StageID: helpers.FromPgInt4Ptr(r.StageID),
		Monto: helpers.FromPgNumeric(r.Monto), Moneda: r.Moneda,
		FechaCierreEsperada: helpers.FromPgDate(r.FechaCierreEsperada), Estado: domain.DealStatus(r.Estado),
		Probabilidad: helpers.FromPgInt4Ptr(r.Probabilidad), Notas: helpers.FromPgText(r.Notas),
		Metadata: helpers.FromJSONB(r.Metadata), AssignedTo: helpers.FromPgInt4Ptr(r.AssignedTo),
		CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}
}

func mapDealListRow(r *sqlc.ListDealsByOrganizationRow) *domain.Deal {
	return &domain.Deal{
		ID: r.ID, OrganizationID: r.OrganizationID, Nombre: r.Nombre,
		ContactID: helpers.FromPgInt4Ptr(r.ContactID), CompanyID: helpers.FromPgInt4Ptr(r.CompanyID),
		PipelineID: r.PipelineID, StageID: helpers.FromPgInt4Ptr(r.StageID),
		Monto: helpers.FromPgNumeric(r.Monto), Moneda: r.Moneda,
		FechaCierreEsperada: helpers.FromPgDate(r.FechaCierreEsperada), Estado: domain.DealStatus(r.Estado),
		Probabilidad: helpers.FromPgInt4Ptr(r.Probabilidad), Notas: helpers.FromPgText(r.Notas),
		Metadata: helpers.FromJSONB(r.Metadata), AssignedTo: helpers.FromPgInt4Ptr(r.AssignedTo),
		CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}
}
