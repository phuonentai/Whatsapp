package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/redis"
	platformstytch "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"go.uber.org/dig"
)

// noopRedis implements redis.Client without storage (enough for wiring tests).
type noopRedis struct{}

func (noopRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}
func (noopRedis) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (noopRedis) Delete(ctx context.Context, key string) error         { return nil }
func (noopRedis) Exists(ctx context.Context, key string) (bool, error) { return false, nil }
func (noopRedis) Ping(ctx context.Context) error                       { return nil }

// TestInitRBACServicePlaceholderFallback verifies the placeholder-detection
// wiring: with a nil Stytch breaker client (development mode) the DI container
// resolves the static fallback RBACService — and never crashes on a missing
// Stytch implementation.
func TestInitRBACServicePlaceholderFallback(t *testing.T) {
	t.Setenv("STYTCH_PROJECT_ID", "REPLACE_ME")
	t.Setenv("STYTCH_SECRET", "REPLACE_ME")

	container := dig.New()
	if err := container.Provide(func() logger.Logger {
		return logger.New(logger.WithLevel(logger.FatalLevel))
	}); err != nil {
		t.Fatal(err)
	}
	if err := container.Provide(func() redis.Client { return noopRedis{} }); err != nil {
		t.Fatal(err)
	}
	// Development mode: the platform provider yields a nil breaker client.
	if err := container.Provide(func() *platformstytch.Client { return nil }); err != nil {
		t.Fatal(err)
	}

	if err := Init(container); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	var svc auth.RBACService
	if err := container.Invoke(func(s auth.RBACService) { svc = s }); err != nil {
		t.Fatalf("failed to resolve RBACService: %v", err)
	}

	// Static fallback must serve the development definitions.
	if roles := svc.GetAllRoles(); len(roles) == 0 {
		t.Fatal("expected static fallback roles in development mode, got none")
	}
}
