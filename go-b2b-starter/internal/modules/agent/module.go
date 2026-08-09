package agent

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/infra/guardrails"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/infra/repositories"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Module wires the agent module dependencies into the DI container.
type Module struct {
	container *dig.Container
}

// NewModule creates the agent module.
func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

// RegisterDependencies provides guardrails, the agent pipeline, and the
// compliance service.
func (m *Module) RegisterDependencies() error {
	if err := m.container.Provide(func(store sqlc.Store) domain.AgentRepository {
		return repositories.NewAgentRepository(store)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(guardrails.NewGuardrailService); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		repo domain.AgentRepository,
		guardrailsService domain.GuardrailService,
		llmClient llmdomain.LLMClient,
		billing billingServices.BillingService,
		outbound crmServices.OutboundService,
		log logger.Logger,
	) services.AgentService {
		return services.NewAgentService(repo, guardrailsService, llmClient, billing, outbound, log)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		repo domain.AgentRepository,
	) services.ComplianceService {
		return services.NewComplianceService(repo)
	}); err != nil {
		return err
	}

	return nil
}
