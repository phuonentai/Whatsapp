package whatsapp

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler *Handler
}

func NewRoutes(handler *Handler) *Routes {
	return &Routes{handler: handler}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	webhooks := router.Group("/v1/webhooks")
	{
		webhooks.GET("/whatsapp", r.handler.HandleVerification)
		webhooks.POST("/whatsapp", r.handler.HandleWebhook)
	}

	mgmt := router.Group("/v1/whatsapp")
	mgmt.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		mgmt.GET("/config",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleGetConfig)

		mgmt.GET("/config/health",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleGetConfigHealth)

		mgmt.PUT("/config",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpsertConfig)

		mgmt.PATCH("/config/toggle",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleToggleConfig)

		mgmt.GET("/signup/meta-config",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleMetaConfig)

		mgmt.POST("/signup/exchange",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleExchangeSignup)

		mgmt.GET("/signup/status",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleSignupStatus)

		mgmt.POST("/config/logs/:id/replay",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleReplayLog)
	}
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
