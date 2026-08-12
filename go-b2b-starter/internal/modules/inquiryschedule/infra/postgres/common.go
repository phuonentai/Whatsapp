// Package postgres implements the inquiry-scheduling domain ports over SQLC,
// consuming the sibling procurement tables (runs, recipients, suppliers,
// products, audit_log) through generated queries — never raw SQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
)

// isNoRows reports whether err is pgx.ErrNoRows (single-row query miss).
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// transactioner is implemented by *sqlc.SQLStore (gen/exec.go) and lets
// repositories compose queries atomically. Non-transactional stores run the
// function directly (test fakes).
type transactioner interface {
	Transaction(ctx context.Context, fn func(sqlc.Store) error) error
}

func inTx(ctx context.Context, store sqlc.Store, fn func(sqlc.Store) error) error {
	if t, ok := store.(transactioner); ok {
		return t.Transaction(ctx, fn)
	}
	return fn(store)
}

// pgTimeToRunTime converts pgtype.Time (microseconds since midnight) to the
// strict "HH:MM" form used by the domain.
func pgTimeToRunTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	totalSec := t.Microseconds / 1_000_000
	hh := totalSec / 3600
	mm := (totalSec % 3600) / 60
	return fmt.Sprintf("%02d:%02d", hh, mm)
}

// runTimeToPgTime converts a strict "HH:MM" string to pgtype.Time.
func runTimeToPgTime(runTime string) (pgtype.Time, error) {
	var hh, mm int
	if _, err := fmt.Sscanf(runTime, "%d:%d", &hh, &mm); err != nil {
		return pgtype.Time{}, fmt.Errorf("invalid run time %q", runTime)
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return pgtype.Time{}, fmt.Errorf("invalid run time %q", runTime)
	}
	return pgtype.Time{Microseconds: int64((hh*3600 + mm*60) * 1_000_000), Valid: true}, nil
}

// mapDays converts a SMALLINT[] column to the domain DaysOfWeek.
func mapDays(days []int16) domain.DaysOfWeek {
	out := make(domain.DaysOfWeek, 0, len(days))
	for _, d := range days {
		out = append(out, domain.DayOfWeek(d))
	}
	return out
}

func daysToInt16(days domain.DaysOfWeek) []int16 {
	out := make([]int16, 0, len(days))
	for _, d := range days {
		out = append(out, int16(d))
	}
	return out
}

// mapSchedule converts a SQLC schedule row to the domain aggregate. Product
// and supplier ids are loaded separately (join rows).
func mapSchedule(row sqlc.ProcurementSchedule) *domain.Schedule {
	return &domain.Schedule{
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
}

// mapScheduleFollowup converts a SQLC follow-up row to the domain value.
func mapScheduleFollowup(row sqlc.ProcurementScheduleFollowup) *domain.FollowUpSettings {
	return &domain.FollowUpSettings{
		OrganizationID:  row.OrganizationID,
		Enabled:         row.Enabled,
		DeadlineHours:   int(row.DeadlineHours),
		MaxNudges:       int(row.MaxNudges),
		MessageTemplate: row.MessageTemplate,
	}
}

func strPtr(s string) *string { return &s }

func i32Ptr(v int32) *int32 { return &v }

// pgInt8Ptr converts a nullable BIGINT to a *int64.
func pgInt8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

// loadLocation resolves the org timezone string with a safe default.
func loadLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return loc
}
