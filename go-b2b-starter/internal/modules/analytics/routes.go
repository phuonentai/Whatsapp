package analytics

import (
	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/registry"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

const ModuleKey = "analytics"

type Routes struct {
	handler         *Handler
	featureProvider features.FeatureProvider
}

func NewRoutes(handler *Handler, featureProvider features.FeatureProvider) *Routes {
	return &Routes{handler: handler, featureProvider: featureProvider}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	group := router.Group("/analytics")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, authcontext.GetRequestContext),
		registry.Require(ModuleKey),
	)

	group.GET("/revenue", auth.RequirePermissionFunc("invoice", "view"), r.handler.Revenue)
	group.GET("/top-customers", auth.RequirePermissionFunc("invoice", "view"), r.handler.TopCustomers)
	group.GET("/funnel", auth.RequirePermissionFunc("deal", "view"), r.handler.Funnel)
	group.GET("/inactive-contacts", auth.RequirePermissionFunc("contact", "view"), r.handler.InactiveContacts)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
