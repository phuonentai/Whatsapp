package guardrails

import (
	"context"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
)

func mustSettings(mutate func(*domain.AgentSettings)) domain.AgentSettings {
	s := domain.DefaultSettings(42)
	if mutate != nil {
		mutate(&s)
	}
	return s
}

func evaluate(t *testing.T, s domain.AgentSettings, input domain.GuardrailInput) domain.GuardrailDecision {
	t.Helper()
	svc := NewGuardrailService()
	dec, err := svc.Evaluate(context.Background(), 42, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return dec
}

func TestEscalateHumanAlwaysAllowed(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) { s.KillSwitch = true })
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:   domain.GuardrailActionEscalate,
		Settings: settings,
	})
	if !dec.Allowed {
		t.Fatalf("escalate_human must always be allowed, got %v", dec.Reasons)
	}
}

func TestKillSwitchBlocksSend(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) { s.KillSwitch = true })
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:   domain.GuardrailActionSendMessage,
		Settings: settings,
		Draft:    "Hola",
	})
	if dec.Allowed {
		t.Fatal("kill switch must deny sends")
	}
	assertReason(t, dec, ReasonKillSwitch)
}

func TestDiscountAboveCapDeniesSend(t *testing.T) {
	settings := mustSettings(nil) // default cap 10%
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:   domain.GuardrailActionSendMessage,
		Settings: settings,
		Draft:    "Te ofrecemos un 12% de descuento",
	})
	if dec.Allowed {
		t.Fatal("discount above cap must deny")
	}
	assertReason(t, dec, ReasonDiscountCap)
}

func TestDiscountWithinCapAllowsSend(t *testing.T) {
	settings := mustSettings(nil)
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:   domain.GuardrailActionSendMessage,
		Settings: settings,
		Draft:    "Te ofrecemos un 8,5% de descuento",
	})
	if !dec.Allowed {
		t.Fatalf("discount within cap must allow, got %v", dec.Reasons)
	}
}

func TestForbiddenTermDeniesSend(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.Guardrails.Never.ForbiddenTerms = []string{"garantía total"}
	})
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:   domain.GuardrailActionSendMessage,
		Settings: settings,
		Draft:    "Incluimos garantía total en tu compra",
	})
	if dec.Allowed {
		t.Fatal("forbidden term must deny")
	}
	assertReason(t, dec, ReasonForbiddenTerm)
}

func TestEscalateTermNeverSendsAutonomously(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.Guardrails.Escalate.Terms = []string{"abogado", "legal"}
	})
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:     domain.GuardrailActionSendMessage,
		Settings:   settings,
		Draft:      "Te contacto con un abogado para revisarlo",
		Autonomous: true,
	})
	if dec.Allowed {
		t.Fatal("escalate match must deny autonomous send")
	}
	assertReason(t, dec, ReasonEscalationMatch)
}

func TestConsentRequiredDeniesAutonomousSend(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = true
		s.AutopilotStart = "00:00"
		s.AutopilotEnd = "23:59"
	})
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:     domain.GuardrailActionSendMessage,
		Settings:   settings,
		Autonomous: true,
		Contact:    domain.ContactFacts{ConsentStatus: domain.ConsentRequested},
		Now:        time.Now(),
	})
	if dec.Allowed {
		t.Fatal("missing consent must deny autonomous send")
	}
	assertReason(t, dec, ReasonConsentRequired)
}

func TestOutOfWindowDeniesAutonomousSend(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = false
		s.AutopilotStart = "09:00"
		s.AutopilotEnd = "17:00"
		s.Timezone = "UTC"
	})
	atNoon := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	atMidnight := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	if dec := evaluate(t, settings, domain.GuardrailInput{
		Action: domain.GuardrailActionSendMessage, Settings: settings, Autonomous: true, Now: atNoon,
	}); !dec.Allowed {
		t.Fatalf("noon must be inside 09:00-17:00, got %v", dec.Reasons)
	}
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action: domain.GuardrailActionSendMessage, Settings: settings, Autonomous: true, Now: atMidnight,
	})
	if dec.Allowed {
		t.Fatal("midnight must be outside 09:00-17:00")
	}
	assertReason(t, dec, ReasonOutsideWindow)
}

func TestOvernightWindow(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = false
		s.AutopilotStart = "21:00"
		s.AutopilotEnd = "07:00"
		s.Timezone = "UTC"
	})
	at23 := time.Date(2026, 8, 8, 23, 0, 0, 0, time.UTC)
	at06 := time.Date(2026, 8, 8, 6, 0, 0, 0, time.UTC)
	at10 := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		now time.Time
		ok  bool
	}{{at23, true}, {at06, true}, {at10, false}} {
		dec := evaluate(t, settings, domain.GuardrailInput{
			Action: domain.GuardrailActionSendMessage, Settings: settings, Autonomous: true, Now: tt.now,
		})
		if dec.Allowed != tt.ok {
			t.Fatalf("window %v at %v: want allowed=%v, got %v", settings.AutopilotStart, tt.now, tt.ok, dec.Reasons)
		}
	}
}

func TestDailyLimitDeniesAutonomousSend(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = false
		s.AutopilotStart = "00:00"
		s.AutopilotEnd = "23:59"
		s.MaxDailyMessages = 5
	})
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:     domain.GuardrailActionSendMessage,
		Settings:   settings,
		Autonomous: true,
		SentToday:  5,
		Now:        time.Now(),
	})
	if dec.Allowed {
		t.Fatal("daily limit reached must deny")
	}
	assertReason(t, dec, ReasonDailyLimit)
}

func TestHumanApprovalBypassesWindowLimitAndConsent(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = true
		s.AutopilotStart = "09:00"
		s.AutopilotEnd = "17:00"
		s.MaxDailyMessages = 1
	})
	// Human-approved send at 23:00, daily limit exhausted, no consent: allow.
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:    domain.GuardrailActionSendMessage,
		Settings:  settings,
		Autonomous: false,
		SentToday: 10,
		Contact:   domain.ContactFacts{ConsentStatus: domain.ConsentNone},
		Now:       time.Date(2026, 8, 8, 23, 0, 0, 0, time.UTC),
	})
	if !dec.Allowed {
		t.Fatalf("human approval must bypass autonomous checks, got %v", dec.Reasons)
	}
}

func TestUnknownActionFailsSafe(t *testing.T) {
	settings := mustSettings(nil)
	svc := NewGuardrailService()
	dec, err := svc.Evaluate(context.Background(), 42, domain.GuardrailInput{
		Action:   "explode_things",
		Settings: settings,
	})
	if err == nil {
		t.Fatal("unknown action must return an error (fail-safe)")
	}
	if dec.Allowed {
		t.Fatal("unknown action must not be allowed")
	}
	assertReason(t, dec, ReasonGuardrailError)
}

func TestInvalidWindowFailsSafe(t *testing.T) {
	settings := mustSettings(func(s *domain.AgentSettings) {
		s.ConsentRequired = false
		s.AutopilotStart = "99:99"
		s.AutopilotEnd = "07:00"
	})
	dec := evaluate(t, settings, domain.GuardrailInput{
		Action:     domain.GuardrailActionSendMessage,
		Settings:   settings,
		Autonomous: true,
		Now:        time.Now(),
	})
	if dec.Allowed {
		t.Fatal("invalid window must not allow autonomous sends")
	}
	assertReason(t, dec, ReasonOutsideWindow)
}

func assertReason(t *testing.T, dec domain.GuardrailDecision, want string) {
	t.Helper()
	for _, r := range dec.Reasons {
		if r == want {
			return
		}
	}
	t.Fatalf("expected reason %q, got %v", want, dec.Reasons)
}
