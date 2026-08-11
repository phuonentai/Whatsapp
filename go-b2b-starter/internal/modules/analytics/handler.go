package analytics

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/analytics/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type Handler struct {
	reportService *services.SalesReportService
}

func NewHandler(reportService *services.SalesReportService) *Handler {
	return &Handler{reportService: reportService}
}

func (h *Handler) Revenue(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	period := c.Query("period")
	if period == "" {
		period = "month"
	}
	from, err := parseOptionalDate(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Fecha inválida en el parámetro from.", domain.ErrInvalidRange)
		return
	}
	to, err := parseOptionalDate(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Fecha inválida en el parámetro to.", domain.ErrInvalidRange)
		return
	}
	points, err := h.reportService.RevenueByPeriod(c.Request.Context(), ctx.OrganizationID, period, from, to)
	if err != nil {
		writeServiceError(c, err, "Error al generar el reporte de ventas")
		return
	}
	response.Success(c, http.StatusOK, points)
}

func (h *Handler) TopCustomers(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	limit, err := parseInt32Param(c.Query("limit"), 0, 50)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Parámetro limit inválido.", domain.ErrInvalidLimit)
		return
	}
	customers, err := h.reportService.TopCustomers(c.Request.Context(), ctx.OrganizationID, limit)
	if err != nil {
		writeServiceError(c, err, "Error al generar el reporte de clientes")
		return
	}
	response.Success(c, http.StatusOK, customers)
}

func (h *Handler) Funnel(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	report, err := h.reportService.Funnel(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		writeServiceError(c, err, "Error al generar el reporte de embudo")
		return
	}
	response.Success(c, http.StatusOK, report)
}

func (h *Handler) InactiveContacts(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	days, err := parseInt32Param(c.Query("days"), 1, 365)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Parámetro days inválido.", domain.ErrInvalidDays)
		return
	}
	contacts, err := h.reportService.InactiveContacts(c.Request.Context(), ctx.OrganizationID, days)
	if err != nil {
		writeServiceError(c, err, "Error al generar el reporte de contactos inactivos")
		return
	}
	response.Success(c, http.StatusOK, contacts)
}

func writeServiceError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, domain.ErrInvalidRange):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, domain.ErrInvalidPeriod):
		response.Error(c, http.StatusBadRequest, "Parámetro period inválido.", err)
	case errors.Is(err, domain.ErrInvalidDays):
		response.Error(c, http.StatusBadRequest, "Parámetro days inválido.", err)
	default:
		response.Error(c, http.StatusInternalServerError, fallback, err)
	}
}

func parseOptionalDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseInt32Param(raw string, min, max int32) (int32, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	if value < int64(min) || value > int64(max) {
		return 0, errors.New("fuera de rango")
	}
	return int32(value), nil
}
