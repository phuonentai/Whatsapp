package instagram

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

type Module struct {
	container *dig.Container
}

func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

func (m *Module) RegisterDependencies() error {
	if err := m.container.Provide(func(
		configRepo domain.ConfigRepository,
		logRepo domain.WebhookLogRepository,
		outboxRepo outbox.Repository,
		log logger.Logger,
	) services.WebhookService {
		return services.NewWebhookService(
			configRepo,
			logRepo,
			outboxRepo,
			graphapi.WebhookVerifyToken(),
			log,
		)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		configRepo domain.ConfigRepository,
		client graphapi.IGClient,
	) services.ConfigService {
		return services.NewConfigService(configRepo, client)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		cfg graphapi.ClientConfig,
	) string {
		return cfg.AppID
	}, dig.Name("instagram_app_id")); err != nil {
		return err
	}
	if err := m.container.Provide(func(
		cfg graphapi.ClientConfig,
	) string {
		return cfg.AppSecret
	}, dig.Name("instagram_app_secret")); err != nil {
		return err
	}

	return nil
}
