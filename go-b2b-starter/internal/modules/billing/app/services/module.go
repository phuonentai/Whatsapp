package services

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/db/adapters"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/mercadopago"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/polar"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/routing"
	payments "github.com/moasq/go-b2b-starter/internal/modules/payments/app/services"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	mercadopagopkg "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	polarpkg "github.com/moasq/go-b2b-starter/internal/platform/polar"
)

// Module handles dependency injection for billing services
// Note: SubscriptionRepository is registered in internal/db/inject.go
type Module struct{}

func NewModule() *Module {
	return &Module{}
}

// providerRouterParams collects the named adapter bindings plus shared
// dependencies for constructing the ProviderRouter.
type providerRouterParams struct {
	dig.In

	PolarAdapter domain.BillingProvider `name:"polar"`
	// MercadoPago is optional: Polar-only deployments skip config/client
	// registration, so the router receives nil and degrades to Polar-only.
	MPAdapter  domain.BillingProvider        `name:"mercadopago" optional:"true"`
	Resolver   routing.BillingProviderResolver
	OrgAdapter domain.OrganizationAdapter
}

// billingServiceParams collects the dependencies for constructing the
// BillingService, including the named MercadoPago adapter used by the
// MP-specific verify flow.
type billingServiceParams struct {
	dig.In

	Repo               domain.SubscriptionRepository
	AiRepo             domain.AiUsageRepository
	OrgAdapter         domain.OrganizationAdapter
	BillingProvider    domain.BillingProvider
	// MPProvider is optional: when MercadoPago is unconfigured the service
	// methods return a clear "mercadopago not configured" error.
	MPProvider  domain.BillingProvider        `name:"mercadopago" optional:"true"`
	Resolver    routing.BillingProviderResolver
	ModuleService      registryServices.ModuleService
	Logger             logger.Logger
	PaymentEventHandler payments.PaymentEventHandler `optional:"true"`
}

// Configure registers all services in the dependency container
func (m *Module) Configure(container *dig.Container) error {
	// Register OrganizationAdapter (uses legacy adapter store for now)
	if err := container.Provide(func(orgStore adapters.OrganizationStore) domain.OrganizationAdapter {
		return repositories.NewOrganizationAdapter(orgStore)
	}); err != nil {
		return err
	}

	// Register PolarAdapter (named so the router can consume it)
	if err := container.Provide(func(client *polarpkg.Client, log logger.Logger) domain.BillingProvider {
		return polar.NewPolarAdapter(client, log)
	}, dig.Name("polar")); err != nil {
		return err
	}

	// Register MPAdapter (named so the router and MP-specific services can consume it)
	if err := container.Provide(func(client *mercadopagopkg.Client, cfg *mercadopagopkg.Config, log logger.Logger) domain.BillingProvider {
		return mercadopago.NewMPAdapter(client, log, cfg)
	}, dig.Name("mercadopago")); err != nil {
		return err
	}

	// Register BillingProviderResolver (reads organizations.billing_provider)
	if err := container.Provide(func(store sqlc.Store) routing.BillingProviderResolver {
		return repositories.NewBillingProviderResolver(store)
	}); err != nil {
		return err
	}

	// Register ProviderRouter as the single unnamed domain.BillingProvider
	// binding, delegating per-organization to PolarAdapter or MPAdapter
	if err := container.Provide(func(p providerRouterParams) domain.BillingProvider {
		return routing.NewProviderRouter(p.PolarAdapter, p.MPAdapter, p.Resolver, p.OrgAdapter)
	}); err != nil {
		return err
	}

	// Register BillingService
	if err := container.Provide(func(p billingServiceParams) BillingService {
		return NewBillingService(p.Repo, p.AiRepo, p.OrgAdapter, p.BillingProvider, p.MPProvider, p.Resolver, p.ModuleService, p.Logger, p.PaymentEventHandler)
	}); err != nil {
		return err
	}

	return nil
}
