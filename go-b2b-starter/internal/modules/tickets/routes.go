package tickets

import (
	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/registry"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

const ModuleKey = "tickets"

type Routes struct {
	handler         *Handler
	featureProvider features.FeatureProvider
}

func NewRoutes(handler *Handler, featureProvider features.FeatureProvider) *Routes {
	return &Routes{handler: handler, featureProvider: featureProvider}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	ticketsGroup := router.Group("/tickets")
	ticketsGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, authcontext.GetRequestContext),
		registry.Require(ModuleKey),
	)

	ticketsGroup.GET("", auth.RequirePermissionFunc("ticket", "view"), r.handler.List)
	ticketsGroup.GET("/:id", auth.RequirePermissionFunc("ticket", "view"), r.handler.Get)
	ticketsGroup.GET("/:id/eventos", auth.RequirePermissionFunc("ticket", "view"), r.handler.ListEvents)
	ticketsGroup.POST("", auth.RequirePermissionFunc("ticket", "manage"), r.handler.Create)
	ticketsGroup.PUT("/:id/estado", auth.RequirePermissionFunc("ticket", "manage"), r.handler.Transition)
	ticketsGroup.PUT("/:id/asignacion", auth.RequirePermissionFunc("ticket", "manage"), r.handler.Assign)
	ticketsGroup.PUT("/:id/prioridad", auth.RequirePermissionFunc("ticket", "manage"), r.handler.SetPriority)
	ticketsGroup.PUT("/:id/etiquetas", auth.RequirePermissionFunc("ticket", "manage"), r.handler.SetTags)
	ticketsGroup.POST("/:id/notas", auth.RequirePermissionFunc("ticket", "manage"), r.handler.AddInternalNote)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
