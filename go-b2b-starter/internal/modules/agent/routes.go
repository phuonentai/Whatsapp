package agent

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

// Routes registers the agent HTTP API.
type Routes struct {
	handler *Handler
}

// NewRoutes builds the agent routes.
func NewRoutes(handler *Handler) *Routes {
	return &Routes{handler: handler}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	group := router.Group("/api/agent")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		group.GET("/suggestions",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListSuggestions)

		group.POST("/suggestions/:id/approve",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleApproveSuggestion)

		group.POST("/suggestions/:id/reject",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleRejectSuggestion)

		group.GET("/settings",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetSettings)

		group.PUT("/settings",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateSettings)

		group.GET("/flows/:conversationId",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetFlowDebug)

		group.GET("/compliance/export/:contactId",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleExportContact)

		group.POST("/compliance/forget/:contactId",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleForgetContact)
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface.
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
