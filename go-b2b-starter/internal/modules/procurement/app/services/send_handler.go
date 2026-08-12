package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	procurementEvents "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// textSender is the outbound Meta seam (pkg/whatsapp client with breaker).
type textSender interface {
	SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error)
}

// sendHandlerOptions lets tests inject the Meta seam; the production wiring
// uses the real circuit-breakered client.
type sendHandlerOptions struct {
	Sender textSender
}

// sendHandler processes procurement durable outbox events. Every handler
// re-validates state transaction-isolated BEFORE invoking the client (D14):
// run not cancelled/escalated, recipient still pending, kill switch off, and
// (for confirmations) consent not withdrawn. Guard failures complete the
// event with an audit and no send.
type sendHandler struct {
	runs     domain.InquiryRunRepository
	orders   domain.OrderRepository
	contacts domain.ContactLookup
	audit    domain.AuditRepository
	configs  whatsappDomain.ConfigRepository
	sender   textSender
	pacer    Pacer
	kill     KillSwitchReader
	metrics  MetricsSink
	log      logger.Logger
}

// NewSendHandler builds the outbox send handler.
func NewSendHandler(
	runs domain.InquiryRunRepository,
	orders domain.OrderRepository,
	contacts domain.ContactLookup,
	audit domain.AuditRepository,
	configs whatsappDomain.ConfigRepository,
	pacer Pacer,
	kill KillSwitchReader,
	metrics MetricsSink,
	log logger.Logger,
	opts ...sendHandlerOptions,
) SendHandler {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	sender := textSender(whatsapp.NewClientWithBreaker(5, 30*time.Second, 2))
	if len(opts) > 0 && opts[0].Sender != nil {
		sender = opts[0].Sender
	}
	return &sendHandler{
		runs: runs, orders: orders, contacts: contacts, audit: audit,
		configs: configs,
		sender:   sender,
		pacer:    pacer, kill: kill, metrics: metrics, log: log,
	}
}

// HandleInquirySend dispatches one recipient send of an inquiry run.
func (h *sendHandler) HandleInquirySend(ctx context.Context, e *procurementEvents.InquirySend) error {
	recipient, err := h.runs.GetRecipient(ctx, e.OrganizationID, e.RecipientID)
	if errors.Is(err, domain.ErrRecipientNotFound) {
		return nil // already gone/processed
	}
	if err != nil {
		return err
	}
	if recipient.Status != domain.RecipientPending {
		return nil // already sent or terminal — idempotent no-op (no double send)
	}

	run, err := h.runs.GetRun(ctx, e.OrganizationID, e.RunID)
	if err != nil {
		return err
	}
	if run.Status == domain.RunCancelled || run.Status == domain.RunEscalated {
		return h.block(ctx, e.OrganizationID, "inquiry_recipient", recipient.ID, "send_skipped", string(run.Status))
	}

	killSwitch, err := h.kill.GetAgentKillSwitch(ctx, e.OrganizationID)
	if err != nil {
		return err
	}
	if killSwitch {
		return h.block(ctx, e.OrganizationID, "inquiry_recipient", recipient.ID, "send_skipped", "kill_switch")
	}

	if !h.pacer.Allow(e.OrganizationID) {
		h.metrics.Inc(MetricSendRetry, map[string]string{"org": itoa(e.OrganizationID), "event": "inquiry_send"})
		return ErrRateLimited
	}

	msgID, err := h.dispatch(ctx, e.OrganizationID, e.To, e.Message)
	if err != nil {
		h.metrics.Inc(MetricSendRetry, map[string]string{"org": itoa(e.OrganizationID), "event": "inquiry_send"})
		return err // dispatcher retries with backoff / dead-letters
	}

	updated, err := h.runs.MarkRecipientSent(ctx, e.OrganizationID, e.RecipientID, msgID)
	if errors.Is(err, domain.ErrRecipientNotFound) {
		return nil // a concurrent dispatch already sent — no double send
	}
	if err != nil {
		return err
	}
	h.metrics.Inc(MetricSendSuccess, map[string]string{"org": itoa(e.OrganizationID), "event": "inquiry_send"})
	h.recordWindowWarning(ctx, e.OrganizationID, e.ContactID, updated.ID)

	return h.afterSend(ctx, e.OrganizationID, e.RunID)
}

// HandleOrderConfirmSend dispatches one order-confirmation send.
func (h *sendHandler) HandleOrderConfirmSend(ctx context.Context, e *procurementEvents.OrderConfirmSend) error {
	order, err := h.orders.GetOrder(ctx, e.OrganizationID, e.OrderID)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if order.Status == domain.OrderConfirmSent || order.Status == domain.OrderSendBlocked {
		return nil // idempotent
	}

	run, err := h.runs.GetRun(ctx, e.OrganizationID, e.RunID)
	if err != nil {
		return err
	}
	if run.Status == domain.RunCancelled {
		return h.block(ctx, e.OrganizationID, "order", order.ID, "confirm_skipped", "run_cancelled")
	}

	killSwitch, err := h.kill.GetAgentKillSwitch(ctx, e.OrganizationID)
	if err != nil {
		return err
	}
	if killSwitch {
		_, _ = h.orders.MarkOrderSendBlocked(ctx, e.OrganizationID, order.ID, "kill_switch")
		return h.block(ctx, e.OrganizationID, "order", order.ID, "confirm_skipped", "kill_switch")
	}

	contact, err := h.contacts.ContactByID(ctx, e.OrganizationID, e.ContactID)
	if err != nil {
		return err
	}
	if contact.ConsentStatus == "withdrawn" {
		_, _ = h.orders.MarkOrderSendBlocked(ctx, e.OrganizationID, order.ID, "consent_withdrawn")
		return h.block(ctx, e.OrganizationID, "order", order.ID, "confirm_skipped", "consent_withdrawn")
	}

	if !h.pacer.Allow(e.OrganizationID) {
		h.metrics.Inc(MetricSendRetry, map[string]string{"org": itoa(e.OrganizationID), "event": "order_confirm"})
		return ErrRateLimited
	}

	if _, err := h.dispatch(ctx, e.OrganizationID, e.To, e.Message); err != nil {
		h.metrics.Inc(MetricSendRetry, map[string]string{"org": itoa(e.OrganizationID), "event": "order_confirm"})
		return err
	}

	if _, err := h.orders.MarkOrderConfirmSent(ctx, e.OrganizationID, order.ID); err != nil {
		return err
	}
	h.metrics.Inc(MetricSendSuccess, map[string]string{"org": itoa(e.OrganizationID), "event": "order_confirm"})
	h.recordWindowWarning(ctx, e.OrganizationID, e.ContactID, order.ID)
	return nil
}

// dispatch resolves the org WhatsApp config and sends through the
// circuit-breakered client, returning the provider message id. Config problems
// are permanent (bubble up as errors); Meta API errors bubble up for outbox
// retry.
func (h *sendHandler) dispatch(ctx context.Context, orgID int32, to, body string) (string, error) {
	config, err := h.configs.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("whatsapp_not_configured: %w", err)
	}
	if !config.IsActive {
		return "", fmt.Errorf("whatsapp_not_configured: config inactive")
	}
	if config.AccessToken == "" {
		return "", fmt.Errorf("whatsapp_no_access_token: access token missing")
	}

	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := config.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		// Mock-auth e2e mode: never call the real Meta Graph API.
		return fmt.Sprintf("wamid.mock.procurement.%d", time.Now().UnixNano()), nil
	}

	return h.sender.SendTextMessage(ctx, config.AccessToken, graphAPIURL, apiVersion, config.PhoneNumberID, to, body)
}

// afterSend re-evaluates the run after a recipient transition: when no
// recipient remains pending, the run moves sending → awaiting_responses (some
// sendable) or sending → failed (none sendable). Guarded transitions make
// concurrent dispatches safe.
func (h *sendHandler) afterSend(ctx context.Context, orgID, runID int32) error {
	recipients, err := h.runs.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return err
	}
	pending := 0
	terminal := 0
	for _, r := range recipients {
		switch r.Status {
		case domain.RecipientPending:
			pending++
		case domain.RecipientFailed:
			terminal++
		}
	}
	if pending > 0 {
		return nil
	}
	if terminal == len(recipients) {
		if _, err := h.runs.TransitionRun(ctx, orgID, runID, domain.RunSending, domain.RunFailed); err != nil {
			if !errors.Is(err, domain.ErrInvalidTransition) {
				return err
			}
		}
		return nil
	}
	if _, err := h.runs.TransitionRun(ctx, orgID, runID, domain.RunSending, domain.RunAwaitingResponses); err != nil {
		if !errors.Is(err, domain.ErrInvalidTransition) {
			return err
		}
	}
	return nil
}

// recordWindowWarning audits the outside_24h_window warning (per
// whatsapp-outbound-send) when the contact's last message is older than 24h
// or absent; the send itself is NOT failed.
func (h *sendHandler) recordWindowWarning(ctx context.Context, orgID, contactID, entityID int32) {
	contact, err := h.contacts.ContactByID(ctx, orgID, contactID)
	if err != nil {
		return
	}
	if contact.LastMessageAt != nil && time.Since(*contact.LastMessageAt) < 24*time.Hour {
		return
	}
	_ = h.audit.Record(ctx, domain.AuditEntry{
		OrganizationID: orgID,
		EntityType:     "contact",
		EntityID:       &entityID,
		Action:         "send_warning",
		Decision:       "allow",
		Reason:         strPtr2("outside_24h_window"),
	})
}

// block audits a dispatch-time guard failure and completes the event with no
// send (D14).
func (h *sendHandler) block(ctx context.Context, orgID int32, entityType string, entityID int32, action, reason string) error {
	h.metrics.Inc(MetricBlock, map[string]string{"org": itoa(orgID), "reason": reason})
	_ = h.audit.Record(ctx, domain.AuditEntry{
		OrganizationID: orgID,
		EntityType:     entityType,
		EntityID:       &entityID,
		Action:         action,
		Decision:       "skip",
		Reason:         strPtr2(reason),
	})
	return nil
}

