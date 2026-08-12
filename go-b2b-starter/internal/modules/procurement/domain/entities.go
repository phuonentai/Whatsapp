// Package domain holds the procurement entities and state machines.
// Constitution rule: no Stytch SDK or transport imports here.
package domain

import (
	"time"
)

// RunStatus is the inquiry run lifecycle status (mirrors agent-governance
// flow semantics).
type RunStatus string

const (
	RunDraft             RunStatus = "draft"
	RunSending           RunStatus = "sending"
	RunAwaitingResponses RunStatus = "awaiting_responses"
	RunCompleted         RunStatus = "completed"
	RunPartiallyAnswered RunStatus = "partially_answered"
	RunFailed            RunStatus = "failed"
	RunEscalated         RunStatus = "escalated"
	RunCancelled         RunStatus = "cancelled"
)

// ValidRunStatus reports whether s is a legal run status.
func ValidRunStatus(s RunStatus) bool {
	switch s {
	case RunDraft, RunSending, RunAwaitingResponses, RunCompleted,
		RunPartiallyAnswered, RunFailed, RunEscalated, RunCancelled:
		return true
	}
	return false
}

// RecipientStatus is the per-supplier send state machine.
type RecipientStatus string

const (
	RecipientPending  RecipientStatus = "pending"
	RecipientSent     RecipientStatus = "sent"
	RecipientAnswered RecipientStatus = "answered"
	RecipientTimedOut RecipientStatus = "timed_out"
	RecipientFailed   RecipientStatus = "failed"
)

// OrderStatus is the order placement/send state.
type OrderStatus string

const (
	OrderPlaced       OrderStatus = "placed"
	OrderConfirmSent  OrderStatus = "confirm_sent"
	OrderSendBlocked  OrderStatus = "send_blocked"
	OrderConfirmFailed OrderStatus = "confirm_failed"
)

// Supplier is a procurement supplier; a supplier IS a CRM contact (NIT
// persona jurídica), so identity, PII, consent, and conversation history all
// live on crm.contacts. DisplayName/PhoneNumber are populated by the list
// view (joined from the contact).
type Supplier struct {
	ID             int32
	OrganizationID int32
	ContactID      int32
	NIT            string
	DeliveryDays   *int32
	MinOrderAmount *float64
	Notes          *string
	IsActive       bool
	DisplayName    string
	PhoneNumber    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Product is an org-scoped SKU catalog entry. Deactivation keeps history.
type Product struct {
	ID             int32
	OrganizationID int32
	Name           string
	SKU            string
	Unit           string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InquiryRun is the flow row for a procurement run (source = 'manual' in this
// change; schedule_ref reserved for the future scheduling change).
type InquiryRun struct {
	ID                int32
	OrganizationID    int32
	Status            RunStatus
	Source            string
	ScheduleRef       *int64
	Nota              *string
	CreatedByMemberID *string
	SentAt            *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CanTransition validates the run state machine. Escalation is ALWAYS allowed
// from any in-progress state (mirrors agent-governance), including when the
// kill switch is on; cancellation is allowed from any in-progress state.
func (r *InquiryRun) CanTransition(next RunStatus) bool {
	if !ValidRunStatus(r.Status) || !ValidRunStatus(next) {
		return false
	}
	switch r.Status {
	case RunDraft:
		return next == RunSending || next == RunEscalated || next == RunCancelled
	case RunSending:
		return next == RunAwaitingResponses || next == RunFailed || next == RunEscalated || next == RunCancelled
	case RunAwaitingResponses:
		return next == RunCompleted || next == RunPartiallyAnswered || next == RunEscalated || next == RunCancelled
	default:
		// Terminal states: no further transitions.
		return false
	}
}

// Escalatable reports whether the run may escalate (always true while the run
// is in-progress, per agent-governance).
func (r *InquiryRun) Escalatable() bool {
	switch r.Status {
	case RunDraft, RunSending, RunAwaitingResponses:
		return true
	}
	return false
}

// InquiryRecipient is one supplier send within a run.
type InquiryRecipient struct {
	ID                int32
	OrganizationID    int32
	RunID             int32
	SupplierID        int32
	ContactID         int32
	DraftedMessage    *string
	Status            RecipientStatus
	ProviderMessageID *string
	SentAt            *time.Time
	AnsweredAt        *time.Time
	FollowupCount     int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ResponseItem is one structured extraction row (quote schema) for a supplier
// reply. Price/moneda are never auto-quoted to customers anywhere.
type ResponseItem struct {
	ProductName          string   `json:"product_name"`
	SKU                  *string  `json:"sku,omitempty"`
	Disponible           bool     `json:"disponible"`
	PrecioUnitario       *float64 `json:"precio_unitario,omitempty"`
	Moneda               string   `json:"moneda,omitempty"`
	CantidadDisponible   *float64 `json:"cantidad_disponible,omitempty"`
	TiempoEntrega        *string  `json:"tiempo_entrega,omitempty"`
	RequiereSeguimiento  bool     `json:"requiere_seguimiento"`
}

// InquiryResponse is the persisted structured extraction, idempotent on
// (recipient_id, raw_message_id).
type InquiryResponse struct {
	ID             int32
	OrganizationID int32
	RecipientID    int32
	RawMessageID   string
	Items          []ResponseItem
	Resumen        string
	Confidence     *float64
	RequiereHumano bool
	CreatedAt      time.Time
}

// OrderItem is a product × quantity line of a placed order.
type OrderItem struct {
	ProductID int32   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
}

// Order is a human-approved order placement (D13: atomic + idempotent).
type Order struct {
	ID                int32
	OrganizationID    int32
	RunID             int32
	SupplierID        int32
	ContactID         int32
	NegocioID         *int32
	Status            OrderStatus
	Items             []OrderItem
	Notes             *string
	ConfirmMessage    *string
	BlockedReason     *string
	CreatedByMemberID *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// BoardRow is one supplier row of the aggregation board, deterministically
// ranked in Go (availability desc, unit price asc, lead time asc).
type BoardRow struct {
	RecipientID     int32
	RecipientStatus RecipientStatus
	SentAt          *time.Time
	AnsweredAt      *time.Time
	ProviderMessageID *string
	SupplierID      int32
	NIT             string
	DeliveryDays    *int32
	MinOrderAmount  *float64
	ContactID       int32
	DisplayName     string
	PhoneNumber     string
	Response        *InquiryResponse
}

// Board is the ranked comparison board for a run.
type Board struct {
	Run     *InquiryRun
	Rows    []BoardRow
	Summary *string
}

// ExtractionResult is the LLM reply-extraction contract.
type ExtractionResult struct {
	Items          []ResponseItem `json:"items"`
	Resumen        string         `json:"resumen"`
	RequiereHumano bool           `json:"requiere_humano"`
}
