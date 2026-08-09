// Package invoicing exposes the invoicing webhook ingress.
package invoicing

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/siigo"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type Handler struct {
	invoicingService services.InvoicingService
	webhookSecret    string
	logger           loggerDomain.Logger
}

func NewHandler(svc services.InvoicingService, cfg *siigo.Config, log loggerDomain.Logger) *Handler {
	return &Handler{
		invoicingService: svc,
		webhookSecret:    cfg.WebhookSecret,
		logger:           log,
	}
}

// ProcessSiigoWebhook godoc
// @Summary Process a Siigo invoice-status webhook
// @Description Verifies the HMAC signature and dispatches the event to the invoicing service. No session required.
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "received"
// @Failure 401 {object} map[string]any "invalid signature"
// @Router /api/v1/webhooks/siigo [post]
func (h *Handler) ProcessSiigoWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to read request body"})
		return
	}

	if h.webhookSecret == "" {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "webhook secret not configured"})
		return
	}

	if err := siigo.VerifyWebhookSignature(rawBody, c.GetHeader(siigo.WebhookSignatureHeader), h.webhookSecret); err != nil {
		h.logger.Warn("[ProcessSiigoWebhook] Signature verification failed", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "invalid signature"})
		return
	}

	if err := h.invoicingService.ProcessWebhookEvent(c.Request.Context(), rawBody); err != nil {
		h.logger.Error("[ProcessSiigoWebhook] Failed to process event", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to process webhook"})
		return
	}

	h.logger.Info("[ProcessSiigoWebhook] Event processed")
	c.JSON(http.StatusOK, map[string]any{"received": true})
}
