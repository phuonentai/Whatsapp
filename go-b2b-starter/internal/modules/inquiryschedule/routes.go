package inquiryschedule

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

// Routes registers the inquiry-scheduling HTTP API.
type Routes struct {
	handler *Handler
}

// NewRoutes builds the inquiry-scheduling routes.
func NewRoutes(handler *Handler) *Routes {
	return &Routes{handler: handler}
}

// RegisterRoutes mounts /procurement/schedules + /procurement/followup-settings
// behind auth + org_context + subscription: org:manage on writes, org:view on
// reads (RBAC per the supplier-inquiries convention).
func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	group := router.Group("/procurement")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		group.GET("/schedules",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListSchedules)
		group.POST("/schedules",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleCreateSchedule)
		group.GET("/schedules/:id",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetSchedule)
		group.PUT("/schedules/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateSchedule)
		group.POST("/schedules/:id/pause",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandlePauseSchedule)
		group.POST("/schedules/:id/resume",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleResumeSchedule)
		group.DELETE("/schedules/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleDeleteSchedule)

		group.GET("/followup-settings",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetFollowUpSettings)
		group.PUT("/followup-settings",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateFollowUpSettings)
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface.
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
