package procurement

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

// Routes registers the procurement HTTP API.
type Routes struct {
	handler *Handler
}

// NewRoutes builds the procurement routes.
func NewRoutes(handler *Handler) *Routes {
	return &Routes{handler: handler}
}

// RegisterRoutes mounts /procurement behind auth + org_context + subscription
// (D9): org:manage on writes/approvals, org:view on reads.
func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	group := router.Group("/procurement")
	group.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		group.GET("/suppliers",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListSuppliers)
		group.POST("/suppliers",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleCreateSupplier)
		group.PUT("/suppliers/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateSupplier)

		group.GET("/products",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListProducts)
		group.POST("/products",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleCreateProduct)
		group.PUT("/products/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleUpdateProduct)

		group.GET("/runs",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListRuns)
		group.POST("/runs",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleCreateRun)
		group.GET("/runs/:id",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleGetRunBoard)
		group.POST("/runs/:id/send",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandleSendRun)
		group.POST("/runs/:id/orders",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.HandlePlaceOrder)
		group.GET("/runs/:id/orders",
			auth.RequirePermissionFunc("org", "view"),
			r.handler.HandleListRunOrders)
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface.
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
