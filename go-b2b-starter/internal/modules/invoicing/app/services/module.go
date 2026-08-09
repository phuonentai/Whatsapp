package services

import (
	"fmt"

	"go.uber.org/dig"

	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// Module handles dependency injection for invoicing services.
type Module struct{}

func NewModule() *Module { return &Module{} }

type serviceParams struct {
	dig.In

	Repo        domain.InvoiceRepository
	Provider    domain.InvoicingProvider
	DealRepo    crmDomain.DealRepository
	CompanyRepo crmDomain.CompanyRepository
	ContactRepo crmDomain.ContactRepository
	ConvRepo    crmDomain.ConversationRepository
	ActivitySvc crmServices.ActivityService
	Outbound    crmServices.OutboundService
	Logger      loggerDomain.Logger
	PaymentLinker PaymentLinker `optional:"true"`
}

func (m *Module) Configure(container *dig.Container) error {
	if err := container.Provide(func(p serviceParams) InvoicingService {
		return NewInvoicingService(
			p.Repo, p.Provider, p.DealRepo, p.CompanyRepo, p.ContactRepo, p.ConvRepo,
			p.ActivitySvc, p.Outbound, p.Logger, p.PaymentLinker,
		)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing service: %w", err)
	}

	if err := container.Provide(func(svc InvoicingService, log loggerDomain.Logger) DealStageListener {
		return NewDealStageListener(svc, log)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing deal stage listener: %w", err)
	}

	return nil
}
