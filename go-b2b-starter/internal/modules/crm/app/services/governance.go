package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

// ManualSendGovernance is the shared seam for the inbox tier: every manual
// send (member with inbox:reply OR admin) SHALL traverse the agent-governance
// guardrails (kill switch, discount cap, forbidden terms, escalation terms,
// consent, send window, daily limit) with the decision recorded in the audit
// ledger — no role exemption.
//
// Implemented by the agent module (which owns the guardrail service, settings,
// usage counters and audit writer); the CRM module depends only on this
// interface (dependency inversion, no import cycle: agent already depends on
// the CRM outbound service).

type ManualSendGovernance interface {
	// GovernManualSend evaluates the send_message guardrails for a manual send
	// and, when allowed, delivers the message through the standard outbound
	// path, returning the created message. When the send is NOT performed the
	// guardrail denial reasons are returned (the denial is already audited);
	// err is non-nil only for infrastructure failures (audit write, settings
	// fetch, delivery).
	GovernManualSend(ctx context.Context, orgID, convID int32, content, actorID string) (deniedReasons []string, msg *domain.Message, err error)
}
