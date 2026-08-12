package tickets

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
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

	// AI triage: platform deps (metered LLM client + billing credits) injected
	// for this service only; TicketService is untouched.
	if err := p.container.Provide(func(
		llm llmdomain.LLMClient,
		billing billingServices.BillingService,
		repo domain.TicketRepository,
	) *ticketsServices.AITriageService {
		return ticketsServices.NewAITriageService(llm, billing, repo)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		ticketService ticketsServices.TicketService,
		aiTriageService *ticketsServices.AITriageService,
	) *Handler {
		return NewHandler(ticketService, aiTriageService)
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
