package crm

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type CRMHandler struct {
	contactService      services.ContactService
	companyService      services.CompanyService
	dealService         services.DealService
	pipelineService     services.PipelineService
	activityService     services.ActivityService
	tagService          services.TagService
	conversationService services.ConversationService
	outboundService     services.OutboundService
	featureProvider     features.FeatureProvider
	logger              logger.Logger
}

func NewCRMHandler(
	contactService services.ContactService, companyService services.CompanyService,
	dealService services.DealService, pipelineService services.PipelineService,
	activityService services.ActivityService, tagService services.TagService,
	conversationService services.ConversationService,
	outboundService services.OutboundService,
	featureProvider features.FeatureProvider, logger logger.Logger,
) *CRMHandler {
	return &CRMHandler{
		contactService: contactService, companyService: companyService,
		dealService: dealService, pipelineService: pipelineService,
		activityService: activityService, tagService: tagService,
		conversationService: conversationService,
		outboundService:     outboundService,
		featureProvider: featureProvider, logger: logger,
	}
}

func (h *CRMHandler) GetEntitlement(c *gin.Context) {
	e := features.GetEntitlement(c)
	if e == nil { response.Error(c, http.StatusOK, "", nil); return }
	response.Success(c, http.StatusOK, map[string]interface{}{
		"funcionalidades": e.Features, "cuotas": e.Quotas, "uso": e.Usage,
		"solo_lectura": e.IsReadOnly, "periodo_gracia": e.IsGracePeriod, "plan": e.PlanName,
		"modulos": e.Modules,
	})
}

// ---- Contactos ----

func (h *CRMHandler) ListContactos(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	limit, offset := parsePagination(c)
	r, err := h.contactService.List(c.Request.Context(), ctx.OrganizationID, c.Query("source"), c.Query("lead_status"), 0, 0, limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar contactos", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) SearchContactos(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.contactService.Search(c.Request.Context(), ctx.OrganizationID, c.Query("q"), limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al buscar contactos", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) GetContacto(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	r, err := h.contactService.GetByID(c.Request.Context(), ctx.OrganizationID, parseID(c))
	if err != nil { response.Error(c, http.StatusNotFound, "Contacto no encontrado", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateContacto(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.CreateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.OrganizationID = ctx.OrganizationID
	r, err := h.contactService.Create(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrContactDuplicateEmail) {
			response.Error(c, http.StatusConflict, domain.ErrContactDuplicateEmail.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al crear contacto", err); return
	}
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdateContacto(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.UpdateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.ID = parseID(c); req.OrganizationID = ctx.OrganizationID
	r, err := h.contactService.Update(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrContactDuplicateEmail) {
			response.Error(c, http.StatusConflict, domain.ErrContactDuplicateEmail.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al actualizar contacto", err); return
	}
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) DeleteContacto(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if err := h.contactService.Delete(c.Request.Context(), ctx.OrganizationID, parseID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al eliminar contacto", err); return
	}
	response.Success(c, http.StatusOK, gin.H{"mensaje": "Contacto eliminado"})
}

// ---- Empresas ----

func (h *CRMHandler) ListEmpresas(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.companyService.List(c.Request.Context(), ctx.OrganizationID, limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar empresas", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) SearchEmpresas(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.companyService.Search(c.Request.Context(), ctx.OrganizationID, c.Query("q"), limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al buscar empresas", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) GetEmpresa(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	r, err := h.companyService.GetByID(c.Request.Context(), ctx.OrganizationID, parseID(c))
	if err != nil { response.Error(c, http.StatusNotFound, "Empresa no encontrada", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateEmpresa(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.companyService.Create(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyDuplicateName) {
			response.Error(c, http.StatusConflict, domain.ErrCompanyDuplicateName.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al crear empresa", err); return
	}
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdateEmpresa(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.ID = parseID(c)
	r, err := h.companyService.Update(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyDuplicateName) {
			response.Error(c, http.StatusConflict, domain.ErrCompanyDuplicateName.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al actualizar empresa", err); return
	}
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) DeleteEmpresa(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if err := h.companyService.Delete(c.Request.Context(), ctx.OrganizationID, parseID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al eliminar empresa", err); return
	}
	response.Success(c, http.StatusOK, gin.H{"mensaje": "Empresa eliminada"})
}

// ---- Negocios ----

func (h *CRMHandler) ListNegocios(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.dealService.List(c.Request.Context(), ctx.OrganizationID,
		parseInt(c.Query("pipeline_id")), parseInt(c.Query("stage_id")), c.Query("estado"),
		parseInt(c.Query("contact_id")), limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar negocios", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) GetNegocio(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	r, err := h.dealService.GetByID(c.Request.Context(), ctx.OrganizationID, parseID(c))
	if err != nil { response.Error(c, http.StatusNotFound, "Negocio no encontrado", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateNegocio(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.CreateDealRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.dealService.Create(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al crear negocio", err); return }
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdateNegocio(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.UpdateDealRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.ID = parseID(c); req.OrganizationID = ctx.OrganizationID
	r, err := h.dealService.Update(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al actualizar negocio", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) MoverEtapa(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req struct{ StageID int32 `json:"stage_id"`; OldStageName string `json:"old_stage_name"`; NewStageName string `json:"new_stage_name"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.dealService.UpdateStage(c.Request.Context(), ctx.OrganizationID, parseID(c), req.StageID, ctx.AccountID, req.OldStageName, req.NewStageName)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al mover etapa", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) DeleteNegocio(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if err := h.dealService.Delete(c.Request.Context(), ctx.OrganizationID, parseID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al eliminar negocio", err); return
	}
	response.Success(c, http.StatusOK, gin.H{"mensaje": "Negocio eliminado"})
}

// ---- Pipelines ----

func (h *CRMHandler) ListPipelines(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	r, err := h.pipelineService.GetOrCreateDefault(c.Request.Context(), ctx.OrganizationID)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar pipelines", err); return }
	all, err := h.pipelineService.List(c.Request.Context(), ctx.OrganizationID)
	if err == nil { response.Success(c, http.StatusOK, all); return }
	response.Success(c, http.StatusOK, []*domain.PipelineWithStages{r})
}
func (h *CRMHandler) CreatePipeline(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.pipelineService.Create(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al crear pipeline", err); return }
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdatePipeline(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.ID = parseID(c)
	r, err := h.pipelineService.Update(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al actualizar pipeline", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateEtapa(c *gin.Context) {
	var req services.CreateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.pipelineService.CreateStage(c.Request.Context(), parseID(c), &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al crear etapa", err); return }
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdateEtapa(c *gin.Context) {
	var req services.UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.pipelineService.UpdateStage(c.Request.Context(), parseInt(c.Param("stageId")), parseID(c), &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al actualizar etapa", err); return }
	response.Success(c, http.StatusOK, r)
}

// ---- Actividades ----

func (h *CRMHandler) ListActividades(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.activityService.ListByOrganization(c.Request.Context(), ctx.OrganizationID,
		c.Query("tipo"), c.Query("entity_type"), parseInt(c.Query("entity_id")), limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar actividades", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateActividad(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req services.CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	req.RealizadaPor = &ctx.AccountID
	r, err := h.activityService.Create(c.Request.Context(), ctx.OrganizationID, &req)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al crear actividad", err); return }
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) ListActividadesByContacto(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.activityService.ListByContact(c.Request.Context(), parseID(c), ctx.OrganizationID, limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar actividades", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) ListActividadesByNegocio(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.activityService.ListByDeal(c.Request.Context(), parseID(c), ctx.OrganizationID, limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar actividades", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) ListActividadesByEmpresa(c *gin.Context) {
	ctx := auth.GetRequestContext(c); limit, offset := parsePagination(c)
	r, err := h.activityService.ListByCompany(c.Request.Context(), parseID(c), ctx.OrganizationID, limit, offset)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar actividades", err); return }
	response.Success(c, http.StatusOK, r)
}

// ---- Etiquetas ----

func (h *CRMHandler) ListEtiquetas(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	r, err := h.tagService.List(c.Request.Context(), ctx.OrganizationID)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar etiquetas", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) CreateEtiqueta(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req struct{ Nombre string `json:"nombre"`; Color string `json:"color"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.tagService.Create(c.Request.Context(), ctx.OrganizationID, req.Nombre, req.Color)
	if err != nil {
		if errors.Is(err, domain.ErrTagDuplicateName) {
			response.Error(c, http.StatusConflict, domain.ErrTagDuplicateName.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al crear etiqueta", err); return
	}
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UpdateEtiqueta(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	var req struct{ Nombre string `json:"nombre"`; Color string `json:"color"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.tagService.Update(c.Request.Context(), ctx.OrganizationID, parseID(c), req.Nombre, req.Color)
	if err != nil {
		if errors.Is(err, domain.ErrTagDuplicateName) {
			response.Error(c, http.StatusConflict, domain.ErrTagDuplicateName.Error(), err); return
		}
		response.Error(c, http.StatusInternalServerError, "Error al actualizar etiqueta", err); return
	}
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) ListEntityEtiquetas(c *gin.Context) {
	entityType := domain.EntityType(c.Param("entityType"))
	switch entityType {
	case domain.EntityTypeContact, domain.EntityTypeCompany, domain.EntityTypeDeal:
	default:
		response.Error(c, http.StatusBadRequest, "Tipo de entidad inválido. Valores: contact, company, deal", nil)
		return
	}
	entityID := parseIDGiven(c.Param("entityId"))
	r, err := h.tagService.ListByEntity(c.Request.Context(), entityType, entityID)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al listar etiquetas", err); return }
	response.Success(c, http.StatusOK, r)
}
func (h *CRMHandler) DeleteEtiqueta(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if err := h.tagService.Delete(c.Request.Context(), ctx.OrganizationID, parseID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al eliminar etiqueta", err); return
	}
	response.Success(c, http.StatusOK, gin.H{"mensaje": "Etiqueta eliminada"})
}
func (h *CRMHandler) TagEntity(c *gin.Context) {
	entityType := domain.EntityType(c.Param("entityType"))
	entityID := parseIDGiven(c.Param("entityId"))
	var req struct{ TagID int32 `json:"tag_id"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.Error(c, http.StatusBadRequest, "Solicitud inválida", err); return }
	r, err := h.tagService.AttachToEntity(c.Request.Context(), req.TagID, entityType, entityID)
	if err != nil { response.Error(c, http.StatusInternalServerError, "Error al etiquetar", err); return }
	response.Success(c, http.StatusCreated, r)
}
func (h *CRMHandler) UntagEntity(c *gin.Context) {
	entityType := domain.EntityType(c.Param("entityType"))
	entityID := parseIDGiven(c.Param("entityId"))
	tagID := parseIDGiven(c.Param("tagId"))
	if err := h.tagService.DetachFromEntity(c.Request.Context(), tagID, entityType, entityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al quitar etiqueta", err); return
	}
	response.Success(c, http.StatusOK, gin.H{"mensaje": "Etiqueta removida"})
}

// ---- Conversaciones ----

func (h *CRMHandler) ListConversaciones(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	limit, offset := parsePagination(c)
	statusFilter := c.Query("status")

	convs, err := h.conversationService.ListConversations(c.Request.Context(), ctx.OrganizationID, limit, offset, statusFilter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar conversaciones", err)
		return
	}
	response.Success(c, http.StatusOK, convs)
}

func (h *CRMHandler) ListMensajes(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	convID := parseID(c)
	limit, offset := parsePagination(c)

	msgs, err := h.conversationService.ListMessages(c.Request.Context(), ctx.OrganizationID, convID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Error al listar mensajes", err)
		return
	}
	response.Success(c, http.StatusOK, msgs)
}

func (h *CRMHandler) UpdateEstadoConversacion(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	convID := parseID(c)

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud invalida", err)
		return
	}

	conv, err := h.conversationService.UpdateStatus(c.Request.Context(), ctx.OrganizationID, convID, domain.ConversationStatus(req.Status))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Error al actualizar estado", err)
		return
	}
	response.Success(c, http.StatusOK, conv)
}

// ---- Outbound Messages ----

func (h *CRMHandler) HandleSendMessage(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	convID := parseID(c)

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud invalida", err)
		return
	}

	msg, err := h.outboundService.SendMessage(c.Request.Context(), ctx.OrganizationID, convID, req.Content)
	if err != nil {
		switch {
		case err.Error() == "message content is required":
			response.Error(c, http.StatusBadRequest, "El contenido del mensaje es requerido", err)
		case contains(err.Error(), "whatsapp_not_configured"):
			response.Error(c, http.StatusBadRequest, "WhatsApp no esta configurado", err)
		case contains(err.Error(), "whatsapp_no_access_token"):
			response.Error(c, http.StatusBadRequest, "Token de acceso no configurado", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al enviar mensaje", err)
		}
		return
	}
	response.Success(c, http.StatusOK, msg)
}

// helpers
func parseID(c *gin.Context) int32 { return parseIDGiven(c.Param("id")) }
func parseIDGiven(s string) int32 {
	var i int32
	for _, ch := range s { if ch >= '0' && ch <= '9' { i = i*10 + int32(ch-'0') } }
	return i
}
func parseInt(s string) int32 { return parseIDGiven(s) }
func parsePagination(c *gin.Context) (int32, int32) {
	limit := parseIDGiven(c.Query("limit"))
	if limit == 0 || limit > 100 { limit = 20 }
	return limit, parseIDGiven(c.Query("offset"))
}
func contains(s, substr string) bool { return strings.Contains(s, substr) }
