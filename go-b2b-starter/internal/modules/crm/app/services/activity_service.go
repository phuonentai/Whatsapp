package services

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type ActivityService interface {
	Create(ctx context.Context, orgID int32, req *CreateActivityRequest) (*domain.Activity, error)
	ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) (ListResult[*domain.ActivityWithActor], error)
	ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error)
	ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error)
	ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error)
}

type CreateActivityRequest struct {
	ContactID        *int32                 `json:"contact_id"`
	CompanyID        *int32                 `json:"company_id"`
	DealID           *int32                 `json:"deal_id"`
	ConversationID   *int32                 `json:"conversation_id"`
	Tipo             domain.ActivityType    `json:"tipo"`
	Asunto           string                 `json:"asunto"`
	Contenido        string                 `json:"contenido"`
	Estado           string                 `json:"estado"`
	FechaVencimiento *string                `json:"fecha_vencimiento"`
	RealizadaPor     *int32                 `json:"realizada_por"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type activityService struct {
	activityRepo    domain.ActivityRepository
	featureProvider features.FeatureProvider
}

func NewActivityService(activityRepo domain.ActivityRepository, featureProvider features.FeatureProvider) ActivityService {
	return &activityService{activityRepo: activityRepo, featureProvider: featureProvider}
}

func (s *activityService) Create(ctx context.Context, orgID int32, req *CreateActivityRequest) (*domain.Activity, error) {
	var fechaVencimiento *time.Time
	if req.FechaVencimiento != nil && *req.FechaVencimiento != "" {
		parsed, err := time.Parse(time.RFC3339, *req.FechaVencimiento)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", *req.FechaVencimiento)
			if err != nil {
				return nil, fmt.Errorf("fecha de vencimiento inválida: %w", err)
			}
		}
		fechaVencimiento = &parsed
	}
	a := &domain.Activity{
		OrganizationID: orgID, ContactID: req.ContactID, CompanyID: req.CompanyID,
		DealID: req.DealID, ConversationID: req.ConversationID, Tipo: req.Tipo,
		Asunto: req.Asunto, Contenido: req.Contenido, Estado: req.Estado,
		FechaVencimiento: fechaVencimiento,
		RealizadaPor:     req.RealizadaPor, Metadata: req.Metadata,
		RealizadaEn: time.Now(),
	}
	return s.activityRepo.Create(ctx, a)
}
func (s *activityService) ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) (ListResult[*domain.ActivityWithActor], error) {
	items, err := s.activityRepo.ListByOrganization(ctx, orgID, tipo, entityType, entityID, limit, offset)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	total, err := s.activityRepo.CountByOrganization(ctx, orgID, tipo, entityType, entityID)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	return ListResult[*domain.ActivityWithActor]{Items: items, Total: total}, nil
}
func (s *activityService) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error) {
	items, err := s.activityRepo.ListByContact(ctx, contactID, orgID, limit, offset)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	total, err := s.activityRepo.CountByContact(ctx, contactID, orgID)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	return ListResult[*domain.ActivityWithActor]{Items: items, Total: total}, nil
}
func (s *activityService) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error) {
	items, err := s.activityRepo.ListByDeal(ctx, dealID, orgID, limit, offset)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	total, err := s.activityRepo.CountByDeal(ctx, dealID, orgID)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	return ListResult[*domain.ActivityWithActor]{Items: items, Total: total}, nil
}
func (s *activityService) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) (ListResult[*domain.ActivityWithActor], error) {
	items, err := s.activityRepo.ListByCompany(ctx, companyID, orgID, limit, offset)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	total, err := s.activityRepo.CountByCompany(ctx, companyID, orgID)
	if err != nil {
		return ListResult[*domain.ActivityWithActor]{}, err
	}
	return ListResult[*domain.ActivityWithActor]{Items: items, Total: total}, nil
}
