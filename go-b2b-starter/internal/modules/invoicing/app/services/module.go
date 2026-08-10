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

// connectionParams wires the connection service. The validator is the Siigo
// adapter (named binding); the cipher is the envelope-encryption infra.
type connectionParams struct {
	dig.In

	Repo      domain.ConnectionRepository
	Validator domain.ConnectionValidator `name:"siigo"`
	Cipher    CredentialCipher
	Logger    loggerDomain.Logger
}

type numerationParams struct {
	dig.In

	Reader  domain.NumerationReader `name:"siigo"`
	Repo    domain.NumerationRepository
	ConnSvc ConnectionService
	Logger  loggerDomain.Logger
}

type importParams struct {
	dig.In

	Reader      domain.CustomerReader `name:"siigo"`
	CompanyRepo crmDomain.CompanyRepository
	ContactRepo crmDomain.ContactRepository
	RunRepo     domain.ImportRunRepository
	Logger      loggerDomain.Logger
}

type testInvoiceParams struct {
	dig.In

	Provider domain.InvoicingProvider
	Repo     domain.InvoiceRepository
	ConnSvc  ConnectionService
	Logger   loggerDomain.Logger
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

	if err := container.Provide(func(p connectionParams) ConnectionService {
		return NewConnectionService(p.Repo, p.Validator, p.Cipher, p.Logger)
	}); err != nil {
		return fmt.Errorf("failed to provide connection service: %w", err)
	}

	if err := container.Provide(func(p numerationParams) NumerationService {
		return NewNumerationService(p.Reader, p.Repo, p.ConnSvc, p.Logger)
	}); err != nil {
		return fmt.Errorf("failed to provide numeration service: %w", err)
	}

	if err := container.Provide(func(p importParams) ImportService {
		return NewImportService(p.Reader, p.CompanyRepo, p.ContactRepo, p.RunRepo, p.Logger)
	}); err != nil {
		return fmt.Errorf("failed to provide import service: %w", err)
	}

	if err := container.Provide(func(p testInvoiceParams) TestInvoiceService {
		return NewTestInvoiceService(p.Provider, p.Repo, p.ConnSvc, p.Logger)
	}); err != nil {
		return fmt.Errorf("failed to provide test invoice service: %w", err)
	}

	if err := container.Provide(func(svc InvoicingService, connSvc ConnectionService, activitySvc crmServices.ActivityService, log loggerDomain.Logger) DealStageListener {
		return NewDealStageListener(svc, connSvc, activitySvc, log)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing deal stage listener: %w", err)
	}

	return nil
}
