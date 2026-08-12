package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

// ModuleService exposes the module catalog and per-org module state/config.
type ModuleService interface {
	// ListCatalog returns the tenant-visible catalog (excludes internal modules).
	ListCatalog(ctx context.Context) ([]*domain.Module, error)
	// ListCatalogInternal returns the full catalog including internal modules.
	ListCatalogInternal(ctx context.Context) ([]*domain.Module, error)
	// ListOrgModules returns modules enabled for the org, with configs.
	ListOrgModules(ctx context.Context, orgID int32) ([]*domain.Module, []*domain.OrganizationModule, error)
	// SaveOrgModuleConfig validates and persists the org's module config.
	SaveOrgModuleConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*domain.OrganizationModule, error)
	// ValidateConfig dry-runs config validation against the module's schema
	// without persisting anything. Used by playbook apply for pre-validation.
	ValidateConfig(ctx context.Context, moduleKey string, config map[string]any) error
	// SyncModulesFromMetadata reconciles org module state with subscription
	// metadata module keys (e.g., "module_tickets"). Disabled keys are removed.
	SyncModulesFromMetadata(ctx context.Context, orgID int32, moduleKeys []string) error
	// ResolveGrantedFeatures returns the feature keys granted by the given enabled
	// module keys, respecting module dependencies and registry membership.
	ResolveGrantedFeatures(ctx context.Context, enabledKeys []string) map[string]bool
}

type moduleService struct {
	moduleRepo domain.ModuleRepository
	orgModRepo domain.OrganizationModuleRepository
	logger     logger.Logger
}

func NewModuleService(
	moduleRepo domain.ModuleRepository,
	orgModRepo domain.OrganizationModuleRepository,
	log logger.Logger,
) ModuleService {
	return &moduleService{
		moduleRepo: moduleRepo,
		orgModRepo: orgModRepo,
		logger:     log,
	}
}

func (s *moduleService) ListCatalog(ctx context.Context) ([]*domain.Module, error) {
	modules, err := s.moduleRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list catalog: %w", err)
	}
	result := make([]*domain.Module, 0, len(modules))
	for _, m := range modules {
		if !m.IsInternal {
			result = append(result, m)
		}
	}
	return result, nil
}

func (s *moduleService) ListCatalogInternal(ctx context.Context) ([]*domain.Module, error) {
	return s.moduleRepo.ListActive(ctx)
}

func (s *moduleService) ListOrgModules(ctx context.Context, orgID int32) ([]*domain.Module, []*domain.OrganizationModule, error) {
	orgMods, err := s.orgModRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, nil, fmt.Errorf("list org modules: %w", err)
	}
	modules := make([]*domain.Module, 0, len(orgMods))
	for _, om := range orgMods {
		m, err := s.moduleRepo.GetByKey(ctx, om.ModuleKey)
		if err != nil {
			if err == domain.ErrModuleNotFound {
				continue
			}
			return nil, nil, fmt.Errorf("resolve org module %q: %w", om.ModuleKey, err)
		}
		modules = append(modules, m)
	}
	return modules, orgMods, nil
}

func (s *moduleService) ValidateConfig(ctx context.Context, moduleKey string, config map[string]any) error {
	module, err := s.moduleRepo.GetByKey(ctx, moduleKey)
	if err != nil {
		return fmt.Errorf("%w: %s", domain.ErrModuleNotFound, moduleKey)
	}
	if !module.IsActive {
		return fmt.Errorf("%w: %s", domain.ErrModuleDisabled, moduleKey)
	}
	return validateModuleConfig(moduleKey, config)
}

func (s *moduleService) SaveOrgModuleConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*domain.OrganizationModule, error) {
	module, err := s.moduleRepo.GetByKey(ctx, moduleKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrModuleNotFound, moduleKey)
	}
	if !module.IsActive {
		return nil, fmt.Errorf("%w: %s", domain.ErrModuleDisabled, moduleKey)
	}
	if err := validateModuleConfig(moduleKey, config); err != nil {
		return nil, err
	}
	return s.orgModRepo.UpsertConfig(ctx, orgID, moduleKey, config)
}

func (s *moduleService) SyncModulesFromMetadata(ctx context.Context, orgID int32, moduleKeys []string) error {
	// Absent/empty metadata key sets express no module change: reconciling
	// against an empty list would disable every org module. Treat as a no-op
	// (defense in depth; the webhook path also stops passing empty keys).
	if len(moduleKeys) == 0 {
		return nil
	}

	enabled := make(map[string]bool, len(moduleKeys))
	for _, k := range moduleKeys {
		enabled[k] = true
	}

	modules, err := s.moduleRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("sync modules from metadata: %w", err)
	}
	byKey := make(map[string]*domain.Module, len(modules))
	for _, m := range modules {
		byKey[m.Key] = m
	}

	for key := range enabled {
		module, ok := byKey[key]
		if !ok {
			s.logger.Warn("module key in metadata not in registry, ignoring", map[string]any{"module_key": key})
			continue
		}
		// Resolve dependencies: a module is only enabled when its dependencies are enabled.
		depsSatisfied := true
		for _, dep := range module.Requires {
			if !enabled[dep] {
				depsSatisfied = false
				break
			}
		}
		if !depsSatisfied {
			s.logger.Warn("module dependencies not satisfied, skipping", map[string]any{"module_key": key})
			continue
		}
		// Preserve existing config when re-syncing (upsert keeps previous config otherwise).
		existing, err := s.orgModRepo.GetByKey(ctx, orgID, key)
		if err != nil {
			return fmt.Errorf("get org module %q: %w", key, err)
		}
		config := map[string]any{}
		if existing != nil {
			config = existing.Config
		}
		if _, err := s.orgModRepo.UpsertConfig(ctx, orgID, key, config); err != nil {
			return fmt.Errorf("enable module %q for org %d: %w", key, orgID, err)
		}
	}

	// Disable modules that are no longer present in metadata.
	existing, err := s.orgModRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list org modules for sync: %w", err)
	}
	for _, om := range existing {
		if !enabled[om.ModuleKey] {
			if err := s.orgModRepo.Delete(ctx, orgID, om.ModuleKey); err != nil {
				return fmt.Errorf("disable module %q for org %d: %w", om.ModuleKey, orgID, err)
			}
		}
	}
	return nil
}

func (s *moduleService) ResolveGrantedFeatures(ctx context.Context, enabledKeys []string) map[string]bool {
	result := make(map[string]bool)
	modules, err := s.moduleRepo.ListActive(ctx)
	if err != nil {
		return result
	}
	enabled := make(map[string]bool, len(enabledKeys))
	for _, k := range enabledKeys {
		enabled[k] = true
	}
	byKey := make(map[string]*domain.Module, len(modules))
	for _, m := range modules {
		byKey[m.Key] = m
	}
	for _, key := range enabledKeys {
		module, ok := byKey[key]
		if !ok {
			continue
		}
		depsSatisfied := true
		for _, dep := range module.Requires {
			if !enabled[dep] {
				depsSatisfied = false
				break
			}
		}
		if !depsSatisfied {
			continue
		}
		for _, feature := range module.GrantedFeatures {
			result[feature] = true
		}
	}
	return result
}

// validateModuleConfig performs a hand-rolled schema check for the tickets
// module config keys (sla_hours, priorities, tags). Unknown modules accept
// any JSON object (validated structurally only).
func validateModuleConfig(moduleKey string, config map[string]any) error {
	if config == nil {
		return nil
	}
	switch moduleKey {
	case "tickets":
		if raw, ok := config["sla_hours"]; ok {
			sla, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: sla_hours debe ser un objeto prioridad->horas", domain.ErrInvalidModuleConfig)
			}
			for priority, hours := range sla {
				switch hours.(type) {
				case float64, int, int64:
				default:
					return fmt.Errorf("%w: sla_hours[%s] debe ser un número", domain.ErrInvalidModuleConfig, priority)
				}
			}
		}
		for _, key := range []string{"priorities", "tags"} {
			if raw, ok := config[key]; ok {
				arr, ok := raw.([]any)
				if !ok {
					return fmt.Errorf("%w: %s debe ser una lista", domain.ErrInvalidModuleConfig, key)
				}
				for i, item := range arr {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("%w: %s[%d] debe ser texto", domain.ErrInvalidModuleConfig, key, i)
					}
				}
			}
		}
	}
	return nil
}
