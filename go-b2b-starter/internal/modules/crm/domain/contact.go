package domain

import "time"

type LeadStatus string

const (
	LeadStatusNuevo         LeadStatus = "nuevo"
	LeadStatusContactado    LeadStatus = "contactado"
	LeadStatusCalificado    LeadStatus = "calificado"
	LeadStatusDescalificado LeadStatus = "descalificado"
	LeadStatusCliente       LeadStatus = "cliente"
)

type ContactSource string

const (
	ContactSourceWhatsApp  ContactSource = "whatsapp"
	ContactSourceInstagram ContactSource = "instagram"
	ContactSourceManual    ContactSource = "manual"
	ContactSourceImport    ContactSource = "import"
	ContactSourceAPI       ContactSource = "api"
)

// ConsentStatus is the Ley 1581 (Habeas Data) consent state of a contact.
// Value "withdrawn" triggers PII masking in every CSV export.
const (
	ConsentStatusNone      string = "none"
	ConsentStatusRequested string = "requested"
	ConsentStatusGranted   string = "granted"
	ConsentStatusWithdrawn string = "withdrawn"
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
	ID                int32                  `json:"id"`
	OrganizationID    int32                  `json:"organization_id"`
	PhoneNumber       string                 `json:"phone_number"`
	InstagramUserID   string                 `json:"instagram_user_id,omitempty"`
	InstagramUsername string                 `json:"instagram_username,omitempty"`
	DisplayName       string                 `json:"display_name,omitempty"`
	Email             string                 `json:"email,omitempty"`
	CompanyID         *int32                 `json:"company_id,omitempty"`
	Source            ContactSource          `json:"source"`
	LeadStatus        LeadStatus             `json:"lead_status"`
	JobTitle          string                 `json:"job_title,omitempty"`
	AssignedTo        *int32                 `json:"assigned_to,omitempty"`
	TipoDocumento     TipoDocumento          `json:"tipo_documento,omitempty"`
	NumeroDocumento   string                 `json:"numero_documento,omitempty"`
	AvatarURL         string                 `json:"avatar_url,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	IsBlocked         bool                   `json:"is_blocked"`
	ConsentStatus     string                 `json:"consent_status,omitempty"`
	LastMessageAt     *time.Time             `json:"last_message_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

func (c *Contact) Validate() error {
	if c.OrganizationID == 0 {
		return ErrContactOrganizationRequired
	}
	if c.PhoneNumber == "" && c.InstagramUserID == "" {
		return ErrContactPhoneRequired
	}
	return nil
}
