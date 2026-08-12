package domain

import (
	"context"
	"errors"
	"fmt"
)

// MfaPolicy controls whether the organization requires MFA for every member.
// Values mirror the Stytch B2B `mfa_policy` organization setting.
type MfaPolicy string

const (
	// MfaPolicyOptional is the default: members are only required to complete
	// MFA when their own `mfa_enrolled` status is true.
	MfaPolicyOptional MfaPolicy = "OPTIONAL"
	// MfaPolicyRequiredForAll requires every member to complete MFA at login.
	MfaPolicyRequiredForAll MfaPolicy = "REQUIRED_FOR_ALL"
)

// MfaMethods controls which MFA methods members of the organization may use.
// Values mirror the Stytch B2B `mfa_methods` organization setting.
type MfaMethods string

const (
	// MfaMethodsAllAllowed permits any supported secondary factor.
	MfaMethodsAllAllowed MfaMethods = "ALL_ALLOWED"
	// MfaMethodsRestricted restricts secondary factors to `allowed_mfa_methods`.
	MfaMethodsRestricted MfaMethods = "RESTRICTED"
)

// MfaMethod is a single supported secondary authentication factor.
// Values mirror the Stytch B2B `allowed_mfa_methods` accepted values.
type MfaMethod string

const (
	// MfaMethodTOTP is a time-based one-time password (authenticator app).
	MfaMethodTOTP MfaMethod = "totp"
	// MfaMethodSMSOTP is a one-time password delivered via SMS.
	MfaMethodSMSOTP MfaMethod = "sms_otp"
)

// Validate reports whether policy is a supported MFA policy value.
func (p MfaPolicy) Validate() error {
	switch p {
	case MfaPolicyOptional, MfaPolicyRequiredForAll:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidMfaPolicy, string(p))
	}
}

// Validate reports whether methods is a supported MFA methods value.
func (m MfaMethods) Validate() error {
	switch m {
	case MfaMethodsAllAllowed, MfaMethodsRestricted:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidMfaMethods, string(m))
	}
}

// Validate reports whether method is a supported secondary factor.
func (m MfaMethod) Validate() error {
	switch m {
	case MfaMethodTOTP, MfaMethodSMSOTP:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidMfaMethod, string(m))
	}
}

// MfaPolicyUpdater updates an organization's MFA policy in the auth provider.
//
// The auth provider is the single source of truth for MFA policy state: no
// local database or audit row stores secrets, recovery codes, or MFA material
// (SSOT constitution). Implementations SHALL route every outbound call through
// the shared circuit breaker; when the breaker is open or the provider is
// unreachable the organization's policy MUST remain unchanged and the returned
// error maps to a 503 structured error at the API boundary.
type MfaPolicyUpdater interface {
	// UpdateMfaPolicy sets the organization's MFA policy, method restriction,
	// and (when methods == RESTRICTED) the allowed method list.
	UpdateMfaPolicy(
		ctx context.Context,
		orgID string,
		policy MfaPolicy,
		methods MfaMethods,
		allowedMethods []MfaMethod,
	) error
}

// ValidateMfaPolicyUpdate validates a full policy-update payload. When methods
// is RESTRICTED the allowed method list MUST be non-empty and contain only
// supported values.
func ValidateMfaPolicyUpdate(policy MfaPolicy, methods MfaMethods, allowedMethods []MfaMethod) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if err := methods.Validate(); err != nil {
		return err
	}
	if methods == MfaMethodsRestricted && len(allowedMethods) == 0 {
		return errors.New("allowed MFA methods are required when methods are restricted")
	}
	seen := make(map[MfaMethod]struct{}, len(allowedMethods))
	for _, m := range allowedMethods {
		if err := m.Validate(); err != nil {
			return err
		}
		if _, dup := seen[m]; dup {
			return fmt.Errorf("duplicate allowed MFA method: %q", string(m))
		}
		seen[m] = struct{}{}
	}
	return nil
}
