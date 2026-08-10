package cmd

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/crm"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	crmEvents "github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	igEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
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

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.EchoListener,
	) error {
		return bus.Subscribe(whatsappEvents.MessageEchoEventType, func(ctx context.Context, event eventbus.Event) error {
			echoEvent, ok := event.(*whatsappEvents.MessageEcho)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleMessageEcho(ctx, echoEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to whatsapp echo events: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.InstagramMessageListener,
	) error {
		return bus.Subscribe(igEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*igEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleMessageReceived(ctx, msgEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to instagram message events: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.InstagramEchoListener,
	) error {
		return bus.Subscribe(igEvents.MessageEchoEventType, func(ctx context.Context, event eventbus.Event) error {
			echoEvent, ok := event.(*igEvents.MessageEcho)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleMessageEcho(ctx, echoEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to instagram echo events: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener *services.ProfileBackfillListener,
	) error {
		return bus.Subscribe(igEvents.ProfileBackfillEventType, func(ctx context.Context, event eventbus.Event) error {
			return listener.Handle(ctx, event)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to instagram profile backfill events: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.DealStageListener,
	) error {
		return bus.Subscribe(crmEvents.DealStageChangedEventType, func(ctx context.Context, event eventbus.Event) error {
			stageEvent, ok := event.(*crmEvents.DealStageChanged)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleStageChanged(ctx, stageEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to deal stage events: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		handler *services.MessageSendHandler,
	) error {
		if err := bus.Subscribe(whatsappEvents.MessageSendEventType, func(ctx context.Context, event eventbus.Event) error {
			return handler.Handle(ctx, event)
		}); err != nil {
			return err
		}
		return bus.Subscribe(igEvents.MessageSendEventType, func(ctx context.Context, event eventbus.Event) error {
			return handler.Handle(ctx, event)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to outbound send events: %w", err)
	}

	return nil
}
