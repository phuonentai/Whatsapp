// Package services implements the client-payments application logic.
package services

import (
	"context"
	"fmt"
	"math"
	"time"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// PaymentsService is the application boundary for client payments.
type PaymentsService interface {
	CreateLink(ctx context.Context, orgID, dealID int32, invoiceID *int32, amountCOP int64) (string, error)
	HandlePaymentEvent(ctx context.Context, eventType, paymentID string) error
}

type paymentsService struct {
	repo        domain.PaymentRepository
	rail        domain.PaymentRail
	dealRepo    crmdomain.DealRepository
	convRepo    crmdomain.ConversationRepository
	activitySvc crmServices.ActivityService
	outbound    crmServices.OutboundService
	logger      loggerDomain.Logger
	commissionRate float64
}

func NewPaymentsService(
	repo domain.PaymentRepository,
	rail domain.PaymentRail,
	dealRepo crmdomain.DealRepository,
	convRepo crmdomain.ConversationRepository,
	activitySvc crmServices.ActivityService,
	outbound crmServices.OutboundService,
	logger loggerDomain.Logger,
	commissionRate float64,
) PaymentsService {
	return &paymentsService{
		repo: repo, rail: rail,
		dealRepo: dealRepo, convRepo: convRepo,
		activitySvc: activitySvc, outbound: outbound, logger: logger,
		commissionRate: commissionRate,
	}
}

// commissionFor returns the platform commission for a base amount in COP.
func (s *paymentsService) commissionFor(amountCOP int64) int64 {
	if s.commissionRate <= 0 {
		return 0
	}
	return int64(math.Round(float64(amountCOP) * s.commissionRate))
}

// CreateLink creates a one-shot payment preference priced at the base amount
// plus platform commission and persists a pending record. The returned link
// is included in the WhatsApp invoice notification.
func (s *paymentsService) CreateLink(ctx context.Context, orgID, dealID int32, invoiceID *int32, amountCOP int64) (string, error) {
	if amountCOP <= 0 {
		return "", fmt.Errorf("payment amount must be positive, got %d", amountCOP)
	}

	commission := s.commissionFor(amountCOP)
	unitPrice := amountCOP + commission

	initPoint, preferenceID, err := s.rail.CreatePreference(ctx, orgID, dealID, unitPrice, "COP")
	if err != nil {
		return "", fmt.Errorf("failed to create payment preference: %w", err)
	}

	if _, err := s.repo.Create(ctx, &domain.ClientPayment{
		OrganizationID: orgID,
		DealID:         dealID,
		InvoiceID:      invoiceID,
		AmountCOP:      amountCOP,
		CommissionCOP:  commission,
		Currency:       "COP",
		Status:         domain.PaymentStatusPending,
		MPPreferenceID: preferenceID,
	}); err != nil {
		return "", fmt.Errorf("failed to persist client payment: %w", err)
	}

	return initPoint, nil
}

// HandlePaymentEvent processes a MercadoPago payment event: correlates the
// provider payment with a tracked preference (directly by payment id, or via
// the provider payment detail's preference id), verifies against the provider,
// and transitions state idempotently. Untracked payments are acknowledged as
// no-ops. The WhatsApp confirmation and deal activity are non-fatal.
func (s *paymentsService) HandlePaymentEvent(ctx context.Context, eventType, paymentID string) error {
	if paymentID == "" {
		return fmt.Errorf("payment event %q missing payment id", eventType)
	}

	payment, err := s.repo.GetByPaymentID(ctx, paymentID)
	if err != nil && err != domain.ErrPaymentNotFound {
		return fmt.Errorf("failed to load payment by id: %w", err)
	}

	if err == domain.ErrPaymentNotFound {
		detail, verr := s.rail.VerifyPayment(ctx, paymentID)
		if verr != nil {
			s.logger.Warn("payment verification failed, leaving payment pending", map[string]any{
				"payment_id": paymentID,
				"error":      verr.Error(),
			})
			return nil
		}
		if detail.PreferenceID != "" {
			payment, err = s.repo.GetByPreferenceID(ctx, detail.PreferenceID)
			if err == domain.ErrPaymentNotFound {
				s.logger.Info("payment event for untracked preference, acknowledged", map[string]any{
					"payment_id": paymentID, "preference_id": detail.PreferenceID,
				})
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to load payment by preference: %w", err)
			}
			if _, aerr := s.repo.AttachPaymentID(ctx, payment.ID, paymentID); aerr != nil {
				s.logger.Warn("failed to attach payment id", map[string]any{
					"payment_id": paymentID, "client_payment_id": payment.ID, "error": aerr.Error(),
				})
			}
		} else {
			s.logger.Info("payment event for untracked payment, acknowledged", map[string]any{"payment_id": paymentID})
			return nil
		}
	}

	detail, err := s.rail.VerifyPayment(ctx, paymentID)
	if err != nil {
		s.logger.Warn("payment verification failed, leaving payment pending", map[string]any{
			"payment_id": paymentID, "error": err.Error(),
		})
		return nil
	}

	switch detail.Status {
	case domain.PaymentStatusPaid:
		if err := s.markPaid(ctx, payment, paymentID); err != nil {
			return err
		}
	case domain.PaymentStatusFailed:
		if _, err := s.repo.Transition(ctx, payment.ID, domain.PaymentStatusFailed, paymentID, nil); err != nil && err != domain.ErrPaymentTerminal {
			return fmt.Errorf("failed to mark payment failed: %w", err)
		}
	default:
		s.logger.Debug("payment not yet approved, leaving pending", map[string]any{
			"payment_id": paymentID, "status": string(detail.Status),
		})
	}

	return nil
}

// markPaid transitions a payment to paid (idempotent) and confirms it to the
// contact inside WhatsApp plus a deal activity; both notifications are
// non-fatal (logged warning only).
func (s *paymentsService) markPaid(ctx context.Context, payment *domain.ClientPayment, paymentID string) error {
	now := time.Now()
	paid, err := s.repo.Transition(ctx, payment.ID, domain.PaymentStatusPaid, paymentID, &now)
	if err == domain.ErrPaymentTerminal {
		// Duplicate event: already transitioned. Still confirm idempotently?
		// No — confirmation already fired for the original transition.
		s.logger.Debug("duplicate payment event, already terminal", map[string]any{
			"payment_id": paymentID, "client_payment_id": payment.ID,
		})
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to mark payment paid: %w", err)
	}

	s.confirm(ctx, paid)
	s.recordActivity(ctx, paid)
	return nil
}

// confirm sends the transactional WhatsApp confirmation. Failures never fail
// the payment flow.
func (s *paymentsService) confirm(ctx context.Context, payment *domain.ClientPayment) {
	deal, err := s.dealRepo.GetByID(ctx, payment.OrganizationID, payment.DealID)
	if err != nil || deal.ContactID == nil {
		s.logger.Warn("payment confirmation skipped: deal/contact not resolvable", map[string]any{
			"organization_id": payment.OrganizationID, "deal_id": payment.DealID, "error": errText(err),
		})
		return
	}
	conv, err := s.convRepo.GetActiveByContact(ctx, payment.OrganizationID, *deal.ContactID)
	if err != nil || conv == nil {
		s.logger.Warn("payment confirmation skipped: no active conversation", map[string]any{
			"organization_id": payment.OrganizationID, "contact_id": *deal.ContactID, "error": errText(err),
		})
		return
	}

	msg := fmt.Sprintf("Pago confirmado por %d COP. ¡Gracias!", payment.AmountCOP+payment.CommissionCOP)
	if _, err := s.outbound.SendMessage(ctx, payment.OrganizationID, conv.ID, msg); err != nil {
		s.logger.Warn("failed to send payment confirmation", map[string]any{
			"organization_id": payment.OrganizationID, "payment_id": payment.ID, "error": err.Error(),
		})
	}
}

// recordActivity writes the deal activity; failures are logged, never fatal.
func (s *paymentsService) recordActivity(ctx context.Context, payment *domain.ClientPayment) {
	if _, err := s.activitySvc.Create(ctx, payment.OrganizationID, &crmServices.CreateActivityRequest{
		DealID:    &payment.DealID,
		Tipo:      crmdomain.ActivityTypeSistema,
		Asunto:    "Pago recibido",
		Contenido: fmt.Sprintf("Pago confirmado de %d COP (comisión plataforma %d COP)", payment.AmountCOP, payment.CommissionCOP),
	}); err != nil {
		s.logger.Warn("failed to record payment activity", map[string]any{
			"organization_id": payment.OrganizationID, "payment_id": payment.ID, "error": err.Error(),
		})
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
