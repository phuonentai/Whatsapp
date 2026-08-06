package whatsapp

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
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
		eventBus eventbus.EventBus,
		log logger.Logger,
	) services.WebhookService {
		return services.NewWebhookService(configRepo, logRepo, eventBus, log)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		configRepo domain.ConfigRepository,
	) services.ConfigService {
		return services.NewConfigService(configRepo)
	}); err != nil {
		return err
	}

	return nil
}
