package cmd

import (
	"fmt"
	"os"
	"strconv"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/adapters"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/features"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/ledger"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/trial"
	organizationsDomain "github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/paywall"
	platformFeatures "github.com/moasq/go-b2b-starter/internal/platform/features"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ProvideDependencies registers all billing module dependencies
func ProvideDependencies(container *dig.Container) error {
	// Use the services module for dependency injection
	servicesModule := services.NewModule()
	if err := servicesModule.Configure(container); err != nil {
		return fmt.Errorf("failed to configure billing services: %w", err)
	}

	// Register the platform FeatureProvider (entitlement derivation from
	// subscription plan + purchased modules). Consumed by CRM, tickets, etc.
	if err := container.Provide(func(
		subRepo domain.SubscriptionRepository,
		aiRepo domain.AiUsageRepository,
		store postgres.Store,
		log loggerDomain.Logger,
		moduleService registryServices.ModuleService,
	) platformFeatures.FeatureProvider {
		return features.NewBillingFeatureProvider(subRepo, aiRepo, store, log, moduleService)
	}); err != nil {
		return fmt.Errorf("failed to provide feature provider: %w", err)
	}

	// Register SubscriptionStatusProvider for the paywall middleware
	// This adapter bridges the billing module to the pkg/paywall middleware
	// Communication is event-driven: webhooks → billing → DB → paywall reads
	if err := container.Provide(func(svc services.BillingService) paywall.SubscriptionStatusProvider {
		return adapters.NewStatusProviderAdapter(svc)
	}); err != nil {
		return fmt.Errorf("failed to provide subscription status provider: %w", err)
	}

	// Register TokenLedger for AI usage metering (consumed by the metered LLM client)
	if err := container.Provide(func(
		aiRepo domain.AiUsageRepository,
		subRepo domain.SubscriptionRepository,
		orgAdapter domain.OrganizationAdapter,
		billingProvider domain.BillingProvider,
		log loggerDomain.Logger,
	) llmdomain.TokenLedger {
		return ledger.NewTokenLedger(aiRepo, subRepo, orgAdapter, billingProvider, log)
	}); err != nil {
		return fmt.Errorf("failed to provide token ledger: %w", err)
	}

	// Register TrialSeeder for organizations-domain interface.
	// Idempotent trial + quota seeding on bootstrap when TRIAL_ENABLED=true.
	if err := container.Provide(func(
		store postgres.Store,
		log loggerDomain.Logger,
	) organizationsDomain.TrialSeeder {
		cfg := trial.Config{
			Enabled: getEnvBool("TRIAL_ENABLED", false),
			Days:    getEnvInt("TRIAL_DAYS", 14),
		}
		return trial.NewSeeder(store, cfg, log)
	}); err != nil {
		return fmt.Errorf("failed to provide trial seeder: %w", err)
	}

	return nil
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}
