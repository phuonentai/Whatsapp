package services

import (
	"fmt"

	"go.uber.org/dig"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// Module handles dependency injection for payments services.
type Module struct{}

func NewModule() *Module { return &Module{} }

type serviceParams struct {
	dig.In

	Repo           domain.PaymentRepository
	Rail           domain.PaymentRail
	DealRepo       crmdomain.DealRepository
	ConvRepo       crmdomain.ConversationRepository
	ActivitySvc    crmServices.ActivityService
	Outbound       crmServices.OutboundService
	Logger         loggerDomain.Logger
	CommissionRate float64 `name:"payments_commission_rate"`
}

func (m *Module) Configure(container *dig.Container) error {
	if err := container.Provide(func(p serviceParams) PaymentsService {
		return NewPaymentsService(
			p.Repo, p.Rail, p.DealRepo, p.ConvRepo, p.ActivitySvc, p.Outbound, p.Logger, p.CommissionRate,
		)
	}); err != nil {
		return fmt.Errorf("failed to provide payments service: %w", err)
	}

	// The billing webhook dispatches payment events through this seam.
	if err := container.Provide(func(svc PaymentsService) PaymentEventHandler {
		return svc.(PaymentEventHandler)
	}); err != nil {
		return fmt.Errorf("failed to provide payment event handler: %w", err)
	}

	return nil
}
