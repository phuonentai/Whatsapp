package instagram

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram/app/services"
)

// handlerParams collects the handler dependencies, including the named
// Instagram app credentials provided by the module (not the unnamed strings
// used by other modules).
type handlerParams struct {
	dig.In

	WebhookService services.WebhookService
	ConfigService  services.ConfigService
	AppID          string `name:"instagram_app_id"`
	AppSecret      string `name:"instagram_app_secret"`
}

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	// domain.ConfigRepository / domain.WebhookLogRepository are registered by
	// the db module (registerDomainStores); do not re-provide them here.

	if err := p.container.Provide(func(hp handlerParams) *Handler {
		return NewHandler(hp.WebhookService, hp.ConfigService, hp.AppID, hp.AppSecret)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(NewRoutes); err != nil {
		return err
	}

	return nil
}
