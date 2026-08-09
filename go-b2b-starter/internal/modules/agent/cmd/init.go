package cmd

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

// Init registers agent dependencies and subscribes the agent pipeline to
// inbound WhatsApp message events.
func Init(container *dig.Container) error {
	module := agent.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register agent dependencies: %w", err)
	}

	provider := agent.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register agent provider: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		agentService services.AgentService,
	) error {
		return bus.Subscribe(whatsappEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*whatsappEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return agentService.HandleMessageReceived(ctx, msgEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to whatsapp events: %w", err)
	}

	return nil
}
