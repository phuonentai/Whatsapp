package services

import (
	"context"
	"errors"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// followUpSendHandler processes inquiry.followup_send outbox events with
// dispatch-time re-validation (D14 pattern): run active, recipient still
// 'sent', kill switch off, consent granted. Guard failures complete the event
// with a skip audit and no send; send failures return an error so the durable
// dispatcher retries with backoff and dead-letters after max attempts.
type followUpSendHandler struct {
	readers domain.RecipientStateReader
	kill    domain.KillSwitchReader
	sender  domain.FollowUpSender
	audit   domain.AuditLogWriter
	log     logger.Logger
}

// NewFollowUpSendHandler builds the inquiry.followup_send handler.
func NewFollowUpSendHandler(
	readers domain.RecipientStateReader,
	kill domain.KillSwitchReader,
	sender domain.FollowUpSender,
	audit domain.AuditLogWriter,
	log logger.Logger,
) FollowUpSendHandler {
	return &followUpSendHandler{readers: readers, kill: kill, sender: sender, audit: audit, log: log}
}

func (h *followUpSendHandler) HandleFollowUpSend(ctx context.Context, e *inqEvents.FollowupSend) error {
	target, err := h.readers.GetFollowUpTarget(ctx, e.OrganizationID, e.RecipientID)
	if errors.Is(err, domain.ErrRecipientNotFound) {
		return nil // already gone/processed
	}
	if err != nil {
		return err
	}

	// Run must still be awaiting replies.
	if target.RunStatus != "sending" && target.RunStatus != "awaiting_responses" {
		return h.skip(ctx, e, "run_not_active", nil)
	}
	// Recipient must still be unanswered.
	if target.RecipientStatus != "sent" {
		return nil // answered/timed out/failed: nothing to send
	}
	// Kill switch re-check inside the dispatch path (fail-safe direction).
	killOn, err := h.kill.IsKillSwitchEnabled(ctx, e.OrganizationID)
	if err != nil {
		return err
	}
	if killOn {
		return h.skip(ctx, e, "kill_switch", nil)
	}
	// Consent re-check: withdrawn consent never sends.
	if target.ConsentStatus != "granted" {
		return h.skip(ctx, e, "consent_withdrawn", nil)
	}

	msg := e.Message
	if msg == "" {
		msg = target.MessageTemplate
	}
	if _, err := h.sender.SendFollowUp(ctx, e.OrganizationID, &domain.FollowUpSend{
		RunID:          e.RunID,
		OrganizationID: e.OrganizationID,
		SupplierID:     e.SupplierID,
		ContactID:      e.ContactID,
		RecipientID:    e.RecipientID,
		ContactPhone:   e.ContactPhone,
		Message:        msg,
		NudgeIndex:     e.NudgeIndex,
	}); err != nil {
		// The durable dispatcher retries with backoff, then dead-letters.
		return err
	}
	return nil
}

// skip completes the event with a skip audit (no send).
func (h *followUpSendHandler) skip(ctx context.Context, e *inqEvents.FollowupSend, reason string, _ map[string]any) error {
	if err := h.audit.Record(ctx, domain.AuditEvent{
		OrganizationID: e.OrganizationID,
		EntityType:     "inquiry_recipient",
		EntityID:       &e.RecipientID,
		Action:         "skip",
		Reason:         strPtrOrNil(reason),
		Metadata:       map[string]any{"run_id": e.RunID, "nudge_index": e.NudgeIndex},
	}); err != nil {
		h.log.Error("audit followup skip failed", map[string]any{"error": err.Error()})
	}
	return nil
}
