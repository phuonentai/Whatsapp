package events

import (
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const DealStageChangedEventType = "crm.negocio.etapa_cambiada"

type DealStageChanged struct {
	eventbus.BaseEvent
	DealID         int32  `json:"deal_id"`
	OrganizationID int32  `json:"organization_id"`
	OldStageID     *int32 `json:"old_stage_id,omitempty"`
	NewStageID     int32  `json:"new_stage_id"`
	OldStageName   string `json:"old_stage_name"`
	NewStageName   string `json:"new_stage_name"`
	ChangedBy      int32  `json:"changed_by"`
}

func (e *DealStageChanged) EventName() string { return DealStageChangedEventType }
