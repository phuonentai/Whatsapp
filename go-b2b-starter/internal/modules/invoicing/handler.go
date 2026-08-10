// Package invoicing exposes the invoicing webhook ingress and the
// organization-facing Siigo connection endpoints.
package invoicing

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/siigo"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type Handler struct {
	invoicingService   services.InvoicingService
	connectionSvc      services.ConnectionService
	numerationSvc      services.NumerationService
	importSvc          services.ImportService
	testInvoiceSvc     services.TestInvoiceService
	sandbox            bool
	webhookSecret      string
	logger             loggerDomain.Logger
}

func NewHandler(
	svc services.InvoicingService,
	connSvc services.ConnectionService,
	numerationSvc services.NumerationService,
	importSvc services.ImportService,
	testInvoiceSvc services.TestInvoiceService,
	cfg *siigo.Config,
	log loggerDomain.Logger,
) *Handler {
	return &Handler{
		invoicingService: svc,
		connectionSvc:    connSvc,
		numerationSvc:    numerationSvc,
		importSvc:        importSvc,
		testInvoiceSvc:   testInvoiceSvc,
		sandbox:          cfg.Sandbox,
		webhookSecret:    cfg.WebhookSecret,
		logger:           log,
	}
}

// GetNumeration godoc
// @Summary Read the live provider numeration for the organization
// @Description Returns the DIAN numeration snapshot as the provider reports
// @Description it (auto mode: Siigo assigns consecutive numbers).
// @Tags siigo
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/numeration [get]
func (h *Handler) GetNumeration(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	info, err := h.numerationSvc.GetLive(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] failed to read numeration", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to read numeration", nil)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"mode": info.Mode, "resolution_id": info.ResolutionID,
		"prefijo": info.Prefix, "next_number": info.NextNumber,
	})
}

// ConfirmNumeration godoc
// @Summary Confirm the organization's DIAN numeration
// @Description Stores the snapshot and advances the connection to
// @Description numeracion_ok. The legal checkpoint before any production
// @Description invoice.
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/confirm-numeration [post]
func (h *Handler) ConfirmNumeration(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	snapshot, err := h.numerationSvc.Confirm(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] failed to confirm numeration", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to confirm numeration", nil)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"mode": snapshot.Mode, "resolution_id": snapshot.ResolutionID,
		"prefijo": snapshot.Prefix, "next_number": snapshot.NextNumber,
		"confirmed_at": snapshot.ConfirmedAt,
	})
}

// PreviewImport godoc
// @Summary Preview a provider customer import
// @Description Pulls provider customers and reports counts without writing.
// @Tags siigo
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/import/preview [get]
func (h *Handler) PreviewImport(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	counts, err := h.importSvc.Preview(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] import preview failed", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to preview import", nil)
		return
	}
	response.Success(c, http.StatusOK, counts)
}

// ConfirmImport godoc
// @Summary Commit the provider customer import
// @Description Upserts companies + linked contacts keyed by NIT and records
// @Description the run.
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/import/confirm [post]
func (h *Handler) ConfirmImport(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	counts, err := h.importSvc.Confirm(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] import confirm failed", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to confirm import", nil)
		return
	}
	response.Success(c, http.StatusOK, counts)
}

// SyncCustomers godoc
// @Summary On-demand delta sync of provider customers
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/sync [post]
func (h *Handler) SyncCustomers(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	counts, err := h.importSvc.DeltaSync(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] delta sync failed", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to sync customers", nil)
		return
	}
	response.Success(c, http.StatusOK, counts)
}

// TestInvoice godoc
// @Summary Create a sandbox test invoice (go-live proof)
// @Description Rejected with HTTP 400 when the provider is not sandboxed.
// @Description Advances the connection to sandbox_ok on a valid status.
// @Tags siigo
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Router /api/v1/org/siigo/test-invoice [post]
func (h *Handler) TestInvoice(c *gin.Context) {
	if !h.sandbox {
		response.Error(c, http.StatusBadRequest, "El proveedor no está en modo sandbox", nil)
		return
	}
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	inv, err := h.testInvoiceSvc.CreateTestInvoice(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] test invoice failed", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to create test invoice", nil)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"invoice_id": inv.ExternalID, "status": inv.Status, "cufe": inv.Cufe,
	})
}

// GetConnectionStatus godoc
// @Summary Get the organization's Siigo connection status
// @Description Returns the onboarding state machine status. Never contains
// @Description credential material.
// @Tags siigo
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/status [get]
func (h *Handler) GetConnectionStatus(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	conn, err := h.connectionSvc.Status(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.logger.Error("[siigo] failed to read connection status", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "failed to read connection status", nil)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"organization_id":   conn.OrganizationID,
		"provider":          conn.Provider,
		"status":            conn.Status,
		"nit":               conn.Nit,
		"siigo_company_name": conn.SiigoCompanyName,
		"last_error":        conn.LastError,
		"paused_at":         conn.PausedAt,
	})
}

// ConnectSiigo godoc
// @Summary Connect the organization's Siigo account
// @Description Validates credentials + NIT, stores them encrypted, advances
// @Description the onboarding state to connected.
// @Tags siigo
// @Accept json
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Router /api/v1/org/siigo/connect [post]
func (h *Handler) ConnectSiigo(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	var req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Nit          string `json:"nit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	conn, err := h.connectionSvc.Connect(c.Request.Context(), ctx.OrganizationID, services.ConnectRequest{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret, Nit: req.Nit,
	})
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"status": conn.Status, "siigo_company_name": conn.SiigoCompanyName,
	})
}

// RequestAssistedSetup godoc
// @Summary Mark the organization for assisted Siigo setup
// @Description Moves the connection to awaiting_setup so an admin can
// @Description provision credentials on the client's behalf.
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/request-assisted [post]
func (h *Handler) RequestAssistedSetup(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	conn, err := h.connectionSvc.RequestAssisted(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{"status": conn.Status})
}

// PauseInvoicing godoc
// @Summary Pause the organization's invoicing (kill-switch)
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/pause [post]
func (h *Handler) PauseInvoicing(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	conn, err := h.connectionSvc.Pause(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{"status": conn.Status})
}

// ResumeInvoicing godoc
// @Summary Resume the organization's invoicing
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/resume [post]
func (h *Handler) ResumeInvoicing(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	conn, err := h.connectionSvc.Resume(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{"status": conn.Status})
}

// ActivateInvoicing godoc
// @Summary Activate invoicing after a successful sandbox test
// @Tags siigo
// @Success 200 {object} map[string]any
// @Router /api/v1/org/siigo/activate [post]
func (h *Handler) ActivateInvoicing(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	if ctx == nil {
		response.Error(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	conn, err := h.connectionSvc.Activate(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{"status": conn.Status})
}

// ProvisionSiigo godoc
// @Summary Admin: provision Siigo credentials for an awaiting organization
// @Description Admin-scoped assisted setup. Same validation path as the
// @Description self-serve connect; writes encrypted credentials.
// @Tags siigo
// @Accept json
// @Success 200 {object} map[string]any
// @Router /api/v1/admin/siigo/provision [post]
func (h *Handler) ProvisionSiigo(c *gin.Context) {
	var req struct {
		OrganizationID int32  `json:"organization_id"`
		ClientID       string `json:"client_id"`
		ClientSecret   string `json:"client_secret"`
		Nit            string `json:"nit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", nil)
		return
	}
	if req.OrganizationID == 0 {
		response.Error(c, http.StatusBadRequest, "organization_id is required", nil)
		return
	}

	conn, err := h.connectionSvc.Provision(c.Request.Context(), req.OrganizationID, services.ConnectRequest{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret, Nit: req.Nit,
	})
	if err != nil {
		h.respondConnectionError(c, err)
		return
	}
	h.logger.Info("audit.siigo_provision", map[string]any{
		"organization_id": req.OrganizationID,
		"status":          conn.Status,
	})
	response.Success(c, http.StatusOK, map[string]any{
		"status": conn.Status, "siigo_company_name": conn.SiigoCompanyName,
	})
}

func (h *Handler) respondConnectionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNitMismatch):
		response.Error(c, http.StatusBadRequest, "El NIT de la empresa en Siigo no coincide con el NIT de la organización", nil)
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "Credenciales de Siigo inválidas", nil)
	case errors.Is(err, domain.ErrInvalidTransition):
		response.Error(c, http.StatusConflict, "Transición de estado no permitida para esta conexión", nil)
	default:
		h.logger.Error("[siigo] connection operation failed", map[string]any{"error": err.Error()})
		response.Error(c, http.StatusInternalServerError, "Operación de conexión fallida", nil)
	}
}

// ProcessSiigoWebhook godoc
// @Summary Process a Siigo invoice-status webhook
// @Description Verifies the HMAC signature and dispatches the event to the invoicing service. No session required.
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "received"
// @Failure 401 {object} map[string]any "invalid signature"
// @Router /api/v1/webhooks/siigo [post]
func (h *Handler) ProcessSiigoWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to read request body"})
		return
	}

	if h.webhookSecret == "" {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "webhook secret not configured"})
		return
	}

	if err := siigo.VerifyWebhookSignature(rawBody, c.GetHeader(siigo.WebhookSignatureHeader), h.webhookSecret); err != nil {
		h.logger.Warn("[ProcessSiigoWebhook] Signature verification failed", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "invalid signature"})
		return
	}

	if err := h.invoicingService.ProcessWebhookEvent(c.Request.Context(), rawBody); err != nil {
		h.logger.Error("[ProcessSiigoWebhook] Failed to process event", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to process webhook"})
		return
	}

	h.logger.Info("[ProcessSiigoWebhook] Event processed")
	c.JSON(http.StatusOK, map[string]any{"received": true})
}
