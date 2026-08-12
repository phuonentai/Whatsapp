// Package inquiryschedule wires the inquiry-scheduling module into the DI
// container (domain → app → infrastructure, no Stytch SDK / transport in
// domain). Consumes the sibling supplier-inquiries capability by reference
// through the drafting seam (procurement DraftingService) and the shared
// per-org send pacer.
package inquiryschedule

import (
	"context"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/infra/postgres"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/infra/scheduler"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/infra/send"
	procurementDomain "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	procurementServices "github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// Module wires the inquiry-scheduling dependencies into the DI container.
type Module struct {
	container *dig.Container
}

// NewModule creates the inquiry-scheduling module.
func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

// RegisterDependencies provides repositories, services, the outbox handlers,
// the follow-up service, and the sender.
func (m *Module) RegisterDependencies() error {
	// ---- clock ----
	if err := m.container.Provide(postgres.NewSystemClock); err != nil {
		return err
	}

	// ---- drafting seam: adapts the sibling's metered DraftingService ----
	if err := m.container.Provide(func(drafting procurementServices.DraftingService) domain.DraftFunc {
		return draftAdapter(drafting)
	}); err != nil {
		return err
	}

	// ---- repositories (infra) ----
	if err := m.container.Provide(postgres.NewRepositories); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.ScheduleRepository { return r.Schedules }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.FollowUpSettingsRepository { return r.FollowUpSettings }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.CatalogReader { return r.Catalog }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.KillSwitchReader { return r.KillSwitch }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.OrgTimezoneReader { return r.Timezone }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.RecipientStateReader { return r.Recipients }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.NudgeIncrementer { return r.Nudges }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.FollowUpEnqueuer { return r.Enqueuer }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.FollowUpEnabledOrgLister { return r.Orgs }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.AuditLogWriter { return r.Audit }); err != nil {
		return err
	}
	if err := m.container.Provide(func(r *postgres.Repositories) domain.InquiryRunCreator { return r.Creator }); err != nil {
		return err
	}

	// ---- sender: shares the procurement per-org pacer ----
	if err := m.container.Provide(func(pacer procurementServices.Pacer) send.RateLimiter {
		return pacer
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(configs whatsappDomain.ConfigRepository, limiter send.RateLimiter) domain.FollowUpSender {
		return send.NewWhatsAppSender(configs, limiter)
	}); err != nil {
		return err
	}

	// ---- app services ----
	if err := m.container.Provide(services.NewScheduleService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewScheduledRunHandler); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewFollowUpService); err != nil {
		return err
	}
	if err := m.container.Provide(services.NewFollowUpSendHandler); err != nil {
		return err
	}

	// ---- ticker + sweep (started in cmd/init.go) ----
	if err := m.container.Provide(func(repo domain.ScheduleRepository, clock domain.Clock, log logger.Logger) *scheduler.ScheduleTicker {
		return scheduler.NewScheduleTicker(repo, clock, log)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(orgs domain.FollowUpEnabledOrgLister, svc *services.FollowUpService, log logger.Logger) *scheduler.FollowUpSweeper {
		return scheduler.NewFollowUpSweeper(orgs, svc, log)
	}); err != nil {
		return err
	}

	return nil
}

// draftAdapter adapts the sibling's DraftingService to the domain DraftFunc
// seam (scheduled run creation reuses the metered supplier-inquiries
// drafting; quantity defaults to 1 per schedule product).
func draftAdapter(drafting procurementServices.DraftingService) domain.DraftFunc {
	return func(ctx context.Context, orgID int32, supplier domain.DraftSupplier, products []domain.DraftProduct) (string, error) {
		s := &procurementDomain.Supplier{
			ID:          supplier.ID,
			ContactID:   supplier.ContactID,
			NIT:         supplier.NIT,
			IsActive:    supplier.IsActive,
			DisplayName: supplier.DisplayName,
		}
		ps := make([]*procurementDomain.Product, 0, len(products))
		quantities := make(map[int32]float64, len(products))
		for _, p := range products {
			ps = append(ps, &procurementDomain.Product{
				ID: p.ID, Name: p.Name, SKU: p.SKU, Unit: p.Unit, IsActive: true,
			})
			quantities[p.ID] = 1
		}
		return drafting.DraftForSupplier(ctx, orgID, s, supplier.DisplayName, ps, quantities)
	}
}
