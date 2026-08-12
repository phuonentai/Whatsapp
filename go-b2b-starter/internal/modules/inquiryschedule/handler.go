// Package inquiryschedule exposes the inquiry-scheduling HTTP API under
// /api/procurement/schedules... behind auth + org_context + subscription,
// with org:manage on writes and org:view on reads (Spanish-first errors).
package inquiryschedule

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

// Handler is the inquiry-scheduling HTTP handler.
type Handler struct {
	service services.InquiryscheduleService
}

// NewHandler builds the handler.
func NewHandler(service services.InquiryscheduleService) *Handler {
	return &Handler{service: service}
}

// ---------- Schedules ----------

func (h *Handler) HandleListSchedules(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	schedules, err := h.service.ListSchedules(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar programaciones", err)
		return
	}
	response.Success(c, http.StatusOK, schedules)
}

type scheduleRequest struct {
	Name        string   `json:"name"`
	RunTime     string   `json:"run_time"`
	DaysOfWeek  []int    `json:"days_of_week"`
	ProductIDs  []int32  `json:"product_ids"`
	SupplierIDs []int32  `json:"supplier_ids"`
	Note        *string  `json:"note"`
}

func (h *Handler) HandleCreateSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	created, err := h.service.CreateSchedule(c.Request.Context(), ctx.OrganizationID, ctx.Identity.UserID, services.CreateScheduleInput{
		Name:        req.Name,
		RunTime:     req.RunTime,
		DaysOfWeek:  intsToDays(req.DaysOfWeek),
		ProductIDs:  req.ProductIDs,
		SupplierIDs: req.SupplierIDs,
		Note:        derefStr(req.Note),
	})
	if err != nil {
		switch {
		case isValidation(err):
			response.Error(c, http.StatusBadRequest, err.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al crear la programación", err)
		}
		return
	}
	response.Success(c, http.StatusCreated, created)
}

func (h *Handler) HandleUpdateSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	updated, err := h.service.UpdateSchedule(c.Request.Context(), ctx.OrganizationID, ctx.Identity.UserID, services.UpdateScheduleInput{
		ID:          id,
		Name:        req.Name,
		RunTime:     req.RunTime,
		DaysOfWeek:  intsToDays(req.DaysOfWeek),
		ProductIDs:  req.ProductIDs,
		SupplierIDs: req.SupplierIDs,
		Note:        derefStr(req.Note),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			response.Error(c, http.StatusNotFound, "Programación no encontrada", err)
		case isValidation(err):
			response.Error(c, http.StatusBadRequest, err.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al actualizar la programación", err)
		}
		return
	}
	response.Success(c, http.StatusOK, updated)
}

func (h *Handler) HandleGetSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	detail, err := h.service.GetScheduleDetail(c.Request.Context(), ctx.OrganizationID, id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			response.Error(c, http.StatusNotFound, "Programación no encontrada", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al consultar la programación", err)
		}
		return
	}
	response.Success(c, http.StatusOK, detail)
}

func (h *Handler) HandlePauseSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	paused, err := h.service.PauseSchedule(c.Request.Context(), ctx.OrganizationID, id, ctx.Identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			response.Error(c, http.StatusNotFound, "Programación no encontrada", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al pausar la programación", err)
		}
		return
	}
	response.Success(c, http.StatusOK, paused)
}

func (h *Handler) HandleResumeSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	resumed, err := h.service.ResumeSchedule(c.Request.Context(), ctx.OrganizationID, id, ctx.Identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			response.Error(c, http.StatusNotFound, "Programación no encontrada", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al reanudar la programación", err)
		}
		return
	}
	response.Success(c, http.StatusOK, resumed)
}

func (h *Handler) HandleDeleteSchedule(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	if err := h.service.DeleteSchedule(c.Request.Context(), ctx.OrganizationID, id, ctx.Identity.UserID); err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			response.Error(c, http.StatusNotFound, "Programación no encontrada", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al eliminar la programación", err)
		}
		return
	}
	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

// ---------- Follow-up settings ----------

func (h *Handler) HandleGetFollowUpSettings(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	settings, err := h.service.GetFollowUpSettings(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al consultar la configuración de recordatorios", err)
		return
	}
	response.Success(c, http.StatusOK, settings)
}

type followUpSettingsRequest struct {
	Enabled         bool    `json:"enabled"`
	DeadlineHours   int     `json:"deadline_hours"`
	MaxNudges       int     `json:"max_nudges"`
	MessageTemplate *string `json:"message_template"`
}

func (h *Handler) HandleUpdateFollowUpSettings(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req followUpSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	settings, err := h.service.UpdateFollowUpSettings(c.Request.Context(), ctx.OrganizationID, services.UpdateFollowUpSettingsInput{
		Enabled:         req.Enabled,
		DeadlineHours:   req.DeadlineHours,
		MaxNudges:       req.MaxNudges,
		MessageTemplate: derefStr(req.MessageTemplate),
	})
	if err != nil {
		switch {
		case isValidation(err):
			response.Error(c, http.StatusBadRequest, err.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al actualizar la configuración de recordatorios", err)
		}
		return
	}
	response.Success(c, http.StatusOK, settings)
}

// ---------- helpers ----------

func parseID(c *gin.Context) int32 {
	var i int32
	for _, ch := range c.Param("id") {
		if ch >= '0' && ch <= '9' {
			i = i*10 + int32(ch-'0')
		}
	}
	return i
}

func intsToDays(days []int) domain.DaysOfWeek {
	out := make(domain.DaysOfWeek, 0, len(days))
	for _, d := range days {
		out = append(out, domain.DayOfWeek(d))
	}
	return out
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isValidation(err error) bool {
	var ve *domain.ValidationError
	return errors.As(err, &ve)
}
