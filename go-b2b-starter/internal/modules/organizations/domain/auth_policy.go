package domain

import (
	"context"
	"errors"
	"fmt"
)

// JitPolicy controls whether new members can be JIT-provisioned into the
// organization when authenticating with a verified company email (via Email
// Magic Link or OAuth). Values map to the Stytch B2B `email_jit_provisioning`
// organization setting.
type JitPolicy string

const (
	// JitPolicyDisabled is the default: JIT provisioning is off for the
	// organization (Stytch `NOT_ALLOWED`).
	JitPolicyDisabled JitPolicy = "DISABLED"
	// JitPolicyDomainRestricted provisions new members whose verified email is
	// on `email_allowed_domains` (Stytch `RESTRICTED`). JIT org *creation* is
	// never enabled — the Stytch contract exposes no org-creating value.
	JitPolicyDomainRestricted JitPolicy = "DOMAIN_RESTRICTED"
)

// SsoJitPolicy controls whether new members can be JIT-provisioned via SSO.
// Values map to the Stytch B2B `sso_jit_provisioning` organization setting.
type SsoJitPolicy string

const (
	// SsoJitPolicyDisabled is the default: SSO JIT provisioning is off
	// (Stytch `NOT_ALLOWED`).
	SsoJitPolicyDisabled SsoJitPolicy = "DISABLED"
	// SsoJitPolicyConnectionRestricted provisions members authenticating via
	// the organization's allowed connections (Stytch `RESTRICTED` +
	// `sso_jit_provisioning_allowed_connections`). Least privilege: org-wide
	// `ALL_ALLOWED` SSO JIT is never written by the platform.
	SsoJitPolicyConnectionRestricted SsoJitPolicy = "CONNECTION_RESTRICTED"
)

// AllowedAuthMethod is a primary authentication method an organization
// permits. Values are the Stytch B2B `allowed_auth_methods` accepted values —
// there is no `email_magic_link`, generic `oauth`, `passkeys`, or `m2m` in
// the contract.
type AllowedAuthMethod string

const (
	// AuthMethodMagicLink is passwordless email magic-link sign-in.
	AuthMethodMagicLink AllowedAuthMethod = "magic_link"
	// AuthMethodEmailOTP is a one-time code delivered by email.
	AuthMethodEmailOTP AllowedAuthMethod = "email_otp"
	// AuthMethodSSO is SAML/OIDC SSO sign-in.
	AuthMethodSSO AllowedAuthMethod = "sso"
	// AuthMethodGoogleOAuth is Google social OAuth sign-in.
	AuthMethodGoogleOAuth AllowedAuthMethod = "google_oauth"
	// AuthMethodMicrosoftOAuth is Microsoft social OAuth sign-in.
	AuthMethodMicrosoftOAuth AllowedAuthMethod = "microsoft_oauth"
)

// Validate reports whether p is a supported email JIT policy value.
func (p JitPolicy) Validate() error {
	switch p {
	case JitPolicyDisabled, JitPolicyDomainRestricted:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAuthPolicy, string(p))
	}
}

// Validate reports whether p is a supported SSO JIT policy value.
func (p SsoJitPolicy) Validate() error {
	switch p {
	case SsoJitPolicyDisabled, SsoJitPolicyConnectionRestricted:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAuthPolicy, string(p))
	}
}

// Validate reports whether m is a supported primary authentication method.
func (m AllowedAuthMethod) Validate() error {
	switch m {
	case AuthMethodMagicLink, AuthMethodEmailOTP, AuthMethodSSO,
		AuthMethodGoogleOAuth, AuthMethodMicrosoftOAuth:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAuthPolicy, string(m))
	}
}

// AuthPolicy mirrors the auth provider's organization auth settings. It is a
// display-only mirror: read from the auth provider on fetch and NEVER
// consulted for authorization — Stytch enforces auth methods and JIT
// provisioning at authentication time (SSOT constitution).
type AuthPolicy struct {
	// EmailJITProvisioning mirrors `email_jit_provisioning`.
	EmailJITProvisioning JitPolicy `json:"email_jit_provisioning"`
	// EmailAllowedDomains mirrors `email_allowed_domains` (enforced when
	// either `email_invites` or `email_jit_provisioning` is RESTRICTED).
	EmailAllowedDomains []string `json:"email_allowed_domains,omitempty"`
	// AuthMethodsRestricted mirrors whether `auth_methods` is RESTRICTED
	// (enforced-list mode); ALL_ALLOWED orgs have never saved a policy.
	AuthMethodsRestricted bool `json:"auth_methods_restricted"`
	// AllowedAuthMethods mirrors `allowed_auth_methods` (the enforced list
	// when AuthMethodsRestricted).
	AllowedAuthMethods []AllowedAuthMethod `json:"allowed_auth_methods,omitempty"`
	// SSOJITProvisioning mirrors `sso_jit_provisioning`. Stytch's documented
	// default `ALL_ALLOWED` collapses to SsoJitPolicyDisabled in the domain
	// model: the platform never writes org-wide ALL_ALLOWED (least privilege).
	SSOJITProvisioning SsoJitPolicy `json:"sso_jit_provisioning"`
	// SSOJITProvisioningAllowedConnections mirrors
	// `sso_jit_provisioning_allowed_connections`.
	SSOJITProvisioningAllowedConnections []string `json:"sso_jit_provisioning_allowed_connections,omitempty"`
	// SSODefaultConnectionID mirrors `sso_default_connection_id` (the default
	// connection used for SSO when the org has multiple active connections).
	SSODefaultConnectionID string `json:"sso_default_connection_id,omitempty"`
	// SSOActiveConnectionIDs mirrors the org's active SSO connection ids.
	// Used by the UI to gate the SSO-JIT toggle and by validation to scope
	// `sso_default_connection_id` to org-owned connections.
	SSOActiveConnectionIDs []string `json:"sso_active_connection_ids,omitempty"`
}

// OrgAuthPolicyUpdater reads and updates an organization's auth policy in the
// auth provider.
//
// The auth provider is the single source of truth for org auth policy state:
// no local database row stores policy decisions and the mirrored values are
// display-only (never consulted for authorization — SSOT constitution).
// Implementations SHALL route every outbound call through the shared circuit
// breaker; when the breaker is open or the provider is unreachable the
// organization's policy MUST remain unchanged and the returned error maps to
// a 503 structured error at the API boundary.
type OrgAuthPolicyUpdater interface {
	// GetAuthPolicy reads the organization's current auth policy from the
	// auth provider as a display-only mirror.
	GetAuthPolicy(ctx context.Context, orgID string) (*AuthPolicy, error)

	// UpdateAuthPolicy sets the organization's auth policy in the auth
	// provider. The write contract always persists `auth_methods: RESTRICTED`
	// (enforced-list mode — required for `allowed_auth_methods` to take
	// effect) together with the org's preserved method set plus the requested
	// additions. emailJitPolicy maps to `email_jit_provisioning`; ssoJitPolicy
	// and ssoJitAllowedConnections map to `sso_jit_provisioning` and
	// `sso_jit_provisioning_allowed_connections`; ssoDefaultConnectionID maps
	// to `sso_default_connection_id`.
	UpdateAuthPolicy(
		ctx context.Context,
		orgID string,
		emailJitPolicy JitPolicy,
		allowedDomains []string,
		allowedAuthMethods []AllowedAuthMethod,
		ssoJitPolicy SsoJitPolicy,
		ssoJitAllowedConnections []string,
		ssoDefaultConnectionID string,
	) error
}

// ValidateAuthPolicyUpdate validates a full auth-policy update payload.
// activeConnectionIDs is the organization's current active SSO connection ids;
// it is required so `sso_default_connection_id` can be checked to reference an
// org-owned connection.
func ValidateAuthPolicyUpdate(
	emailJitPolicy JitPolicy,
	allowedDomains []string,
	allowedAuthMethods []AllowedAuthMethod,
	ssoJitPolicy SsoJitPolicy,
	ssoJitAllowedConnections []string,
	ssoDefaultConnectionID string,
	activeConnectionIDs []string,
) error {
	if err := emailJitPolicy.Validate(); err != nil {
		return err
	}
	if err := ssoJitPolicy.Validate(); err != nil {
		return err
	}
	if emailJitPolicy == JitPolicyDomainRestricted && len(allowedDomains) == 0 {
		return errors.New("allowed domains are required when email JIT provisioning is domain-restricted")
	}
	if len(allowedAuthMethods) == 0 {
		return errors.New("at least one allowed auth method is required")
	}
	seen := make(map[AllowedAuthMethod]struct{}, len(allowedAuthMethods))
	for _, m := range allowedAuthMethods {
		if err := m.Validate(); err != nil {
			return err
		}
		if _, dup := seen[m]; dup {
			return fmt.Errorf("duplicate allowed auth method: %q", string(m))
		}
		seen[m] = struct{}{}
	}
	if ssoJitPolicy == SsoJitPolicyConnectionRestricted && len(ssoJitAllowedConnections) == 0 {
		return errors.New("allowed connections are required when SSO JIT provisioning is connection-restricted")
	}
	if ssoDefaultConnectionID != "" && !containsString(activeConnectionIDs, ssoDefaultConnectionID) {
		return fmt.Errorf("%w: sso_default_connection_id must reference an org-owned connection", ErrInvalidAuthPolicy)
	}
	return nil
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
