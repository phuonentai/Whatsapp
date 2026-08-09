package agent

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

// Handler exposes the agent HTTP API.
type Handler struct {
	agent     services.AgentService
	compliance services.ComplianceService
}

// NewHandler builds the agent HTTP handler.
func NewHandler(agent services.AgentService, compliance services.ComplianceService) *Handler {
	return &Handler{agent: agent, compliance: compliance}
}

// HandleListSuggestions returns the org's pending suggestions.
func (h *Handler) HandleListSuggestions(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	suggestions, err := h.agent.ListPendingSuggestions(c.Request.Context(), orgID, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "agent_list_failed", "No se pudieron cargar las sugerencias.",
		))
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// HandleApproveSuggestion sends an approved draft.
func (h *Handler) HandleApproveSuggestion(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID
	memberID := auth.MustGetIdentity(c).UserID

	suggestionID, err := pathID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_id", "Identificador inválido."))
		return
	}

	var body struct {
		EditedBody string `json:"edited_body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_body", "Cuerpo de solicitud inválido."))
		return
	}

	suggestion, err := h.agent.ApproveSuggestion(c.Request.Context(), orgID, suggestionID, body.EditedBody, memberID)
	if err != nil {
		var denial *services.DenialError
		if errors.As(err, &denial) {
			c.JSON(http.StatusConflict, httperr.NewHTTPError(
				http.StatusConflict, "agent_action_denied", "La acción fue rechazada por las políticas del agente.",
			))
			return
		}
		if errors.Is(err, domain.ErrSuggestionNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound, "suggestion_not_found", "La sugerencia no existe o ya fue resuelta.",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "agent_approve_failed", "No se pudo aprobar la sugerencia.",
		))
		return
	}
	c.JSON(http.StatusOK, suggestion)
}

// HandleRejectSuggestion marks a suggestion as rejected.
func (h *Handler) HandleRejectSuggestion(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	suggestionID, err := pathID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_id", "Identificador inválido."))
		return
	}

	suggestion, err := h.agent.RejectSuggestion(c.Request.Context(), orgID, suggestionID)
	if err != nil {
		if errors.Is(err, domain.ErrSuggestionNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound, "suggestion_not_found", "La sugerencia no existe o ya fue resuelta.",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "agent_reject_failed", "No se pudo rechazar la sugerencia.",
		))
		return
	}
	c.JSON(http.StatusOK, suggestion)
}

// HandleGetSettings returns the org's agent settings.
func (h *Handler) HandleGetSettings(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	settings, err := h.agent.GetSettings(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "agent_settings_failed", "No se pudieron cargar los ajustes del asistente.",
		))
		return
	}
	c.JSON(http.StatusOK, settings)
}

// HandleUpdateSettings persists the org's agent settings.
func (h *Handler) HandleUpdateSettings(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	var settings domain.AgentSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_body", "Cuerpo de solicitud inválido."))
		return
	}

	updated, err := h.agent.UpdateSettings(c.Request.Context(), orgID, &settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest, "invalid_settings", "Ajustes del asistente inválidos.",
		))
		return
	}
	c.JSON(http.StatusOK, updated)
}

// HandleGetFlowDebug returns the active flow for a conversation.
func (h *Handler) HandleGetFlowDebug(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	convID, err := pathID(c, "conversationId")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_id", "Identificador inválido."))
		return
	}

	debug, err := h.agent.GetFlowDebug(c.Request.Context(), orgID, convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "agent_flow_failed", "No se pudo cargar el estado del flujo.",
		))
		return
	}
	c.JSON(http.StatusOK, debug)
}

// HandleExportContact exports a contact's data (Ley 1581).
func (h *Handler) HandleExportContact(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	contactID, err := pathID(c, "contactId")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_id", "Identificador inválido."))
		return
	}

	bundle, err := h.compliance.ExportContact(c.Request.Context(), orgID, contactID)
	if err != nil {
		if errors.Is(err, domain.ErrContactNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(http.StatusNotFound, "contact_not_found", "Contacto no encontrado."))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "export_failed", "No se pudo exportar la información del contacto.",
		))
		return
	}
	c.JSON(http.StatusOK, bundle)
}

// HandleForgetContact anonymizes a contact (Ley 1581).
func (h *Handler) HandleForgetContact(c *gin.Context) {
	reqCtx := auth.MustGetRequestContext(c)
	orgID := reqCtx.OrganizationID

	contactID, err := pathID(c, "contactId")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(http.StatusBadRequest, "invalid_id", "Identificador inválido."))
		return
	}

	if err := h.compliance.ForgetContact(c.Request.Context(), orgID, contactID); err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError, "forget_failed", "No se pudo anonimizar el contacto.",
		))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "anonymized"})
}

func pathID(c *gin.Context, name string) (int32, error) {
	raw := c.Param(name)
	id, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}
