package crm

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	gen "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/scope"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler         *CRMHandler
	featureProvider features.FeatureProvider
	requestCtx      features.RequestContextFunc
	pool            *pgxpool.Pool
	store           gen.Store
	logger          logger.Logger
}

func NewRoutes(handler *CRMHandler, featureProvider features.FeatureProvider, pool *pgxpool.Pool, store gen.Store, log logger.Logger) *Routes {
	return &Routes{handler: handler, featureProvider: featureProvider, pool: pool, store: store, logger: log}
}

// rlsEnabledFromEnv lee el flag opt-in de RLS (POSTGRES_RLS_ENABLED).
// La migración 000042 (RLS) y el middleware deben desplegarse juntos.
func rlsEnabledFromEnv() bool {
	raw := os.Getenv("POSTGRES_RLS_ENABLED")
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return v
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	crmGroup := router.Group("/crm")
	crmGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, authcontext.GetRequestContext),
	)

	crmGroup.GET("/entitlement", r.handler.GetEntitlement)

	crmGroup.GET("/export/contactos.csv", auth.RequirePermissionFunc("contact", "export"), r.handler.ExportContactos)
	crmGroup.GET("/export/empresas.csv", features.Require(crmDomain.FeatureCompanies), auth.RequirePermissionFunc("contact", "export"), r.handler.ExportEmpresas)
	crmGroup.GET("/export/negocios.csv", features.Require(crmDomain.FeatureDeals), auth.RequirePermissionFunc("deal", "export"), r.handler.ExportNegocios)
	crmGroup.GET("/export/actividades.csv", features.Require(crmDomain.FeatureActivities), auth.RequirePermissionFunc("activity", "export"), r.handler.ExportActividades)
	crmGroup.GET("/import/contactos/template.csv", auth.RequirePermissionFunc("contact", "view"), r.handler.ImportContactosTemplate)
	crmGroup.POST("/import/contactos", auth.RequirePermissionFunc("contact", "manage"), r.handler.ImportContactos)

	contactos := crmGroup.Group("/contactos")
	contactos.GET("", auth.RequirePermissionFunc("contact", "view"), r.handler.ListContactos)
	contactos.GET("/search", auth.RequirePermissionFunc("contact", "view"), r.handler.SearchContactos)
	contactos.GET("/:id", auth.RequirePermissionFunc("contact", "view"), r.handler.GetContacto)
	contactos.POST("", auth.RequirePermissionFunc("contact", "manage"), r.handler.CreateContacto)
	contactos.PUT("/:id", auth.RequirePermissionFunc("contact", "manage"), r.handler.UpdateContacto)
	contactos.DELETE("/:id", auth.RequirePermissionFunc("contact", "delete"), r.handler.DeleteContacto)

	empresas := crmGroup.Group("/empresas")
	empresas.Use(features.Require(crmDomain.FeatureCompanies))
	empresas.GET("", auth.RequirePermissionFunc("contact", "view"), r.handler.ListEmpresas)
	empresas.GET("/search", auth.RequirePermissionFunc("contact", "view"), r.handler.SearchEmpresas)
	empresas.GET("/:id", auth.RequirePermissionFunc("contact", "view"), r.handler.GetEmpresa)
	empresas.POST("", auth.RequirePermissionFunc("contact", "manage"), r.handler.CreateEmpresa)
	empresas.PUT("/:id", auth.RequirePermissionFunc("contact", "manage"), r.handler.UpdateEmpresa)
	empresas.DELETE("/:id", auth.RequirePermissionFunc("contact", "delete"), r.handler.DeleteEmpresa)

	negocios := crmGroup.Group("/negocios")
	negocios.Use(features.Require(crmDomain.FeatureDeals))
	negocios.GET("", auth.RequirePermissionFunc("deal", "view"), r.handler.ListNegocios)
	negocios.GET("/:id", auth.RequirePermissionFunc("deal", "view"), r.handler.GetNegocio)
	negocios.POST("", auth.RequirePermissionFunc("deal", "manage"), r.handler.CreateNegocio)
	negocios.PUT("/:id", auth.RequirePermissionFunc("deal", "manage"), r.handler.UpdateNegocio)
	negocios.PUT("/:id/etapa", auth.RequirePermissionFunc("deal", "manage"), r.handler.MoverEtapa)
	negocios.DELETE("/:id", auth.RequirePermissionFunc("deal", "manage"), r.handler.DeleteNegocio)

	pipelines := crmGroup.Group("/pipelines")
	pipelines.Use(features.Require(crmDomain.FeatureDeals))
	pipelines.GET("", auth.RequirePermissionFunc("pipeline", "view"), r.handler.ListPipelines)
	pipelines.POST("", auth.RequirePermissionFunc("pipeline", "manage"), r.handler.CreatePipeline)
	pipelines.PUT("/:id", auth.RequirePermissionFunc("pipeline", "manage"), r.handler.UpdatePipeline)
	pipelines.POST("/:id/etapas", auth.RequirePermissionFunc("pipeline", "manage"), r.handler.CreateEtapa)
	pipelines.PUT("/:id/etapas/:stageId", auth.RequirePermissionFunc("pipeline", "manage"), r.handler.UpdateEtapa)

	actividades := crmGroup.Group("/actividades")
	actividades.Use(features.Require(crmDomain.FeatureActivities))
	actividades.GET("", auth.RequirePermissionFunc("contact", "view"), r.handler.ListActividades)
	actividades.POST("", auth.RequirePermissionFunc("contact", "manage"), r.handler.CreateActividad)
	actividades.GET("/contacto/:id", auth.RequirePermissionFunc("contact", "view"), r.handler.ListActividadesByContacto)
	actividades.GET("/negocio/:id", auth.RequirePermissionFunc("deal", "view"), r.handler.ListActividadesByNegocio)
	actividades.GET("/empresa/:id", auth.RequirePermissionFunc("contact", "view"), r.handler.ListActividadesByEmpresa)

	etiquetas := crmGroup.Group("/etiquetas")
	etiquetas.Use(features.Require(crmDomain.FeatureTags))
	etiquetas.GET("", auth.RequirePermissionFunc("contact", "view"), r.handler.ListEtiquetas)
	etiquetas.POST("", auth.RequirePermissionFunc("contact", "manage"), r.handler.CreateEtiqueta)
	etiquetas.PUT("/:id", auth.RequirePermissionFunc("contact", "manage"), r.handler.UpdateEtiqueta)
	etiquetas.DELETE("/:id", auth.RequirePermissionFunc("contact", "manage"), r.handler.DeleteEtiqueta)
	etiquetas.GET("/entity/:entityType/:entityId", auth.RequirePermissionFunc("contact", "view"), r.handler.ListEntityEtiquetas)
	etiquetas.POST("/entity/:entityType/:entityId", auth.RequirePermissionFunc("contact", "manage"), r.handler.TagEntity)
	etiquetas.DELETE("/entity/:entityType/:entityId/:tagId", auth.RequirePermissionFunc("contact", "manage"), r.handler.UntagEntity)

	conversaciones := crmGroup.Group("/conversaciones")
	// Inbox tier: reading (list + thread) requires inbox:view (or org:manage);
	// manual replies require inbox:reply (or org:manage); close/reopen and
	// template sends remain admin-only (org:manage). Enforcement is
	// server-side; the UI additionally hides what the member cannot use.
	//
	// conversation-row-scoping: el middleware de scope resuelve el scope del
	// miembro (flag de entitlement + permisos) y, con RLS activa, abre la
	// transacción del request y setea las session vars (SET LOCAL).
	conversaciones.Use(scope.NewMiddleware(r.pool, r.store, scope.MiddlewareConfig{
		RLSEnabled: rlsEnabledFromEnv(),
	}, r.logger))
	conversaciones.GET("",
		auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage),
		r.handler.ListConversaciones)
	conversaciones.GET("/:id/mensajes",
		auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage),
		r.handler.ListMensajes)
	conversaciones.POST("/:id/mensajes",
		auth.RequireAnyPermissionFunc(auth.PermInboxReply, auth.PermOrgManage),
		r.handler.HandleSendMessage)
	conversaciones.POST("/:id/mensajes/template",
		auth.RequirePermissionFunc("org", "manage"),
		r.handler.HandleSendTemplateMessage)
	conversaciones.PATCH("/:id/status",
		auth.RequirePermissionFunc("org", "manage"),
		r.handler.UpdateEstadoConversacion)
	// Re-asignación (conversation-row-scoping): permiso inbox:reassign;
	// destino validado contra el directorio Stytch; auditoría actor/origen/
	// destino; 503 member_directory_unavailable si el directorio no está.
	conversaciones.GET("/directorio",
		auth.RequireAnyPermissionFunc(auth.PermInboxReassign, auth.PermOrgManage),
		r.handler.HandleListMemberDirectory)
	conversaciones.PATCH("/:id/assignee",
		auth.RequireAnyPermissionFunc(auth.PermInboxReassign, auth.PermOrgManage),
		r.handler.HandleUpdateAssignee)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
