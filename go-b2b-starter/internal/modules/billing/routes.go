package billing

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

func (h *Handler) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	subscriptions := router.Group("/subscriptions")
	subscriptions.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
	)
	{
		subscriptions.GET("/status",
			auth.RequirePermissionFunc("resource", "view"),
			h.GetBillingStatus)

		subscriptions.POST("/create-mp-checkout",
			auth.RequirePermissionFunc("org", "manage"),
			h.CreateMPCheckout)

		subscriptions.POST("/verify-mp-payment",
			h.VerifyMPPayment)

		subscriptions.POST("/mp-cancel",
			auth.RequirePermissionFunc("org", "manage"),
			h.CancelMPSubscription)
	}

	router.POST("/subscriptions/verify-payment",
		resolver.Get("auth"),
		h.VerifyPayment)

	// AI usage endpoint (read-only, org-scoped)
	usage := router.Group("/crm/usage/ai")
	usage.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
	)
	{
		usage.GET("",
			auth.RequirePermissionFunc("resource", "view"),
			h.GetAiUsage)
	}

	// Webhook ingress (signature-only, no session required)
	// Mounted as /v1/webhooks under the already-prefixed /api mount (the
	// whatsapp pattern), so the effective paths are /api/v1/webhooks/polar and
	// /api/v1/webhooks/mercadopago — not /api/api/v1/webhooks/*.
	webhooks := router.Group("/v1/webhooks")
	{
		webhooks.POST("/polar", h.ProcessPolarWebhook)
		webhooks.POST("/mercadopago", h.ProcessMPWebhook)
	}
}
