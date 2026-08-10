package services

import (
	"context"
	"fmt"
	"strings"

	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

type segmentService struct {
	repo      domain.SegmentRepository
	evaluator domain.SegmentEvaluator
	tagRepo   crmDomain.TagRepository
}

func NewSegmentService(repo domain.SegmentRepository, evaluator domain.SegmentEvaluator, tagRepo crmDomain.TagRepository) SegmentService {
	return &segmentService{repo: repo, evaluator: evaluator, tagRepo: tagRepo}
}

func (s *segmentService) Create(ctx context.Context, orgID int32, nombre string, spec []domain.Filter, createdBy string) (*domain.Segment, error) {
	if strings.TrimSpace(nombre) == "" {
		return nil, fmt.Errorf("%w: el nombre es obligatorio", domain.ErrInvalidFilterSpec)
	}
	if err := domain.ValidateFilterSpec(spec); err != nil {
		return nil, err
	}
	if err := verifyTagIDs(ctx, s.tagRepo, orgID, spec); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, orgID, strings.TrimSpace(nombre), spec, createdBy)
}

func (s *segmentService) Update(ctx context.Context, orgID, id int32, nombre string, spec []domain.Filter) (*domain.Segment, error) {
	if strings.TrimSpace(nombre) == "" {
		return nil, fmt.Errorf("%w: el nombre es obligatorio", domain.ErrInvalidFilterSpec)
	}
	if err := domain.ValidateFilterSpec(spec); err != nil {
		return nil, err
	}
	if err := verifyTagIDs(ctx, s.tagRepo, orgID, spec); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, orgID, id, strings.TrimSpace(nombre), spec)
}

func (s *segmentService) Delete(ctx context.Context, orgID, id int32) error {
	return s.repo.Delete(ctx, orgID, id)
}

func (s *segmentService) Get(ctx context.Context, orgID, id int32) (*domain.Segment, error) {
	return s.repo.Get(ctx, orgID, id)
}

func (s *segmentService) List(ctx context.Context, orgID int32) ([]*domain.Segment, error) {
	return s.repo.List(ctx, orgID)
}

func (s *segmentService) Preview(ctx context.Context, orgID int32, spec []domain.Filter) (*domain.EvalResult, error) {
	if err := domain.ValidateFilterSpec(spec); err != nil {
		return nil, err
	}
	return s.evaluator.Count(ctx, orgID, spec)
}
