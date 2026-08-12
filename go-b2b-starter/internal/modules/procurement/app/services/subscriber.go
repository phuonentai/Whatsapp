package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// ProcurementSubscriber is the independent subscriber on
// whatsapp.message.received (D7): it resolves the sender to an active run
// recipient and runs exactly one metered extraction per eligible reply.
// Non-recipient messages return without any LLM call; handled outcomes
// (including escalations) return nil so the eventbus is never crashed.
type ProcurementSubscriber struct {
	runs       domain.InquiryRunRepository
	audit      domain.AuditRepository
	extraction ExtractionService
	metrics    MetricsSink
	log        logger.Logger
}

// NewProcurementSubscriber builds the inbound subscriber.
func NewProcurementSubscriber(
	runs domain.InquiryRunRepository,
	audit domain.AuditRepository,
	extraction ExtractionService,
	metrics MetricsSink,
	log logger.Logger,
) *ProcurementSubscriber {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &ProcurementSubscriber{runs: runs, audit: audit, extraction: extraction, metrics: metrics, log: log}
}

// HandleMessageReceived processes one inbound WhatsApp message. Errors are
// returned only for unexpected infrastructure failures (so the outbox
// dispatcher may retry); all business outcomes complete successfully.
func (s *ProcurementSubscriber) HandleMessageReceived(ctx context.Context, event *whatsappEvents.MessageReceived) error {
	if event.MessageType != "text" || event.Content == "" {
		return nil
	}

	phone, err := whatsapp.CanonicalizeE164(event.From)
	if err != nil {
		s.log.Warn("procurement: invalid sender phone", map[string]any{"from": event.From, "error": err.Error()})
		return nil
	}

	recipients, err := s.runs.ListActiveRecipientsByPhone(ctx, event.OrganizationID, phone)
	if err != nil {
		return fmt.Errorf("procurement: resolve active recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil // not a procurement reply — no LLM call
	}

	for _, recipient := range recipients {
		if err := s.processRecipient(ctx, event, recipient); err != nil {
			return err
		}
	}
	return nil
}

// IsActiveRecipientByPhone implements services.ActiveRecipientChecker (the
// seam consumed by the agent skip check, task 10): tenant-scoped by the
// event's organization id.
func (s *ProcurementSubscriber) IsActiveRecipientByPhone(ctx context.Context, orgID int32, phoneNumber string) (bool, error) {
	recipients, err := s.runs.ListActiveRecipientsByPhone(ctx, orgID, phoneNumber)
	if err != nil {
		return false, err
	}
	return len(recipients) > 0, nil
}

func (s *ProcurementSubscriber) processRecipient(ctx context.Context, event *whatsappEvents.MessageReceived, recipient *domain.InquiryRecipient) error {
	// Redelivery guard: the message was already extracted for this recipient.
	if _, err := s.runs.GetResponseByRecipientMessage(ctx, recipient.ID, event.MessageSID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrResponseNotFound) {
		return err
	}

	result, err := s.extraction.ExtractReply(ctx, event.OrganizationID, event.Content)
	if err != nil {
		s.escalate(ctx, event.OrganizationID, recipient.RunID, fmt.Sprintf("extraction failed: %v", err))
		return nil // handled: escalation audited, no fabricated response persisted
	}

	resp := &domain.InquiryResponse{
		OrganizationID: event.OrganizationID,
		RecipientID:    recipient.ID,
		RawMessageID:   event.MessageSID,
		Items:          result.Items,
		Resumen:        result.Resumen,
		RequiereHumano: result.RequiereHumano,
	}
	if _, err := s.runs.SaveResponse(ctx, resp); err != nil {
		if errors.Is(err, domain.ErrDuplicateResponse) {
			return nil // concurrent redelivery already persisted
		}
		return err
	}

	if _, err := s.runs.MarkRecipientAnswered(ctx, event.OrganizationID, recipient.ID); err != nil {
		return err
	}

	if result.RequiereHumano {
		s.metrics.Inc(MetricExtractionEscalated, map[string]string{"org": itoa(event.OrganizationID)})
		s.escalateRun(ctx, event.OrganizationID, recipient.RunID)
		return nil
	}

	// Run completion: completed when every recipient answered,
	// partially_answered when some answered and the rest timed_out/failed.
	if err := s.evaluateCompletion(ctx, event.OrganizationID, recipient.RunID); err != nil {
		return err
	}
	return nil
}

// escalateRun transitions the run to escalated (allowed from any in-progress
// state, agent-governance parity) and audits it.
func (s *ProcurementSubscriber) escalateRun(ctx context.Context, orgID, runID int32) {
	for _, from := range []domain.RunStatus{domain.RunAwaitingResponses, domain.RunSending} {
		if _, err := s.runs.TransitionRun(ctx, orgID, runID, from, domain.RunEscalated); err == nil {
			break
		}
	}
	_ = s.audit.Record(ctx, domain.AuditEntry{
		OrganizationID: orgID,
		EntityType:     "inquiry_run",
		EntityID:       &runID,
		Action:         "escalate",
		Decision:       "skip",
		Reason:         strPtr2("requiere_humano"),
	})
}

func (s *ProcurementSubscriber) escalate(ctx context.Context, orgID, runID int32, reason string) {
	s.escalateRun(ctx, orgID, runID)
	s.log.Warn("procurement: extraction escalated", map[string]any{"run_id": runID, "reason": reason})
}

// evaluateCompletion mirrors the board service's completion evaluation for
// the extraction path.
func (s *ProcurementSubscriber) evaluateCompletion(ctx context.Context, orgID, runID int32) error {
	recipients, err := s.runs.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return err
	}
	total := len(recipients)
	if total == 0 {
		return nil
	}
	answered := 0
	terminal := 0
	for _, r := range recipients {
		switch r.Status {
		case domain.RecipientAnswered:
			answered++
		case domain.RecipientTimedOut, domain.RecipientFailed:
			terminal++
		}
	}
	switch {
	case answered == total:
		_, err = s.runs.TransitionRun(ctx, orgID, runID, domain.RunAwaitingResponses, domain.RunCompleted)
	case answered+terminal == total && answered > 0:
		_, err = s.runs.TransitionRun(ctx, orgID, runID, domain.RunAwaitingResponses, domain.RunPartiallyAnswered)
	default:
		err = nil
	}
	if err != nil && errors.Is(err, domain.ErrInvalidTransition) {
		return nil
	}
	return err
}
