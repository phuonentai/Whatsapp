// Package domain contains the inquiry-scheduling domain: org-scoped recurring
// schedules that durably create inquiry runs through the supplier-inquiries
// capability, and per-recipient follow-ups with a configurable deadline.
//
// Pure Go by design (repo constitution): no Stytch SDKs and no transport
// packages are imported here. Persistence, ticking and send-side effects live
// in infrastructure adapters implementing the ports in ports.go.
package domain

import "errors"

// Sentinel errors for lookup failures.
var (
	// ErrScheduleNotFound is returned when a schedule does not exist for the
	// organization.
	ErrScheduleNotFound = errors.New("schedule not found")

	// ErrFollowUpSettingsNotFound is returned when no follow-up settings row
	// exists (callers should fall back to DefaultFollowUpSettings instead).
	ErrFollowUpSettingsNotFound = errors.New("follow-up settings not found")

	// ErrRecipientNotFound is returned when a run recipient does not exist
	// for the organization.
	ErrRecipientNotFound = errors.New("recipient not found")

	// ErrCreditsExhausted is returned by the drafting seam when the org's AI
	// credits are exhausted (scheduled runs escalate instead of burning
	// unmetered tokens).
	ErrCreditsExhausted = errors.New("AI credits exhausted")
)

// ValidationError is a field-level validation failure. Message carries the
// user-facing Spanish copy (repo convention: Spanish-first UI).
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
