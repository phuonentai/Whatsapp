// Package services implements the invoicing application logic.
package services

import (
	"context"
	"encoding/json"
	"fmt"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// InvoicingService is the application boundary for the invoicing capability.
type InvoicingService interface {
	CreateForDeal(ctx context.Context, orgID, dealID int32) (*domain.Invoice, error)
	ProcessWebhookEvent(ctx context.Context, rawBody []byte) error
	PollPending(ctx context.Context) (int, error)
}

type invoicingService struct {
	repo          domain.InvoiceRepository
	provider      domain.InvoicingProvider
	dealRepo      crmdomain.DealRepository
	companyRepo   crmdomain.CompanyRepository
	contactRepo   crmdomain.ContactRepository
	convRepo      crmdomain.ConversationRepository
	activitySvc   crmServices.ActivityService
	outbound      crmServices.OutboundService
	logger        loggerDomain.Logger
	paymentLinker PaymentLinker
}

// PaymentLinker supplies the payment link appended to invoice notifications.
// Returns "" when unavailable (e.g. MercadoPago not configured for the org).
type PaymentLinker interface {
	PaymentLink(ctx context.Context, orgID int32) (string, error)
}

func NewInvoicingService(
	repo domain.InvoiceRepository,
	provider domain.InvoicingProvider,
	dealRepo crmdomain.DealRepository,
	companyRepo crmdomain.CompanyRepository,
	contactRepo crmdomain.ContactRepository,
	convRepo crmdomain.ConversationRepository,
	activitySvc crmServices.ActivityService,
	outbound crmServices.OutboundService,
	logger loggerDomain.Logger,
	paymentLinker PaymentLinker,
) InvoicingService {
	if paymentLinker == nil {
		paymentLinker = noopPaymentLinker{}
	}
	return &invoicingService{
		repo: repo, provider: provider,
		dealRepo: dealRepo, companyRepo: companyRepo, contactRepo: contactRepo, convRepo: convRepo,
		activitySvc: activitySvc, outbound: outbound, logger: logger,
		paymentLinker: paymentLinker,
	}
}

// CreateForDeal creates an invoice for a deal that reached the invoicing stage.
// Idempotent: one invoice per (org, deal); a re-trigger returns the existing row.
func (s *invoicingService) CreateForDeal(ctx context.Context, orgID, dealID int32) (*domain.Invoice, error) {
	existing, err := s.repo.GetByDeal(ctx, orgID, dealID)
	if err == nil {
		return existing, nil
	}
	if err != domain.ErrInvoiceNotFound {
		return nil, fmt.Errorf("failed to check existing invoice: %w", err)
	}

	deal, err := s.dealRepo.GetByID(ctx, orgID, dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to load deal: %w", err)
	}

	customer, err := s.buildCustomer(ctx, orgID, deal)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve customer: %w", err)
	}

	if _, err := s.provider.UpsertCustomer(ctx, orgID, customer); err != nil {
		return nil, fmt.Errorf("failed to upsert customer at provider: %w", err)
	}

	created, err := s.provider.CreateInvoice(ctx, orgID, &domain.InvoiceRequest{
		OrganizationID: orgID,
		DealID:         dealID,
		Customer:       customer,
		Amount:         deal.Monto,
		Currency:       deal.Moneda,
		Description:    deal.Nombre,
	})
	if err != nil {
		return nil, err
	}

	stored, err := s.repo.Insert(ctx, created)
	if err != nil {
		if err == domain.ErrInvoiceExists {
			// A concurrent trigger created it first; return the existing row.
			return s.repo.GetByDeal(ctx, orgID, dealID)
		}
		return nil, fmt.Errorf("failed to store invoice: %w", err)
	}

	s.recordActivity(ctx, orgID, dealID, "Factura electrónica creada", fmt.Sprintf("Factura %s creada para el negocio %s", stored.ExternalID, deal.Nombre))
	s.notify(ctx, stored)

	return stored, nil
}

func (s *invoicingService) buildCustomer(ctx context.Context, orgID int32, deal *crmdomain.DealWithRefs) (domain.CustomerInfo, error) {
	customer := domain.CustomerInfo{}

	if deal.CompanyID != nil {
		company, err := s.companyRepo.GetByID(ctx, orgID, *deal.CompanyID)
		if err != nil {
			return customer, fmt.Errorf("failed to load company: %w", err)
		}
		customer.Name = company.Name
		customer.Identification = company.Nit
		customer.IdentificationType = "NIT"
		if company.Phone != "" {
			customer.Phone = company.Phone
		}
	}

	if deal.ContactID != nil {
		contact, err := s.contactRepo.GetByID(ctx, orgID, *deal.ContactID)
		if err != nil {
			return customer, fmt.Errorf("failed to load contact: %w", err)
		}
		if customer.Name == "" {
			customer.Name = contact.DisplayName
			if customer.Name == "" {
				customer.Name = contact.PhoneNumber
			}
		}
		if customer.Identification == "" {
			customer.Identification = contact.NumeroDocumento
			customer.IdentificationType = string(contact.TipoDocumento)
		}
		customer.Phone = contact.PhoneNumber
		customer.Email = contact.Email
	}

	if customer.Identification == "" {
		customer.Identification = "C.C. 000000000" // provider requires an identification; masked placeholder
		customer.IdentificationType = "CC"
	}
	if customer.Name == "" {
		customer.Name = fmt.Sprintf("Cliente %d", deal.ID)
	}

	return customer, nil
}

// ProcessWebhookEvent applies a provider status notification idempotently and
// transaction-safely: status regressions and duplicates are ignored.
func (s *invoicingService) ProcessWebhookEvent(ctx context.Context, rawBody []byte) error {
	var payload struct {
		ID     string `json:"id"`
		Number string `json:"number"`
		Cufe   string `json:"cufe"`
		Status string `json:"status"`
		PdfURL string `json:"pdf_url"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return fmt.Errorf("invalid webhook payload: %w", err)
	}
	if payload.ID == "" {
		return fmt.Errorf("webhook payload missing invoice id")
	}

	inv, err := s.repo.GetByExternalID(ctx, payload.ID)
	if err != nil {
		if err == domain.ErrInvoiceNotFound {
			s.logger.Warn("invoice webhook for unknown external id", map[string]any{"external_id": payload.ID})
			return nil
		}
		return fmt.Errorf("failed to load invoice by external id: %w", err)
	}

	newStatus := domain.InvoiceStatus(payload.Status)
	if newStatus == "" {
		return fmt.Errorf("webhook payload missing status")
	}

	// Idempotency + regression guard: never move a final invoice backwards.
	if newStatus == inv.Status {
		return nil
	}
	if inv.Status.IsFinal() {
		return nil
	}

	if _, err := s.repo.UpdateStatus(ctx, inv.ID, newStatus, payload.Cufe, payload.PdfURL); err != nil {
		return fmt.Errorf("failed to update invoice status: %w", err)
	}

	inv.Status = newStatus
	inv.Cufe = payload.Cufe
	inv.PdfURL = payload.PdfURL
	s.notify(ctx, inv)

	return nil
}

// PollPending reconciles non-final invoices via the provider as a safety net
// for missed webhooks. Returns the number of invoices reconciled.
func (s *invoicingService) PollPending(ctx context.Context) (int, error) {
	pending, err := s.repo.ListByStatus(ctx, domain.InvoiceStatusPending, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to list pending invoices: %w", err)
	}

	reconciled := 0
	for _, inv := range pending {
		if inv.ExternalID == "" {
			continue
		}
		remote, err := s.provider.GetInvoiceStatus(ctx, inv.OrganizationID, inv.ExternalID)
		if err != nil {
			s.logger.Warn("poll failed for invoice", map[string]any{"invoice_id": inv.ID, "error": err.Error()})
			continue
		}
		if remote.Status == inv.Status {
			continue
		}
		if _, err := s.repo.UpdateStatus(ctx, inv.ID, remote.Status, remote.Cufe, remote.PdfURL); err != nil {
			s.logger.Warn("poll update failed for invoice", map[string]any{"invoice_id": inv.ID, "error": err.Error()})
			continue
		}
		inv.Status = remote.Status
		inv.Cufe = remote.Cufe
		inv.PdfURL = remote.PdfURL
		s.notify(ctx, inv)
		reconciled++
	}

	return reconciled, nil
}

// notify sends the WhatsApp invoice notification once per status transition.
// Notification failures never fail the invoicing flow (log warning only).
func (s *invoicingService) notify(ctx context.Context, inv *domain.Invoice) {
	if inv.NotifiedStatus == inv.Status && inv.NotifiedStatus != "" {
		return
	}

	deal, err := s.dealRepo.GetByID(ctx, inv.OrganizationID, inv.DealID)
	if err != nil || deal.ContactID == nil {
		return
	}
	conv, err := s.convRepo.GetActiveByContact(ctx, inv.OrganizationID, *deal.ContactID)
	if err != nil || conv == nil {
		return
	}

	paymentLink := ""
	if s.paymentLinker != nil {
		paymentLink, _ = s.paymentLinker.PaymentLink(ctx, inv.OrganizationID)
	}

	msg := fmt.Sprintf("Factura %s · estado: %s", inv.ExternalID, inv.Status)
	if inv.PdfURL != "" {
		msg += "\nEnlace: " + inv.PdfURL
	}
	if paymentLink != "" {
		msg += "\nPagar: " + paymentLink
	}

	if _, err := s.outbound.SendMessage(ctx, inv.OrganizationID, conv.ID, msg); err != nil {
		s.logger.Warn("failed to send invoice notification", map[string]any{
			"organization_id": inv.OrganizationID,
			"invoice_id":      inv.ID,
			"error":           err.Error(),
		})
		return
	}

	if _, err := s.repo.MarkNotified(ctx, inv.ID, inv.Status); err != nil {
		s.logger.Warn("failed to mark invoice notified", map[string]any{"invoice_id": inv.ID, "error": err.Error()})
	}
}

func (s *invoicingService) recordActivity(ctx context.Context, orgID, dealID int32, asunto, contenido string) {
	if _, err := s.activitySvc.Create(ctx, orgID, &crmServices.CreateActivityRequest{
		DealID:    &dealID,
		Tipo:      crmdomain.ActivityTypeSistema,
		Asunto:    asunto,
		Contenido: contenido,
	}); err != nil {
		s.logger.Warn("failed to record invoicing activity", map[string]any{"deal_id": dealID, "error": err.Error()})
	}
}

type noopPaymentLinker struct{}

func (noopPaymentLinker) PaymentLink(ctx context.Context, orgID int32) (string, error) { return "", nil }
