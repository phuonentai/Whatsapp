package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// scheduledRunHandler processes inquiry_run.scheduled outbox events: kill
// switch pre-check, then the transaction-isolated idempotent run creation
// (dedupe per occurrence inside the creator), then the supplier-inquiries
// run-creation service. Scheduling itself never invokes an LLM and never
// writes the ai-usage ledger directly (the drafting seam is the sibling's
// metered step; credit exhaustion escalates the run instead).
type scheduledRunHandler struct {
	kill    domain.KillSwitchReader
	creator domain.InquiryRunCreator
	audit   domain.AuditLogWriter
	log     logger.Logger
}

// NewScheduledRunHandler builds the inquiry_run.scheduled handler.
func NewScheduledRunHandler(
	kill domain.KillSwitchReader,
	creator domain.InquiryRunCreator,
	audit domain.AuditLogWriter,
	log logger.Logger,
) ScheduledRunHandler {
	return &scheduledRunHandler{kill: kill, creator: creator, audit: audit, log: log}
}

func (h *scheduledRunHandler) HandleScheduledRun(ctx context.Context, e *inqEvents.InquiryRunScheduled) error {
	killOn, err := h.kill.IsKillSwitchEnabled(ctx, e.OrganizationID)
	if err != nil {
		return err
	}
	if killOn {
		if err := h.audit.Record(ctx, domain.AuditEvent{
			OrganizationID: e.OrganizationID,
			EntityType:     "schedule",
			EntityID:       &e.ScheduleID,
			Action:         "skip",
			Reason:         strPtrOrNil("kill_switch"),
			Metadata:       map[string]any{"occurrence_at": e.OccurrenceAt},
		}); err != nil {
			h.log.Error("audit skip/kill_switch failed", map[string]any{"error": err.Error()})
		}
		return nil
	}

	result, err := h.creator.CreateScheduledRun(ctx, domain.CreateScheduledRunInput{
		OrganizationID: e.OrganizationID,
		ScheduleID:     e.ScheduleID,
		OccurrenceAt:   e.OccurrenceAt,
		ProductIDs:     e.ProductIDs,
		SupplierIDs:    e.SupplierIDs,
		Note:           e.Note,
	})
	if err != nil {
		return err // outbox dispatcher retries with backoff, then dead-letters
	}
	if result.Skipped {
		// duplicate_occurrence: audited inside the creator's transaction.
		h.log.Info("scheduled run skipped (duplicate occurrence)", map[string]any{
			"schedule_id": e.ScheduleID, "occurrence_at": e.OccurrenceAt,
		})
		return nil
	}
	if result.Escalated {
		h.log.Warn("scheduled run created as escalated", map[string]any{
			"schedule_id": e.ScheduleID, "run_id": result.RunID,
		})
	}
	return nil
}
