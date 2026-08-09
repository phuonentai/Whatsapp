package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type TagService interface {
	Create(ctx context.Context, orgID int32, nombre, color string) (*domain.Tag, error)
	List(ctx context.Context, orgID int32) ([]*domain.Tag, error)
	Update(ctx context.Context, orgID, tagID int32, nombre, color string) (*domain.Tag, error)
	Delete(ctx context.Context, orgID, tagID int32) error
	AttachToEntity(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) (*domain.EntityTag, error)
	DetachFromEntity(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) error
	ListByEntity(ctx context.Context, entityType domain.EntityType, entityID int32) ([]*domain.Tag, error)
}

type tagService struct {
	tagRepo         domain.TagRepository
	entityTagRepo   domain.EntityTagRepository
	featureProvider features.FeatureProvider
}

func NewTagService(tagRepo domain.TagRepository, entityTagRepo domain.EntityTagRepository, featureProvider features.FeatureProvider) TagService {
	return &tagService{tagRepo: tagRepo, entityTagRepo: entityTagRepo, featureProvider: featureProvider}
}

func (s *tagService) Create(ctx context.Context, orgID int32, nombre, color string) (*domain.Tag, error) {
	return s.tagRepo.Create(ctx, &domain.Tag{OrganizationID: orgID, Nombre: nombre, Color: color})
}
func (s *tagService) List(ctx context.Context, orgID int32) ([]*domain.Tag, error) { return s.tagRepo.List(ctx, orgID) }
func (s *tagService) Update(ctx context.Context, orgID, tagID int32, nombre, color string) (*domain.Tag, error) {
	updated, err := s.tagRepo.Update(ctx, &domain.Tag{ID: tagID, OrganizationID: orgID, Nombre: nombre, Color: color})
	if err != nil {
		if isUniqueViolationOn(err, "tags_organization_id_nombre_key") {
			return nil, domain.ErrTagDuplicateName
		}
		return nil, err
	}
	return updated, nil
}
func (s *tagService) Delete(ctx context.Context, orgID, tagID int32) error { return s.tagRepo.Delete(ctx, orgID, tagID) }
func (s *tagService) AttachToEntity(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) (*domain.EntityTag, error) {
	return s.entityTagRepo.Attach(ctx, tagID, entityType, entityID)
}
func (s *tagService) DetachFromEntity(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) error {
	return s.entityTagRepo.Detach(ctx, tagID, entityType, entityID)
}
func (s *tagService) ListByEntity(ctx context.Context, entityType domain.EntityType, entityID int32) ([]*domain.Tag, error) {
	return s.entityTagRepo.ListByEntity(ctx, entityType, entityID)
}
