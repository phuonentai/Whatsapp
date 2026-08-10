package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type pipelineRepository struct{ store sqlc.Store }
type stageRepository struct{ store sqlc.Store }

func NewPipelineRepository(store sqlc.Store) domain.PipelineRepository {
	return &pipelineRepository{store: store}
}

func NewPipelineStageRepository(store sqlc.Store) domain.PipelineStageRepository {
	return &stageRepository{store: store}
}

func (r *pipelineRepository) Create(ctx context.Context, p *domain.Pipeline) (*domain.Pipeline, error) {
	result, err := r.store.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID: p.OrganizationID, Nombre: p.Nombre,
		EsPredeterminado: p.EsPredeterminado, Orden: p.Orden,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}
	return mapPipeline(&result), nil
}

func (r *pipelineRepository) GetByID(ctx context.Context, orgID, pipelineID int32) (*domain.Pipeline, error) {
	result, err := r.store.GetPipelineByID(ctx, sqlc.GetPipelineByIDParams{ID: pipelineID, OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}
	return mapPipeline(&result), nil
}

func (r *pipelineRepository) List(ctx context.Context, orgID int32) ([]*domain.PipelineWithStages, error) {
	results, err := r.store.ListPipelinesByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	pipelines := make([]*domain.PipelineWithStages, len(results))
	for i := range results {
		pipelines[i] = mapPipelineRow(&results[i])
	}
	return pipelines, nil
}

func (r *pipelineRepository) GetDefault(ctx context.Context, orgID int32) (*domain.Pipeline, error) {
	result, err := r.store.GetDefaultPipelineByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default pipeline: %w", err)
	}
	return mapPipeline(&result), nil
}

func (r *pipelineRepository) Update(ctx context.Context, p *domain.Pipeline) (*domain.Pipeline, error) {
	result, err := r.store.UpdatePipeline(ctx, sqlc.UpdatePipelineParams{
		ID: p.ID, OrganizationID: p.OrganizationID,
		Column3:          helpers.ToPgText(p.Nombre),
		EsPredeterminado: p.EsPredeterminado, Orden: p.Orden,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}
	return mapPipeline(&result), nil
}

func (r *pipelineRepository) Delete(ctx context.Context, orgID, pipelineID int32) error {
	return r.store.DeletePipeline(ctx, sqlc.DeletePipelineParams{ID: pipelineID, OrganizationID: orgID})
}

func mapPipeline(c *sqlc.CrmPipeline) *domain.Pipeline {
	return &domain.Pipeline{
		ID: c.ID, OrganizationID: c.OrganizationID, Nombre: c.Nombre,
		EsPredeterminado: c.EsPredeterminado, Orden: c.Orden,
		CreatedAt: c.CreatedAt.Time, UpdatedAt: c.UpdatedAt.Time,
	}
}

func mapPipelineRow(r *sqlc.ListPipelinesByOrganizationRow) *domain.PipelineWithStages {
	p := &domain.PipelineWithStages{
		Pipeline: domain.Pipeline{
			ID: r.ID, OrganizationID: r.OrganizationID, Nombre: r.Nombre,
			EsPredeterminado: r.EsPredeterminado, Orden: r.Orden,
			CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
		},
		Etapas: parseStagesJSON(r.Etapas),
	}
	return p
}

func parseStagesJSON(data interface{}) []domain.PipelineStage {
	if data == nil {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var stages []domain.PipelineStage
	if err := json.Unmarshal(raw, &stages); err != nil {
		return nil
	}
	return stages
}

// PipelineStageRepository

func (r *stageRepository) Create(ctx context.Context, s *domain.PipelineStage) (*domain.PipelineStage, error) {
	result, err := r.store.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: s.PipelineID, Nombre: s.Nombre, Orden: s.Orden,
		Color: helpers.ToPgText(s.Color), Probabilidad: helpers.ToPgInt4Ptr(s.Probabilidad),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}
	return mapStage(&result), nil
}

func (r *stageRepository) ListByPipeline(ctx context.Context, pipelineID int32) ([]*domain.PipelineStage, error) {
	results, err := r.store.ListStagesByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages: %w", err)
	}
	stages := make([]*domain.PipelineStage, len(results))
	for i := range results {
		stages[i] = mapStage(&results[i])
	}
	return stages, nil
}

func (r *stageRepository) GetByID(ctx context.Context, stageID int32) (*domain.PipelineStage, error) {
	result, err := r.store.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage: %w", err)
	}
	return mapStage(&result), nil
}

func (r *stageRepository) Update(ctx context.Context, s *domain.PipelineStage) (*domain.PipelineStage, error) {
	result, err := r.store.UpdatePipelineStage(ctx, sqlc.UpdatePipelineStageParams{
		ID: s.ID, PipelineID: s.PipelineID,
		Column3: helpers.ToPgText(s.Nombre), Orden: s.Orden,
		Column5: helpers.ToPgText(s.Color), Probabilidad: helpers.ToPgInt4Ptr(s.Probabilidad),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}
	return mapStage(&result), nil
}

func (r *stageRepository) Delete(ctx context.Context, stageID, pipelineID int32) error {
	return r.store.DeletePipelineStage(ctx, sqlc.DeletePipelineStageParams{ID: stageID, PipelineID: pipelineID})
}

func mapStage(c *sqlc.CrmPipelineStage) *domain.PipelineStage {
	return &domain.PipelineStage{
		ID: c.ID, PipelineID: c.PipelineID, Nombre: c.Nombre,
		Orden: c.Orden, Color: helpers.FromPgText(c.Color),
		Probabilidad: helpers.FromPgInt4Ptr(c.Probabilidad),
		CreatedAt:    c.CreatedAt.Time, UpdatedAt: c.UpdatedAt.Time,
	}
}
