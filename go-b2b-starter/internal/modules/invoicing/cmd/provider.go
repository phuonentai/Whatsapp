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
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/secrets"
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

	// Envelope encryption (SIIGO_MASTER_KEY). Used for per-org credential
	// storage; also satisfies the service CredentialCipher seam.
	if err := container.Provide(func(cfg *siigo.Config) (*secrets.Envelope, error) {
		env, err := secrets.NewEnvelope(cfg.MasterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to init secrets envelope: %w", err)
		}
		return env, nil
	}); err != nil {
		return fmt.Errorf("failed to provide secrets envelope: %w", err)
	}

	// Local invoice repository (system of record).
	if err := container.Provide(func(store sqlc.Store) domain.InvoiceRepository {
		return repositories.NewInvoiceRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide invoice repository: %w", err)
	}

	// Local org connection repository.
	if err := container.Provide(func(store sqlc.Store) domain.ConnectionRepository {
		return repositories.NewConnectionRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide connection repository: %w", err)
	}

	// Numeration snapshot + import-run repositories.
	if err := container.Provide(func(store sqlc.Store) domain.NumerationRepository {
		return repositories.NewNumerationRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide numeration repository: %w", err)
	}
	if err := container.Provide(func(store sqlc.Store) domain.ImportRunRepository {
		return repositories.NewImportRunRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide import run repository: %w", err)
	}

	// Per-org credential resolver (repository + envelope decrypt).
	if err := container.Provide(func(repo domain.ConnectionRepository, env *secrets.Envelope) domain.CredentialProvider {
		return siigo.NewCredentialResolver(repo, env)
	}); err != nil {
		return fmt.Errorf("failed to provide credential resolver: %w", err)
	}

	// Siigo adapter (named binding) + per-org resolver + router as the single
	// unnamed domain.InvoicingProvider. Mirrors billing ProviderRouter wiring.
	if err := container.Provide(func(cfg *siigo.Config, creds domain.CredentialProvider) domain.InvoicingProvider {
		return siigo.NewAdapter(cfg, nil, creds)
	}, dig.Name("siigo")); err != nil {
		return fmt.Errorf("failed to provide siigo adapter: %w", err)
	}

	// The adapter also verifies raw credential pairs against Siigo; expose it
	// through the ConnectionValidator seam used by the connection service.
	type validatorParams struct {
		dig.In
		SiigoAdapter domain.InvoicingProvider `name:"siigo"`
	}
	if err := container.Provide(func(p validatorParams) domain.ConnectionValidator {
		return p.SiigoAdapter.(domain.ConnectionValidator)
	}, dig.Name("siigo")); err != nil {
		return fmt.Errorf("failed to provide siigo connection validator: %w", err)
	}

	// The adapter also lists provider customers for onboarding imports; expose
	// it through the CustomerReader seam.
	if err := container.Provide(func(p validatorParams) domain.CustomerReader {
		return p.SiigoAdapter.(domain.CustomerReader)
	}, dig.Name("siigo")); err != nil {
		return fmt.Errorf("failed to provide siigo customer reader: %w", err)
	}

	// The adapter also reads provider numeration; expose it through the
	// NumerationReader seam.
	if err := container.Provide(func(p validatorParams) domain.NumerationReader {
		return p.SiigoAdapter.(domain.NumerationReader)
	}, dig.Name("siigo")); err != nil {
		return fmt.Errorf("failed to provide siigo numeration reader: %w", err)
	}

	// The secrets envelope satisfies the service CredentialCipher seam.
	if err := container.Provide(func(env *secrets.Envelope) services.CredentialCipher {
		return env
	}); err != nil {
		return fmt.Errorf("failed to provide credential cipher: %w", err)
	}

	// Provider resolver driven by the org connection state: only live
	// connections route to siigo; everything else routes to the explicit
	// "none" no-op provider.
	if err := container.Provide(func(repo domain.ConnectionRepository) routing.ProviderResolver {
		return routing.NewConnectionProviderResolver(repo)
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

	// Webhook handler + org-facing connection endpoints.
	if err := container.Provide(func() domain.WebhookVerifier {
		return siigo.NewVerifier()
	}); err != nil {
		return fmt.Errorf("failed to provide siigo webhook verifier: %w", err)
	}

	if err := container.Provide(func(
		svc services.InvoicingService,
		connSvc services.ConnectionService,
		numerationSvc services.NumerationService,
		importSvc services.ImportService,
		testInvoiceSvc services.TestInvoiceService,
		numerationRepo domain.NumerationRepository,
		importRunRepo domain.ImportRunRepository,
		webhookVerifier domain.WebhookVerifier,
		cfg *siigo.Config,
		log loggerDomain.Logger,
	) *invoicing.Handler {
		return invoicing.NewHandler(svc, connSvc, numerationSvc, importSvc, testInvoiceSvc, numerationRepo, importRunRepo, webhookVerifier, cfg.Sandbox, cfg.WebhookSecret, log)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing handler: %w", err)
	}

	return nil
}
