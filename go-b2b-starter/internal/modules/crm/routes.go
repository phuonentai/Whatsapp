package crm

import (
	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type Routes struct {
	handler         *CRMHandler
	featureProvider features.FeatureProvider
	requestCtx      features.RequestContextFunc
}

func NewRoutes(handler *CRMHandler, featureProvider features.FeatureProvider) *Routes {
	return &Routes{handler: handler, featureProvider: featureProvider}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	crmGroup := router.Group("/crm")
	crmGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, auth.GetRequestContext),
	)

	crmGroup.GET("/entitlement", r.handler.GetEntitlement)

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
	etiquetas.DELETE("/:id", auth.RequirePermissionFunc("contact", "manage"), r.handler.DeleteEtiqueta)
	etiquetas.POST("/entity/:entityType/:entityId", auth.RequirePermissionFunc("contact", "manage"), r.handler.TagEntity)
	etiquetas.DELETE("/entity/:entityType/:entityId/:tagId", auth.RequirePermissionFunc("contact", "manage"), r.handler.UntagEntity)

	conversaciones := crmGroup.Group("/conversaciones")
	conversaciones.GET("", auth.RequirePermissionFunc("contact", "view"), r.handler.ListConversaciones)
	conversaciones.GET("/:id/mensajes", auth.RequirePermissionFunc("contact", "view"), r.handler.ListMensajes)
	conversaciones.POST("/:id/mensajes", auth.RequirePermissionFunc("contact", "manage"), r.handler.HandleSendMessage)
	conversaciones.PATCH("/:id/status", auth.RequirePermissionFunc("contact", "manage"), r.handler.UpdateEstadoConversacion)
}

func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
