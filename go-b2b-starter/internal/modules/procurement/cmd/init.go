package cmd

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

// Init registers procurement dependencies, the outbox event codecs, the
// independent whatsapp.message.received subscriber (D7), and the outbox
// send handlers (D14).
func Init(container *dig.Container) error {
	module := procurement.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register procurement dependencies: %w", err)
	}

	provider := procurement.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register procurement provider: %w", err)
	}

	// Outbox codecs: the dispatcher decodes procurement events and republishes
	// them on the bus for our handlers.
	if err := container.Invoke(func(registry *outbox.Registry) error {
		for _, eventType := range []string{events.InquirySendEventType, events.OrderConfirmSendEventType} {
			codec, err := events.Codec(eventType)
			if err != nil {
				return err
			}
			registry.Register(eventType, codec)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to register procurement outbox codecs: %w", err)
	}

	// Inbound subscriber: independent of the CRM/agent listeners (D7).
	if err := container.Invoke(func(
		bus eventbus.EventBus,
		sub *services.ProcurementSubscriber,
	) error {
		return bus.Subscribe(whatsappEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*whatsappEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return sub.HandleMessageReceived(ctx, msgEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe procurement inbound listener: %w", err)
	}

	// Outbox send handlers (dispatch-time re-validation, D14).
	if err := container.Invoke(func(
		bus eventbus.EventBus,
		handler services.SendHandler,
	) error {
		if err := bus.Subscribe(events.InquirySendEventType, func(ctx context.Context, event eventbus.Event) error {
			e, ok := event.(*events.InquirySend)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return handler.HandleInquirySend(ctx, e)
		}); err != nil {
			return err
		}
		return bus.Subscribe(events.OrderConfirmSendEventType, func(ctx context.Context, event eventbus.Event) error {
			e, ok := event.(*events.OrderConfirmSend)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return handler.HandleOrderConfirmSend(ctx, e)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe procurement outbox handlers: %w", err)
	}

	return nil
}
