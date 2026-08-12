package stytch

import (
	"fmt"

	"go.uber.org/dig"
)

// ProvideDependencies registers Stytch package dependencies in the DI container.
//
// The Stytch `Client` (with the shared two-tier circuit breaker) is the only
// Stytch dependency provided here; the RBAC policy service is consolidated in
// `internal/modules/auth/adapters/stytch` (cache key `auth:stytch:rbac:policy:v3`)
// and wired by `auth/cmd.Init` — no duplicate policy service is provided.
func ProvideDependencies(container *dig.Container) error {
	// Provide Stytch client
	if err := container.Provide(func(cfg Config) (*Client, error) {
		return NewClient(cfg)
	}); err != nil {
		return fmt.Errorf("failed to provide stytch client: %w", err)
	}

	return nil
}
