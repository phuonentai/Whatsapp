package playbooks

import (
	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler         *Handler
	featureProvider features.FeatureProvider
}

func NewRoutes(handler *Handler, featureProvider features.FeatureProvider) *Routes {
	return &Routes{handler: handler, featureProvider: featureProvider}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	playbooksGroup := router.Group("/playbooks")
	playbooksGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, authcontext.GetRequestContext),
	)

	playbooksGroup.GET("", r.handler.GetCatalog)
	playbooksGroup.POST("/:key/apply", auth.RequirePermissionFunc("org", "manage"), r.handler.Apply)
	playbooksGroup.POST("/:key/reset", auth.RequirePermissionFunc("org", "manage"), r.handler.Reset)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
