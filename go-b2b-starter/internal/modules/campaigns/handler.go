package campaigns

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type Handler struct {
	segmentService services.SegmentService
	campaignService services.CampaignService
	aiBuilder      services.AudienceBuilderService
}

func NewHandler(
	segmentService services.SegmentService,
	campaignService services.CampaignService,
	aiBuilder services.AudienceBuilderService,
) *Handler {
	return &Handler{segmentService: segmentService, campaignService: campaignService, aiBuilder: aiBuilder}
}

// ---- Segments ----

func (h *Handler) ListSegments(c *gin.Context) {
	orgID := auth.GetRequestContext(c).OrganizationID
	segments, err := h.segmentService.List(c.Request.Context(), orgID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar segmentos", err)
		return
	}
	response.Success(c, http.StatusOK, segments)
}

func (h *Handler) CreateSegment(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	var body struct {
		Nombre     string          `json:"nombre" binding:"required"`
		FilterSpec []domain.Filter `json:"filter_spec" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	segment, err := h.segmentService.Create(c.Request.Context(), reqCtx.OrganizationID, body.Nombre, body.FilterSpec, memberID(reqCtx))
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, segment)
}

func (h *Handler) UpdateSegment(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	id, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de segmento inválido", nil)
		return
	}
	var body struct {
		Nombre     string          `json:"nombre" binding:"required"`
		FilterSpec []domain.Filter `json:"filter_spec" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	segment, err := h.segmentService.Update(c.Request.Context(), reqCtx.OrganizationID, id, body.Nombre, body.FilterSpec)
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, segment)
}

func (h *Handler) DeleteSegment(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	id, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de segmento inválido", nil)
		return
	}
	if err := h.segmentService.Delete(c.Request.Context(), reqCtx.OrganizationID, id); err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al eliminar segmento", err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) PreviewSegment(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	id, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de segmento inválido", nil)
		return
	}
	segment, err := h.segmentService.Get(c.Request.Context(), reqCtx.OrganizationID, id)
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	preview, err := h.segmentService.Preview(c.Request.Context(), reqCtx.OrganizationID, segment.FilterSpec)
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, preview)
}

func (h *Handler) GetSegment(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	id, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de segmento inválido", nil)
		return
	}
	segment, err := h.segmentService.Get(c.Request.Context(), reqCtx.OrganizationID, id)
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, segment)
}

func (h *Handler) PreviewSpec(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	var body struct {
		FilterSpec []domain.Filter `json:"filter_spec" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	preview, err := h.segmentService.Preview(c.Request.Context(), reqCtx.OrganizationID, body.FilterSpec)
	if err != nil {
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, preview)
}

// AiBuild converts natural language into a validated candidate filter spec
// with preview. Persists nothing.
func (h *Handler) AiBuild(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	var body struct {
		Descripcion string `json:"descripcion" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	result, err := h.aiBuilder.Build(c.Request.Context(), reqCtx.OrganizationID, body.Descripcion)
	if err != nil {
		if errors.Is(err, domain.ErrAiCreditsExhausted) {
			response.Error(c, http.StatusPaymentRequired, domain.ErrAiCreditsExhausted.Error(), err)
			return
		}
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result)
}

// ---- Campaigns ----

func (h *Handler) ListCampaigns(c *gin.Context) {
	orgID := auth.GetRequestContext(c).OrganizationID
	campaigns, err := h.campaignService.List(c.Request.Context(), orgID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar campañas", err)
		return
	}
	response.Success(c, http.StatusOK, campaigns)
}

func (h *Handler) CreateCampaign(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	var body struct {
		Nombre    string `json:"nombre" binding:"required"`
		SegmentID int32  `json:"segment_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	campaign, err := h.campaignService.Create(c.Request.Context(), reqCtx.OrganizationID, body.Nombre, body.SegmentID, memberID(reqCtx))
	if err != nil {
		if errors.Is(err, domain.ErrSegmentNotFound) {
			response.Error(c, http.StatusBadRequest, domain.ErrSegmentNotFound.Error(), err)
			return
		}
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, campaign)
}

func (h *Handler) LaunchCampaign(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	id, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de campaña inválido", nil)
		return
	}
	campaign, err := h.campaignService.Launch(c.Request.Context(), reqCtx.OrganizationID, id, memberID(reqCtx))
	if err != nil {
		if errors.Is(err, domain.ErrCampaignNotDraft) {
			response.Error(c, http.StatusConflict, domain.ErrCampaignNotDraft.Error(), err)
			return
		}
		if errors.Is(err, domain.ErrCampaignNotFound) {
			response.Error(c, http.StatusNotFound, domain.ErrCampaignNotFound.Error(), err)
			return
		}
		writeSegmentError(c, err)
		return
	}
	response.Success(c, http.StatusOK, campaign)
}

func (h *Handler) ListRecipients(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	campaignID, ok := parseID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "ID de campaña inválido", nil)
		return
	}
	limit, offset := parsePagination(c)
	recipients, err := h.campaignService.ListRecipients(c.Request.Context(), reqCtx.OrganizationID, campaignID, limit, offset)
	if err != nil {
		if errors.Is(err, domain.ErrCampaignNotFound) {
			response.Error(c, http.StatusNotFound, domain.ErrCampaignNotFound.Error(), err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al listar destinatarios", err)
		return
	}
	response.Success(c, http.StatusOK, recipients)
}

// ---- helpers ----

func writeSegmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidFilterSpec):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, domain.ErrSegmentNotFound):
		response.Error(c, http.StatusNotFound, domain.ErrSegmentNotFound.Error(), err)
	default:
		response.Error(c, http.StatusInternalServerError, "Error al procesar la solicitud", err)
	}
}

func memberID(reqCtx *auth.RequestContext) string {
	if reqCtx.Identity == nil {
		return ""
	}
	return reqCtx.Identity.UserID
}

func parseID(c *gin.Context) (int32, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil || id <= 0 {
		return 0, false
	}
	return int32(id), true
}

func parsePagination(c *gin.Context) (int32, int32) {
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)
	if offset < 0 {
		offset = 0
	}
	return int32(limit), int32(offset)
}
