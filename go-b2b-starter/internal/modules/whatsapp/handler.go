package whatsapp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/app/services"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

type Handler struct {
	webhookService services.WebhookService
	configService  services.ConfigService
}

func NewHandler(webhookService services.WebhookService, configService services.ConfigService) *Handler {
	return &Handler{
		webhookService: webhookService,
		configService:  configService,
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
	reqCtx := auth.MustGetRequestContext(c)
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
	reqCtx := auth.MustGetRequestContext(c)
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
	reqCtx := auth.MustGetRequestContext(c)
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

func (h *Handler) HandleToggleConfig(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
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
