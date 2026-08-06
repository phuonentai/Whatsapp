package repositories

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type tagRepository struct {
	store sqlc.Store
}

func NewTagRepository(store sqlc.Store) domain.TagRepository {
	return &tagRepository{store: store}
}

func (r *tagRepository) Create(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	result, err := r.store.CreateTag(ctx, sqlc.CreateTagParams{
		OrganizationID: tag.OrganizationID,
		Nombre:         tag.Nombre,
		Color:          helpers.ToPgText(tag.Color),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *tagRepository) GetByID(ctx context.Context, orgID, tagID int32) (*domain.Tag, error) {
	result, err := r.store.GetTagByID(ctx, sqlc.GetTagByIDParams{ID: tagID, OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *tagRepository) List(ctx context.Context, orgID int32) ([]*domain.Tag, error) {
	results, err := r.store.ListTagsByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	tags := make([]*domain.Tag, len(results))
	for i := range results {
		tags[i] = r.mapToDomain(&results[i])
	}
	return tags, nil
}

func (r *tagRepository) Update(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	result, err := r.store.UpdateTag(ctx, sqlc.UpdateTagParams{
		ID: tag.ID, OrganizationID: tag.OrganizationID,
		Column3: helpers.ToPgText(tag.Nombre), Column4: helpers.ToPgText(tag.Color),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *tagRepository) Delete(ctx context.Context, orgID, tagID int32) error {
	return r.store.DeleteTag(ctx, sqlc.DeleteTagParams{ID: tagID, OrganizationID: orgID})
}

func (r *tagRepository) mapToDomain(c *sqlc.CrmTag) *domain.Tag {
	return &domain.Tag{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Nombre:         c.Nombre,
		Color:          helpers.FromPgText(c.Color),
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
}

type entityTagRepository struct {
	store sqlc.Store
}

func NewEntityTagRepository(store sqlc.Store) domain.EntityTagRepository {
	return &entityTagRepository{store: store}
}

func (r *entityTagRepository) Attach(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) (*domain.EntityTag, error) {
	result, err := r.store.AttachTag(ctx, sqlc.AttachTagParams{
		TagID:      tagID,
		EntityType: string(entityType),
		EntityID:   entityID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach tag: %w", err)
	}
	return &domain.EntityTag{
		ID:         result.ID,
		TagID:      result.TagID,
		EntityType: domain.EntityType(result.EntityType),
		EntityID:   result.EntityID,
		CreatedAt:  result.CreatedAt.Time,
	}, nil
}

func (r *entityTagRepository) Detach(ctx context.Context, tagID int32, entityType domain.EntityType, entityID int32) error {
	return r.store.DetachTag(ctx, sqlc.DetachTagParams{
		TagID:      tagID,
		EntityType: string(entityType),
		EntityID:   entityID,
	})
}

func (r *entityTagRepository) ListByEntity(ctx context.Context, entityType domain.EntityType, entityID int32) ([]*domain.Tag, error) {
	results, err := r.store.ListTagsByEntity(ctx, sqlc.ListTagsByEntityParams{
		EntityType: string(entityType),
		EntityID:   entityID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags by entity: %w", err)
	}
	tags := make([]*domain.Tag, len(results))
	for i := range results {
		tags[i] = &domain.Tag{
			ID:     results[i].ID,
			Nombre: results[i].Nombre,
			Color:  helpers.FromPgText(results[i].Color),
		}
	}
	return tags, nil
}

func (r *entityTagRepository) ListByTag(ctx context.Context, tagID int32) ([]*domain.EntityTag, error) {
	results, err := r.store.ListEntitiesByTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by tag: %w", err)
	}
	entities := make([]*domain.EntityTag, len(results))
	for i := range results {
		entities[i] = &domain.EntityTag{
			TagID:      tagID,
			EntityType: domain.EntityType(results[i].EntityType),
			EntityID:   results[i].EntityID,
		}
	}
	return entities, nil
}
