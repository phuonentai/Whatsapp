package domain

import (
	"time"
)

type WhatsAppConfig struct {
	ID             int32                  `json:"id"`
	OrganizationID int32                  `json:"organization_id"`
	PhoneNumberID  string                 `json:"phone_number_id"`
	BusinessPhone  string                 `json:"business_phone"`
	WebhookSecret  string                 `json:"webhook_secret"`
	VerifyToken    string                 `json:"verify_token"`
	AppID          string                 `json:"app_id,omitempty"`
	WABAID         string                 `json:"waba_id,omitempty"`
	AccessToken    string                 `json:"access_token,omitempty"`
	APIVersion     string                 `json:"api_version,omitempty"`
	GraphAPIURL    string                 `json:"graph_api_url,omitempty"`
	IsActive       bool                   `json:"is_active"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (c *WhatsAppConfig) Validate() error {
	if c.OrganizationID == 0 {
		return ErrOrgRequired
	}
	if c.PhoneNumberID == "" {
		return ErrPhoneNumberIDRequired
	}
	if c.WebhookSecret == "" {
		return ErrWebhookSecretRequired
	}
	return nil
}
