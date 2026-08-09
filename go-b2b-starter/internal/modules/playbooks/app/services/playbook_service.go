package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	registryDomain "github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

// PlaybookService exposes the playbook catalog and per-org apply/reset.
type PlaybookService interface {
	// ListCatalog returns the tenant-visible catalog with each org's applied
	// state and the guiones of applied playbooks.
	ListCatalog(ctx context.Context, orgID int32) ([]*CatalogEntry, error)
	// Apply seeds the playbook for the org (one-way, idempotent).
	Apply(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error)
	// Reset removes playbook-seeded data only.
	Reset(ctx context.Context, orgID int32, key string) error
}

// CatalogEntry joins a playbook with the org's applied state and guiones.
type CatalogEntry struct {
	Playbook *domain.Playbook
	Applied  *domain.OrganizationPlaybook
	Guiones  []domain.Guion
}

type playbookService struct {
	playbookRepo  domain.PlaybookRepository
	orgPbRepo     domain.OrganizationPlaybookRepository
	applyRepo     domain.PlaybookApplyRepository
	moduleService registryServices.ModuleService
	logger        logger.Logger
}

func NewPlaybookService(
	playbookRepo domain.PlaybookRepository,
	orgPbRepo domain.OrganizationPlaybookRepository,
	applyRepo domain.PlaybookApplyRepository,
	moduleService registryServices.ModuleService,
	log logger.Logger,
) PlaybookService {
	return &playbookService{
		playbookRepo:  playbookRepo,
		orgPbRepo:     orgPbRepo,
		applyRepo:     applyRepo,
		moduleService: moduleService,
		logger:        log,
	}
}

func (s *playbookService) ListCatalog(ctx context.Context, orgID int32) ([]*CatalogEntry, error) {
	playbooks, err := s.playbookRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list playbook catalog: %w", err)
	}
	applied, err := s.orgPbRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org playbooks: %w", err)
	}
	appliedByKey := make(map[string]*domain.OrganizationPlaybook, len(applied))
	for _, ap := range applied {
		appliedByKey[ap.PlaybookKey] = ap
	}

	entries := make([]*CatalogEntry, 0, len(playbooks))
	for _, pb := range playbooks {
		entry := &CatalogEntry{Playbook: pb, Applied: appliedByKey[pb.Key]}
		if entry.Applied != nil {
			payload, err := ParsePayload(pb.Payload)
			if err == nil {
				entry.Guiones = payload.Guiones
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *playbookService) Apply(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error) {
	pb, err := s.playbookRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	payload, err := ParsePayload(pb.Payload)
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]bool)
	_, orgMods, err := s.moduleService.ListOrgModules(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org modules: %w", err)
	}
	for _, om := range orgMods {
		enabled[om.ModuleKey] = true
	}
	for _, req := range pb.RequiresModules {
		if !enabled[req] {
			return nil, fmt.Errorf("%w: %s", domain.ErrPlaybookRequiresModules, req)
		}
	}

	// Dry-run all presets before any mutation so invalid presets abort with
	// no partial persistence.
	for moduleKey, preset := range payload.ModuleConfigs {
		if err := s.moduleService.ValidateConfig(ctx, moduleKey, preset); err != nil {
			if err == registryDomain.ErrModuleNotFound {
				return nil, fmt.Errorf("%w: módulo %q no existe en el registro", domain.ErrInvalidModuleConfigPreset, moduleKey)
			}
			return nil, fmt.Errorf("%w: %v", domain.ErrInvalidModuleConfigPreset, err)
		}
	}

	state, err := s.applyRepo.Apply(ctx, orgID, pb, payload)
	if err != nil {
		return nil, fmt.Errorf("apply playbook %q: %w", key, err)
	}
	s.logger.Info("playbook applied", map[string]any{"org_id": orgID, "playbook": key})
	return state, nil
}

func (s *playbookService) Reset(ctx context.Context, orgID int32, key string) error {
	pb, err := s.playbookRepo.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	payload, err := ParsePayload(pb.Payload)
	if err != nil {
		return err
	}
	if err := s.applyRepo.Reset(ctx, orgID, pb, payload); err != nil {
		return fmt.Errorf("reset playbook %q: %w", key, err)
	}
	s.logger.Info("playbook reset", map[string]any{"org_id": orgID, "playbook": key})
	return nil
}

// ParsePayload decodes and structurally validates a playbook payload.
func ParsePayload(raw []byte) (*domain.PlaybookPayload, error) {
	var payload domain.PlaybookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: payload JSON inválido", domain.ErrInvalidPlaybookPayload)
	}
	if payload.Pipeline.Nombre == "" {
		return nil, fmt.Errorf("%w: el pipeline debe tener nombre", domain.ErrInvalidPlaybookPayload)
	}
	if len(payload.Pipeline.Etapas) == 0 {
		return nil, fmt.Errorf("%w: el pipeline debe tener al menos una etapa", domain.ErrInvalidPlaybookPayload)
	}
	seen := map[int32]bool{}
	for i, etapa := range payload.Pipeline.Etapas {
		if etapa.Nombre == "" {
			return nil, fmt.Errorf("%w: etapa %d sin nombre", domain.ErrInvalidPlaybookPayload, i)
		}
		if etapa.Orden < 1 {
			return nil, fmt.Errorf("%w: etapa %q con orden inválido", domain.ErrInvalidPlaybookPayload, etapa.Nombre)
		}
		if seen[etapa.Orden] {
			return nil, fmt.Errorf("%w: orden duplicado %d en etapas", domain.ErrInvalidPlaybookPayload, etapa.Orden)
		}
		seen[etapa.Orden] = true
	}
	for _, guion := range payload.Guiones {
		if guion.ID == "" || guion.Titulo == "" || guion.Mensaje == "" {
			return nil, fmt.Errorf("%w: guion incompleto (id/titulo/mensaje requeridos)", domain.ErrInvalidPlaybookPayload)
		}
	}
	return &payload, nil
}
