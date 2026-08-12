package domain

import (
	"context"
	"strings"
	"testing"
	"time"
)

func bogotaLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatalf("load America/Bogota: %v", err)
	}
	return loc
}

func at(t *testing.T, loc *time.Location, y int, m time.Month, d, hh, mm int) time.Time {
	t.Helper()
	return time.Date(y, m, d, hh, mm, 0, 0, loc)
}

// newSchedule builds a valid schedule (Mon-Fri 08:00, one product, one
// supplier) with the given overrides.
func newSchedule(override func(*Schedule)) *Schedule {
	s := &Schedule{
		OrganizationID: 1,
		Name:           "Cotización diaria",
		RunTime:        "08:00",
		DaysOfWeek:     DaysOfWeek{Monday, Tuesday, Wednesday, Thursday, Friday},
		ProductIDs:     []int32{11},
		SupplierIDs:    []int32{21},
		IsActive:       true,
	}
	if override != nil {
		override(s)
	}
	return s
}

// Anchor weekdays (all verified): 2026-08-10 Mon, 08-12 Wed, 08-13 Thu,
// 08-14 Fri, 08-15 Sat, 08-16 Sun, 08-17 Mon, 08-23 Sun.
func TestNextOccurrenceAfter(t *testing.T) {
	loc := bogotaLoc(t)
	cases := []struct {
		name string
		s    *Schedule
		now  time.Time
		want time.Time
	}{
		{
			name: "same day before run time",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 12, 7, 0), // Wed 07:00
			want: at(t, loc, 2026, time.August, 12, 8, 0), // Wed 08:00
		},
		{
			name: "same day after run time rolls to tomorrow",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 12, 8, 30), // Wed 08:30
			want: at(t, loc, 2026, time.August, 13, 8, 0),  // Thu 08:00
		},
		{
			name: "exactly at run time is strictly after",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 12, 8, 0), // Wed 08:00:00
			want: at(t, loc, 2026, time.August, 13, 8, 0), // Thu 08:00
		},
		{
			name: "friday afternoon skips weekend",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 14, 12, 0), // Fri 12:00
			want: at(t, loc, 2026, time.August, 17, 8, 0),  // Mon 08:00
		},
		{
			name: "saturday rolls to monday",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 15, 10, 0), // Sat 10:00
			want: at(t, loc, 2026, time.August, 17, 8, 0),  // Mon 08:00
		},
		{
			name: "sunday only scans full week",
			s: newSchedule(func(s *Schedule) {
				s.DaysOfWeek = DaysOfWeek{Sunday}
			}),
			now:  at(t, loc, 2026, time.August, 16, 9, 0), // Sun 09:00
			want: at(t, loc, 2026, time.August, 23, 8, 0), // next Sun 08:00
		},
		{
			name: "monday morning same day",
			s:    newSchedule(nil),
			now:  at(t, loc, 2026, time.August, 10, 7, 0), // Mon 07:00
			want: at(t, loc, 2026, time.August, 10, 8, 0), // Mon 08:00
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.s.NextOccurrenceAfter(tc.now, loc)
			if err != nil {
				t.Fatalf("NextOccurrenceAfter: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
			if got.Location() != loc {
				t.Fatalf("result in %s, want %s", got.Location(), loc)
			}
		})
	}
}

func TestNextOccurrenceAfterInvalidSpec(t *testing.T) {
	loc := bogotaLoc(t)
	now := at(t, loc, 2026, time.August, 12, 7, 0)
	if _, err := newSchedule(func(s *Schedule) { s.RunTime = "" }).NextOccurrenceAfter(now, loc); err == nil {
		t.Fatal("expected error for empty run_time")
	}
	if _, err := newSchedule(func(s *Schedule) { s.RunTime = "8:00" }).NextOccurrenceAfter(now, loc); err == nil {
		t.Fatal("expected error for malformed run_time")
	}
	if _, err := newSchedule(func(s *Schedule) { s.DaysOfWeek = nil }).NextOccurrenceAfter(now, loc); err == nil {
		t.Fatal("expected error for empty days_of_week")
	}
}

func TestScheduleValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Schedule)
		wantMsg string
	}{
		{"missing run_time", func(s *Schedule) { s.RunTime = "" }, "hora de ejecución es requerida"},
		{"malformed run_time", func(s *Schedule) { s.RunTime = "8:00" }, "formato HH:MM"},
		{"empty days", func(s *Schedule) { s.DaysOfWeek = nil }, "al menos un día de la semana"},
		{"day out of range", func(s *Schedule) { s.DaysOfWeek = DaysOfWeek{Monday, 7} }, "entre 0 (domingo) y 6 (sábado)"},
		{"duplicate days", func(s *Schedule) { s.DaysOfWeek = DaysOfWeek{Monday, Monday} }, "no pueden repetirse"},
		{"no products", func(s *Schedule) { s.ProductIDs = nil }, "al menos un producto"},
		{"no suppliers", func(s *Schedule) { s.SupplierIDs = nil }, "al menos un proveedor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSchedule(tc.mutate)
			err := s.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error %q does not contain %q", err, tc.wantMsg)
			}
		})
	}

	t.Run("valid schedule passes", func(t *testing.T) {
		if err := newSchedule(nil).Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

type fakeCatalog struct {
	products  map[int32]bool
	suppliers map[int32]bool
	err       error
}

func (f *fakeCatalog) ProductMembership(_ context.Context, _ int32, ids []int32) (map[int32]bool, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int32]bool, len(ids))
	for _, id := range ids {
		out[id] = f.products[id]
	}
	return out, nil
}

func (f *fakeCatalog) SupplierMembership(_ context.Context, _ int32, ids []int32) (map[int32]bool, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int32]bool, len(ids))
	for _, id := range ids {
		out[id] = f.suppliers[id]
	}
	return out, nil
}

func TestValidateOrgScope(t *testing.T) {
	ctx := context.Background()
	all := &fakeCatalog{
		products:  map[int32]bool{11: true, 12: true},
		suppliers: map[int32]bool{21: true},
	}
	t.Run("all references belong to org", func(t *testing.T) {
		s := newSchedule(func(s *Schedule) { s.ProductIDs = []int32{11, 12} })
		if err := s.ValidateOrgScope(ctx, all); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("foreign product rejected with Spanish message", func(t *testing.T) {
		s := newSchedule(func(s *Schedule) { s.ProductIDs = []int32{11, 99} })
		err := s.ValidateOrgScope(ctx, all)
		if err == nil {
			t.Fatal("expected org-scope error")
		}
		for _, want := range []string{"productos", "99"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not mention %q", err, want)
			}
		}
	})
	t.Run("foreign supplier rejected", func(t *testing.T) {
		s := newSchedule(func(s *Schedule) { s.SupplierIDs = []int32{21, 77} })
		err := s.ValidateOrgScope(ctx, all)
		if err == nil {
			t.Fatal("expected org-scope error")
		}
		if !strings.Contains(err.Error(), "proveedores") {
			t.Fatalf("error %q does not mention proveedores", err)
		}
	})
	t.Run("catalog failure propagates", func(t *testing.T) {
		broken := &fakeCatalog{err: context.DeadlineExceeded}
		s := newSchedule(nil)
		if _, ok := s.ValidateOrgScope(ctx, broken).(error); !ok {
			t.Fatal("expected catalog error to propagate")
		}
	})
	t.Run("nil catalog is a no-op", func(t *testing.T) {
		s := newSchedule(nil)
		if err := s.ValidateOrgScope(ctx, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
