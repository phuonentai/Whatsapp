// Package events defines the inquiry-scheduling durable outbox payloads. The
// outbox dispatcher decodes each payload into an eventbus event and the
// module subscribes to the resulting events (cmd/init.go).
package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

// InquiryRunScheduledEventType is enqueued by the scheduler claim transaction:
// one durable event per fired occurrence (exactly-once effect, at-least-once
// delivery).
const InquiryRunScheduledEventType = "inquiry_run.scheduled"

// FollowupSendEventType is enqueued by the follow-up service: one durable
// event per nudge (guarded by the atomic followup_count increment).
const FollowupSendEventType = "inquiry.followup_send"

// InquiryRunScheduled is the payload of a fired schedule occurrence.
type InquiryRunScheduled struct {
	eventbus.BaseEvent
	ScheduleID     int32     `json:"schedule_id"`
	OrganizationID int32     `json:"organization_id"`
	ProductIDs     []int32   `json:"product_ids"`
	SupplierIDs    []int32   `json:"supplier_ids"`
	Note           string    `json:"note"`
	OccurrenceAt   time.Time `json:"occurrence_at"`
}

// NewInquiryRunScheduled builds the claim event (enqueue side).
func NewInquiryRunScheduled(scheduleID, orgID int32, productIDs, supplierIDs []int32, note string, occurrenceAt time.Time) *InquiryRunScheduled {
	return &InquiryRunScheduled{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      InquiryRunScheduledEventType,
			CreatedAt: time.Now(),
			Meta:      map[string]any{},
		},
		ScheduleID:     scheduleID,
		OrganizationID: orgID,
		ProductIDs:     productIDs,
		SupplierIDs:    supplierIDs,
		Note:           note,
		OccurrenceAt:   occurrenceAt,
	}
}

// FollowupSend is the payload of one recipient follow-up send.
type FollowupSend struct {
	eventbus.BaseEvent
	RunID          int32  `json:"run_id"`
	OrganizationID int32  `json:"organization_id"`
	SupplierID     int32  `json:"supplier_id"`
	ContactID      int32  `json:"contact_id"`
	RecipientID    int32  `json:"recipient_id"`
	ContactPhone   string `json:"contact_phone"`
	Message        string `json:"message"`
	NudgeIndex     int32  `json:"nudge_index"`
}

// NewFollowupSend builds the follow-up event (enqueue side).
func NewFollowupSend(runID, orgID, supplierID, contactID, recipientID int32, contactPhone, message string, nudgeIndex int32) *FollowupSend {
	return &FollowupSend{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      FollowupSendEventType,
			CreatedAt: time.Now(),
			Meta:      map[string]any{},
		},
		RunID:          runID,
		OrganizationID: orgID,
		SupplierID:     supplierID,
		ContactID:      contactID,
		RecipientID:    recipientID,
		ContactPhone:   contactPhone,
		Message:        message,
		NudgeIndex:     nudgeIndex,
	}
}

// Codec returns a payload codec for an event type, for the outbox registry.
func Codec(eventType string) (func(payload json.RawMessage) (eventbus.Event, error), error) {
	switch eventType {
	case InquiryRunScheduledEventType:
		return decodeInquiryRunScheduled, nil
	case FollowupSendEventType:
		return decodeFollowupSend, nil
	default:
		return nil, fmt.Errorf("inquiryschedule: unknown outbox event type %q", eventType)
	}
}

func decodeInquiryRunScheduled(payload json.RawMessage) (eventbus.Event, error) {
	var e InquiryRunScheduled
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, fmt.Errorf("decode inquiry_run.scheduled payload: %w", err)
	}
	e.Name = InquiryRunScheduledEventType
	e.CreatedAt = time.Now()
	e.Meta = map[string]any{}
	return &e, nil
}

func decodeFollowupSend(payload json.RawMessage) (eventbus.Event, error) {
	var e FollowupSend
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, fmt.Errorf("decode inquiry.followup_send payload: %w", err)
	}
	e.Name = FollowupSendEventType
	e.CreatedAt = time.Now()
	e.Meta = map[string]any{}
	return &e, nil
}
