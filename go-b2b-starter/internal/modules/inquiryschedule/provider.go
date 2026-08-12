package inquiryschedule

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
)

// Provider registers the inquiry-scheduling HTTP handler and routes.
type Provider struct {
	container *dig.Container
}

// NewProvider creates the inquiry-scheduling provider.
func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

// RegisterDependencies provides the handler and routes.
func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(service services.InquiryscheduleService) *Handler {
		return NewHandler(service)
	}); err != nil {
		return err
	}
	return p.container.Provide(NewRoutes)
}
