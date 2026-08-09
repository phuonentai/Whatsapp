package cmd

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

const pollInterval = 5 * time.Minute

// Init registers invoicing dependencies, subscribes to deal stage changes, and
// starts the polling safety net for non-final invoices.
func Init(container *dig.Container) error {
	if err := ProvideDependencies(container); err != nil {
		return err
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		listener services.DealStageListener,
	) error {
		return bus.Subscribe(events.DealStageChangedEventType, func(ctx context.Context, event eventbus.Event) error {
			stageEvent, ok := event.(*events.DealStageChanged)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return listener.HandleStageChanged(ctx, stageEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to deal stage events: %w", err)
	}

	if err := container.Invoke(func(
		svc services.InvoicingService,
		log loggerDomain.Logger,
	) error {
		go startPoller(svc, log)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to start invoicing poller: %w", err)
	}

	return nil
}

// startPoller periodically reconciles pending invoices as a webhook fallback.
func startPoller(svc services.InvoicingService, log loggerDomain.Logger) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		reconciled, err := svc.PollPending(ctx)
		cancel()
		if err != nil {
			log.Warn("invoicing poll cycle failed", map[string]any{"error": err.Error()})
			continue
		}
		if reconciled > 0 {
			log.Info("invoicing poll reconciled invoices", map[string]any{"count": reconciled})
		}
	}
}
