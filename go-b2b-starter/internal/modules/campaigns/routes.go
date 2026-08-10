package campaigns

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler *Handler
}

func NewRoutes(handler *Handler) *Routes {
	return &Routes{handler: handler}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	group := router.Group("/crm/campanas")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
		features.Require(domain.FeatureCampaigns),
	)
	{
		group.GET("/segments",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.ListSegments)

		group.POST("/segments",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.CreateSegment)

		group.POST("/segments/preview",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.PreviewSpec)

		group.POST("/segments/ai-build",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.AiBuild)

		group.GET("/segments/:id",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.GetSegment)

		group.GET("/segments/:id/preview",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.PreviewSegment)

		group.PUT("/segments/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.UpdateSegment)

		group.DELETE("/segments/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.DeleteSegment)

		group.GET("",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.ListCampaigns)

		group.POST("",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.CreateCampaign)

		group.POST("/:id/launch",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.LaunchCampaign)

		group.GET("/:id/recipients",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.ListRecipients)
	}
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
