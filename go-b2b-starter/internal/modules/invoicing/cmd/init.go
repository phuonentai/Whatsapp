// Package cmd registers invoicing module dependencies in the dig container.
package cmd

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

const (
	pollInterval  = 5 * time.Minute
	deltaSyncHour = 2 // nightly customer delta sync at 02:00 local
)

// Init registers invoicing dependencies, subscribes to deal stage changes,
// and starts the polling safety net for non-final invoices plus the nightly
// customer delta sync.
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

	if err := container.Invoke(func(
		importSvc services.ImportService,
		connRepo domain.ConnectionRepository,
		log loggerDomain.Logger,
	) error {
		go startNightlyDeltaSync(importSvc, connRepo, log)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to start nightly delta sync: %w", err)
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

// startNightlyDeltaSync re-pulls provider customers for live organizations
// once per day at deltaSyncHour. Failures are logged and never crash the loop.
func startNightlyDeltaSync(importSvc services.ImportService, connRepo domain.ConnectionRepository, log loggerDomain.Logger) {
	waitUntilNextHour(deltaSyncHour)
	for {
		runDeltaSyncOnce(importSvc, connRepo, log)
		time.Sleep(24 * time.Hour)
	}
}

func runDeltaSyncOnce(importSvc services.ImportService, connRepo domain.ConnectionRepository, log loggerDomain.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	connections, err := connRepo.ListByStatus(ctx, "siigo", domain.ConnStatusLive)
	if err != nil {
		log.Warn("nightly delta sync failed to list live connections", map[string]any{"error": err.Error()})
		return
	}
	for _, conn := range connections {
		if _, err := importSvc.DeltaSync(ctx, conn.OrganizationID); err != nil {
			log.Warn("nightly delta sync failed for organization", map[string]any{
				"organization_id": conn.OrganizationID,
				"error":           err.Error(),
			})
		}
	}
}

// waitUntilNextHour blocks until the next occurrence of the given local hour.
func waitUntilNextHour(hour int) {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	time.Sleep(time.Until(next))
}
