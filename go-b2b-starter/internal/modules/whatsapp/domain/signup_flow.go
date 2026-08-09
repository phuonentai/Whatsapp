package domain

import (
	"fmt"
	"time"
)

type SignupStatus string

const (
	SignupStatusExchanging  SignupStatus = "exchanging"
	SignupStatusRegistering SignupStatus = "registering"
	SignupStatusVerifying   SignupStatus = "verifying"
	SignupStatusConnected   SignupStatus = "connected"
	SignupStatusFailed      SignupStatus = "failed"
)

func (s SignupStatus) IsValid() bool {
	switch s {
	case SignupStatusExchanging, SignupStatusRegistering, SignupStatusVerifying, SignupStatusConnected, SignupStatusFailed:
		return true
	}
	return false
}

// InProgress reports whether the flow is mid-flight (not terminal).
func (s SignupStatus) InProgress() bool {
	return s == SignupStatusExchanging || s == SignupStatusRegistering || s == SignupStatusVerifying
}

// SignupFlow is the embedded-signup provisioning state for one organization.
type SignupFlow struct {
	ID             int64                  `json:"id"`
	OrganizationID int32                  `json:"organization_id"`
	Status         SignupStatus           `json:"status"`
	Step           string                 `json:"step,omitempty"`
	ErrorCode      string                 `json:"error_code,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (f *SignupFlow) Validate() error {
	if f.OrganizationID == 0 {
		return ErrOrgRequired
	}
	if !f.Status.IsValid() {
		return fmt.Errorf("invalid signup flow status: %s", f.Status)
	}
	return nil
}
