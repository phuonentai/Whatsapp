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

// updateMfaPolicyRequest is the payload for PUT /organizations/mfa-policy.
// Values mirror the Stytch B2B organization settings (mfa_policy,
// mfa_methods, allowed_mfa_methods).
type updateMfaPolicyRequest struct {
	Policy          string   `json:"mfa_policy" binding:"required"`
	Methods         string   `json:"mfa_methods" binding:"required"`
	AllowedMethods  []string `json:"allowed_mfa_methods"`
}

// UpdateMfaPolicy sets the current organization's MFA policy in the auth
// provider (SSOT). Authenticated + org:manage gated at the route.
//
//	@Summary Update organization MFA policy
//	@Description Sets mfa_policy (OPTIONAL|REQUIRED_FOR_ALL), mfa_methods (ALL_ALLOWED|RESTRICTED) and allowed_mfa_methods (totp, sms_otp) on the Stytch organization. Persisted via the Go backend; outbound calls are circuit-breaker protected.
//	@Tags auth
//	@Accept json
//	@Produce json
//	@Param body body updateMfaPolicyRequest true "MFA policy payload"
//	@Success 200 {object} map[string]bool "Policy updated"
//	@Failure 400 {object} map[string]any "Invalid payload or organization context"
//	@Failure 401 {object} map[string]any "Authentication required"
//	@Failure 403 {object} map[string]any "Insufficient permissions (org:manage)"
//	@Failure 503 {object} httperr.HTTPError "Auth provider unavailable (circuit breaker open / 5xx)"
//	@Router /organizations/mfa-policy [put]
func (h *OrganizationHandler) UpdateMfaPolicy(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		h.logger.Error("missing request context", nil)
		response.Error(c, http.StatusBadRequest, "organization context is required", nil)
		return
	}

	var req updateMfaPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request payload", map[string]interface{}{"error": err.Error()})
		response.Error(c, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	policy := domain.MfaPolicy(req.Policy)
	methods := domain.MfaMethods(req.Methods)
	allowedMethods := make([]domain.MfaMethod, 0, len(req.AllowedMethods))
	for _, m := range req.AllowedMethods {
		allowedMethods = append(allowedMethods, domain.MfaMethod(m))
	}

	if err := h.orgService.UpdateMfaPolicy(
		c.Request.Context(),
		reqCtx.ProviderOrgID,
		policy,
		methods,
		allowedMethods,
	); err != nil {
		h.logger.Error("failed to update organization MFA policy", map[string]interface{}{
			"org_id": reqCtx.ProviderOrgID,
			"error":  err.Error(),
		})

		switch {
		case errors.Is(err, domain.ErrMfaPolicyUnavailable),
			errors.Is(err, domain.ErrAuthConnection):
			// Breaker open or Stytch unreachable/5xx: policy unchanged, answer
			// 503 with a structured error code.
			c.JSON(http.StatusServiceUnavailable, httperr.NewHTTPError(
				http.StatusServiceUnavailable,
				"mfa_policy_update_unavailable",
				"The MFA policy service is temporarily unavailable. Your organization's policy was not changed.",
			))
			return
		case errors.Is(err, domain.ErrInvalidMfaPolicy),
			errors.Is(err, domain.ErrInvalidMfaMethods),
			errors.Is(err, domain.ErrInvalidMfaMethod),
			errors.Is(err, domain.ErrAuthOrganizationIDRequired):
			response.Error(c, http.StatusBadRequest, "invalid MFA policy payload", err)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update MFA policy", err)
			return
		}
	}

	response.Success(c, http.StatusOK, gin.H{"updated": true})
}
