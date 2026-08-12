// Package events defines the procurement durable outbox event payloads.
// The outbox dispatcher decodes each payload into an eventbus event and the
// procurement module subscribes to the resulting events (cmd/init.go).
package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

// InquirySendEventType is the durable fan-out event (one per recipient).
const InquirySendEventType = "procurement.inquiry_send"

// OrderConfirmSendEventType is the durable order-confirmation send event.
const OrderConfirmSendEventType = "procurement.order_confirm_send"

// InquirySend is the payload of a single recipient send of an inquiry run.
type InquirySend struct {
	eventbus.BaseEvent
	OrganizationID int32  `json:"organization_id"`
	RunID          int32  `json:"run_id"`
	RecipientID    int32  `json:"recipient_id"`
	SupplierID     int32  `json:"supplier_id"`
	ContactID      int32  `json:"contact_id"`
	To             string `json:"to"`
	Message        string `json:"message"`
}

// NewInquirySend builds the event (used by the enqueue side).
func NewInquirySend(orgID, runID, recipientID, supplierID, contactID int32, to, message string) *InquirySend {
	return &InquirySend{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      InquirySendEventType,
			CreatedAt: time.Now(),
			Meta:      map[string]any{},
		},
		OrganizationID: orgID,
		RunID:          runID,
		RecipientID:    recipientID,
		SupplierID:     supplierID,
		ContactID:      contactID,
		To:             to,
		Message:        message,
	}
}

// OrderConfirmSend is the payload of an order-confirmation send.
type OrderConfirmSend struct {
	eventbus.BaseEvent
	OrganizationID int32  `json:"organization_id"`
	OrderID        int32  `json:"order_id"`
	RunID          int32  `json:"run_id"`
	SupplierID     int32  `json:"supplier_id"`
	ContactID      int32  `json:"contact_id"`
	To             string `json:"to"`
	Message        string `json:"message"`
}

// NewOrderConfirmSend builds the event (used by the enqueue side).
func NewOrderConfirmSend(orgID, orderID, runID, supplierID, contactID int32, to, message string) *OrderConfirmSend {
	return &OrderConfirmSend{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      OrderConfirmSendEventType,
			CreatedAt: time.Now(),
			Meta:      map[string]any{},
		},
		OrganizationID: orgID,
		OrderID:        orderID,
		RunID:          runID,
		SupplierID:     supplierID,
		ContactID:      contactID,
		To:             to,
		Message:        message,
	}
}

// Codec returns a payload codec for an event type, for the outbox registry.
func Codec(eventType string) (func(payload json.RawMessage) (eventbus.Event, error), error) {
	switch eventType {
	case InquirySendEventType:
		return decodeInquirySend, nil
	case OrderConfirmSendEventType:
		return decodeOrderConfirmSend, nil
	default:
		return nil, fmt.Errorf("procurement: unknown outbox event type %q", eventType)
	}
}

func decodeInquirySend(payload json.RawMessage) (eventbus.Event, error) {
	var e InquirySend
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, fmt.Errorf("decode inquiry send payload: %w", err)
	}
	e.Name = InquirySendEventType
	e.CreatedAt = time.Now()
	e.Meta = map[string]any{}
	return &e, nil
}

func decodeOrderConfirmSend(payload json.RawMessage) (eventbus.Event, error) {
	var e OrderConfirmSend
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, fmt.Errorf("decode order confirm send payload: %w", err)
	}
	e.Name = OrderConfirmSendEventType
	e.CreatedAt = time.Now()
	e.Meta = map[string]any{}
	return &e, nil
}
