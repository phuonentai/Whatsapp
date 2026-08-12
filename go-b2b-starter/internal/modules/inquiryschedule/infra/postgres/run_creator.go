package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// runCreator implements the handler-side idempotent scheduled run creation:
// ONE transaction that locks the schedule row FOR UPDATE, skips when the
// occurrence was already fired, creates the run (source='scheduled',
// schedule_ref), drafts one metered message per supplier, stores one
// recipient per supplier, records the fired occurrence, and audits — so a
// redelivered inquiry_run.scheduled event can never create a second run.
type runCreator struct {
	store sqlc.Store
	draft domain.DraftFunc
	log   logger.Logger
}

// NewRunCreator builds the InquiryRunCreator adapter.
func NewRunCreator(store sqlc.Store, draft domain.DraftFunc, log logger.Logger) domain.InquiryRunCreator {
	return &runCreator{store: store, draft: draft, log: log}
}

func (c *runCreator) CreateScheduledRun(ctx context.Context, in domain.CreateScheduledRunInput) (*domain.ScheduledRunResult, error) {
	var result domain.ScheduledRunResult
	err := inTx(ctx, c.store, func(s sqlc.Store) error {
		row, err := s.GetScheduleForUpdate(ctx, sqlc.GetScheduleForUpdateParams{
			ID:             in.ScheduleID,
			OrganizationID: in.OrganizationID,
		})
		if isNoRows(err) {
			return domain.ErrScheduleNotFound
		}
		if err != nil {
			return err
		}
		schedule := mapSchedule(row)

		// Dedupe: a run for (schedule_id, occurrence_at) already exists.
		if schedule.LastRunOccurrenceAt != nil && schedule.LastRunOccurrenceAt.Equal(in.OccurrenceAt) {
			result.Skipped = true
			return auditRecord(ctx, s, in.OrganizationID, "schedule", &in.ScheduleID, "skip",
				strPtr("duplicate_occurrence"), map[string]any{
					"schedule_id":   in.ScheduleID,
					"occurrence_at": in.OccurrenceAt,
				})
		}

		runRow, err := s.InsertScheduledRun(ctx, sqlc.InsertScheduledRunParams{
			OrganizationID: in.OrganizationID,
			ScheduleRef:    pgtype.Int8{Int64: int64(in.ScheduleID), Valid: true},
			Nota:           helpers.ToPgTextPtr(strPtr(in.Note)),
		})
		if err != nil {
			return err
		}
		result.RunID = runRow.ID

		if c.draft != nil {
			escalated, err := c.draftAndCreateRecipients(ctx, s, in, runRow.ID)
			if err != nil {
				return err
			}
			result.Escalated = escalated
			if escalated {
				// Mark the occurrence as fired anyway: the run exists
				// (escalated) and must not be re-created on redelivery.
				if _, err := s.UpdateScheduleLastRun(ctx, lastRunParams(in)); err != nil {
					return err
				}
				return auditRecord(ctx, s, in.OrganizationID, "schedule", &in.ScheduleID, "inquiry_run_scheduled",
					strPtr("allow"), map[string]any{
						"schedule_id":   in.ScheduleID,
						"run_id":        runRow.ID,
						"occurrence_at": in.OccurrenceAt,
						"escalated":     true,
					})
			}
		}

		if _, err := s.UpdateScheduleLastRun(ctx, lastRunParams(in)); err != nil {
			return err
		}
		return auditRecord(ctx, s, in.OrganizationID, "schedule", &in.ScheduleID, "inquiry_run_scheduled",
			strPtr("allow"), map[string]any{
				"schedule_id":   in.ScheduleID,
				"run_id":        runRow.ID,
				"occurrence_at": in.OccurrenceAt,
			})
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// draftAndCreateRecipients drafts one message per supplier and stores one
// recipient per supplier. On credit exhaustion or drafting failure the run is
// escalated (draft -> escalated) with an audit and NO further unmetered calls
// happen; the escalated run is returned. Returns (escalated, error).
func (c *runCreator) draftAndCreateRecipients(ctx context.Context, s sqlc.Store, in domain.CreateScheduledRunInput, runID int32) (bool, error) {
	suppliers, err := s.ListSuppliersWithDisplay(ctx, sqlc.ListSuppliersWithDisplayParams{
		OrganizationID: in.OrganizationID,
		Column2:        in.SupplierIDs,
	})
	if err != nil {
		return false, err
	}
	products, err := s.ListProductsByIDs(ctx, sqlc.ListProductsByIDsParams{
		OrganizationID: in.OrganizationID,
		Column2:        in.ProductIDs,
	})
	if err != nil {
		return false, err
	}
	draftProducts := make([]domain.DraftProduct, 0, len(products))
	for _, p := range products {
		draftProducts = append(draftProducts, domain.DraftProduct{ID: p.ID, Name: p.Name, SKU: p.Sku, Unit: p.Unit})
	}

	escalate := func(cause error) error {
		if _, tErr := s.UpdateRunStatusFrom(ctx, sqlc.UpdateRunStatusFromParams{
			ID:             runID,
			OrganizationID: in.OrganizationID,
			Status:         "draft",
			Status_2:       "escalated",
		}); tErr != nil {
			c.log.Error("escalate scheduled run failed", map[string]any{"run_id": runID, "error": tErr.Error()})
		}
		if aErr := auditRecord(ctx, s, in.OrganizationID, "inquiry_run", &runID, "escalate",
			strPtr("skip"), map[string]any{"reason": cause.Error()}); aErr != nil {
			c.log.Error("audit scheduled escalate failed", map[string]any{"error": aErr.Error()})
		}
		return nil
	}

	for _, sup := range suppliers {
		if !sup.IsActive {
			return true, escalate(fmt.Errorf("supplier %d inactive", sup.ID))
		}
		msg, err := c.draft(ctx, in.OrganizationID, domain.DraftSupplier{
			ID:          sup.ID,
			NIT:         sup.Nit,
			DisplayName: sup.DisplayName,
			ContactID:   sup.ContactID,
			IsActive:    sup.IsActive,
		}, draftProducts)
		if err != nil {
			if errors.Is(err, domain.ErrCreditsExhausted) {
				return true, escalate(domain.ErrCreditsExhausted)
			}
			return true, escalate(fmt.Errorf("drafting failed: %w", err))
		}
		if _, err := s.CreateInquiryRecipient(ctx, sqlc.CreateInquiryRecipientParams{
			OrganizationID: in.OrganizationID,
			RunID:          runID,
			SupplierID:     sup.ID,
			ContactID:      sup.ContactID,
			DraftedMessage: helpers.ToPgTextPtr(strPtr(msg)),
		}); err != nil {
			return false, err
		}
	}
	return false, nil
}

func lastRunParams(in domain.CreateScheduledRunInput) sqlc.UpdateScheduleLastRunParams {
	return sqlc.UpdateScheduleLastRunParams{
		ID:                  in.ScheduleID,
		OrganizationID:      in.OrganizationID,
		LastRunAt:           helpers.ToPgTimestamptzPtr(&in.OccurrenceAt),
		LastRunOccurrenceAt: helpers.ToPgTimestamptzPtr(&in.OccurrenceAt),
	}
}
