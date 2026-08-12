package procurement

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/infra/repositories"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Module wires the procurement dependencies into the DI container
// (domain → app → infrastructure, no Stytch SDK / transport in domain).
type Module struct {
	container *dig.Container
}

// NewModule creates the procurement module.
func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

// RegisterDependencies provides repositories, services, the subscriber, and
// the agent skip-check seam.
func (m *Module) RegisterDependencies() error {
	// ---- repositories (infra) ----
	if err := m.container.Provide(func(store sqlc.Store) domain.SupplierRepository {
		return repositories.NewSupplierRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.ProductRepository {
		return repositories.NewProductRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.InquiryRunRepository {
		return repositories.NewRunRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.OrderRepository {
		return repositories.NewOrderRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.AuditRepository {
		return repositories.NewAuditRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.ContactLookup {
		return repositories.NewContactLookup(store)
	}); err != nil {
		return err
	}
	// The store itself implements services.KillSwitchReader (GetAgentKillSwitch).
	if err := m.container.Provide(func(store sqlc.Store) services.KillSwitchReader {
		return store
	}); err != nil {
		return err
	}

	// ---- platform services ----
	if err := m.container.Provide(services.NewCounterSink); err != nil {
		return err
	}
	// The counter sink implements services.MetricsSink; bind the interface so
	// subscribers (ProcurementSubscriber, drafting/extraction/run) can resolve it.
	if err := m.container.Provide(func(sink *services.CounterSink) services.MetricsSink {
		return sink
	}); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewTokenBucketPacer); err != nil {
		return err
	}

	// ---- app services ----
	if err := m.container.Provide(services.NewDraftingService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewExtractionService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewRunService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewBoardService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewOrderService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewSendHandler); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewProcurementSubscriber); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewProcurementService); err != nil {
		return err
	}

	// ---- agent skip-check seam (task 10) ----
	if err := m.container.Provide(func(sub *services.ProcurementSubscriber) agent.ActiveInquiryChecker {
		return sub
	}); err != nil {
		return err
	}

	return nil
}
