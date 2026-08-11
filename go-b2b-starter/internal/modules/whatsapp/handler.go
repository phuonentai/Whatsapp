package whatsapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/app/services"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

type Handler struct {
	webhookService services.WebhookService
	configService  services.ConfigService
	signupService  services.SignupService
}

func NewHandler(webhookService services.WebhookService, configService services.ConfigService, signupService services.SignupService) *Handler {
	return &Handler{
		webhookService: webhookService,
		configService:  configService,
		signupService:  signupService,
	}
}

func (h *Handler) HandleWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_body",
			"Failed to read request body",
		))
		return
	}

	signatureHeader := c.GetHeader("X-Hub-Signature-256")

	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_json",
			"Failed to parse request body as JSON",
		))
		return
	}

	if err := h.webhookService.ProcessWebhook(
		c.Request.Context(),
		rawBody,
		payload,
		signatureHeader,
	); err != nil {
		if errors.Is(err, whatsappDomain.ErrInvalidSignature) {
			c.JSON(http.StatusUnauthorized, httperr.NewHTTPError(
				http.StatusUnauthorized,
				"invalid_signature",
				err.Error(),
			))
			return
		}
		if errors.Is(err, whatsappDomain.ErrUnknownPhoneNumber) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"unknown_phone_number",
				err.Error(),
			))
			return
		}

		status := http.StatusInternalServerError
		code := "webhook_processing_failed"

		msg := err.Error()
		c.JSON(status, httperr.NewHTTPError(status, code, msg))
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) HandleVerification(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	challengeResponse, err := h.webhookService.VerifyChallenge(c.Request.Context(), mode, token, challenge)
	if err != nil {
		c.JSON(http.StatusForbidden, httperr.NewHTTPError(
			http.StatusForbidden,
			"verification_failed",
			err.Error(),
		))
		return
	}

	c.String(http.StatusOK, challengeResponse)
}

func (h *Handler) HandleGetConfig(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	config, err := h.configService.GetConfig(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"config_not_found",
				"No WhatsApp configuration found for this organization",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"config_fetch_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *Handler) HandleGetConfigHealth(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	stats, err := h.webhookService.GetWebhookLogStats(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"health_fetch_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, stats)
}

// HandleReplayLog re-enqueues the events of a stored webhook log from its
// raw payload (operator recovery action for lost/dead-lettered events).
func (h *Handler) HandleReplayLog(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	logID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_log_id",
			"webhook log id must be an integer",
		))
		return
	}

	count, err := h.webhookService.Replay(c.Request.Context(), orgID, int32(logID))
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrWebhookLogNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"webhook_log_not_found",
				"Webhook log not found for this organization",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"replay_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{"replayed_events": count})
}

type UpsertConfigRequest struct {
	PhoneNumberID string                 `json:"phone_number_id"`
	BusinessPhone string                 `json:"business_phone"`
	WebhookSecret string                 `json:"webhook_secret"`
	VerifyToken   string                 `json:"verify_token"`
	AppID         string                 `json:"app_id,omitempty"`
	WabaID        string                 `json:"waba_id,omitempty"`
	AccessToken   string                 `json:"access_token,omitempty"`
	APIVersion    string                 `json:"api_version,omitempty"`
	GraphAPIURL   string                 `json:"graph_api_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

func (h *Handler) HandleUpsertConfig(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	var req UpsertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request body: "+err.Error(),
		))
		return
	}

	input := &whatsappDomain.WhatsAppConfig{
		PhoneNumberID: req.PhoneNumberID,
		BusinessPhone: req.BusinessPhone,
		WebhookSecret: req.WebhookSecret,
		VerifyToken:   req.VerifyToken,
		AppID:         req.AppID,
		WABAID:        req.WabaID,
		AccessToken:   req.AccessToken,
		APIVersion:    req.APIVersion,
		GraphAPIURL:   req.GraphAPIURL,
		Metadata:      req.Metadata,
	}

	config, err := h.configService.UpsertConfig(c.Request.Context(), orgID, input)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrPhoneNumberIDRequired) ||
			errors.Is(err, whatsappDomain.ErrWebhookSecretRequired) ||
			errors.Is(err, whatsappDomain.ErrOrgRequired) {
			c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
				http.StatusBadRequest,
				"validation_error",
				err.Error(),
			))
			return
		}
		if err.Error() == "phone_number_id_conflict: this phone number ID is already in use by another organization" {
			c.JSON(http.StatusConflict, httperr.NewHTTPError(
				http.StatusConflict,
				"phone_number_id_conflict",
				"This phone number ID is already in use by another organization",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"config_upsert_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *Handler) HandleMetaConfig(c *gin.Context) {
	metaConfig, err := h.signupService.MetaConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"meta_config_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, metaConfig)
}

type ExchangeSignupRequest struct {
	Code string `json:"code"`
}

func (h *Handler) HandleExchangeSignup(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID
	actorMemberID := ""
	if reqCtx.Identity != nil {
		actorMemberID = reqCtx.Identity.UserID
	}

	var req ExchangeSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request body: "+err.Error(),
		))
		return
	}

	result, err := h.signupService.Exchange(c.Request.Context(), orgID, req.Code, actorMemberID)
	if err != nil {
		var failedErr *whatsappDomain.SignupFailedError
		switch {
		case errors.Is(err, whatsappDomain.ErrSignupCodeRequired):
			c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
				http.StatusBadRequest,
				"validation_error",
				"Authorization code is required",
			))
		case errors.Is(err, whatsappDomain.ErrSignupInProgress):
			c.JSON(http.StatusConflict, httperr.NewHTTPError(
				http.StatusConflict,
				"signup_in_progress",
				"A signup is already in progress for this organization",
			))
		case errors.Is(err, whatsappDomain.ErrSignupAlreadyConnected):
			c.JSON(http.StatusConflict, httperr.NewHTTPError(
				http.StatusConflict,
				"signup_already_connected",
				"This organization is already connected",
			))
		case errors.As(err, &failedErr):
			c.JSON(http.StatusBadGateway, httperr.NewHTTPError(
				http.StatusBadGateway,
				"signup_failed",
				fmt.Sprintf("Signup failed with error %s", failedErr.Code),
			))
		default:
			c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
				http.StatusInternalServerError,
				"signup_exchange_failed",
				err.Error(),
			))
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) HandleSignupStatus(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	flow, err := h.signupService.Status(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrSignupNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"signup_not_found",
				"No signup flow found for this organization",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"signup_status_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, flow)
}

func (h *Handler) HandleToggleConfig(c *gin.Context) {
	reqCtx := authcontext.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	config, err := h.configService.ToggleConfig(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"config_not_found",
				"No WhatsApp configuration found for this organization",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"config_toggle_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, config)
}
