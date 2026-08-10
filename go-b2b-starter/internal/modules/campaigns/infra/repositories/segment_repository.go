package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

type segmentRepository struct {
	store sqlc.Store
}

func NewSegmentRepository(store sqlc.Store) domain.SegmentRepository {
	return &segmentRepository{store: store}
}

func (r *segmentRepository) Create(ctx context.Context, orgID int32, nombre string, spec []domain.Filter, createdBy string) (*domain.Segment, error) {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter spec: %w", err)
	}
	result, err := r.store.CreateSegment(ctx, sqlc.CreateSegmentParams{
		OrganizationID: orgID,
		Nombre:         nombre,
		FilterSpec:     specJSON,
		CreatedBy:      pgText(createdBy),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create segment: %w", err)
	}
	return mapSegment(&result), nil
}

func (r *segmentRepository) Update(ctx context.Context, orgID, id int32, nombre string, spec []domain.Filter) (*domain.Segment, error) {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter spec: %w", err)
	}
	result, err := r.store.UpdateSegment(ctx, sqlc.UpdateSegmentParams{
		ID:             id,
		OrganizationID: orgID,
		Nombre:         nombre,
		FilterSpec:     specJSON,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSegmentNotFound
		}
		return nil, fmt.Errorf("failed to update segment: %w", err)
	}
	return mapSegment(&result), nil
}

func (r *segmentRepository) Delete(ctx context.Context, orgID, id int32) error {
	return r.store.DeleteSegment(ctx, sqlc.DeleteSegmentParams{ID: id, OrganizationID: orgID})
}

func (r *segmentRepository) Get(ctx context.Context, orgID, id int32) (*domain.Segment, error) {
	result, err := r.store.GetSegment(ctx, sqlc.GetSegmentParams{ID: id, OrganizationID: orgID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSegmentNotFound
		}
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}
	return mapSegment(&result), nil
}

func (r *segmentRepository) List(ctx context.Context, orgID int32) ([]*domain.Segment, error) {
	results, err := r.store.ListSegments(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}
	segments := make([]*domain.Segment, len(results))
	for i := range results {
		segments[i] = mapSegment(&results[i])
	}
	return segments, nil
}

func mapSegment(s *sqlc.CrmSegment) *domain.Segment {
	var spec []domain.Filter
	if len(s.FilterSpec) > 0 {
		_ = json.Unmarshal(s.FilterSpec, &spec)
	}
	return &domain.Segment{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		Nombre:         s.Nombre,
		FilterSpec:     spec,
		CreatedBy:      s.CreatedBy.String,
		CreatedAt:      s.CreatedAt.Time,
		UpdatedAt:      s.UpdatedAt.Time,
	}
}
