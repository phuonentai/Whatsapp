package domain

import (
	"errors"
	"strings"
)

// Follow-up policy defaults (spec: org row absent => behave as if defaults
// applied). Follow-ups are opt-in: Enabled defaults to false so no
// organization receives automatic nudges before explicitly enabling them.
const (
	DefaultFollowUpDeadlineHours = 4
	DefaultFollowUpMaxNudges     = 1
	DefaultFollowUpTemplate      = "Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos."

	minDeadlineHours = 1
	maxDeadlineHours = 168
	minMaxNudges     = 0
	maxMaxNudges     = 5
)

// FollowUpSettings is the org-level follow-up configuration applied to every
// inquiry run of the organization (scheduled and manual): one row per
// organization in procurement.schedule_followups.
type FollowUpSettings struct {
	OrganizationID  int32
	Enabled         bool
	DeadlineHours   int
	MaxNudges       int
	MessageTemplate string
}

// DefaultFollowUpSettings returns the settings that apply when no row exists
// for the organization: follow-ups disabled with the spec defaults.
func DefaultFollowUpSettings(orgID int32) FollowUpSettings {
	return FollowUpSettings{
		OrganizationID:  orgID,
		DeadlineHours:   DefaultFollowUpDeadlineHours,
		MaxNudges:       DefaultFollowUpMaxNudges,
		MessageTemplate: DefaultFollowUpTemplate,
	}
}

// Validate checks the follow-up policy ranges and returns Spanish validation
// errors (joined).
func (f *FollowUpSettings) Validate() error {
	var errs []error
	if f.DeadlineHours < minDeadlineHours || f.DeadlineHours > maxDeadlineHours {
		errs = append(errs, &ValidationError{
			Field:   "deadline_hours",
			Message: "Las horas de plazo deben estar entre 1 y 168.",
		})
	}
	if f.MaxNudges < minMaxNudges || f.MaxNudges > maxMaxNudges {
		errs = append(errs, &ValidationError{
			Field:   "max_nudges",
			Message: "El número de recordatorios debe estar entre 0 y 5.",
		})
	}
	if f.Enabled && strings.TrimSpace(f.MessageTemplate) == "" {
		errs = append(errs, &ValidationError{
			Field:   "message_template",
			Message: "El mensaje de recordatorio es requerido cuando los recordatorios están habilitados.",
		})
	}
	return errors.Join(errs...)
}
