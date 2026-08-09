package agent

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
)

// Provider registers the agent HTTP handler and routes.
type Provider struct {
	container *dig.Container
}

// NewProvider creates the agent provider.
func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

// RegisterDependencies provides the handler and routes.
func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(
		agentService services.AgentService,
		compliance services.ComplianceService,
	) *Handler {
		return NewHandler(agentService, compliance)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(NewRoutes); err != nil {
		return err
	}

	return nil
}
