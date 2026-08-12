package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// followUpService detects overdue unanswered recipients and enqueues exactly
// one inquiry.followup_send per nudge through the atomic nudge guard (the
// double-nudge guard for sweep/reply races and dispatcher redelivery).
// Consent gating: only 'granted' recipients are nudged; 'withdrawn' escalates
// with a skip audit. The kill switch cancels all nudges with a skip audit.
type FollowUpService struct {
	readers  domain.RecipientStateReader
	enqueuer domain.FollowUpEnqueuer
	kill     domain.KillSwitchReader
	audit    domain.AuditLogWriter
	log      logger.Logger
}

// NewFollowUpService builds the follow-up service.
func NewFollowUpService(
	readers domain.RecipientStateReader,
	enqueuer domain.FollowUpEnqueuer,
	kill domain.KillSwitchReader,
	audit domain.AuditLogWriter,
	log logger.Logger,
) *FollowUpService {
	return &FollowUpService{readers: readers, enqueuer: enqueuer, kill: kill, audit: audit, log: log}
}

// SweepOrg runs the periodic candidate scan for one organization.
func (s *FollowUpService) SweepOrg(ctx context.Context, orgID int32, limit int32) (int, error) {
	candidates, err := s.readers.ListFollowUpCandidates(ctx, orgID, limit)
	if err != nil {
		return 0, err
	}
	nudged := 0
	for _, c := range candidates {
		ok, err := s.nudge(ctx, c)
		if err != nil {
			return nudged, err
		}
		if ok {
			nudged++
		}
	}
	return nudged, nil
}

// OnReplyArrival runs the cheap reply-trigger check: a whatsapp.message.received
// event answered the contact's recipients; nudge the run's OTHER overdue
// recipients, excluding the just-answered ones (the sibling subscriber marks
// them answered concurrently, so the exclusion is conservative).
func (s *FollowUpService) OnReplyArrival(ctx context.Context, orgID int32, phoneNumber string) error {
	answering, err := s.readers.ActiveRecipientsByPhone(ctx, orgID, phoneNumber)
	if err != nil {
		return err
	}
	if len(answering) == 0 {
		return nil
	}
	excluded := make(map[int32]bool, len(answering))
	seenRuns := make(map[int32]bool, len(answering))
	for _, r := range answering {
		excluded[r.RecipientID] = true
	}
	for _, r := range answering {
		if seenRuns[r.RunID] {
			continue
		}
		seenRuns[r.RunID] = true
		candidates, err := s.readers.ListOverdueRecipientsForRun(ctx, orgID, r.RunID)
		if err != nil {
			return err
		}
		for _, c := range candidates {
			if excluded[c.RecipientID] {
				continue
			}
			if _, err := s.nudge(ctx, c); err != nil {
				return err
			}
		}
	}
	return nil
}

// nudge enqueues one follow-up send for a candidate with the atomic guard.
// Returns (enqueued, error); enqueued=false when the candidate is already at
// the nudge cap (escalated to a human), the guard was at the cap (another
// path already nudged to the limit), or a governed skip.
func (s *FollowUpService) nudge(ctx context.Context, c *domain.FollowUpCandidate) (bool, error) {
	// Defense-in-depth: the candidate query filters followup_count < max;
	// re-check here so an at-cap candidate is never nudged (escalated).
	if c.FollowupCount >= c.MaxNudges {
		return false, nil
	}

	// Kill switch re-check (fail-safe direction).
	killOn, err := s.kill.IsKillSwitchEnabled(ctx, c.OrganizationID)
	if err != nil {
		return false, err
	}
	if killOn {
		if err := s.audit.Record(ctx, domain.AuditEvent{
			OrganizationID: c.OrganizationID,
			EntityType:     "inquiry_recipient",
			EntityID:       &c.RecipientID,
			Action:         "skip",
			Reason:         strPtrOrNil("kill_switch"),
			Metadata:       map[string]any{"run_id": c.RunID},
		}); err != nil {
			s.log.Error("audit skip/kill_switch failed", map[string]any{"error": err.Error()})
		}
		return false, nil
	}

	// Consent gating (Ley 1581): only granted recipients are nudged.
	if c.ConsentStatus != "granted" {
		if err := s.audit.Record(ctx, domain.AuditEvent{
			OrganizationID: c.OrganizationID,
			EntityType:     "inquiry_recipient",
			EntityID:       &c.RecipientID,
			Action:         "skip",
			Reason:         strPtrOrNil("consent_withdrawn"),
			Metadata:       map[string]any{"run_id": c.RunID, "consent": c.ConsentStatus},
		}); err != nil {
			s.log.Error("audit skip/consent_withdrawn failed", map[string]any{"error": err.Error()})
		}
		// Flag for human escalation: the overdue badge is computed at read
		// time (recipient 'sent' with followup_count >= max_nudges is
		// surfaced as overdue; no further nudges are enqueued).
		return false, nil
	}

	// The Spanish template with [proveedor] replaced by the supplier name.
	message := strings.ReplaceAll(c.MessageTemplate, "[proveedor]", c.SupplierDisplayName)
	nudgeIndex := c.FollowupCount + 1
	ev := inqEvents.NewFollowupSend(
		c.RunID, c.OrganizationID, c.SupplierID, c.ContactID, c.RecipientID,
		c.ContactPhone, message, nudgeIndex,
	)
	payload, err := json.Marshal(ev)
	if err != nil {
		return false, err
	}

	ok, err := s.enqueuer.EnqueueNudge(ctx, c.OrganizationID, c.RecipientID, c.MaxNudges, domain.OutboxEventInput{
		EventType: inqEvents.FollowupSendEventType,
		Payload:   payload,
	})
	if err != nil {
		return false, err
	}
	if !ok {
		// Double-nudge guard: a concurrent sweep/reply path already nudged
		// this recipient to the cap. At most max_nudges sends per recipient.
		s.log.Info("follow-up nudge skipped (guard at cap)", map[string]any{
			"recipient_id": c.RecipientID, "run_id": c.RunID,
		})
		return false, nil
	}
	// When the nudge reaches max_nudges the recipient is flagged for human
	// escalation (overdue badge) — surfaced by the status queries.
	return true, nil
}
