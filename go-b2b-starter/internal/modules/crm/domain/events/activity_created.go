package events

import "github.com/moasq/go-b2b-starter/internal/platform/eventbus"

const ActivityCreatedEventType = "crm.actividad.creada"

type ActivityCreated struct {
	eventbus.BaseEvent
	ActivityID     int32  `json:"activity_id"`
	OrganizationID int32  `json:"organization_id"`
	ContactID      *int32 `json:"contact_id,omitempty"`
	DealID         *int32 `json:"deal_id,omitempty"`
	Tipo           string `json:"tipo"`
}

func (e *ActivityCreated) EventName() string { return ActivityCreatedEventType }
