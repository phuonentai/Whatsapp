package agent

import "context"

// ActiveInquiryChecker lets the agent pipeline skip analysis for messages
// consumed by an active supplier inquiry run (procurement independent
// subscriber, D7). Implemented by the procurement module and injected at the
// composition root; the agent module never imports procurement.
type ActiveInquiryChecker interface {
	// IsActiveRecipientByPhone reports whether the phone belongs to a
	// recipient of a run in sending/awaiting_responses, tenant-scoped by the
	// event's organization id.
	IsActiveRecipientByPhone(ctx context.Context, organizationID int32, phoneNumber string) (bool, error)
}
