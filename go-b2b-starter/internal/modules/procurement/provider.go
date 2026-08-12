package procurement

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
)

// Provider registers the procurement HTTP handler and routes.
type Provider struct {
	container *dig.Container
}

// NewProvider creates the procurement provider.
func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

// RegisterDependencies provides the handler and routes.
func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(service services.ProcurementService) *Handler {
		return NewHandler(service)
	}); err != nil {
		return err
	}
	return p.container.Provide(NewRoutes)
}
