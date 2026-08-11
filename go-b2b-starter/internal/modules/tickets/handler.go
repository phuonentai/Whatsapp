package tickets

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type Handler struct {
	ticketService ticketsServices.TicketService
}

func NewHandler(ticketService ticketsServices.TicketService) *Handler {
	return &Handler{ticketService: ticketService}
}

func (h *Handler) List(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	limit, offset := parsePagination(c)
	status := c.Query("status")
	assignee := c.Query("assignee")
	tickets, err := h.ticketService.List(c.Request.Context(), ctx.OrganizationID, status, assignee, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar tickets", err)
		return
	}
	response.Success(c, http.StatusOK, mapTickets(tickets))
}

func (h *Handler) Get(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	ticket, err := h.ticketService.Get(c.Request.Context(), ctx.OrganizationID, parseID(c))
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al obtener ticket", err)
		return
	}
	events, err := h.ticketService.ListEvents(c.Request.Context(), ticket.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al obtener historial", err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ticket": mapTicket(ticket), "eventos": mapEvents(events)})
}

func (h *Handler) Create(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req ticketsServices.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	ticket, err := h.ticketService.Create(c.Request.Context(), ctx.OrganizationID, &req, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPriority) {
			response.Error(c, http.StatusBadRequest, domain.ErrInvalidPriority.Error(), err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al crear ticket", err)
		return
	}
	response.Success(c, http.StatusCreated, mapTicket(ticket))
}

func (h *Handler) Transition(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	ticket, err := h.ticketService.Transition(c.Request.Context(), ctx.OrganizationID, parseID(c), domain.TicketStatus(req.Status), actorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTicketNotFound):
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
		case errors.Is(err, domain.ErrInvalidTransition):
			response.Error(c, http.StatusBadRequest, domain.ErrInvalidTransition.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al cambiar estado", err)
		}
		return
	}
	response.Success(c, http.StatusOK, mapTicket(ticket))
}

func (h *Handler) Assign(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req struct {
		Assignee string `json:"assignee"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	ticket, err := h.ticketService.Assign(c.Request.Context(), ctx.OrganizationID, parseID(c), req.Assignee, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al asignar ticket", err)
		return
	}
	response.Success(c, http.StatusOK, mapTicket(ticket))
}

func (h *Handler) SetPriority(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req struct {
		Priority string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	ticket, err := h.ticketService.SetPriority(c.Request.Context(), ctx.OrganizationID, parseID(c), domain.TicketPriority(req.Priority), actorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTicketNotFound):
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
		case errors.Is(err, domain.ErrInvalidPriority):
			response.Error(c, http.StatusBadRequest, domain.ErrInvalidPriority.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al cambiar prioridad", err)
		}
		return
	}
	response.Success(c, http.StatusOK, mapTicket(ticket))
}

func (h *Handler) SetTags(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req struct {
		Tags []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	ticket, err := h.ticketService.SetTags(c.Request.Context(), ctx.OrganizationID, parseID(c), req.Tags, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al actualizar etiquetas", err)
		return
	}
	response.Success(c, http.StatusOK, mapTicket(ticket))
}

func (h *Handler) AddInternalNote(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	actor := authcontext.GetIdentity(c)
	actorID := ""
	if actor != nil {
		actorID = actor.UserID
	}
	event, err := h.ticketService.AddInternalNote(c.Request.Context(), ctx.OrganizationID, parseID(c), req.Body, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			response.Error(c, http.StatusNotFound, "Ticket no encontrado", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al agregar nota", err)
		return
	}
	response.Success(c, http.StatusCreated, mapEvent(event))
}

func (h *Handler) ListEvents(c *gin.Context) {
	events, err := h.ticketService.ListEvents(c.Request.Context(), parseID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al obtener historial", err)
		return
	}
	response.Success(c, http.StatusOK, mapEvents(events))
}

func parseID(c *gin.Context) int32 {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 32)
	return int32(id)
}

func parsePagination(c *gin.Context) (int32, int32) {
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32)
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)
	return int32(limit), int32(offset)
}

func mapTicket(t *domain.Ticket) map[string]any {
	return map[string]any{
		"id":             t.ID,
		"organization_id": t.OrganizationID,
		"contact_id":     t.ContactID,
		"conversation_id": t.ConversationID,
		"title":          t.Title,
		"description":    t.Description,
		"status":         t.Status,
		"priority":       t.Priority,
		"tags":           t.Tags,
		"assignee":       t.AssigneeStytchMember,
		"sla_due_at":     t.SLADueAt,
		"overdue":        t.IsOverdue(time.Now()),
		"created_at":     t.CreatedAt,
		"updated_at":     t.UpdatedAt,
	}
}

func mapTickets(tickets []*domain.Ticket) []map[string]any {
	result := make([]map[string]any, len(tickets))
	for i, t := range tickets {
		result[i] = mapTicket(t)
	}
	return result
}

func mapEvent(e *domain.TicketEvent) map[string]any {
	return map[string]any{
		"id":         e.ID,
		"event_type": e.EventType,
		"actor":      e.ActorStytchMember,
		"payload":    e.Payload,
		"created_at": e.CreatedAt,
	}
}

func mapEvents(events []*domain.TicketEvent) []map[string]any {
	result := make([]map[string]any, len(events))
	for i, e := range events {
		result[i] = mapEvent(e)
	}
	return result
}
