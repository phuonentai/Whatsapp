package instagram

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/repositories"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(store sqlc.Store) domain.ConfigRepository {
		return repositories.NewConfigRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		webhookService services.WebhookService,
		configService services.ConfigService,
		appID string,
		appSecret string,
	) *Handler {
		return NewHandler(webhookService, configService, appID, appSecret)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(NewRoutes); err != nil {
		return err
	}

	return nil
}
