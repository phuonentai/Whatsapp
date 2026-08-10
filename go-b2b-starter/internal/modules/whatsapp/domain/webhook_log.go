package domain

import (
	"encoding/json"
	"time"
)

type WebhookLog struct {
	ID             int32           `json:"id"`
	OrganizationID *int32          `json:"organization_id"`
	Status         string          `json:"status"`
	EventType      string          `json:"event_type,omitempty"`
	PhoneNumberID  string          `json:"phone_number_id,omitempty"`
	RawHeaders     json.RawMessage `json:"raw_headers,omitempty"`
	RawBody        json.RawMessage `json:"raw_body,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
	DeliveryKey    string          `json:"delivery_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// OutboxEventInput is a durable event payload to be committed atomically
// with the webhook log row.
type OutboxEventInput struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}
