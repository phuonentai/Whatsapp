package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	stytchcfg "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/organizations"
)

type stytchOrganizationRepository struct {
	client       *stytchcfg.Client
	logger       loggerDomain.Logger
	localOrgRepo domain.OrganizationRepository
}

// Ensure stytchOrganizationRepository implements the organization repository,
// the MFA policy updater, and the org auth policy updater domain contracts.
var _ domain.AuthOrganizationRepository = (*stytchOrganizationRepository)(nil)
var _ domain.MfaPolicyUpdater = (*stytchOrganizationRepository)(nil)
var _ domain.OrgAuthPolicyUpdater = (*stytchOrganizationRepository)(nil)

// NewStytchOrganizationRepository creates a Stytch-backed organization
// repository. It returns the concrete type so the same instance can satisfy
// both the AuthOrganizationRepository and MfaPolicyUpdater domain contracts.
func NewStytchOrganizationRepository(
	client *stytchcfg.Client,
	logger loggerDomain.Logger,
	localOrgRepo domain.OrganizationRepository,
) *stytchOrganizationRepository {
	return &stytchOrganizationRepository{
		client:       client,
		logger:       logger,
		localOrgRepo: localOrgRepo,
	}
}

func (r *stytchOrganizationRepository) CreateOrganization(ctx context.Context, req *domain.CreateAuthOrganizationRequest) (*domain.AuthOrganization, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create organization request: %w", err)
	}

	// Generate base slug from display name (infrastructure concern)
	baseSlug := generateSlug(req.DisplayName)

	// Prepare email invites parameter
	emailInvites := "NOT_ALLOWED"
	if req.EmailInvitesAllowed {
		emailInvites = "ALL_ALLOWED"
	}

	// Retry loop for duplicate slug handling (infrastructure concern)
	const maxAttempts = 5
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Generate slug with suffix if needed
		slug := generateSlugWithSuffix(baseSlug, attempt)

		r.logger.Debug("attempting to create organization", loggerDomain.Fields{
			"display_name": req.DisplayName,
			"slug":         slug,
			"attempt":      attempt,
		})

		// Try to create in Stytch
		params := &organizations.CreateParams{
			OrganizationSlug: slug,
			OrganizationName: req.DisplayName,
			EmailInvites:     emailInvites,
		}

		resp, err := r.client.API().Organizations.Create(ctx, params)

		// Success - return immediately
		if err == nil {
			if attempt > 1 {
				r.logger.Info("created organization with retry", loggerDomain.Fields{
					"display_name": req.DisplayName,
					"final_slug":   slug,
					"attempts":     attempt,
				})
			}
			return mapToAuthOrganization(resp.Organization), nil
		}

		// Check if duplicate slug error - retry
		if stytchcfg.IsDuplicateSlugError(err) {
			r.logger.Debug("slug already exists, retrying", loggerDomain.Fields{
				"attempted_slug": slug,
				"attempt":        attempt,
				"max_attempts":   maxAttempts,
			})
			lastErr = err
			continue // Try next suffix
		}

		// Other error - fail immediately
		return nil, fmt.Errorf("stytch create organization: %w", stytchcfg.MapError(err))
	}

	// All attempts exhausted
	r.logger.Error("failed to create organization after retries", loggerDomain.Fields{
		"display_name": req.DisplayName,
		"base_slug":    baseSlug,
		"attempts":     maxAttempts,
	})
	return nil, fmt.Errorf("failed to create organization after %d attempts, slug conflicts: %w",
		maxAttempts, stytchcfg.MapError(lastErr))
}

func (r *stytchOrganizationRepository) GetOrganization(ctx context.Context, organizationID string) (*domain.AuthOrganization, error) {
	if organizationID == "" {
		return nil, domain.ErrAuthOrganizationIDRequired
	}

	if r.client == nil {
		// Development mode: placeholder credentials mean no Stytch client.
		// Fail cleanly instead of nil-pointer dereferencing.
		return nil, domain.ErrAuthConnection
	}

	resp, err := r.client.API().Organizations.Get(ctx, &organizations.GetParams{OrganizationID: organizationID})
	if err != nil {
		return nil, fmt.Errorf("stytch get organization: %w", stytchcfg.MapError(err))
	}

	return mapToAuthOrganization(resp.Organization), nil
}

func (r *stytchOrganizationRepository) UpdateOrganization(ctx context.Context, organizationID, displayName string) (*domain.AuthOrganization, error) {
	if organizationID == "" {
		return nil, domain.ErrAuthOrganizationIDRequired
	}
	if displayName == "" {
		return nil, domain.ErrAuthOrganizationDisplayNameRequired
	}

	if r.client == nil {
		// Development mode: placeholder credentials mean no Stytch client.
		// Fail cleanly instead of nil-pointer dereferencing.
		return nil, domain.ErrAuthConnection
	}

	var resp *organizations.UpdateResponse
	err := r.client.Run(ctx, func() error {
		var callErr error
		resp, callErr = r.client.API().Organizations.Update(ctx, &organizations.UpdateParams{
			OrganizationID:   organizationID,
			OrganizationName: displayName,
		})
		if callErr != nil {
			return fmt.Errorf("stytch update organization: %w", stytchcfg.MapError(callErr))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return mapToAuthOrganization(resp.Organization), nil
}

func (r *stytchOrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	if organizationID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}

	_, err := r.client.API().Organizations.Delete(ctx, &organizations.DeleteParams{OrganizationID: organizationID})
	if err != nil {
		return fmt.Errorf("stytch delete organization: %w", stytchcfg.MapError(err))
	}

	return nil
}

// UpdateMfaPolicy sets the organization's MFA policy in Stytch via
// `PUT /v1/b2b/organizations` (mfa_policy, mfa_methods, allowed_mfa_methods).
//
// The whole call runs behind the shared Stytch circuit breaker: when the
// breaker is open it fails fast with stytchcfg.ErrCircuitOpen; Stytch 5xx
// responses surface as stytchcfg.ErrInternal. Both are mapped to the domain
// ErrMfaPolicyUnavailable so the API boundary can answer 503 with a structured
// error, and the organization's policy stays unchanged (Stytch is the SSOT —
// no local state is touched).
//
// It implements domain.MfaPolicyUpdater.
func (r *stytchOrganizationRepository) UpdateMfaPolicy(
	ctx context.Context,
	orgID string,
	policy domain.MfaPolicy,
	methods domain.MfaMethods,
	allowedMethods []domain.MfaMethod,
) error {
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	if err := domain.ValidateMfaPolicyUpdate(policy, methods, allowedMethods); err != nil {
		return err
	}

	if r.client == nil {
		// Development mode: placeholder credentials mean no Stytch client.
		// Fail cleanly instead of nil-pointer dereferencing.
		return domain.ErrAuthConnection
	}

	allowed := make([]string, 0, len(allowedMethods))
	for _, m := range allowedMethods {
		allowed = append(allowed, string(m))
	}

	runErr := r.client.Run(ctx, func() error {
		_, callErr := r.client.API().Organizations.Update(ctx, &organizations.UpdateParams{
			OrganizationID:    orgID,
			MFAPolicy:         string(policy),
			MFAMethods:        string(methods),
			AllowedMFAMethods: allowed,
		})
		if callErr != nil {
			return fmt.Errorf("stytch update mfa policy: %w", stytchcfg.MapError(callErr))
		}
		return nil
	})
	if runErr != nil {
		// The breaker can fail the call fast (ErrCircuitOpen) without reaching
		// fn; Stytch 5xx collapses to ErrInternal via MapError. Both are
		// availability failures: map to the 503 domain sentinel.
		if errors.Is(runErr, stytchcfg.ErrCircuitOpen) || errors.Is(runErr, stytchcfg.ErrInternal) {
			return fmt.Errorf("stytch update mfa policy: %w", domain.ErrMfaPolicyUnavailable)
		}
		return runErr
	}
	return nil
}

func (r *stytchOrganizationRepository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("email cannot be empty")
	}

	r.logger.Debug("checking if email exists", loggerDomain.Fields{
		"email": email,
	})

	// Use the local organization repository to check if email exists
	// GetByUserEmail returns organization if email is found (and account is active)
	_, err := r.localOrgRepo.GetByUserEmail(ctx, email)
	if err != nil {
		// Check if it's a "not found" error using proper error comparison
		if errors.Is(err, domain.ErrOrganizationNotFound) || errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug("email not found", loggerDomain.Fields{
				"email": email,
			})
			return false, nil
		}
		// Other errors are real failures
		r.logger.Error("failed to check email existence", loggerDomain.Fields{
			"email": email,
			"error": err.Error(),
		})
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	r.logger.Debug("email exists", loggerDomain.Fields{
		"email": email,
	})
	return true, nil
}

// GetAuthPolicy reads the organization's auth settings from Stytch via
// `GET /v1/b2b/organizations/{organization_id}` and maps them to a
// display-only domain mirror (`domain.AuthPolicy`). The mirror is read from
// the auth provider on fetch and is NEVER consulted for authorization.
//
// The read runs behind the shared Stytch circuit breaker: breaker-open or
// Stytch 5xx map to domain.ErrAuthPolicyUnavailable (503 at the API
// boundary). Client 4xx surfaces as a normal error.
//
// It implements domain.OrgAuthPolicyUpdater.
func (r *stytchOrganizationRepository) GetAuthPolicy(ctx context.Context, orgID string) (*domain.AuthPolicy, error) {
	if orgID == "" {
		return nil, domain.ErrAuthOrganizationIDRequired
	}
	if r.client == nil {
		// Development mode: placeholder credentials mean no Stytch client.
		// Fail cleanly instead of nil-pointer dereferencing.
		return nil, domain.ErrAuthConnection
	}

	var org *organizations.Organization
	runErr := r.client.Run(ctx, func() error {
		resp, callErr := r.client.API().Organizations.Get(ctx, &organizations.GetParams{OrganizationID: orgID})
		if callErr != nil {
			return fmt.Errorf("stytch get organization auth policy: %w", stytchcfg.MapError(callErr))
		}
		org = &resp.Organization
		return nil
	})
	if runErr != nil {
		if errors.Is(runErr, stytchcfg.ErrCircuitOpen) || errors.Is(runErr, stytchcfg.ErrInternal) {
			return nil, fmt.Errorf("stytch get organization auth policy: %w", domain.ErrAuthPolicyUnavailable)
		}
		return nil, runErr
	}

	return mapToAuthPolicy(*org), nil
}

// UpdateAuthPolicy sets the organization's auth policy in Stytch via
// `PUT /v1/b2b/organizations/{organization_id}`.
//
// The write contract always persists `auth_methods: RESTRICTED` (enforced-list
// mode — required for `allowed_auth_methods` to take effect) together with
// `email_jit_provisioning`, `email_allowed_domains`, `allowed_auth_methods`,
// `sso_jit_provisioning`, `sso_jit_provisioning_allowed_connections`, and
// `sso_default_connection_id`.
//
// First-write default preservation: an org still on `auth_methods:
// ALL_ALLOWED` (the provider default) keeps its current effective method set —
// the project-enabled primary methods relevant to the org (at minimum
// `magic_link`, plus `sso` when the org has active SSO connections) — which is
// persisted as `RESTRICTED` + the preserved set + the requested additions, so
// the first save never silently removes methods the org already used.
//
// The read (for preservation + connection scoping) and the write both run
// behind the shared Stytch circuit breaker: breaker-open or Stytch 5xx map to
// domain.ErrAuthPolicyUnavailable (503 at the API boundary) and the
// organization's policy stays unchanged. Client 4xx surfaces as a normal
// error.
//
// It implements domain.OrgAuthPolicyUpdater.
func (r *stytchOrganizationRepository) UpdateAuthPolicy(
	ctx context.Context,
	orgID string,
	emailJitPolicy domain.JitPolicy,
	allowedDomains []string,
	allowedAuthMethods []domain.AllowedAuthMethod,
	ssoJitPolicy domain.SsoJitPolicy,
	ssoJitAllowedConnections []string,
	ssoDefaultConnectionID string,
) error {
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	if r.client == nil {
		// Development mode: placeholder credentials mean no Stytch client.
		// Fail cleanly instead of nil-pointer dereferencing.
		return domain.ErrAuthConnection
	}

	// Read the org first: (a) the first write preserves an org still on
	// `auth_methods: ALL_ALLOWED` (no surprise method removal), and (b) the
	// org's active SSO connections scope the `sso_default_connection_id`
	// ownership check and the SSO-JIT allowlist.
	var org *organizations.Organization
	readErr := r.client.Run(ctx, func() error {
		resp, callErr := r.client.API().Organizations.Get(ctx, &organizations.GetParams{OrganizationID: orgID})
		if callErr != nil {
			return fmt.Errorf("stytch read organization for auth policy: %w", stytchcfg.MapError(callErr))
		}
		org = &resp.Organization
		return nil
	})
	if readErr != nil {
		if errors.Is(readErr, stytchcfg.ErrCircuitOpen) || errors.Is(readErr, stytchcfg.ErrInternal) {
			return fmt.Errorf("stytch read organization for auth policy: %w", domain.ErrAuthPolicyUnavailable)
		}
		return readErr
	}

	activeConnectionIDs := make([]string, 0, len(org.SSOActiveConnections))
	for _, conn := range org.SSOActiveConnections {
		activeConnectionIDs = append(activeConnectionIDs, conn.ConnectionID)
	}

	if err := domain.ValidateAuthPolicyUpdate(
		emailJitPolicy,
		allowedDomains,
		allowedAuthMethods,
		ssoJitPolicy,
		ssoJitAllowedConnections,
		ssoDefaultConnectionID,
		activeConnectionIDs,
	); err != nil {
		return err
	}

	allowedMethods := computeAllowedAuthMethods(org.AuthMethods, org.AllowedAuthMethods, activeConnectionIDs, allowedAuthMethods)

	params := &organizations.UpdateParams{
		OrganizationID:                       orgID,
		EmailJITProvisioning:                 mapJitPolicyToStytch(emailJitPolicy),
		EmailAllowedDomains:                  allowedDomains,
		AuthMethods:                          "RESTRICTED",
		AllowedAuthMethods:                   allowedMethods,
		SSOJITProvisioning:                   mapSsoJitPolicyToStytch(ssoJitPolicy),
		SSOJITProvisioningAllowedConnections: ssoJitAllowedConnections,
		SSODefaultConnectionID:               ssoDefaultConnectionID,
	}

	runErr := r.client.Run(ctx, func() error {
		_, callErr := r.client.API().Organizations.Update(ctx, params)
		if callErr != nil {
			return fmt.Errorf("stytch update auth policy: %w", stytchcfg.MapError(callErr))
		}
		return nil
	})
	if runErr != nil {
		// The breaker can fail the call fast (ErrCircuitOpen) without reaching
		// fn; Stytch 5xx collapses to ErrInternal via MapError. Both are
		// availability failures: map to the 503 domain sentinel.
		if errors.Is(runErr, stytchcfg.ErrCircuitOpen) || errors.Is(runErr, stytchcfg.ErrInternal) {
			return fmt.Errorf("stytch update auth policy: %w", domain.ErrAuthPolicyUnavailable)
		}
		return runErr
	}
	return nil
}

func mapToAuthOrganization(src organizations.Organization) *domain.AuthOrganization {
	var createdAt, updatedAt time.Time
	if src.CreatedAt != nil {
		createdAt = src.CreatedAt.UTC()
	}
	if src.UpdatedAt != nil {
		updatedAt = src.UpdatedAt.UTC()
	}

	return &domain.AuthOrganization{
		OrganizationID:    src.OrganizationID,
		Slug:              src.OrganizationSlug,
		DisplayName:       src.OrganizationName,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		MFAPolicy:         src.MFAPolicy,
		MFAMethods:        src.MFAMethods,
		AllowedMFAMethods: src.AllowedMFAMethods,
	}
}

// mapToAuthPolicy converts a Stytch organization object into the display-only
// domain mirror. Unknown/empty provider values collapse to the safe disabled
// domain values; org-wide `ALL_ALLOWED` SSO JIT (Stytch's documented default)
// maps to SsoJitPolicyDisabled because the platform never writes ALL_ALLOWED.
func mapToAuthPolicy(org organizations.Organization) *domain.AuthPolicy {
	activeIDs := make([]string, 0, len(org.SSOActiveConnections))
	for _, conn := range org.SSOActiveConnections {
		activeIDs = append(activeIDs, conn.ConnectionID)
	}

	return &domain.AuthPolicy{
		EmailJITProvisioning:                 mapJitPolicy(org.EmailJITProvisioning),
		EmailAllowedDomains:                  org.EmailAllowedDomains,
		AuthMethodsRestricted:                org.AuthMethods == "RESTRICTED",
		AllowedAuthMethods:                   mapAllowedAuthMethods(org.AllowedAuthMethods),
		SSOJITProvisioning:                   mapSsoJitPolicy(org.SSOJITProvisioning),
		SSOJITProvisioningAllowedConnections: org.SSOJITProvisioningAllowedConnections,
		SSODefaultConnectionID:               org.SSODefaultConnectionID,
		SSOActiveConnectionIDs:               activeIDs,
	}
}

// mapJitPolicy converts a Stytch `email_jit_provisioning` value to the domain
// JitPolicy. The provider default is `NOT_ALLOWED` (disabled).
func mapJitPolicy(v string) domain.JitPolicy {
	if v == "RESTRICTED" {
		return domain.JitPolicyDomainRestricted
	}
	return domain.JitPolicyDisabled
}

// mapSsoJitPolicy converts a Stytch `sso_jit_provisioning` value to the domain
// SsoJitPolicy. Only `RESTRICTED` maps to connection-restricted; both
// `NOT_ALLOWED` and the provider-default `ALL_ALLOWED` collapse to disabled in
// the domain model (the platform never writes org-wide ALL_ALLOWED). E2E task
// 7.3a verifies the provider's actual create default in the test project.
func mapSsoJitPolicy(v string) domain.SsoJitPolicy {
	if v == "RESTRICTED" {
		return domain.SsoJitPolicyConnectionRestricted
	}
	return domain.SsoJitPolicyDisabled
}

// mapAllowedAuthMethods converts a Stytch `allowed_auth_methods` value list to
// domain AllowedAuthMethod values, skipping values outside the domain enum
// (e.g. `password` or provider-specific oauth values) — the mirror only
// exposes the methods this platform governs.
func mapAllowedAuthMethods(values []string) []domain.AllowedAuthMethod {
	if len(values) == 0 {
		return nil
	}
	out := make([]domain.AllowedAuthMethod, 0, len(values))
	for _, v := range values {
		m := domain.AllowedAuthMethod(v)
		if m.Validate() == nil {
			out = append(out, m)
		}
	}
	return out
}

// mapJitPolicyToStytch converts a domain JitPolicy to the Stytch
// `email_jit_provisioning` value. Only RESTRICTED/NOT_ALLOWED are ever
// written — the Stytch contract exposes no org-creating value.
func mapJitPolicyToStytch(p domain.JitPolicy) string {
	if p == domain.JitPolicyDomainRestricted {
		return "RESTRICTED"
	}
	return "NOT_ALLOWED"
}

// mapSsoJitPolicyToStytch converts a domain SsoJitPolicy to the Stytch
// `sso_jit_provisioning` value. Org-wide ALL_ALLOWED is never written
// (least privilege).
func mapSsoJitPolicyToStytch(p domain.SsoJitPolicy) string {
	if p == domain.SsoJitPolicyConnectionRestricted {
		return "RESTRICTED"
	}
	return "NOT_ALLOWED"
}

// computeAllowedAuthMethods derives the `allowed_auth_methods` list for a
// policy write.
//
// Orgs still on the provider default (`auth_methods: ALL_ALLOWED`) get their
// current effective method set preserved on first write: the project-enabled
// primary methods relevant to the org (at minimum `magic_link`, plus `sso`
// when the org has active SSO connections) — the full effective set, not just
// the `magic_link` floor, so the first save never silently removes methods the
// org already used. Orgs already restricted keep their current allowlist.
// Requested additions are unioned in and the result is deduplicated.
func computeAllowedAuthMethods(
	currentAuthMethods string,
	currentAllowed []string,
	activeConnectionIDs []string,
	requested []domain.AllowedAuthMethod,
) []string {
	var preserved []string
	if currentAuthMethods != "RESTRICTED" {
		preserved = []string{
			string(domain.AuthMethodMagicLink),
			string(domain.AuthMethodEmailOTP),
			string(domain.AuthMethodGoogleOAuth),
			string(domain.AuthMethodMicrosoftOAuth),
		}
		if len(activeConnectionIDs) > 0 {
			preserved = append(preserved, string(domain.AuthMethodSSO))
		}
	} else {
		preserved = append(preserved, currentAllowed...)
	}

	seen := make(map[string]struct{}, len(preserved)+len(requested))
	merged := make([]string, 0, len(preserved)+len(requested))
	for _, m := range preserved {
		if _, dup := seen[m]; dup {
			continue
		}
		seen[m] = struct{}{}
		merged = append(merged, m)
	}
	for _, m := range requested {
		v := string(m)
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		merged = append(merged, v)
	}
	return merged
}
