package instagram

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/app/services"
	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

type Handler struct {
	webhookService services.WebhookService
	configService  services.ConfigService
	appID          string
	appSecret      string
}

func NewHandler(webhookService services.WebhookService, configService services.ConfigService, appID, appSecret string) *Handler {
	return &Handler{
		webhookService: webhookService,
		configService:  configService,
		appID:          appID,
		appSecret:      appSecret,
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
		if errors.Is(err, igDomain.ErrInvalidSignature) {
			c.JSON(http.StatusUnauthorized, httperr.NewHTTPError(
				http.StatusUnauthorized,
				"invalid_signature",
				err.Error(),
			))
			return
		}
		if errors.Is(err, igDomain.ErrUnknownIGUser) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"unknown_ig_user",
				err.Error(),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"webhook_processing_failed",
			err.Error(),
		))
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) HandleVerification(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if err := h.webhookService.VerifyChallenge(c.Request.Context(), mode, token, challenge); err != nil {
		c.JSON(http.StatusForbidden, httperr.NewHTTPError(
			http.StatusForbidden,
			"verification_failed",
			err.Error(),
		))
		return
	}

	c.String(http.StatusOK, challenge)
}

func (h *Handler) HandleGetConfig(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	config, err := h.configService.GetConfig(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, igDomain.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"config_not_found",
				"No Instagram configuration found for this organization",
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

// HandleReplayLog re-enqueues the events of a stored webhook log from its
// raw payload (operator recovery action for lost/dead-lettered events).
func (h *Handler) HandleReplayLog(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
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
		if errors.Is(err, igDomain.ErrWebhookLogNotFound) {
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
	IGUserID       string                 `json:"ig_user_id"`
	IGUsername     string                 `json:"ig_username,omitempty"`
	FBPageID       string                 `json:"fb_page_id,omitempty"`
	AccessToken    string                 `json:"access_token,omitempty"`
	TokenExpiresAt *string                `json:"token_expires_at,omitempty"`
	WebhookSecret  string                 `json:"webhook_secret"`
	VerifyToken    string                 `json:"verify_token"`
	APIVersion     string                 `json:"api_version,omitempty"`
	GraphAPIURL    string                 `json:"graph_api_url,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
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

	var tokenExpiresAt *time.Time
	if req.TokenExpiresAt != nil && *req.TokenExpiresAt != "" {
		t, parseErr := time.Parse(time.RFC3339, *req.TokenExpiresAt)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
				http.StatusBadRequest,
				"invalid_request",
				"token_expires_at must be an RFC3339 timestamp",
			))
			return
		}
		tokenExpiresAt = &t
	}

	input := &igDomain.InstagramConfig{
		IGUserID:       req.IGUserID,
		IGUsername:     req.IGUsername,
		FBPageID:       req.FBPageID,
		AccessToken:    req.AccessToken,
		TokenExpiresAt: tokenExpiresAt,
		WebhookSecret:  req.WebhookSecret,
		VerifyToken:    req.VerifyToken,
		APIVersion:     req.APIVersion,
		GraphAPIURL:    req.GraphAPIURL,
		Metadata:       req.Metadata,
	}

	config, err := h.configService.UpsertConfig(c.Request.Context(), orgID, input)
	if err != nil {
		if errors.Is(err, igDomain.ErrIGUserIDRequired) ||
			errors.Is(err, igDomain.ErrWebhookSecretRequired) ||
			errors.Is(err, igDomain.ErrOrgRequired) ||
			err.Error() == "access token is required" {
			c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
				http.StatusBadRequest,
				"validation_error",
				err.Error(),
			))
			return
		}
		if errors.Is(err, igDomain.ErrIGUserIDConflict) {
			c.JSON(http.StatusConflict, httperr.NewHTTPError(
				http.StatusConflict,
				"ig_user_id_conflict",
				"This IG user ID is already in use by another organization",
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
		if errors.Is(err, igDomain.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"config_not_found",
				"No Instagram configuration found for this organization",
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

func (h *Handler) HandleRefreshToken(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	config, err := h.configService.RefreshToken(c.Request.Context(), orgID, h.appID, h.appSecret)
	if err != nil {
		if errors.Is(err, igDomain.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"config_not_found",
				"No Instagram configuration found for this organization",
			))
			return
		}
		if errors.Is(err, igDomain.ErrTokenRefreshFailed) {
			c.JSON(http.StatusBadGateway, httperr.NewHTTPError(
				http.StatusBadGateway,
				"instagram_token_refresh_failed",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"config_refresh_failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, config)
}
