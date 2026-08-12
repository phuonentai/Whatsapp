// Package cmd provides initialization for the auth module.
package cmd

import (
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/auth/adapters/stytch"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/redis"
	platformstytch "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"go.uber.org/dig"
)

//
// This sets up:
//   - stytch.Config
//   - auth.AuthProvider (Stytch adapter)
//
// Note: The auth middleware is NOT initialized here because it requires
// organization/account resolvers from the organizations module.
// Use InitMiddleware after the organizations module is initialized.
//
// # Prerequisites
//
// The following modules must be initialized first:
//   - redis (for caching)
//   - logger
//
// # Usage
//
//	// In main/cmd/init_mods.go:
//	if err := authCmd.Init(container); err != nil {
//	    panic(err)
//	}
func Init(container *dig.Container) error {
	// Stytch configuration
	if err := container.Provide(func() (*stytch.Config, error) {
		return stytch.LoadConfig()
	}); err != nil {
		return fmt.Errorf("failed to provide stytch config: %w", err)
	}

	// Stytch Auth Adapter (implements auth.AuthProvider)
	if err := container.Provide(func(
		cfg *stytch.Config,
		breakerClient *platformstytch.Client,
		redisClient redis.Client,
		log logger.Logger,
	) (auth.AuthProvider, error) {
		// Check for placeholder credentials
		if isPlaceholderCredentials(cfg) {
			log.Warn("Stytch credentials are placeholders - using development mode", map[string]any{
				"project_id": cfg.ProjectID,
				"message":    "Update STYTCH_PROJECT_ID and STYTCH_SECRET in app.env with real credentials",
			})
			return stytch.NewMockAuthAdapter(log), nil
		}

		adapter, err := stytch.NewStytchAuthAdapter(cfg, breakerClient, redisClient, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create stytch adapter: %w", err)
		}
		return adapter, nil
	}); err != nil {
		return fmt.Errorf("failed to provide auth provider: %w", err)
	}

	// RBAC Service backed by the Stytch policy (runtime SSOT): the service
	// served by `GET /api/rbac/roles` derives roles/permissions from the Stytch
	// RBAC policy, cached in Redis with a 5-minute TTL, and fetches through the
	// two-tier circuit breaker (`platform/stytch.Client.Run`).
	//
	// Same placeholder-detection pattern as the AuthProvider above: when the
	// breaker client is nil (placeholder credentials / development mode) the
	// static definitions fallback (`defaultRBACService`) is used and MUST NOT
	// be reachable in production (the platform client is always non-nil there).
	if err := container.Provide(func(
		breakerClient *platformstytch.Client,
		redisClient redis.Client,
		log logger.Logger,
	) auth.RBACService {
		if breakerClient == nil {
			log.Warn("Stytch client unavailable - using static RBAC fallback (development mode only)", map[string]any{
				"message": "GET /api/rbac/roles will serve static role definitions until real STYTCH_PROJECT_ID/STYTCH_SECRET are configured",
			})
			return auth.NewRBACService()
		}

		policyService := stytch.NewRBACPolicyServiceWithBreaker(
			breakerClient.API(),
			breakerClient,
			redisClient,
			log,
		)
		return stytch.NewStytchRBACService(policyService)
	}); err != nil {
		return fmt.Errorf("failed to provide rbac service: %w", err)
	}

	// Member directory service for inbox:reassign (conversation-row-scoping):
	// lista de miembros activos del org vía Stytch B2B Members.Search, con
	// circuit-breaker de dos niveles + cache Redis 5-min TTL. Devuelve solo
	// stytch_member_id. En development (breaker nil) devuelve nil — los
	// handlers de re-asignación responden 503 member_directory_unavailable
	// (degradación visible, bandeja funcional).
	if err := container.Provide(func(
		breakerClient *platformstytch.Client,
		redisClient redis.Client,
		log logger.Logger,
	) *stytch.MemberDirectoryService {
		if breakerClient == nil {
			log.Warn("Stytch client unavailable - member directory disabled (development mode only)", map[string]any{
				"message": "reassignment endpoints will respond 503 member_directory_unavailable",
			})
			return nil
		}
		return stytch.NewMemberDirectoryService(
			breakerClient.API(),
			breakerClient,
			redisClient,
			log,
		)
	}); err != nil {
		return fmt.Errorf("failed to provide member directory service: %w", err)
	}

	return nil
}
//
// This must be called after the organizations module is initialized,
// as it depends on organization and account repositories.
//
// # Prerequisites
//
// The following must be available in the container:
//   - auth.AuthProvider (from Init)
//   - auth.OrganizationResolver
//   - auth.AccountResolver
//   - serverDomain.Server (for registering named middlewares)
//
// # Usage
//
//	// After organizations module init:
//	if err := authCmd.InitMiddleware(container); err != nil {
//	    panic(err)
//	}
func InitMiddleware(container *dig.Container) error {
	if err := auth.SetupMiddleware(container); err != nil {
		return fmt.Errorf("failed to setup auth middleware: %w", err)
	}
	return nil
}

// isPlaceholderCredentials checks if the Stytch credentials are placeholder values.
func isPlaceholderCredentials(cfg *stytch.Config) bool {
	return strings.Contains(cfg.ProjectID, "REPLACE") ||
		strings.Contains(cfg.Secret, "REPLACE") ||
		cfg.ProjectID == "" ||
		cfg.Secret == ""
}
