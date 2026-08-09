package guardrails

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
)

// Reason strings recorded in the audit ledger.
const (
	ReasonKillSwitch        = "kill_switch"
	ReasonDiscountCap       = "discount_exceeds_cap"
	ReasonForbiddenTerm     = "forbidden_term"
	ReasonEscalationMatch   = "escalation_match"
	ReasonOutsideWindow     = "outside_send_window"
	ReasonDailyLimit        = "daily_limit_reached"
	ReasonConsentRequired   = "consent_required"
	ReasonGuardrailError    = "guardrail_error"
)

var discountPattern = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*%`)

// guardrailService implements the deterministic, data-defined guardrail
// rules from the agent-governance spec. Pure logic, no dependencies:
// all tunables come from domain.AgentSettings (parameters-as-data).
type guardrailService struct{}

// NewGuardrailService creates the guardrail evaluation service.
func NewGuardrailService() domain.GuardrailService {
	return &guardrailService{}
}

func (g *guardrailService) Evaluate(ctx context.Context, orgID int32, input domain.GuardrailInput) (domain.GuardrailDecision, error) {
	switch input.Action {
	case domain.GuardrailActionEscalate:
		// Escalation must never be blocked: a human can always be reached.
		return domain.GuardrailDecision{Allowed: true}, nil
	case domain.GuardrailActionGenerateDraft:
		// Drafting has no side effect; only send_message is governable.
		return domain.GuardrailDecision{Allowed: true}, nil
	case domain.GuardrailActionSendMessage:
		return g.evaluateSend(ctx, orgID, input), nil
	default:
		return domain.GuardrailDecision{Allowed: false, Reasons: []string{ReasonGuardrailError}}, fmt.Errorf("unknown guardrail action %q", input.Action)
	}
}

func (g *guardrailService) evaluateSend(ctx context.Context, orgID int32, input domain.GuardrailInput) domain.GuardrailDecision {
	dec := domain.GuardrailDecision{Allowed: true}

	// Hard: kill switch blocks ALL sends, including human approvals.
	if input.Settings.KillSwitch {
		dec.Allowed = false
		dec.Reasons = append(dec.Reasons, ReasonKillSwitch)
		return dec
	}

	// Never rules: deterministic draft checks, deny always wins.
	if input.Settings.Guardrails.Never != nil {
		if cap := input.Settings.Guardrails.Never.MaxDiscountPercent; cap != nil && *cap > 0 {
			if parsed, ok := maxPercentInDraft(input.Draft); ok && parsed > *cap {
				dec.Allowed = false
				dec.Reasons = append(dec.Reasons, ReasonDiscountCap)
				return dec
			}
		}
		for _, term := range input.Settings.Guardrails.Never.ForbiddenTerms {
			if term != "" && strings.Contains(strings.ToLower(input.Draft), strings.ToLower(term)) {
				dec.Allowed = false
				dec.Reasons = append(dec.Reasons, ReasonForbiddenTerm)
				return dec
			}
		}
	}

	// Escalate rules: matching topics must go to a human, never autonomously.
	if input.Settings.Guardrails.Escalate != nil {
		for _, term := range input.Settings.Guardrails.Escalate.Terms {
			if term != "" && strings.Contains(strings.ToLower(input.Draft), strings.ToLower(term)) {
				dec.Allowed = false
				dec.Reasons = append(dec.Reasons, ReasonEscalationMatch)
				return dec
			}
		}
	}

	// Autonomous-only hard checks (human approvals bypass window/limit/consent).
	if input.Autonomous {
		if input.Settings.ConsentRequired && input.Contact.ConsentStatus != domain.ConsentGranted {
			dec.Allowed = false
			dec.Reasons = append(dec.Reasons, ReasonConsentRequired)
			return dec
		}
		if !withinWindow(input.Settings, input.Now) {
			dec.Allowed = false
			dec.Reasons = append(dec.Reasons, ReasonOutsideWindow)
			return dec
		}
		if input.Settings.MaxDailyMessages > 0 && input.SentToday >= int64(input.Settings.MaxDailyMessages) {
			dec.Allowed = false
			dec.Reasons = append(dec.Reasons, ReasonDailyLimit)
			return dec
		}
	}

	return dec
}

// maxPercentInDraft returns the largest percentage value found in a draft.
func maxPercentInDraft(draft string) (float64, bool) {
	matches := discountPattern.FindAllStringSubmatch(draft, -1)
	if len(matches) == 0 {
		return 0, false
	}
	max := 0.0
	for _, m := range matches {
		v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		if err != nil {
			continue
		}
		if v > max {
			max = v
		}
	}
	return max, true
}

// withinWindow reports whether t falls inside the configured autopilot window.
// Missing bounds mean "no window constraint". Timezone failures fall back to
// UTC (documented; the window rule still applies in that case).
func withinWindow(settings domain.AgentSettings, t time.Time) bool {
	if settings.AutopilotStart == "" || settings.AutopilotEnd == "" {
		return true
	}
	loc := time.UTC
	if settings.Timezone != "" {
		if l, err := time.LoadLocation(settings.Timezone); err == nil {
			loc = l
		}
	}
	now := t.In(loc)
	start, err1 := parseHHMM(settings.AutopilotStart)
	end, err2 := parseHHMM(settings.AutopilotEnd)
	if err1 != nil || err2 != nil {
		return false
	}
	nowMinutes := now.Hour()*60 + now.Minute()
	if start == end {
		return true
	}
	if start < end {
		return nowMinutes >= start && nowMinutes < end
	}
	// Overnight window (e.g. 21:00–07:00).
	return nowMinutes >= start || nowMinutes < end
}

func parseHHMM(v string) (int, error) {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time %q", v)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid time %q", v)
	}
	return h*60 + m, nil
}
