package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type ActivityService interface {
	Create(ctx context.Context, orgID int32, req *CreateActivityRequest) (*domain.Activity, error)
	ListByOrganization(ctx context.Context, orgID int32, tipo string, limit, offset int32) ([]*domain.ActivityWithActor, error)
	ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error)
	ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error)
	ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error)
}

type CreateActivityRequest struct {
	ContactID        *int32
	CompanyID        *int32
	DealID           *int32
	ConversationID   *int32
	Tipo             domain.ActivityType
	Asunto           string
	Contenido        string
	Estado           string
	FechaVencimiento *string
	RealizadaPor     *int32
	Metadata         map[string]interface{}
}

type activityService struct {
	activityRepo    domain.ActivityRepository
	featureProvider features.FeatureProvider
}

func NewActivityService(activityRepo domain.ActivityRepository, featureProvider features.FeatureProvider) ActivityService {
	return &activityService{activityRepo: activityRepo, featureProvider: featureProvider}
}

func (s *activityService) Create(ctx context.Context, orgID int32, req *CreateActivityRequest) (*domain.Activity, error) {
	a := &domain.Activity{
		OrganizationID: orgID, ContactID: req.ContactID, CompanyID: req.CompanyID,
		DealID: req.DealID, ConversationID: req.ConversationID, Tipo: req.Tipo,
		Asunto: req.Asunto, Contenido: req.Contenido, Estado: req.Estado,
		RealizadaPor: req.RealizadaPor, Metadata: req.Metadata,
	}
	return s.activityRepo.Create(ctx, a)
}
func (s *activityService) ListByOrganization(ctx context.Context, orgID int32, tipo string, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return s.activityRepo.ListByOrganization(ctx, orgID, tipo, limit, offset)
}
func (s *activityService) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return s.activityRepo.ListByContact(ctx, contactID, orgID, limit, offset)
}
func (s *activityService) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return s.activityRepo.ListByDeal(ctx, dealID, orgID, limit, offset)
}
func (s *activityService) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return s.activityRepo.ListByCompany(ctx, companyID, orgID, limit, offset)
}
