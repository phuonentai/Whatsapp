package cmd

import (
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

// Init registers outbox dependencies in the DI container.
func Init(container *dig.Container) error {
	if err := container.Provide(outbox.LoadConfig); err != nil {
		return fmt.Errorf("failed to provide outbox config: %w", err)
	}
	if err := container.Provide(func(store sqlc.Store) outbox.Repository {
		return outbox.NewSQLCRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide outbox repository: %w", err)
	}
	if err := container.Provide(outbox.NewRegistry); err != nil {
		return fmt.Errorf("failed to provide outbox registry: %w", err)
	}
	if err := container.Provide(func(
		repo outbox.Repository,
		bus eventbus.EventBus,
		registry *outbox.Registry,
		log domain.Logger,
		cfg outbox.Config,
	) *outbox.Dispatcher {
		return outbox.NewDispatcher(repo, bus, registry, log, cfg)
	}); err != nil {
		return fmt.Errorf("failed to provide outbox dispatcher: %w", err)
	}
	return nil
}
