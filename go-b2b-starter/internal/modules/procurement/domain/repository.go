package domain

import (
	"context"
	"encoding/json"
	"time"
)

// ContactInput carries the CRM contact fields created for a supplier
// (D11: NIT persona jurídica, org-declared consent granted).
type ContactInput struct {
	PhoneNumber string
	DisplayName string
	NIT         string
}

// SupplierRepository provides org-scoped supplier CRUD. Create is
// transactional: it creates the linked CRM contact (with consent granted) and
// audits supplier_created + consent_grant in the same transaction.
type SupplierRepository interface {
	Create(ctx context.Context, orgID int32, supplier *Supplier, contact ContactInput, memberID string) (*Supplier, error)
	GetByID(ctx context.Context, orgID, id int32) (*Supplier, error)
	GetByContactID(ctx context.Context, orgID, contactID int32) (*Supplier, error)
	List(ctx context.Context, orgID int32, limit, offset int32) ([]*Supplier, error)
	ListActive(ctx context.Context, orgID int32) ([]*Supplier, error)
	ListByIDs(ctx context.Context, orgID int32, ids []int32) ([]*Supplier, error)
	Update(ctx context.Context, orgID int32, supplier *Supplier) (*Supplier, error)
}

// ProductRepository provides org-scoped product CRUD.
type ProductRepository interface {
	Create(ctx context.Context, orgID int32, product *Product) (*Product, error)
	GetByID(ctx context.Context, orgID, id int32) (*Product, error)
	List(ctx context.Context, orgID int32, limit, offset int32) ([]*Product, error)
	ListByIDs(ctx context.Context, orgID int32, ids []int32) ([]*Product, error)
	Update(ctx context.Context, orgID int32, product *Product) (*Product, error)
}

// SupplierWithDisplay pairs a supplier with its contact display name (the
// greeting name allowed through the business-identity allowlist, D11).
type SupplierWithDisplay struct {
	Supplier    *Supplier
	DisplayName string
}

// RecipientWithPhone pairs a recipient with its contact phone (fan-out target).
type RecipientWithPhone struct {
	Recipient    *InquiryRecipient
	ContactPhone string
}

// InquiryRunRepository owns runs, recipients, and responses with
// transaction-isolated, conditional state transitions (idempotent under
// outbox/webhook redelivery).
type InquiryRunRepository interface {
	CreateRun(ctx context.Context, orgID int32, nota *string, memberID string) (*InquiryRun, error)
	GetRun(ctx context.Context, orgID, runID int32) (*InquiryRun, error)
	ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*InquiryRun, error)
	// TransitionRun applies a guarded status change: only when current == from.
	// Returns ErrInvalidTransition when the guard fails.
	TransitionRun(ctx context.Context, orgID, runID int32, from, to RunStatus) (*InquiryRun, error)

	CreateRecipient(ctx context.Context, orgID, runID, supplierID, contactID int32, draftedMessage *string) (*InquiryRecipient, error)
	GetRecipient(ctx context.Context, orgID, recipientID int32) (*InquiryRecipient, error)
	ListRunRecipients(ctx context.Context, orgID, runID int32) ([]*InquiryRecipient, error)
	// ListSuppliersWithDisplay returns suppliers joined with contact display
	// names (drafting greeting).
	ListSuppliersWithDisplay(ctx context.Context, orgID int32, ids []int32) ([]SupplierWithDisplay, error)
	// ListRunRecipientsWithPhone returns recipients joined with contact phones
	// (fan-out target).
	ListRunRecipientsWithPhone(ctx context.Context, orgID, runID int32) ([]RecipientWithPhone, error)
	// MarkRecipientSent transitions pending -> sent with the provider message
	// id; a second dispatch is a no-op returning ErrRecipientNotPending.
	MarkRecipientSent(ctx context.Context, orgID, recipientID int32, providerMessageID string) (*InquiryRecipient, error)
	// MarkRecipientAnswered transitions pending|sent -> answered.
	MarkRecipientAnswered(ctx context.Context, orgID, recipientID int32) (*InquiryRecipient, error)
	// MarkRecipientTimedOut transitions sent -> timed_out (lazy timeout).
	MarkRecipientTimedOut(ctx context.Context, orgID, recipientID int32) (*InquiryRecipient, error)
	// MarkRecipientFailed transitions pending|sent -> failed.
	MarkRecipientFailed(ctx context.Context, orgID, recipientID int32) (*InquiryRecipient, error)

	// ListActiveRecipientsByPhone is the hot inbound lookup shared by the
	// procurement subscriber and the agent skip check (tenant-scoped).
	ListActiveRecipientsByPhone(ctx context.Context, orgID int32, phoneNumber string) ([]*InquiryRecipient, error)
	// ListAwaitingRecipients returns sent-but-unanswered recipients of a run.
	ListAwaitingRecipients(ctx context.Context, orgID, runID int32) ([]*InquiryRecipient, error)
	// ListExpiredSentRecipients returns recipients sent before the window
	// (lazy read-time timeout reconciliation, D12).
	ListExpiredSentRecipients(ctx context.Context, orgID, runID int32, windowHours int32) ([]*InquiryRecipient, error)

	// SaveResponse persists a structured extraction idempotently on
	// (recipient_id, raw_message_id); a duplicate returns ErrDuplicateResponse.
	SaveResponse(ctx context.Context, resp *InquiryResponse) (*InquiryResponse, error)
	// GetResponseByRecipientMessage is the redelivery pre-check.
	GetResponseByRecipientMessage(ctx context.Context, recipientID int32, rawMessageID string) (*InquiryResponse, error)
	// ListRunResponses returns all responses of a run (board feed).
	ListRunResponses(ctx context.Context, orgID, runID int32) ([]*InquiryResponse, error)

	// RunBoardRows loads the raw board feed; deterministic ranking happens in
	// the app layer (D5).
	RunBoardRows(ctx context.Context, orgID, runID int32) ([]BoardRow, error)

	// SendFanOut atomically transitions the run draft→sending and enqueues one
	// durable outbox event per recipient (no send-without-enqueue or
	// enqueue-without-transition).
	SendFanOut(ctx context.Context, orgID, runID int32, events []OutboxEventInput) (*InquiryRun, error)
}

// OrderRepository owns orders (D13: atomic, idempotent on (run_id, supplier_id)).
type OrderRepository interface {
	// PlaceOrderTx atomically: inserts the order marker, enqueues the
	// confirmation outbox event (unless blocked at placement), creates the
	// negocio + actividad in the default pipeline, links the negocio, and
	// audits order_placed (or the block). A duplicate (run_id, supplier_id)
	// returns ErrOrderAlreadyPlaced.
	PlaceOrderTx(ctx context.Context, in PlaceOrderTxParams) (*Order, error)
	// CreateOrder inserts the order marker; a duplicate (run_id, supplier_id)
	// returns ErrOrderAlreadyPlaced (no-op for retried POSTs).
	CreateOrder(ctx context.Context, order *Order) (*Order, error)
	GetOrderByRunSupplier(ctx context.Context, runID, supplierID int32) (*Order, error)
	GetOrder(ctx context.Context, orgID, orderID int32) (*Order, error)
	MarkOrderConfirmSent(ctx context.Context, orgID, orderID int32) (*Order, error)
	MarkOrderSendBlocked(ctx context.Context, orgID, orderID int32, reason string) (*Order, error)
	MarkOrderConfirmFailed(ctx context.Context, orgID, orderID int32) (*Order, error)
	ListRunOrders(ctx context.Context, orgID, runID int32) ([]*Order, error)
}

// AuditEntry is one append-only procurement audit row.
type AuditEntry struct {
	OrganizationID int32
	EntityType     string
	EntityID       *int32
	Action         string
	Decision       string // allow | deny | skip
	Reason         *string
	MemberID       *string
	Metadata       map[string]any
}

// AuditRepository records append-only audit entries.
type AuditRepository interface {
	Record(ctx context.Context, entry AuditEntry) error
}

// InquiryStateReader is the minimal tenant-scoped state seam consumed by the
// agent skip check (task 10): "is this sender an active run recipient?".
type InquiryStateReader interface {
	// IsActiveRecipientByPhone reports whether the phone belongs to a
	// recipient of a run in sending/awaiting_responses.
	IsActiveRecipientByPhone(ctx context.Context, orgID int32, phoneNumber string) (bool, error)
}

// RawExtraction is the JSON contract the drafting/extraction LLM calls
// return. Drafting: {"message": "..."}; extraction: the quote schema.
type RawExtraction struct {
	Message *string `json:"message,omitempty"`
}

// ContactRef is the minimal tenant-scoped contact read used by dispatch-time
// guards (D14): consent status and the last-message window for the
// outside_24h_window warning.
type ContactRef struct {
	ID            int32
	PhoneNumber   string
	ConsentStatus string
	LastMessageAt *time.Time
}

// ContactLookup provides tenant-scoped contact reads.
type ContactLookup interface {
	ContactByID(ctx context.Context, orgID, contactID int32) (*ContactRef, error)
}

// OutboxEventInput is a durable outbox event to enqueue inside a
// procurement transaction (fan-out / order confirmation).
type OutboxEventInput struct {
	EventType string
	Payload   json.RawMessage
}

// PlaceOrderTxParams carries the atomic order-placement transaction (D13).
type PlaceOrderTxParams struct {
	Order             *Order
	ConfirmEvent      *OutboxEventInput // nil when the send is blocked at placement
	DealNombre        string
	ActividadAsunto   string
	ActividadContenido string
}

// MarshalOrderItems renders order items for JSONB storage.
func MarshalOrderItems(items []OrderItem) (json.RawMessage, error) {
	if items == nil {
		items = []OrderItem{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	return b, nil
}
