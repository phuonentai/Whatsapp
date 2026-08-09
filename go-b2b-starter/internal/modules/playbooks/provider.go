package playbooks

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	playbooksServices "github.com/moasq/go-b2b-starter/internal/modules/playbooks/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/infra/repositories"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
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
	if err := p.container.Provide(func(store sqlc.Store) *repositories.PlaybookRepository {
		return repositories.NewPlaybookRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(r *repositories.PlaybookRepository) domain.PlaybookRepository {
		return r
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(r *repositories.PlaybookRepository) domain.OrganizationPlaybookRepository {
		return r
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(r *repositories.PlaybookRepository) domain.PlaybookApplyRepository {
		return r
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		playbookRepo domain.PlaybookRepository,
		orgPbRepo domain.OrganizationPlaybookRepository,
		applyRepo domain.PlaybookApplyRepository,
		moduleService registryServices.ModuleService,
		log logger.Logger,
	) playbooksServices.PlaybookService {
		return playbooksServices.NewPlaybookService(playbookRepo, orgPbRepo, applyRepo, moduleService, log)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		playbookService playbooksServices.PlaybookService,
	) *Handler {
		return NewHandler(playbookService)
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
