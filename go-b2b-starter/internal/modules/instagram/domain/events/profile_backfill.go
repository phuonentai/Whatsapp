package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const ProfileBackfillEventType = "instagram.profile.backfill"

// ProfileBackfill requests the async resolution of an Instagram contact's
// username and avatar from the Graph API. Transient failures retry with the
// outbox backoff; permanent failures dead-letter leaving the username NULL.
type ProfileBackfill struct {
	eventbus.BaseEvent
	OrganizationID int32           `json:"organization_id"`
	ContactID      int32           `json:"contact_id"`
	IGUserID       string          `json:"ig_user_id"`
	RawPayload     json.RawMessage `json:"raw_payload,omitempty"`
}

func NewProfileBackfill(orgID, contactID int32, igUserID string) *ProfileBackfill {
	return &ProfileBackfill{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      ProfileBackfillEventType,
			CreatedAt: time.Now(),
			Meta:      make(map[string]interface{}),
		},
		OrganizationID: orgID,
		ContactID:      contactID,
		IGUserID:       igUserID,
	}
}
