package domain

import "errors"

// Sentinel errors for the procurement module. Handlers map these to Spanish
// HTTP responses; services use them for control flow.
var (
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrSupplierAlreadyExists = errors.New("supplier already exists for this NIT")
	ErrProductNotFound       = errors.New("product not found")
	ErrRunNotFound           = errors.New("inquiry run not found")
	ErrRecipientNotFound     = errors.New("inquiry recipient not found")
	ErrInvalidTransition     = errors.New("invalid run transition")
	ErrInvalidRunStatus      = errors.New("invalid run status")
	ErrRunNotDraft           = errors.New("run is not in draft status")
	ErrRunNotSendable        = errors.New("run is not in a sendable state")
	ErrNoDraftedMessages     = errors.New("run has no drafted messages")
	ErrCreditsExhausted      = errors.New("AI credits exhausted")
	ErrMalformedLLMResponse  = errors.New("LLM returned malformed response")
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderAlreadyPlaced    = errors.New("order already placed for this supplier")
	ErrResponseNotAnswered   = errors.New("supplier response is not answered")
	ErrResponseRequiresHuman = errors.New("supplier response requires human review")
	ErrKillSwitchEnabled     = errors.New("kill switch enabled")
	ErrConsentWithdrawn      = errors.New("supplier consent withdrawn")
	ErrDuplicateResponse     = errors.New("response already persisted for this message")
	ErrResponseNotFound      = errors.New("response not found")
	ErrContactNotFound       = errors.New("contact not found")
	ErrDefaultPipelineMissing = errors.New("default pipeline not found")
	ErrNoSuppliers           = errors.New("no suppliers selected")
	ErrNoProducts            = errors.New("no products selected")
	ErrSupplierInactive      = errors.New("supplier is inactive")
	ErrResponseWindow        = errors.New("response window expired")
)
