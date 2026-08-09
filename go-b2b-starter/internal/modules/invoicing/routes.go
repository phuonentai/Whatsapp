package invoicing

import (
	"github.com/gin-gonic/gin"

	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

func (h *Handler) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	// Webhook ingress (signature-only, no session required).
	// Follows the per-provider pattern established by whatsapp and billing.
	webhooks := router.Group("/api/v1/webhooks")
	{
		webhooks.POST("/siigo", h.ProcessSiigoWebhook)
	}
}
