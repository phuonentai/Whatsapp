package documents

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler *Handler
}

func NewRoutes(handler *Handler) *Routes {
	return &Routes{
		handler: handler,
	}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	docsGroup := router.Group("/example_documents")
	docsGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		resolver.Get("subscription"),
	)
	{
		// Upload document — admin-only in v1 (a document index is a compliance
		// surface: org:manage is required server-side, not just hidden in the UI).
		docsGroup.POST("/upload",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.UploadDocument)

		// List documents — members see workspace docs; the handler filters
		// admin_only docs server-side for members without org:manage.
		docsGroup.GET("",
			auth.RequirePermissionFunc("resource", "view"),
			r.handler.ListDocuments)

		// Document detail — same visibility filter as list (restricted doc =
		// 404 for members without org:manage).
		docsGroup.GET("/:id",
			auth.RequirePermissionFunc("resource", "view"),
			r.handler.GetDocument)

		// Update (title/visibility) — admin-only in v1.
		docsGroup.PATCH("/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.UpdateDocument)

		// Reprocess (Retry) — re-extract + re-embed; admin-only in v1.
		docsGroup.POST("/:id/reprocess",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.ReprocessDocument)

		// Delete document — admin-only in v1.
		docsGroup.DELETE("/:id",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.DeleteDocument)

		// Compliance export (Ley 1581) — indexed documents with visibility.
		// Admin-only: the export is a compliance surface.
		docsGroup.GET("/export/compliance.csv",
			auth.RequirePermissionFunc("org", "manage"),
			r.handler.ExportCompliance)
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
