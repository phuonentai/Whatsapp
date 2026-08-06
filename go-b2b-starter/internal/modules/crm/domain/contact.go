package domain

import "time"

type LeadStatus string

const (
	LeadStatusNuevo        LeadStatus = "nuevo"
	LeadStatusContactado   LeadStatus = "contactado"
	LeadStatusCalificado   LeadStatus = "calificado"
	LeadStatusDescalificado LeadStatus = "descalificado"
	LeadStatusCliente      LeadStatus = "cliente"
)

type ContactSource string

const (
	ContactSourceWhatsApp ContactSource = "whatsapp"
	ContactSourceManual   ContactSource = "manual"
	ContactSourceImport   ContactSource = "import"
	ContactSourceAPI      ContactSource = "api"
)

type TipoDocumento string

const (
	TipoDocCC  TipoDocumento = "CC"
	TipoDocNIT TipoDocumento = "NIT"
	TipoDocCE  TipoDocumento = "CE"
	TipoDocTI  TipoDocumento = "TI"
	TipoDocPP  TipoDocumento = "PP"
)

type Contact struct {
	ID              int32                  `json:"id"`
	OrganizationID  int32                  `json:"organization_id"`
	PhoneNumber     string                 `json:"phone_number"`
	DisplayName     string                 `json:"display_name,omitempty"`
	Email           string                 `json:"email,omitempty"`
	CompanyID       *int32                 `json:"company_id,omitempty"`
	Source          ContactSource          `json:"source"`
	LeadStatus      LeadStatus             `json:"lead_status"`
	JobTitle        string                 `json:"job_title,omitempty"`
	AssignedTo      *int32                 `json:"assigned_to,omitempty"`
	TipoDocumento   TipoDocumento          `json:"tipo_documento,omitempty"`
	NumeroDocumento string                 `json:"numero_documento,omitempty"`
	AvatarURL       string                 `json:"avatar_url,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	IsBlocked       bool                   `json:"is_blocked"`
	LastMessageAt   *time.Time             `json:"last_message_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func (c *Contact) Validate() error {
	if c.OrganizationID == 0 {
		return ErrContactOrganizationRequired
	}
	if c.PhoneNumber == "" {
		return ErrContactPhoneRequired
	}
	return nil
}
