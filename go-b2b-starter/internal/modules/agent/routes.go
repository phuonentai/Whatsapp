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
	group := router.Group("/agent")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		// Suggestions are an admin/agent surface in the member tier: members
		// only read and reply manually (inbox:view/inbox:reply).
		group.GET("/suggestions",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleListSuggestions)

		group.POST("/suggestions/:id/approve",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleApproveSuggestion)

		group.POST("/suggestions/:id/reject",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleRejectSuggestion)

		// Mock-auth test seeding endpoint (AUTH_MOCK_ENABLED=true only).
		group.POST("/suggestions/seed",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleSeedSuggestion)

		group.GET("/settings",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetSettings)

		group.PUT("/settings",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateSettings)

		group.GET("/flows/:conversationId",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetFlowDebug)

		// Conversation context is readable by inbox members (consent-gated
		// server-side) and admins; the UI degrades visibly on withdrawn consent.
		group.GET("/conversations/:id/context",
			auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage),
			r.handler.HandleGetConversationContext)

		group.GET("/compliance/export/:contactId",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleExportContact)

		group.POST("/compliance/forget/:contactId",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleForgetContact)

		// Writing assist: transforms a user-authored draft (rephrase/formal/
		// casual/summarize). Drafting is view-level; sending stays org:manage.
		// Writing assist requires org:manage in the member tier (members reply
		// manually without AI transformation).
		group.POST("/rephrase",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleRephrase)
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface.
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
