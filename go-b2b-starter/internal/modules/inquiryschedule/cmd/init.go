// Package cmd registers inquiry-scheduling dependencies in the dig container,
// subscribes the outbox handlers and the reply-arrival trigger, and starts
// the schedule claim ticker and the follow-up sweep with the module.
package cmd

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/infra/scheduler"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// Init registers inquiry-scheduling dependencies, the outbox event codecs,
// the whatsapp.message.received reply-arrival trigger, the outbox handlers
// (inquiry_run.scheduled, inquiry.followup_send), and starts the two loops.
func Init(container *dig.Container) error {
	module := inquiryschedule.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register inquiryschedule dependencies: %w", err)
	}

	provider := inquiryschedule.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register inquiryschedule provider: %w", err)
	}

	// Outbox codecs: the dispatcher decodes inquiry-scheduling events and
	// republishes them on the bus for our handlers.
	if err := container.Invoke(func(registry *outbox.Registry) error {
		for _, eventType := range []string{events.InquiryRunScheduledEventType, events.FollowupSendEventType} {
			codec, err := events.Codec(eventType)
			if err != nil {
				return err
			}
			registry.Register(eventType, codec)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to register inquiryschedule outbox codecs: %w", err)
	}

	// Reply-arrival trigger: a whatsapp.message.received event marks a
	// recipient answered; run the cheap overdue check for the run's other
	// recipients (the answering recipients are excluded conservatively).
	if err := container.Invoke(func(
		bus eventbus.EventBus,
		svc *services.FollowUpService,
	) error {
		return bus.Subscribe(whatsappEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*whatsappEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			phone, err := whatsapp.CanonicalizeE164(msgEvent.From)
			if err != nil {
				return nil // not a canonical phone: nothing to check
			}
			return svc.OnReplyArrival(ctx, msgEvent.OrganizationID, phone)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe inquiryschedule reply trigger: %w", err)
	}

	// Outbox handlers.
	if err := container.Invoke(func(
		bus eventbus.EventBus,
		handler services.ScheduledRunHandler,
	) error {
		return bus.Subscribe(events.InquiryRunScheduledEventType, func(ctx context.Context, event eventbus.Event) error {
			e, ok := event.(*events.InquiryRunScheduled)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return handler.HandleScheduledRun(ctx, e)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe inquiry_run.scheduled handler: %w", err)
	}
	if err := container.Invoke(func(
		bus eventbus.EventBus,
		handler services.FollowUpSendHandler,
	) error {
		return bus.Subscribe(events.FollowupSendEventType, func(ctx context.Context, event eventbus.Event) error {
			e, ok := event.(*events.FollowupSend)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return handler.HandleFollowUpSend(ctx, e)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe inquiry.followup_send handler: %w", err)
	}

	// Loops: claim ticker (30–60s) + follow-up sweep (≤15 min). Both stop on
	// process exit; failures are logged and never crash the loops.
	if err := container.Invoke(func(
		ticker *scheduler.ScheduleTicker,
		sweep *scheduler.FollowUpSweeper,
		log logger.Logger,
	) error {
		go ticker.Run(context.Background())
		go sweep.Run(context.Background())
		log.Info("inquiryschedule loops started", map[string]any{
			"claim_interval": (45 * time.Second).String(),
			"sweep_interval": (15 * time.Minute).String(),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to start inquiryschedule loops: %w", err)
	}

	return nil
}
