package registry

import (
	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
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
	modulesGroup := router.Group("/modules")
	modulesGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, auth.GetRequestContext),
	)

	modulesGroup.GET("", r.handler.GetCatalog)
	modulesGroup.GET("/me", r.handler.GetMyModules)
	modulesGroup.GET("/org", auth.RequirePermissionFunc("org", "manage"), r.handler.GetOrgModules)
	modulesGroup.PUT("/:key/config", auth.RequirePermissionFunc("org", "manage"), r.handler.SaveModuleConfig)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
