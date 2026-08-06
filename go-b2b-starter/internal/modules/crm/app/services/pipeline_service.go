package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type PipelineService interface {
	Create(ctx context.Context, orgID int32, req *CreatePipelineRequest) (*domain.Pipeline, error)
	GetByID(ctx context.Context, orgID, pipelineID int32) (*domain.Pipeline, error)
	List(ctx context.Context, orgID int32) ([]*domain.PipelineWithStages, error)
	Update(ctx context.Context, orgID int32, req *UpdatePipelineRequest) (*domain.Pipeline, error)
	Delete(ctx context.Context, orgID, pipelineID int32) error
	GetOrCreateDefault(ctx context.Context, orgID int32) (*domain.PipelineWithStages, error)
	CreateStage(ctx context.Context, pipelineID int32, req *CreateStageRequest) (*domain.PipelineStage, error)
	UpdateStage(ctx context.Context, stageID, pipelineID int32, req *UpdateStageRequest) (*domain.PipelineStage, error)
}

type CreatePipelineRequest struct {
	Nombre string
	Orden  int32
}
type UpdatePipelineRequest struct {
	ID     int32
	Nombre string
	Orden  int32
}
type CreateStageRequest struct {
	Nombre       string
	Orden        int32
	Color        string
	Probabilidad *int32
}
type UpdateStageRequest struct {
	Nombre       string
	Orden        int32
	Color        string
	Probabilidad *int32
}

type pipelineService struct {
	pipelineRepo    domain.PipelineRepository
	stageRepo       domain.PipelineStageRepository
	featureProvider features.FeatureProvider
}

func NewPipelineService(
	pipelineRepo domain.PipelineRepository,
	stageRepo domain.PipelineStageRepository,
	featureProvider features.FeatureProvider,
) PipelineService {
	return &pipelineService{pipelineRepo: pipelineRepo, stageRepo: stageRepo, featureProvider: featureProvider}
}

func (s *pipelineService) GetOrCreateDefault(ctx context.Context, orgID int32) (*domain.PipelineWithStages, error) {
	pipelines, err := s.pipelineRepo.List(ctx, orgID)
	if err != nil { return nil, err }
	if len(pipelines) > 0 { return pipelines[0], nil }

	p, err := s.pipelineRepo.Create(ctx, &domain.Pipeline{
		OrganizationID: orgID, Nombre: "Pipeline de Ventas", EsPredeterminado: true, Orden: 1,
	})
	if err != nil { return nil, err }

	defaultStages := []struct {
		Nombre string; Orden int32; Color string; Probabilidad int32
	}{
		{"Prospección", 1, "#6B7280", 10},
		{"Calificado", 2, "#3B82F6", 25},
		{"Propuesta", 3, "#8B5CF6", 50},
		{"Negociación", 4, "#F59E0B", 75},
		{"Cerrado Ganado", 5, "#10B981", 100},
		{"Cerrado Perdido", 6, "#EF4444", 0},
	}
	etapas := make([]domain.PipelineStage, 0, len(defaultStages))
	for _, st := range defaultStages {
		prob := st.Probabilidad
		stage, err := s.stageRepo.Create(ctx, &domain.PipelineStage{
			PipelineID: p.ID, Nombre: st.Nombre, Orden: st.Orden,
			Color: st.Color, Probabilidad: &prob,
		})
		if err != nil { continue }
		etapas = append(etapas, *stage)
	}
	return &domain.PipelineWithStages{Pipeline: *p, Etapas: etapas}, nil
}

func (s *pipelineService) Create(ctx context.Context, orgID int32, req *CreatePipelineRequest) (*domain.Pipeline, error) {
	return s.pipelineRepo.Create(ctx, &domain.Pipeline{OrganizationID: orgID, Nombre: req.Nombre, Orden: req.Orden})
}
func (s *pipelineService) GetByID(ctx context.Context, orgID, pipelineID int32) (*domain.Pipeline, error) {
	return s.pipelineRepo.GetByID(ctx, orgID, pipelineID)
}
func (s *pipelineService) List(ctx context.Context, orgID int32) ([]*domain.PipelineWithStages, error) {
	return s.pipelineRepo.List(ctx, orgID)
}
func (s *pipelineService) Update(ctx context.Context, orgID int32, req *UpdatePipelineRequest) (*domain.Pipeline, error) {
	return s.pipelineRepo.Update(ctx, &domain.Pipeline{ID: req.ID, OrganizationID: orgID, Nombre: req.Nombre, Orden: req.Orden})
}
func (s *pipelineService) Delete(ctx context.Context, orgID, pipelineID int32) error {
	return s.pipelineRepo.Delete(ctx, orgID, pipelineID)
}
func (s *pipelineService) CreateStage(ctx context.Context, pipelineID int32, req *CreateStageRequest) (*domain.PipelineStage, error) {
	return s.stageRepo.Create(ctx, &domain.PipelineStage{PipelineID: pipelineID, Nombre: req.Nombre, Orden: req.Orden, Color: req.Color, Probabilidad: req.Probabilidad})
}
func (s *pipelineService) UpdateStage(ctx context.Context, stageID, pipelineID int32, req *UpdateStageRequest) (*domain.PipelineStage, error) {
	return s.stageRepo.Update(ctx, &domain.PipelineStage{ID: stageID, PipelineID: pipelineID, Nombre: req.Nombre, Orden: req.Orden, Color: req.Color, Probabilidad: req.Probabilidad})
}
