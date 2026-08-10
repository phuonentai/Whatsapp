package domain

import "time"

// Channel constants shared across CRM and provider modules.
const (
	ChannelWhatsapp  = "whatsapp"
	ChannelInstagram = "instagram"
)

type InstagramConfig struct {
	ID             int32                  `json:"id"`
	OrganizationID int32                  `json:"organization_id"`
	IGUserID       string                 `json:"ig_user_id"`
	IGUsername     string                 `json:"ig_username,omitempty"`
	FBPageID       string                 `json:"fb_page_id,omitempty"`
	AccessToken    string                 `json:"access_token,omitempty"`
	TokenExpiresAt *time.Time             `json:"token_expires_at,omitempty"`
	WebhookSecret  string                 `json:"webhook_secret"`
	VerifyToken    string                 `json:"verify_token"`
	APIVersion     string                 `json:"api_version,omitempty"`
	GraphAPIURL    string                 `json:"graph_api_url,omitempty"`
	IsActive       bool                   `json:"is_active"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (c *InstagramConfig) Validate() error {
	if c.OrganizationID == 0 {
		return ErrOrgRequired
	}
	if c.IGUserID == "" {
		return ErrIGUserIDRequired
	}
	if c.WebhookSecret == "" {
		return ErrWebhookSecretRequired
	}
	return nil
}

// TokenExpiryWarning reports whether the stored token expires within 7 days
// or has no recorded expiry at all.
func (c *InstagramConfig) TokenExpiryWarning() bool {
	if c.TokenExpiresAt == nil {
		return true
	}
	return time.Until(*c.TokenExpiresAt) < 7*24*time.Hour
}
