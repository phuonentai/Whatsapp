package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
)

type scheduleRepository struct {
	store sqlc.Store
	clock domain.Clock
}

// NewScheduleRepository builds the schedule + follow-up-settings repository.
func NewScheduleRepository(store sqlc.Store, clock domain.Clock) *scheduleRepository {
	return &scheduleRepository{store: store, clock: clock}
}

// ------------- ScheduleRepository -------------

func (r *scheduleRepository) Create(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	var created *domain.Schedule
	err := inTx(ctx, r.store, func(tx sqlc.Store) error {
		runTime, err := runTimeToPgTime(s.RunTime)
		if err != nil {
			return err
		}
		row, err := tx.InsertSchedule(ctx, sqlc.InsertScheduleParams{
			OrganizationID: orgID,
			Name:           s.Name,
			RunTime:        runTime,
			DaysOfWeek:     daysToInt16(s.DaysOfWeek),
			Note:           helpers.ToPgText(s.Note),
			IsActive:       s.IsActive,
			NextRunAt:      helpers.ToPgTimestamptz(s.NextRunAt),
		})
		if err != nil {
			return err
		}
		created = mapSchedule(row)
		created.ProductIDs = append([]int32(nil), s.ProductIDs...)
		created.SupplierIDs = append([]int32(nil), s.SupplierIDs...)
		return r.replaceJoinRows(ctx, tx, orgID, row.ID, s.ProductIDs, s.SupplierIDs)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *scheduleRepository) Get(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	row, err := r.store.GetSchedule(ctx, sqlc.GetScheduleParams{ID: id, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.withJoinRows(ctx, orgID, mapSchedule(row))
}

func (r *scheduleRepository) GetForUpdate(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	row, err := r.store.GetScheduleForUpdate(ctx, sqlc.GetScheduleForUpdateParams{ID: id, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.withJoinRows(ctx, orgID, mapSchedule(row))
}

func (r *scheduleRepository) List(ctx context.Context, orgID int32) ([]*domain.Schedule, error) {
	rows, err := r.store.ListSchedulesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Schedule, 0, len(rows))
	for i := range rows {
		s, err := r.withJoinRows(ctx, orgID, mapSchedule(rows[i]))
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *scheduleRepository) Update(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	var updated *domain.Schedule
	err := inTx(ctx, r.store, func(tx sqlc.Store) error {
		runTime, err := runTimeToPgTime(s.RunTime)
		if err != nil {
			return err
		}
		row, err := tx.UpdateSchedule(ctx, sqlc.UpdateScheduleParams{
			ID:             s.ID,
			OrganizationID: orgID,
			Name:           s.Name,
			RunTime:        runTime,
			DaysOfWeek:     daysToInt16(s.DaysOfWeek),
			Note:           helpers.ToPgText(s.Note),
			NextRunAt:      helpers.ToPgTimestamptz(s.NextRunAt),
		})
		if isNoRows(err) {
			return domain.ErrScheduleNotFound
		}
		if err != nil {
			return err
		}
		updated = mapSchedule(row)
		updated.ProductIDs = append([]int32(nil), s.ProductIDs...)
		updated.SupplierIDs = append([]int32(nil), s.SupplierIDs...)
		return r.replaceJoinRows(ctx, tx, orgID, row.ID, s.ProductIDs, s.SupplierIDs)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *scheduleRepository) Delete(ctx context.Context, orgID, id int32) error {
	if err := r.store.DeleteSchedule(ctx, sqlc.DeleteScheduleParams{ID: id, OrganizationID: orgID}); err != nil {
		return err
	}
	return nil
}

func (r *scheduleRepository) Pause(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	row, err := r.store.PauseSchedule(ctx, sqlc.PauseScheduleParams{ID: id, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.withJoinRows(ctx, orgID, mapSchedule(row))
}

func (r *scheduleRepository) Resume(ctx context.Context, orgID, id int32, nextRunAt time.Time) (*domain.Schedule, error) {
	row, err := r.store.ResumeSchedule(ctx, sqlc.ResumeScheduleParams{
		ID:             id,
		OrganizationID: orgID,
		NextRunAt:      helpers.ToPgTimestamptz(nextRunAt),
	})
	if isNoRows(err) {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.withJoinRows(ctx, orgID, mapSchedule(row))
}

func (r *scheduleRepository) MarkFiredOccurrence(ctx context.Context, orgID, id int32, occurrence time.Time) (*domain.Schedule, error) {
	row, err := r.store.UpdateScheduleLastRun(ctx, sqlc.UpdateScheduleLastRunParams{
		ID:                  id,
		OrganizationID:      orgID,
		LastRunAt:           helpers.ToPgTimestamptzPtr(&occurrence),
		LastRunOccurrenceAt: helpers.ToPgTimestamptzPtr(&occurrence),
	})
	if isNoRows(err) {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapSchedule(row), nil
}

// ClaimDue returns due schedules claimed atomically (FOR UPDATE SKIP LOCKED).
// Locked rows are held until the surrounding transaction commits; use
// ClaimAndAdvanceAndEnqueue for the full claim+advance+enqueue cycle.
func (r *scheduleRepository) ClaimDue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	rows, err := r.store.ClaimDueSchedules(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Schedule, 0, len(rows))
	for i := range rows {
		sch := mapSchedule(rows[i])
		sch.ProductIDs, sch.SupplierIDs, err = r.joinRowIDs(ctx, r.store, sch.ID, sch.OrganizationID)
		if err != nil {
			return nil, err
		}
		out = append(out, sch)
	}
	return out, nil
}

// ClaimAndAdvanceAndEnqueue claims up to limit due schedules and, for each,
// computes+persists the next next_run_at and enqueues the
// inquiry_run.scheduled outbox event — all inside ONE transaction so the
// FOR UPDATE SKIP LOCKED claim holds until commit (a crash before commit
// rolls back and a later tick re-fires; at-least-once at the tick level).
func (r *scheduleRepository) ClaimAndAdvanceAndEnqueue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	var claimed []*domain.Schedule
	err := inTx(ctx, r.store, func(tx sqlc.Store) error {
		rows, err := tx.ClaimDueSchedules(ctx, limit)
		if err != nil {
			return err
		}
		now := r.clock.Now()
		for i := range rows {
			sch := mapSchedule(rows[i])
			sch.ProductIDs, sch.SupplierIDs, err = r.joinRowIDs(ctx, tx, sch.ID, sch.OrganizationID)
			if err != nil {
				return err
			}
			tzName, err := tx.GetOrgTimezone(ctx, sch.OrganizationID)
			if err != nil {
				return err
			}
			next, err := sch.NextOccurrenceAfter(now, loadLocation(tzName))
			if err != nil {
				// No valid next occurrence (e.g. config drift): skip and keep
				// the schedule at its current next_run_at; log.
				continue
			}
			if _, err := tx.SetNextRunAt(ctx, sqlc.SetNextRunAtParams{
				ID:             sch.ID,
				OrganizationID: sch.OrganizationID,
				NextRunAt:      helpers.ToPgTimestamptz(next),
			}); err != nil {
				return err
			}
			ev := inqEvents.NewInquiryRunScheduled(sch.ID, sch.OrganizationID, sch.ProductIDs, sch.SupplierIDs, sch.Note, now)
			payload, err := json.Marshal(ev)
			if err != nil {
				return err
			}
			if _, err := tx.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
				EventType:      inqEvents.InquiryRunScheduledEventType,
				Payload:        payload,
				OrganizationID: helpers.ToPgInt4Ptr(&sch.OrganizationID),
			}); err != nil {
				return fmt.Errorf("enqueue %s: %w", inqEvents.InquiryRunScheduledEventType, err)
			}
			claimed = append(claimed, sch)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r *scheduleRepository) ListWithStatus(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error) {
	rows, err := r.store.ListSchedulesWithStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ScheduleStatus, 0, len(rows))
	for i := range rows {
		row := rows[i]
		sch := &domain.Schedule{
			ID:                  row.ID,
			OrganizationID:      row.OrganizationID,
			Name:                row.Name,
			RunTime:             pgTimeToRunTime(row.RunTime),
			DaysOfWeek:          mapDays(row.DaysOfWeek),
			Note:                helpers.FromPgText(row.Note),
			IsActive:            row.IsActive,
			NextRunAt:           row.NextRunAt.Time,
			LastRunAt:           helpers.FromPgTimestamptzPtr(row.LastRunAt),
			LastRunOccurrenceAt: helpers.FromPgTimestamptzPtr(row.LastRunOccurrenceAt),
			CreatedAt:           row.CreatedAt.Time,
			UpdatedAt:           row.UpdatedAt.Time,
		}
		sch.ProductIDs, sch.SupplierIDs, err = r.joinRowIDs(ctx, r.store, sch.ID, orgID)
		if err != nil {
			return nil, err
		}
		status := &domain.ScheduleStatus{Schedule: sch}
		if row.LastRunStatus != "" {
			status.HasLastRun = true
			status.LastRunID = row.LastRunID
			status.LastRunStatus = row.LastRunStatus
			if row.LastRunCreatedAt.Valid {
				status.LastRunAt = &row.LastRunCreatedAt.Time
			}
		}
		out = append(out, status)
	}
	return out, nil
}

func (r *scheduleRepository) RecentRuns(ctx context.Context, orgID, id int32, limit int32) ([]*domain.ScheduledRun, error) {
	rows, err := r.store.ListScheduleRuns(ctx, sqlc.ListScheduleRunsParams{
		OrganizationID: orgID,
		ScheduleRef:    pgtype.Int8{Int64: int64(id), Valid: true},
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ScheduledRun, 0, len(rows))
	for i := range rows {
		row := rows[i]
		out = append(out, &domain.ScheduledRun{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			Status:         row.Status,
			Source:         row.Source,
			ScheduleRef:    pgInt8Ptr(row.ScheduleRef),
			Nota:           helpers.FromPgTextPtr(row.Nota),
			CreatedAt:      row.CreatedAt.Time,
		})
	}
	return out, nil
}

func (r *scheduleRepository) CountOverdueRecipients(ctx context.Context, orgID, id int32) (int32, error) {
	count, err := r.store.CountOverdueRecipientsForSchedule(ctx, sqlc.CountOverdueRecipientsForScheduleParams{
		OrganizationID: orgID,
		ScheduleRef:    pgtype.Int8{Int64: int64(id), Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// ------------- FollowUpSettingsRepository -------------

func (r *scheduleRepository) GetByOrg(ctx context.Context, orgID int32) (*domain.FollowUpSettings, error) {
	row, err := r.store.GetFollowUpSettingsByOrg(ctx, orgID)
	if isNoRows(err) {
		return nil, domain.ErrFollowUpSettingsNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapScheduleFollowup(row), nil
}

func (r *scheduleRepository) Upsert(ctx context.Context, settings *domain.FollowUpSettings) (*domain.FollowUpSettings, error) {
	row, err := r.store.UpsertFollowUpSettings(ctx, sqlc.UpsertFollowUpSettingsParams{
		OrganizationID:  settings.OrganizationID,
		Enabled:         settings.Enabled,
		DeadlineHours:   int32(settings.DeadlineHours),
		MaxNudges:       int32(settings.MaxNudges),
		MessageTemplate: settings.MessageTemplate,
	})
	if err != nil {
		return nil, err
	}
	return mapScheduleFollowup(row), nil
}

// ------------- join rows -------------

func (r *scheduleRepository) replaceJoinRows(ctx context.Context, tx sqlc.Store, orgID, scheduleID int32, productIDs, supplierIDs []int32) error {
	if err := tx.DeleteScheduleProducts(ctx, sqlc.DeleteScheduleProductsParams{ScheduleID: scheduleID, OrganizationID: orgID}); err != nil {
		return err
	}
	if err := tx.DeleteScheduleSuppliers(ctx, sqlc.DeleteScheduleSuppliersParams{ScheduleID: scheduleID, OrganizationID: orgID}); err != nil {
		return err
	}
	for _, pid := range productIDs {
		if err := tx.InsertScheduleProduct(ctx, sqlc.InsertScheduleProductParams{
			OrganizationID: orgID, ScheduleID: scheduleID, ProductID: pid,
		}); err != nil {
			return err
		}
	}
	for _, sid := range supplierIDs {
		if err := tx.InsertScheduleSupplier(ctx, sqlc.InsertScheduleSupplierParams{
			OrganizationID: orgID, ScheduleID: scheduleID, SupplierID: sid,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *scheduleRepository) joinRowIDs(ctx context.Context, store sqlc.Store, scheduleID, orgID int32) (productIDs, supplierIDs []int32, err error) {
	prods, err := store.ListScheduleProducts(ctx, sqlc.ListScheduleProductsParams{ScheduleID: scheduleID, OrganizationID: orgID})
	if err != nil {
		return nil, nil, err
	}
	for _, p := range prods {
		productIDs = append(productIDs, p.ProductID)
	}
	sups, err := store.ListScheduleSuppliers(ctx, sqlc.ListScheduleSuppliersParams{ScheduleID: scheduleID, OrganizationID: orgID})
	if err != nil {
		return nil, nil, err
	}
	for _, s := range sups {
		supplierIDs = append(supplierIDs, s.SupplierID)
	}
	return productIDs, supplierIDs, nil
}

func (r *scheduleRepository) withJoinRows(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	productIDs, supplierIDs, err := r.joinRowIDs(ctx, r.store, s.ID, orgID)
	if err != nil {
		return nil, err
	}
	s.ProductIDs = productIDs
	s.SupplierIDs = supplierIDs
	return s, nil
}
