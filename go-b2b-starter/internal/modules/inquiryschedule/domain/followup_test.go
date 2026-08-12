package domain

import (
	"strings"
	"testing"
)

func TestDefaultFollowUpSettings(t *testing.T) {
	d := DefaultFollowUpSettings(7)
	if d.OrganizationID != 7 {
		t.Fatalf("org id = %d, want 7", d.OrganizationID)
	}
	if d.Enabled {
		t.Fatal("follow-ups must default to disabled (opt-in)")
	}
	if d.DeadlineHours != DefaultFollowUpDeadlineHours {
		t.Fatalf("deadline = %d, want %d", d.DeadlineHours, DefaultFollowUpDeadlineHours)
	}
	if d.MaxNudges != DefaultFollowUpMaxNudges {
		t.Fatalf("max_nudges = %d, want %d", d.MaxNudges, DefaultFollowUpMaxNudges)
	}
	if strings.TrimSpace(d.MessageTemplate) == "" {
		t.Fatal("default template must not be empty")
	}
	if !strings.Contains(d.MessageTemplate, "[proveedor]") {
		t.Fatalf("default template %q must contain the [proveedor] placeholder", d.MessageTemplate)
	}
}

func TestFollowUpSettingsValidate(t *testing.T) {
	valid := func() FollowUpSettings {
		return FollowUpSettings{
			OrganizationID:  7,
			Enabled:         true,
			DeadlineHours:   4,
			MaxNudges:       1,
			MessageTemplate: DefaultFollowUpTemplate,
		}
	}
	cases := []struct {
		name    string
		mutate  func(*FollowUpSettings)
		wantMsg string
	}{
		{"deadline zero", func(f *FollowUpSettings) { f.DeadlineHours = 0 }, "entre 1 y 168"},
		{"deadline above max", func(f *FollowUpSettings) { f.DeadlineHours = 169 }, "entre 1 y 168"},
		{"max_nudges above max", func(f *FollowUpSettings) { f.MaxNudges = 9 }, "entre 0 y 5"},
		{"max_nudges negative", func(f *FollowUpSettings) { f.MaxNudges = -1 }, "entre 0 y 5"},
		{"enabled without template", func(f *FollowUpSettings) { f.MessageTemplate = "" }, "mensaje de recordatorio es requerido"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := valid()
			tc.mutate(&f)
			err := f.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error %q does not contain %q", err, tc.wantMsg)
			}
		})
	}
	t.Run("boundaries accepted", func(t *testing.T) {
		for _, d := range []int{1, 168} {
			f := valid()
			f.DeadlineHours = d
			if err := f.Validate(); err != nil {
				t.Fatalf("deadline %d should be valid: %v", d, err)
			}
		}
		for _, n := range []int{0, 5} {
			f := valid()
			f.MaxNudges = n
			if err := f.Validate(); err != nil {
				t.Fatalf("max_nudges %d should be valid: %v", n, err)
			}
		}
	})
	t.Run("disabled without template is valid", func(t *testing.T) {
		f := valid()
		f.Enabled = false
		f.MessageTemplate = ""
		if err := f.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
