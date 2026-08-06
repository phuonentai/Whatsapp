package domain

import "time"

type ActivityType string

const (
	ActivityTypeNota           ActivityType = "nota"
	ActivityTypeLlamada        ActivityType = "llamada"
	ActivityTypeCorreo         ActivityType = "correo"
	ActivityTypeReunion        ActivityType = "reunion"
	ActivityTypeTarea          ActivityType = "tarea"
	ActivityTypeWhatsAppMessage ActivityType = "whatsapp_message"
	ActivityTypeSistema        ActivityType = "sistema"
)

type Activity struct {
	ID               int32                  `json:"id"`
	OrganizationID   int32                  `json:"organization_id"`
	ContactID        *int32                 `json:"contact_id,omitempty"`
	CompanyID        *int32                 `json:"company_id,omitempty"`
	DealID           *int32                 `json:"deal_id,omitempty"`
	ConversationID   *int32                 `json:"conversation_id,omitempty"`
	Tipo             ActivityType           `json:"tipo"`
	Asunto           string                 `json:"asunto,omitempty"`
	Contenido        string                 `json:"contenido,omitempty"`
	Estado           string                 `json:"estado,omitempty"`
	FechaVencimiento *time.Time             `json:"fecha_vencimiento,omitempty"`
	RealizadaPor     *int32                 `json:"realizada_por,omitempty"`
	RealizadaEn      time.Time              `json:"realizada_en"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type ActivityWithActor struct {
	Activity
	RealizadaPorNombre string `json:"realizada_por_nombre,omitempty"`
}
