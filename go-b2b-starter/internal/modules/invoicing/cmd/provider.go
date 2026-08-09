// Package cmd registers invoicing module dependencies in the dig container.
package cmd

import (
	"fmt"

	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/routing"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/siigo"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ProvideDependencies registers all invoicing module dependencies.
func ProvideDependencies(container *dig.Container) error {
	// Siigo config (env-split, sandbox default). Required like MP/Polar configs.
	if err := container.Provide(func() (*siigo.Config, error) {
		config, err := siigo.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load siigo configuration: %w", err)
		}
		return &config, nil
	}); err != nil {
		return fmt.Errorf("failed to provide siigo config: %w", err)
	}

	// Local invoice repository (system of record).
	if err := container.Provide(func(store sqlc.Store) domain.InvoiceRepository {
		return repositories.NewInvoiceRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide invoice repository: %w", err)
	}

	// Siigo adapter (named binding) + per-org resolver + router as the single
	// unnamed domain.InvoicingProvider. Mirrors billing ProviderRouter wiring.
	if err := container.Provide(func(cfg *siigo.Config) domain.InvoicingProvider {
		return siigo.NewAdapter(cfg, nil)
	}, dig.Name("siigo")); err != nil {
		return fmt.Errorf("failed to provide siigo adapter: %w", err)
	}

	if err := container.Provide(func() routing.ProviderResolver {
		return routing.NewStaticResolver("siigo")
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing provider resolver: %w", err)
	}

	type routerParams struct {
		dig.In
		SiigoAdapter domain.InvoicingProvider `name:"siigo"`
		Resolver     routing.ProviderResolver
	}

	if err := container.Provide(func(p routerParams) domain.InvoicingProvider {
		return routing.NewInvoiceRouter(p.SiigoAdapter, p.Resolver)
	}); err != nil {
		return fmt.Errorf("failed to provide invoice router: %w", err)
	}

	// Application services (DI via the services module).
	servicesModule := services.NewModule()
	if err := servicesModule.Configure(container); err != nil {
		return fmt.Errorf("failed to configure invoicing services: %w", err)
	}

	// Webhook handler.
	if err := container.Provide(func(svc services.InvoicingService, cfg *siigo.Config, log loggerDomain.Logger) *invoicing.Handler {
		return invoicing.NewHandler(svc, cfg, log)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing handler: %w", err)
	}

	return nil
}
