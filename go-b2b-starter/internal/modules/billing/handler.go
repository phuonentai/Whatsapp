package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

type Handler struct {
	billingService      billingServices.BillingService
	logger              logger.Logger
	webhookVerifier     domain.WebhookVerifier
	polarWebhookSecret  string
	mpWebhookSecret     string
}

func NewHandler(billingService billingServices.BillingService, webhookVerifier domain.WebhookVerifier, polarWebhookSecret, mpWebhookSecret string, log logger.Logger) *Handler {
	return &Handler{
		billingService:     billingService,
		logger:             log,
		webhookVerifier:    webhookVerifier,
		polarWebhookSecret: polarWebhookSecret,
		mpWebhookSecret:    mpWebhookSecret,
	}
}

// GetBillingStatus godoc
// @Summary Get current billing and quota status
// @Description Retrieve the current subscription billing status and invoice quota information for the organization
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} domain.BillingStatus "Current billing and quota status"
// @Failure 400 {object} httperr.HTTPError "Invalid request parameters or missing organization context"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/subscriptions/status [get]
func (h *Handler) GetBillingStatus(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	// Call service layer to get billing status
	billingStatus, err := h.billingService.GetBillingStatus(c.Request.Context(), reqCtx.OrganizationID)
	if err != nil {
		// Check if subscription not found - this is not necessarily an error
		// Organization might not have a subscription yet
		if err == domain.ErrSubscriptionNotFound {
			// Return a response indicating no active subscription
			c.JSON(http.StatusOK, domain.BillingStatus{
				OrganizationID:        reqCtx.OrganizationID,
				HasActiveSubscription: false,
				CanProcessInvoices:    false,
				InvoiceCount:          0,
				Reason:                "No active subscription found",
				CheckedAt:             time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"billing_status_failed",
			fmt.Sprintf("Failed to retrieve billing status: %v", err),
		))
		return
	}

	c.JSON(http.StatusOK, billingStatus)
}

// VerifyPaymentRequest represents the request payload for verifying a payment
type VerifyPaymentRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// VerifyPayment godoc
// @Summary Verify payment from checkout session
// @Description Verifies a payment by checking the Polar checkout session and updates subscription status. This is the primary mechanism for "Verification on Redirect" pattern when user returns from payment page.
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body VerifyPaymentRequest true "Checkout session ID"
// @Success 200 {object} domain.BillingStatus "Verification result with updated billing status"
// @Failure 400 {object} httperr.HTTPError "Invalid request parameters or checkout session failed"
// @Failure 404 {object} httperr.HTTPError "Checkout session not found"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/subscriptions/verify-payment [post]
func (h *Handler) VerifyPayment(c *gin.Context) {
	h.logger.Info("[VerifyPayment] Starting payment verification request", nil)

	// Bind request
	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("[VerifyPayment] Failed to bind request JSON", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			fmt.Sprintf("Invalid request: %v", err),
		))
		return
	}

	h.logger.Info("[VerifyPayment] Request parsed successfully", map[string]any{
		"session_id": req.SessionID,
	})

	// Validate session_id is not empty
	if req.SessionID == "" {
		h.logger.Warn("[VerifyPayment] Missing session_id in request", nil)
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_session_id",
			"Checkout session ID is required",
		))
		return
	}

	h.logger.Info("[VerifyPayment] Calling billing service to verify payment", map[string]any{
		"session_id": req.SessionID,
	})

	// Call service to verify payment
	billingStatus, err := h.billingService.VerifyPaymentFromCheckout(c.Request.Context(), req.SessionID)
	if err != nil {
		// Check if it's a checkout session not found error
		if errors.Is(err, domain.ErrCheckoutSessionNotFound) {
			h.logger.Warn("[VerifyPayment] Checkout session not found", map[string]any{
				"session_id": req.SessionID,
			})
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"session_not_found",
				fmt.Sprintf("Checkout session not found: %s", req.SessionID),
			))
			return
		}

		h.logger.Error("[VerifyPayment] Failed to verify payment", map[string]any{
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"verification_failed",
			fmt.Sprintf("Failed to verify payment: %v", err),
		))
		return
	}

	h.logger.Info("[VerifyPayment] Billing service returned status", map[string]any{
		"session_id":              req.SessionID,
		"has_active_subscription": billingStatus.HasActiveSubscription,
		"can_process_invoices":    billingStatus.CanProcessInvoices,
		"invoice_count":           billingStatus.InvoiceCount,
		"reason":                  billingStatus.Reason,
	})

	// If checkout session is not succeeded, return 400 with reason
	if !billingStatus.HasActiveSubscription && billingStatus.Reason != "Payment verified successfully" {
		h.logger.Warn("[VerifyPayment] Payment not completed", map[string]any{
			"session_id": req.SessionID,
			"reason":     billingStatus.Reason,
		})
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"payment_not_completed",
			billingStatus.Reason,
		))
		return
	}

	h.logger.Info("[VerifyPayment] Payment verification completed successfully", map[string]any{
		"session_id":      req.SessionID,
		"organization_id": billingStatus.OrganizationID,
		"invoice_count":   billingStatus.InvoiceCount,
	})

	c.JSON(http.StatusOK, billingStatus)
}

// CreateMPCheckoutRequest represents the request payload for creating a MercadoPago checkout
type CreateMPCheckoutRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CreateMPCheckout godoc
// @Summary Create a MercadoPago checkout session
// @Description Creates a MercadoPago preapproval for the given plan and returns the hosted Checkout Pro redirect URL (init_point).
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body CreateMPCheckoutRequest true "Plan ID"
// @Success 200 {object} map[string]any "checkoutUrl for the hosted checkout"
// @Failure 400 {object} httperr.HTTPError "Invalid request or checkout creation failed"
// @Failure 401 {object} httperr.HTTPError "Missing or invalid session"
// @Failure 403 {object} httperr.HTTPError "Missing org:manage permission"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/subscriptions/create-mp-checkout [post]
func (h *Handler) CreateMPCheckout(c *gin.Context) {
	var req CreateMPCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			fmt.Sprintf("Invalid request: %v", err),
		))
		return
	}

	// RequireOrganization populates the Gin context key; the request context
	// never carries it, so reading ctx.Value here would always miss.
	stytchOrgID := c.GetString("stytch_org_id")
	if stytchOrgID == "" {
		c.JSON(http.StatusUnauthorized, httperr.NewHTTPError(
			http.StatusUnauthorized,
			"missing_org_context",
			"Organization context is required",
		))
		return
	}

	billingStatus, err := h.billingService.CreateMPCheckout(c.Request.Context(), stytchOrgID, req.PlanID)
	if err != nil {
		h.logger.Error("[CreateMPCheckout] Failed to create checkout", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"checkout_creation_failed",
			fmt.Sprintf("Failed to create MercadoPago checkout: %v", err),
		))
		return
	}

	if billingStatus.CheckoutURL == "" {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"checkout_url_missing",
			"MercadoPago checkout did not return a redirect URL",
		))
		return
	}

	h.logger.Info("[CreateMPCheckout] Checkout created", map[string]any{
		"organization_id": billingStatus.OrganizationID,
		"has_init":        true,
	})

	c.JSON(http.StatusOK, map[string]any{
		"checkout_url": billingStatus.CheckoutURL,
		"checkoutUrl":  billingStatus.CheckoutURL,
		"message":      billingStatus.Reason,
		"checked_at":   billingStatus.CheckedAt,
	})
}

// VerifyMPPaymentRequest represents the request payload for verifying a MercadoPago payment
type VerifyMPPaymentRequest struct {
	PaymentID string `json:"payment_id" binding:"required"`
}

// VerifyMPPayment godoc
// @Summary Verify a MercadoPago payment
// @Description Verifies a MercadoPago payment by polling the payments API and updates subscription + quota records on approval.
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body VerifyMPPaymentRequest true "MercadoPago payment ID"
// @Success 200 {object} map[string]any "Billing status with updated subscription state"
// @Failure 400 {object} httperr.HTTPError "Invalid request or payment not completed"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/subscriptions/verify-mp-payment [post]
func (h *Handler) VerifyMPPayment(c *gin.Context) {
	var req VerifyMPPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			fmt.Sprintf("Invalid request: %v", err),
		))
		return
	}

	billingStatus, err := h.billingService.VerifyMPPayment(c.Request.Context(), req.PaymentID)
	if err != nil {
		h.logger.Error("[VerifyMPPayment] Failed to verify payment", map[string]any{
			"payment_id": req.PaymentID,
			"error":      err.Error(),
		})
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"verification_failed",
			fmt.Sprintf("Failed to verify payment: %v", err),
		))
		return
	}

	h.logger.Info("[VerifyMPPayment] Payment verification completed", map[string]any{
		"payment_id":             req.PaymentID,
		"organization_id":        billingStatus.OrganizationID,
		"has_active_subscription": billingStatus.HasActiveSubscription,
		"reason":                 billingStatus.Reason,
	})

	c.JSON(http.StatusOK, map[string]any{
		"organization_id":         billingStatus.OrganizationID,
		"external_id":             billingStatus.ExternalID,
		"has_active_subscription": billingStatus.HasActiveSubscription,
		"can_process_invoices":    billingStatus.CanProcessInvoices,
		"invoice_count":           billingStatus.InvoiceCount,
		"reason":                  billingStatus.Reason,
		"checked_at":              billingStatus.CheckedAt,
	})
}

// CancelMPSubscriptionRequest represents the request payload for cancelling a MercadoPago subscription
type CancelMPSubscriptionRequest struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
}

// CancelMPSubscription godoc
// @Summary Cancel a MercadoPago subscription
// @Description Cancels a MercadoPago preapproval via the API and marks the local subscription as canceled.
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body CancelMPSubscriptionRequest true "MercadoPago preapproval (subscription) ID"
// @Success 200 {object} map[string]any "Cancellation result"
// @Failure 400 {object} httperr.HTTPError "Invalid request or cancellation failed"
// @Failure 403 {object} httperr.HTTPError "Missing org:manage permission"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/subscriptions/mp-cancel [post]
func (h *Handler) CancelMPSubscription(c *gin.Context) {
	var req CancelMPSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			fmt.Sprintf("Invalid request: %v", err),
		))
		return
	}

	// RequireOrganization populates the Gin context key; pass it explicitly
	// into the service (the request context never carries it).
	stytchOrgID := c.GetString("stytch_org_id")
	if stytchOrgID == "" {
		c.JSON(http.StatusUnauthorized, httperr.NewHTTPError(
			http.StatusUnauthorized,
			"missing_org_context",
			"Organization context is required",
		))
		return
	}

	billingStatus, err := h.billingService.CancelMPSubscription(c.Request.Context(), stytchOrgID, req.SubscriptionID)
	if err != nil {
		h.logger.Error("[CancelMPSubscription] Failed to cancel subscription", map[string]any{
			"subscription_id": req.SubscriptionID,
			"error":           err.Error(),
		})
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"cancellation_failed",
			fmt.Sprintf("Failed to cancel MercadoPago subscription: %v", err),
		))
		return
	}

	h.logger.Info("[CancelMPSubscription] Subscription cancelled", map[string]any{
		"subscription_id": req.SubscriptionID,
		"reason":          billingStatus.Reason,
	})

	c.JSON(http.StatusOK, map[string]any{
		"status":   "cancelled",
		"reason":   billingStatus.Reason,
		"checked_at": billingStatus.CheckedAt,
	})
}

// ProcessPolarWebhook godoc
// @Summary Process a Polar.sh webhook
// @Description Verifies the Svix-style webhook signature and dispatches the event to the Polar webhook service. No session required.
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "received"
// @Failure 401 {object} map[string]any "invalid signature"
// @Router /api/v1/webhooks/polar [post]
func (h *Handler) ProcessPolarWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to read request body"})
		return
	}

	msgID := c.GetHeader(domain.PolarWebhookIDHeader)
	msgTimestamp := c.GetHeader(domain.PolarWebhookTimestampHeader)
	signature := c.GetHeader(domain.PolarWebhookSignatureHeader)

	if h.polarWebhookSecret == "" {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "webhook secret not configured"})
		return
	}

	if err := h.webhookVerifier.VerifyPolar(rawBody, msgID, msgTimestamp, signature, h.polarWebhookSecret); err != nil {
		h.logger.Warn("[ProcessPolarWebhook] Signature verification failed", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "invalid signature"})
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}

	eventType, _ := payload["type"].(string)
	if eventType == "" {
		// Polar nests the event type under data.type in some payloads
		if data, ok := payload["data"].(map[string]any); ok {
			eventType, _ = data["type"].(string)
		}
	}

	if err := h.billingService.ProcessWebhookEvent(c.Request.Context(), eventType, payload); err != nil {
		h.logger.Error("[ProcessPolarWebhook] Failed to process event", map[string]any{
			"event_type": eventType,
			"error":      err.Error(),
		})
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to process webhook"})
		return
	}

	h.logger.Info("[ProcessPolarWebhook] Event processed", map[string]any{
		"event_type": eventType,
	})
	c.JSON(http.StatusOK, map[string]any{"received": true})
}

// ProcessMPWebhook godoc
// @Summary Process a MercadoPago IPN webhook
// @Description Verifies the x-signature HMAC and dispatches the event to the MercadoPago webhook service. No session required.
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "received"
// @Failure 401 {object} map[string]any "invalid signature"
// @Router /api/v1/webhooks/mercadopago [post]
func (h *Handler) ProcessMPWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to read request body"})
		return
	}

	secret := h.mpWebhookSecret
	if secret == "" {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "webhook secret not configured"})
		return
	}

	if err := h.webhookVerifier.VerifyMercadoPago(rawBody, c.GetHeader(domain.MercadoPagoWebhookSignatureHeader), secret); err != nil {
		h.logger.Warn("[ProcessMPWebhook] Signature verification failed", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "invalid signature"})
		return
	}

	if err := h.billingService.ProcessMPWebhookEvent(c.Request.Context(), rawBody); err != nil {
		h.logger.Error("[ProcessMPWebhook] Failed to process event", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to process webhook"})
		return
	}

	h.logger.Info("[ProcessMPWebhook] Event processed")
	c.JSON(http.StatusOK, map[string]any{"received": true})
}

// aiUsageResponse is the JSON shape of GET /api/crm/usage/ai.
type aiUsageResponse struct {
	TokensInput      int64  `json:"tokens_input"`
	TokensOutput     int64  `json:"tokens_output"`
	TokensEmbedding  int64  `json:"tokens_embedding"`
	CreditsUsed      int32  `json:"credits_used"`
	CreditsMax       int32  `json:"credits_max"`
	CreditsRemaining int32  `json:"credits_remaining"`
	PeriodStart      string `json:"period_start"`
	PeriodEnd        string `json:"period_end"`
}

// GetAiUsage godoc
// @Summary Get AI usage for the current billing period
// @Description Returns the organization's AI token/credit consumption for the current billing period
// @Tags usage
// @Accept json
// @Produce json
// @Success 200 {object} aiUsageResponse "AI usage for the current period"
// @Failure 400 {object} httperr.HTTPError "Missing organization context"
// @Failure 500 {object} httperr.HTTPError "Internal server error"
// @Router /api/crm/usage/ai [get]
func (h *Handler) GetAiUsage(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	status, err := h.billingService.GetAiUsageStatus(c.Request.Context(), reqCtx.OrganizationID)
	if err != nil {
		h.logger.Error("failed to get ai usage status", map[string]any{
			"organization_id": reqCtx.OrganizationID,
			"error":           err.Error(),
		})
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"usage_failed",
			"Failed to load AI usage: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, aiUsageResponse{
		TokensInput:      status.TokensInput,
		TokensOutput:     status.TokensOutput,
		TokensEmbedding:  status.TokensEmbedding,
		CreditsUsed:      status.CreditsUsed,
		CreditsMax:       status.CreditsMax,
		CreditsRemaining: status.CreditsRemaining,
		PeriodStart:      status.PeriodStart.Format(time.RFC3339),
		PeriodEnd:        status.PeriodEnd.Format(time.RFC3339),
	})
}
