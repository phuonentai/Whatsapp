package domain

import (
	"context"
	"fmt"
)

// MetaConfig carries the browser bootstrap values for the embedded signup SDK.
type MetaConfig struct {
	AppID       string `json:"app_id"`
	ConfigID    string `json:"config_id"`
	RedirectURI string `json:"redirect_uri"`
}

// SignupResult is the outcome of an exchange attempt, with the config
// returned masked.
type SignupResult struct {
	Status    SignupStatus    `json:"status"`
	ErrorCode string          `json:"error_code,omitempty"`
	Config    *WhatsAppConfig `json:"config,omitempty"`
}

// SignupService is the transport-free contract for the embedded signup flow.
type SignupService interface {
	MetaConfig(ctx context.Context) (*MetaConfig, error)
	Exchange(ctx context.Context, orgID int32, code string, actorMemberID string) (*SignupResult, error)
	Status(ctx context.Context, orgID int32) (*SignupFlow, error)
}

// SignupFailedError carries the recorded error code for a terminal failure.
type SignupFailedError struct {
	Code string
	Err  error
}

func (e *SignupFailedError) Error() string {
	if e.Code == "" {
		return "signup failed"
	}
	return fmt.Sprintf("signup failed: %s", e.Code)
}

func (e *SignupFailedError) Unwrap() error {
	return e.Err
}
