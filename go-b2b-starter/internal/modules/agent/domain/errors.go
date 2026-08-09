package domain

import "errors"

// ErrFlowNotFound is returned when a flow does not exist for an organization.
var ErrFlowNotFound = errors.New("flow not found")

// ErrSettingsNotFound is returned when no agent settings row exists.
var ErrSettingsNotFound = errors.New("agent settings not found")

// ErrSuggestionNotFound is returned when a suggestion is missing or already
// resolved.
var ErrSuggestionNotFound = errors.New("suggestion not found")

// ErrContactNotFound is returned when a contact cannot be resolved.
var ErrContactNotFound = errors.New("contact not found")

// ErrConversationNotFound is returned when a conversation cannot be resolved.
var ErrConversationNotFound = errors.New("conversation not found")

// ErrActionDenied is returned by the pipeline when the guardrail layer denied
// an action. The denial is already audited.
var ErrActionDenied = errors.New("agent action denied by guardrails")

// ErrNodeNotFound is returned when a node is not claimable (already claimed
// or missing) for a flow.
var ErrNodeNotFound = errors.New("node not found")

// ErrNodeNotRunnable is returned when a node cannot be claimed because its
// dependencies are not satisfied.
var ErrNodeNotRunnable = errors.New("node not runnable")

// ErrNoCreditsRemaining is returned when the organization has exhausted its
// AI credit allowance.
var ErrNoCreditsRemaining = errors.New("ai credits exhausted")

// ErrNodeSkipped is returned by a node handler to signal the node must be
// marked skipped without retries or escalation (e.g. AI credits exhausted).
var ErrNodeSkipped = errors.New("agent node skipped")
