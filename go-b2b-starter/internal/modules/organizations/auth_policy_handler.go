package organizations

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

// authPolicyResponse is the display-only mirror of the organization's auth
// policy returned by GET /organizations/auth-policy. Values mirror the Stytch
// B2B organization settings; the mirror is NEVER consulted for authorization
// (Stytch enforces auth methods and JIT provisioning at authentication time).
type authPolicyResponse struct {
	EmailJITProvisioning                 domain.JitPolicy           `json:"email_jit_provisioning"`
	EmailAllowedDomains                  []string                   `json:"email_allowed_domains,omitempty"`
	AuthMethodsRestricted                bool                       `json:"auth_methods_restricted"`
	AllowedAuthMethods                   []domain.AllowedAuthMethod `json:"allowed_auth_methods,omitempty"`
	SSOJITProvisioning                   domain.SsoJitPolicy        `json:"sso_jit_provisioning"`
	SSOJITProvisioningAllowedConnections []string                   `json:"sso_jit_provisioning_allowed_connections,omitempty"`
	SSODefaultConnectionID               string                     `json:"sso_default_connection_id,omitempty"`
	SSOActiveConnectionIDs               []string                   `json:"sso_active_connection_ids,omitempty"`
}

// updateAuthPolicyRequest is the payload for PUT /organizations/auth-policy.
// The write contract is the full org auth policy; email_jit_provisioning and
// sso_jit_provisioning default to their disabled values when omitted.
type updateAuthPolicyRequest struct {
	EmailJITProvisioning                 string   `json:"email_jit_provisioning"`
	EmailAllowedDomains                  []string `json:"email_allowed_domains"`
	AllowedAuthMethods                   []string `json:"allowed_auth_methods"`
	SSOJITProvisioning                   string   `json:"sso_jit_provisioning"`
	SSOJITProvisioningAllowedConnections []string `json:"sso_jit_provisioning_allowed_connections"`
	SSODefaultConnectionID               string   `json:"sso_default_connection_id"`
}

// GetAuthPolicy returns the current organization's auth policy mirror from the
// auth provider (SSOT). Authenticated + org:manage gated at the route.
//
//	@Summary Get organization auth policy mirror
//	@Description Returns the display-only mirror of the organization's auth settings (email JIT provisioning, allowed domains, allowed auth methods, SSO JIT) read from Stytch. The mirror is never consulted for authorization.
//	@Tags auth
//	@Produce json
//	@Success 200 {object} authPolicyResponse "Auth policy mirror"
//	@Failure 400 {object} map[string]any "Organization context required"
//	@Failure 401 {object} map[string]any "Authentication required"
//	@Failure 403 {object} map[string]any "Insufficient permissions (org:manage)"
//	@Failure 503 {object} httperr.HTTPError "Auth provider unavailable (circuit breaker open / 5xx)"
//	@Router /organizations/auth-policy [get]
func (h *OrganizationHandler) GetAuthPolicy(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		h.logger.Error("missing request context", nil)
		response.Error(c, http.StatusBadRequest, "organization context is required", nil)
		return
	}

	policy, err := h.orgService.GetAuthPolicy(c.Request.Context(), reqCtx.ProviderOrgID)
	if err != nil {
		h.logger.Error("failed to get organization auth policy", map[string]interface{}{
			"org_id": reqCtx.ProviderOrgID,
			"error":  err.Error(),
		})

		switch {
		case errors.Is(err, domain.ErrAuthPolicyUnavailable),
			errors.Is(err, domain.ErrAuthConnection):
			// Breaker open or Stytch unreachable/5xx: answer 503 with a
			// structured error code.
			c.JSON(http.StatusServiceUnavailable, httperr.NewHTTPError(
				http.StatusServiceUnavailable,
				"auth_policy_unavailable",
				"The auth policy service is temporarily unavailable.",
			))
			return
		case errors.Is(err, domain.ErrAuthOrganizationIDRequired):
			response.Error(c, http.StatusBadRequest, "organization context is required", err)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "failed to get auth policy", err)
			return
		}
	}

	response.Success(c, http.StatusOK, authPolicyResponse{
		EmailJITProvisioning:                 policy.EmailJITProvisioning,
		EmailAllowedDomains:                  policy.EmailAllowedDomains,
		AuthMethodsRestricted:                policy.AuthMethodsRestricted,
		AllowedAuthMethods:                   policy.AllowedAuthMethods,
		SSOJITProvisioning:                   policy.SSOJITProvisioning,
		SSOJITProvisioningAllowedConnections: policy.SSOJITProvisioningAllowedConnections,
		SSODefaultConnectionID:               policy.SSODefaultConnectionID,
		SSOActiveConnectionIDs:               policy.SSOActiveConnectionIDs,
	})
}

// UpdateAuthPolicy sets the current organization's auth policy in the auth
// provider (SSOT). Authenticated + org:manage gated at the route.
//
//	@Summary Update organization auth policy
//	@Description Sets email_jit_provisioning (DISABLED|DOMAIN_RESTRICTED), email_allowed_domains, allowed_auth_methods (magic_link, email_otp, sso, google_oauth, microsoft_oauth), sso_jit_provisioning (DISABLED|CONNECTION_RESTRICTED), sso_jit_provisioning_allowed_connections and sso_default_connection_id on the Stytch organization. Always writes auth_methods=RESTRICTED (enforced-list mode) with the org's preserved method set plus the requested additions. Persisted via the Go backend; outbound calls are circuit-breaker protected.
//	@Tags auth
//	@Accept json
//	@Produce json
//	@Param body body updateAuthPolicyRequest true "Auth policy payload"
//	@Success 200 {object} map[string]bool "Policy updated"
//	@Failure 400 {object} map[string]any "Invalid payload or organization context"
//	@Failure 401 {object} map[string]any "Authentication required"
//	@Failure 403 {object} map[string]any "Insufficient permissions (org:manage)"
//	@Failure 503 {object} httperr.HTTPError "Auth provider unavailable (circuit breaker open / 5xx)"
//	@Router /organizations/auth-policy [put]
func (h *OrganizationHandler) UpdateAuthPolicy(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		h.logger.Error("missing request context", nil)
		response.Error(c, http.StatusBadRequest, "organization context is required", nil)
		return
	}

	var req updateAuthPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request payload", map[string]interface{}{"error": err.Error()})
		response.Error(c, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	emailJitPolicy := domain.JitPolicy(req.EmailJITProvisioning)
	if emailJitPolicy == "" {
		emailJitPolicy = domain.JitPolicyDisabled
	}
	ssoJitPolicy := domain.SsoJitPolicy(req.SSOJITProvisioning)
	if ssoJitPolicy == "" {
		ssoJitPolicy = domain.SsoJitPolicyDisabled
	}
	allowedAuthMethods := make([]domain.AllowedAuthMethod, 0, len(req.AllowedAuthMethods))
	for _, m := range req.AllowedAuthMethods {
		allowedAuthMethods = append(allowedAuthMethods, domain.AllowedAuthMethod(m))
	}

	if err := h.orgService.UpdateAuthPolicy(
		c.Request.Context(),
		reqCtx.ProviderOrgID,
		emailJitPolicy,
		req.EmailAllowedDomains,
		allowedAuthMethods,
		ssoJitPolicy,
		req.SSOJITProvisioningAllowedConnections,
		req.SSODefaultConnectionID,
	); err != nil {
		h.logger.Error("failed to update organization auth policy", map[string]interface{}{
			"org_id": reqCtx.ProviderOrgID,
			"error":  err.Error(),
		})

		switch {
		case errors.Is(err, domain.ErrAuthPolicyUnavailable),
			errors.Is(err, domain.ErrAuthConnection):
			// Breaker open or Stytch unreachable/5xx: policy unchanged, answer
			// 503 with a structured error code.
			c.JSON(http.StatusServiceUnavailable, httperr.NewHTTPError(
				http.StatusServiceUnavailable,
				"auth_policy_update_unavailable",
				"The auth policy service is temporarily unavailable. Your organization's policy was not changed.",
			))
			return
		case errors.Is(err, domain.ErrInvalidAuthPolicy),
			errors.Is(err, domain.ErrAuthOrganizationIDRequired):
			response.Error(c, http.StatusBadRequest, "invalid auth policy payload", err)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update auth policy", err)
			return
		}
	}

	response.Success(c, http.StatusOK, gin.H{"updated": true})
}
