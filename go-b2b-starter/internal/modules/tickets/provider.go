package tickets

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(store sqlc.Store) domain.TicketRepository {
		return repositories.NewTicketRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		repo domain.TicketRepository,
		moduleService registryServices.ModuleService,
	) ticketsServices.TicketService {
		return ticketsServices.NewTicketService(repo, moduleService)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		ticketService ticketsServices.TicketService,
	) *Handler {
		return NewHandler(ticketService)
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
