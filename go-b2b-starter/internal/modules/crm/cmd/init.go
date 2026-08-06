package cmd

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/crm"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

func Init(container *dig.Container) error {
	module := crm.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register crm dependencies: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.MessageListener,
	) error {
		return bus.Subscribe(whatsappEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*whatsappEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleMessageReceived(ctx, msgEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to whatsapp events: %w", err)
	}

	return nil
}
