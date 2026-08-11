package invoicing

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

func (h *Handler) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	// Organization-facing Siigo connection endpoints (org-scoped).
	// Mounted under ApiPrefix "/api" → /api/v1/org/siigo/...
	org := router.Group("/v1/org/siigo")
	org.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
	)
	{
		org.GET("/status",
			auth.RequirePermissionFunc("resource", "view"),
			h.GetConnectionStatus)

		org.POST("/connect",
			auth.RequirePermissionFunc("org", "manage"),
			h.ConnectSiigo)

		org.POST("/request-assisted",
			auth.RequirePermissionFunc("org", "manage"),
			h.RequestAssistedSetup)

		org.POST("/pause",
			auth.RequirePermissionFunc("org", "manage"),
			h.PauseInvoicing)

		org.POST("/resume",
			auth.RequirePermissionFunc("org", "manage"),
			h.ResumeInvoicing)

		org.POST("/activate",
			auth.RequirePermissionFunc("org", "manage"),
			h.ActivateInvoicing)

		org.GET("/numeration",
			auth.RequirePermissionFunc("resource", "view"),
			h.GetNumeration)

		org.POST("/confirm-numeration",
			auth.RequirePermissionFunc("org", "manage"),
			h.ConfirmNumeration)

		org.GET("/import/preview",
			auth.RequirePermissionFunc("resource", "view"),
			h.PreviewImport)

		org.POST("/import/confirm",
			auth.RequirePermissionFunc("org", "manage"),
			h.ConfirmImport)

		org.POST("/sync",
			auth.RequirePermissionFunc("org", "manage"),
			h.SyncCustomers)

		org.POST("/test-invoice",
			auth.RequirePermissionFunc("org", "manage"),
			h.TestInvoice)
	}

	// Admin-scoped assisted provisioning (no org context: target org is in
	// the request body; the admin role gate is org:manage).
	admin := router.Group("/v1/admin/siigo")
	admin.Use(
		resolver.Get("auth"),
		auth.RequirePermissionFunc("org", "manage"),
	)
	{
		admin.GET("/connections", h.AdminListConnections)
		admin.POST("/provision", h.ProvisionSiigo)
	}

	// Webhook ingress (signature-only, no session required).
	// Follows the per-provider pattern established by whatsapp and billing.
	webhooks := router.Group("/v1/webhooks")
	{
		webhooks.POST("/siigo", h.ProcessSiigoWebhook)
	}
}
