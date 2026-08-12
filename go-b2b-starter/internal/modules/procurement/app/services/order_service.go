package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	procurementEvents "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// orderService implements human-approved order placement (D10/D13): guards,
// atomic placement transaction, idempotent retries, and dispatch-time
// re-checks happen in the send handler (D14).
type orderService struct {
	runs      domain.InquiryRunRepository
	orders    domain.OrderRepository
	contacts  domain.ContactLookup
	audit     domain.AuditRepository
	kill      KillSwitchReader
	metrics   MetricsSink
	log       logger.Logger
}

// NewOrderService builds the order placement service.
func NewOrderService(
	runs domain.InquiryRunRepository,
	orders domain.OrderRepository,
	contacts domain.ContactLookup,
	audit domain.AuditRepository,
	kill KillSwitchReader,
	metrics MetricsSink,
	log logger.Logger,
) *orderService {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &orderService{runs: runs, orders: orders, contacts: contacts, audit: audit, kill: kill, metrics: metrics, log: log}
}

// PlaceOrder validates the supplier response, applies the kill-switch and
// consent guards, and runs the atomic placement transaction. A retried POST
// (same run + supplier) returns the existing order without a second send or
// deal.
func (s *orderService) PlaceOrder(ctx context.Context, orgID int32, memberID string, in PlaceOrderInput) (*domain.Order, error) {
	if in.SupplierID == 0 || len(in.Items) == 0 {
		return nil, domain.ErrResponseNotAnswered
	}

	run, err := s.runs.GetRun(ctx, orgID, in.RunID)
	if err != nil {
		return nil, err
	}
	if run.Status != domain.RunAwaitingResponses && run.Status != domain.RunCompleted &&
		run.Status != domain.RunPartiallyAnswered && run.Status != domain.RunEscalated {
		return nil, domain.ErrInvalidRunStatus
	}

	// Find the recipient for this supplier and require an answered response
	// with requiere_humano = false (or an explicit org:manage override).
	recipient, response, err := s.latestResponseForSupplier(ctx, orgID, in.RunID, in.SupplierID)
	if err != nil {
		return nil, err
	}
	if recipient == nil || response == nil {
		return nil, domain.ErrResponseNotAnswered
	}
	if recipient.Status != domain.RecipientAnswered {
		return nil, domain.ErrResponseNotAnswered
	}
	if response.RequiereHumano && !in.Override {
		return nil, domain.ErrResponseRequiresHuman
	}

	contact, err := s.contacts.ContactByID(ctx, orgID, recipient.ContactID)
	if err != nil {
		return nil, err
	}

	// Guards: kill switch and consent withdrawn block the SEND but the
	// order/negocio/actividad are still recorded (D10).
	blockedReason := s.guardBlock(ctx, orgID, contact)
	order := &domain.Order{
		OrganizationID:    orgID,
		RunID:             in.RunID,
		SupplierID:        in.SupplierID,
		ContactID:         recipient.ContactID,
		Status:            domain.OrderPlaced,
		Items:             in.Items,
		Notes:             in.Notes,
		CreatedByMemberID: strPtr(memberID),
	}
	if blockedReason != nil {
		order.Status = domain.OrderSendBlocked
		order.BlockedReason = blockedReason
	}

	confirmMessage := buildConfirmMessage(run, contact.PhoneNumber, in.Items, in.Notes)
	order.ConfirmMessage = strPtr(confirmMessage)

	var confirmEvent *domain.OutboxEventInput
	if blockedReason == nil {
		ev := procurementEvents.NewOrderConfirmSend(orgID, 0, in.RunID, in.SupplierID, recipient.ContactID, contact.PhoneNumber, confirmMessage)
		payload, err := json.Marshal(ev)
		if err != nil {
			return nil, err
		}
		confirmEvent = &domain.OutboxEventInput{
			EventType: procurementEvents.OrderConfirmSendEventType,
			Payload:   payload,
		}
	}

	dealName := "Pedido a proveedor - " + runNotaLabel(run)
	placed, err := s.orders.PlaceOrderTx(ctx, domain.PlaceOrderTxParams{
		Order:              order,
		ConfirmEvent:       confirmEvent,
		DealNombre:         dealName,
		ActividadAsunto:    "Pedido a proveedor",
		ActividadContenido: confirmMessage,
	})
	if errors.Is(err, domain.ErrOrderAlreadyPlaced) {
		// Idempotent retry: return the existing order (no second send/deal).
		existing, gerr := s.orders.GetOrderByRunSupplier(ctx, in.RunID, in.SupplierID)
		if gerr != nil {
			return nil, gerr
		}
		return existing, nil
	}
	if err != nil {
		return nil, err
	}

	s.metrics.Inc(MetricOrderPlaced, map[string]string{"org": itoa(orgID)})
	if blockedReason != nil {
		s.metrics.Inc(MetricBlock, map[string]string{"org": itoa(orgID), "reason": *blockedReason})
	}
	return placed, nil
}

func (s *orderService) ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error) {
	return s.orders.ListRunOrders(ctx, orgID, runID)
}

// guardBlock returns the blocking reason when the send must not happen
// (kill switch, consent withdrawn), else nil.
func (s *orderService) guardBlock(ctx context.Context, orgID int32, contact *domain.ContactRef) *string {
	killSwitch, err := s.kill.GetAgentKillSwitch(ctx, orgID)
	if err == nil && killSwitch {
		return strPtr("kill_switch")
	}
	if contact.ConsentStatus == "withdrawn" {
		return strPtr("consent_withdrawn")
	}
	return nil
}

func (s *orderService) latestResponseForSupplier(ctx context.Context, orgID, runID, supplierID int32) (*domain.InquiryRecipient, *domain.InquiryResponse, error) {
	recipients, err := s.runs.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return nil, nil, err
	}
	var recipient *domain.InquiryRecipient
	for _, r := range recipients {
		if r.SupplierID == supplierID {
			recipient = r
			break
		}
	}
	if recipient == nil {
		return nil, nil, domain.ErrRecipientNotFound
	}
	responses, err := s.runs.ListRunResponses(ctx, orgID, runID)
	if err != nil {
		return nil, nil, err
	}
	var latest *domain.InquiryResponse
	for i := range responses {
		if responses[i].RecipientID == recipient.ID {
			if latest == nil || responses[i].ID > latest.ID {
				latest = responses[i]
			}
		}
	}
	return recipient, latest, nil
}

// buildConfirmMessage composes the pre-composed Spanish order confirmation
// (plain text; template infra is a separate change).
func buildConfirmMessage(run *domain.InquiryRun, phone string, items []domain.OrderItem, notes *string) string {
	var b strings.Builder
	b.WriteString("Hola, confirmamos nuestro pedido:\n")
	for _, it := range items {
		b.WriteString(fmt.Sprintf("- %s\n", formatOrderLine(it)))
	}
	if notes != nil && *notes != "" {
		b.WriteString("Notas: " + *notes + "\n")
	}
	b.WriteString("Quedamos atentos a la entrega. ¡Gracias!")
	return b.String()
}

func formatOrderLine(it domain.OrderItem) string {
	return fmt.Sprintf("Producto %d × %g", it.ProductID, it.Quantity)
}

func runNotaLabel(run *domain.InquiryRun) string {
	return "cotización"
}
