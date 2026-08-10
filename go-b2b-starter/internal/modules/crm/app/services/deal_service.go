package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type DealService interface {
	Create(ctx context.Context, orgID int32, req *CreateDealRequest) (*domain.Deal, error)
	GetByID(ctx context.Context, orgID, dealID int32) (*domain.DealWithRefs, error)
	List(ctx context.Context, orgID int32, pipelineID, stageID int32, estado string, contactID, limit, offset int32) ([]*domain.DealWithRefs, error)
	Update(ctx context.Context, orgID int32, req *UpdateDealRequest) (*domain.Deal, error)
	UpdateStage(ctx context.Context, orgID, dealID, stageID, changedBy int32, oldStageName, newStageName string) (*domain.Deal, error)
	Delete(ctx context.Context, orgID, dealID int32) error
}

type CreateDealRequest struct {
	Nombre              string   `json:"nombre"`
	ContactID           *int32   `json:"contact_id"`
	CompanyID           *int32   `json:"company_id"`
	PipelineID          int32    `json:"pipeline_id"`
	StageID             *int32   `json:"stage_id"`
	Monto               *float64 `json:"monto"`
	Moneda              string   `json:"moneda"`
	FechaCierreEsperada *string  `json:"fecha_cierre_esperada"`
	Notas               string   `json:"notas"`
	AssignedTo          *int32   `json:"assigned_to"`
}
type UpdateDealRequest struct {
	ID                  int32    `json:"id"`
	OrganizationID      int32    `json:"organization_id"`
	Nombre              string   `json:"nombre"`
	ContactID           *int32   `json:"contact_id"`
	CompanyID           *int32   `json:"company_id"`
	Monto               *float64 `json:"monto"`
	Moneda              string   `json:"moneda"`
	FechaCierreEsperada *string  `json:"fecha_cierre_esperada"`
	Estado              string   `json:"estado"`
	Notas               string   `json:"notas"`
	AssignedTo          *int32   `json:"assigned_to"`
}

type dealService struct {
	dealRepo        domain.DealRepository
	pipelineRepo    domain.PipelineRepository
	activityRepo    domain.ActivityRepository
	featureProvider features.FeatureProvider
	eventBus        eventbus.EventBus
}

func NewDealService(
	dealRepo domain.DealRepository, pipelineRepo domain.PipelineRepository,
	activityRepo domain.ActivityRepository, featureProvider features.FeatureProvider,
	eventBus eventbus.EventBus,
) DealService {
	return &dealService{dealRepo: dealRepo, pipelineRepo: pipelineRepo, activityRepo: activityRepo, featureProvider: featureProvider, eventBus: eventBus}
}

func (s *dealService) Create(ctx context.Context, orgID int32, req *CreateDealRequest) (*domain.Deal, error) {
	deal := &domain.Deal{
		OrganizationID: orgID, Nombre: req.Nombre, ContactID: req.ContactID,
		CompanyID: req.CompanyID, PipelineID: req.PipelineID, StageID: req.StageID,
		Monto: req.Monto, Moneda: req.Moneda, Estado: domain.DealStatusAbierto,
		Notas: req.Notas, AssignedTo: req.AssignedTo,
	}
	if deal.Moneda == "" {
		deal.Moneda = "COP"
	}
	if err := deal.Validate(); err != nil {
		return nil, err
	}
	return s.dealRepo.Create(ctx, deal)
}
func (s *dealService) GetByID(ctx context.Context, orgID, dealID int32) (*domain.DealWithRefs, error) {
	return s.dealRepo.GetByID(ctx, orgID, dealID)
}
func (s *dealService) List(ctx context.Context, orgID int32, pipelineID, stageID int32, estado string, contactID, limit, offset int32) ([]*domain.DealWithRefs, error) {
	return s.dealRepo.List(ctx, orgID, pipelineID, stageID, estado, contactID, limit, offset)
}
func (s *dealService) Update(ctx context.Context, orgID int32, req *UpdateDealRequest) (*domain.Deal, error) {
	deal := &domain.Deal{
		ID: req.ID, OrganizationID: orgID, Nombre: req.Nombre,
		ContactID: req.ContactID, CompanyID: req.CompanyID, Monto: req.Monto,
		Moneda: req.Moneda, Estado: domain.DealStatus(req.Estado), Notas: req.Notas,
		AssignedTo: req.AssignedTo,
	}
	return s.dealRepo.Update(ctx, deal)
}
func (s *dealService) UpdateStage(ctx context.Context, orgID, dealID, stageID, changedBy int32, oldStageName, newStageName string) (*domain.Deal, error) {
	_, err := s.dealRepo.GetByID(ctx, orgID, dealID)
	if err != nil {
		return nil, err
	}
	updated, err := s.dealRepo.UpdateStage(ctx, orgID, dealID, stageID)
	if err != nil {
		return nil, err
	}
	if s.eventBus != nil {
		s.eventBus.Publish(ctx, &events.DealStageChanged{
			DealID: dealID, OrganizationID: orgID, NewStageID: stageID, ChangedBy: changedBy,
			OldStageName: oldStageName, NewStageName: newStageName,
		})
	}
	return updated, nil
}
func (s *dealService) Delete(ctx context.Context, orgID, dealID int32) error {
	return s.dealRepo.Delete(ctx, orgID, dealID)
}
