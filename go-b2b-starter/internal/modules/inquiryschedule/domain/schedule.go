package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DayOfWeek is a weekday in the range 0=Sunday..6=Saturday, matching
// PostgreSQL EXTRACT(DOW) so the domain value maps 1:1 to the
// days_of_week SMALLINT[] column.
type DayOfWeek int

const (
	Sunday DayOfWeek = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

// DaysOfWeek is the ordered set of weekdays a schedule fires on.
type DaysOfWeek []DayOfWeek

// Contains reports whether dow is in the set.
func (d DaysOfWeek) Contains(dow DayOfWeek) bool {
	for _, v := range d {
		if v == dow {
			return true
		}
	}
	return false
}

// Schedule is the org-scoped recurring inquiry-run configuration. A schedule
// fires run_time on every day in DaysOfWeek; next_run_at is computed in the
// organization's timezone at save/claim time.
type Schedule struct {
	ID                  int32
	OrganizationID      int32
	Name                string
	RunTime             string // "HH:MM" (24h), interpreted in the org timezone
	DaysOfWeek          DaysOfWeek
	ProductIDs          []int32
	SupplierIDs         []int32
	Note                string
	IsActive            bool
	NextRunAt           time.Time
	LastRunAt           *time.Time
	LastRunOccurrenceAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// validateRunSpec validates the fields NextOccurrenceAfter depends on:
// run_time presence/format and a non-empty days_of_week. Shared by Validate
// so next-run computation and full validation agree.
func (s *Schedule) validateRunSpec() error {
	if strings.TrimSpace(s.RunTime) == "" {
		return &ValidationError{Field: "run_time", Message: "La hora de ejecución es requerida."}
	}
	if _, _, err := parseRunTime(s.RunTime); err != nil {
		return &ValidationError{Field: "run_time", Message: "La hora de ejecución debe tener el formato HH:MM (ej. 08:00)."}
	}
	if len(s.DaysOfWeek) == 0 {
		return &ValidationError{Field: "days_of_week", Message: "Debe seleccionar al menos un día de la semana."}
	}
	return nil
}

// Validate checks all schedule invariants and returns Spanish validation
// errors (joined). It does not check org-scope of referenced products or
// suppliers; use ValidateOrgScope for that.
func (s *Schedule) Validate() error {
	var errs []error
	if err := s.validateRunSpec(); err != nil {
		errs = append(errs, err)
	}
	seen := make(map[DayOfWeek]bool, len(s.DaysOfWeek))
	for _, dow := range s.DaysOfWeek {
		if dow < Sunday || dow > Saturday {
			errs = append(errs, &ValidationError{Field: "days_of_week", Message: "Los días de la semana deben estar entre 0 (domingo) y 6 (sábado)."})
			break
		}
		if seen[dow] {
			errs = append(errs, &ValidationError{Field: "days_of_week", Message: "Los días de la semana no pueden repetirse."})
			break
		}
		seen[dow] = true
	}
	if len(s.ProductIDs) == 0 {
		errs = append(errs, &ValidationError{Field: "product_ids", Message: "Debe seleccionar al menos un producto."})
	}
	if len(s.SupplierIDs) == 0 {
		errs = append(errs, &ValidationError{Field: "supplier_ids", Message: "Debe seleccionar al menos un proveedor."})
	}
	return errors.Join(errs...)
}

// parseRunTime parses a strict "HH:MM" (24h, zero-padded) string into hour and
// minute. The length/colon check rejects lenient single-digit forms such as
// "8:00" that time.Parse otherwise accepts.
func parseRunTime(runTime string) (hour, minute int, err error) {
	rt := strings.TrimSpace(runTime)
	if len(rt) != 5 || rt[2] != ':' {
		return 0, 0, fmt.Errorf("invalid run time %q", runTime)
	}
	t, err := time.Parse("15:04", rt)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

// NextOccurrenceAfter returns the next occurrence of RunTime on a day in
// DaysOfWeek strictly after now, interpreted in the IANA timezone tz. It never
// backfills: occurrences that fell while the schedule was paused or before the
// last claim are skipped. tz defaults to UTC when nil. DST shifts are handled
// by time.Date normalization; no deeper DST logic is a non-goal of the change.
func (s *Schedule) NextOccurrenceAfter(now time.Time, tz *time.Location) (time.Time, error) {
	if err := s.validateRunSpec(); err != nil {
		return time.Time{}, err
	}
	hh, mm, err := parseRunTime(s.RunTime)
	if err != nil {
		return time.Time{}, err
	}
	if tz == nil {
		tz = time.UTC
	}
	// Scan today plus the next 7 days: any non-empty weekday set has an
	// occurrence strictly after now within that window.
	day := now.In(tz)
	for i := 0; i <= 7; i++ {
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, tz)
		if candidate.After(now) && s.DaysOfWeek.Contains(DayOfWeek(candidate.Weekday())) {
			return candidate, nil
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, fmt.Errorf("no next occurrence found for schedule %d", s.ID)
}

// ValidateOrgScope returns an error when any referenced product or supplier
// does not belong to the schedule's organization, with a Spanish message
// naming the offending ids. The catalog nil-check keeps callers that have no
// catalog wired (e.g. pure next-run tests) working; production always passes
// the sibling-backed CatalogReader adapter.
func (s *Schedule) ValidateOrgScope(ctx context.Context, catalog CatalogReader) error {
	if catalog == nil {
		return nil
	}
	var errs []error
	if len(s.ProductIDs) > 0 {
		membership, err := catalog.ProductMembership(ctx, s.OrganizationID, s.ProductIDs)
		if err != nil {
			return err
		}
		if missing := missingIDs(s.ProductIDs, membership); len(missing) > 0 {
			errs = append(errs, &ValidationError{
				Field:   "product_ids",
				Message: fmt.Sprintf("Los productos con id %s no pertenecen a su organización.", joinIDs(missing)),
			})
		}
	}
	if len(s.SupplierIDs) > 0 {
		membership, err := catalog.SupplierMembership(ctx, s.OrganizationID, s.SupplierIDs)
		if err != nil {
			return err
		}
		if missing := missingIDs(s.SupplierIDs, membership); len(missing) > 0 {
			errs = append(errs, &ValidationError{
				Field:   "supplier_ids",
				Message: fmt.Sprintf("Los proveedores con id %s no pertenecen a su organización.", joinIDs(missing)),
			})
		}
	}
	return errors.Join(errs...)
}

// missingIDs returns the ids whose membership is false or unknown.
func missingIDs(ids []int32, membership map[int32]bool) []int32 {
	var out []int32
	for _, id := range ids {
		if !membership[id] {
			out = append(out, id)
		}
	}
	return out
}

func joinIDs(ids []int32) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ", ")
}
