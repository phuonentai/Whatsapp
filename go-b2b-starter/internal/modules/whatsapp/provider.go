package whatsapp

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/app/services"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(
		webhookService services.WebhookService,
		configService services.ConfigService,
		signupService services.SignupService,
	) *Handler {
		return NewHandler(webhookService, configService, signupService)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(NewRoutes); err != nil {
		return err
	}

	return nil
}
