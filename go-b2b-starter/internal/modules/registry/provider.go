package registry

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/registry/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(store sqlc.Store) domain.ModuleRepository {
		return repositories.NewModuleRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(store sqlc.Store) domain.OrganizationModuleRepository {
		return repositories.NewOrgModuleRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		moduleRepo domain.ModuleRepository,
		orgModRepo domain.OrganizationModuleRepository,
		log logger.Logger,
	) registryServices.ModuleService {
		return registryServices.NewModuleService(moduleRepo, orgModRepo, log)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		moduleService registryServices.ModuleService,
	) *Handler {
		return NewHandler(moduleService)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		handler *Handler,
		featureProvider features.FeatureProvider,
	) *Routes {
		return NewRoutes(handler, featureProvider)
	}); err != nil {
		return err
	}

	return nil
}
