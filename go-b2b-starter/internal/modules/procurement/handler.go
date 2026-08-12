// Package procurement exposes the procurement HTTP API under
// /api/procurement/... behind auth + org_context + subscription, with
// org:manage on writes/approvals and org:view on reads (Spanish-first errors).
package procurement

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

// Handler is the procurement HTTP handler.
type Handler struct {
	service services.ProcurementService
}

// NewHandler builds the procurement handler.
func NewHandler(service services.ProcurementService) *Handler {
	return &Handler{service: service}
}

// ---------- Suppliers ----------

func (h *Handler) HandleListSuppliers(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	suppliers, err := h.service.ListSuppliers(c.Request.Context(), ctx.OrganizationID, 100, 0)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar proveedores", err)
		return
	}
	response.Success(c, http.StatusOK, suppliers)
}

type createSupplierRequest struct {
	NIT            string   `json:"nit"`
	PhoneNumber    string   `json:"phone"`
	DisplayName    string   `json:"display_name"`
	DeliveryDays   *int32   `json:"delivery_days"`
	MinOrderAmount *float64 `json:"min_order_amount"`
	Notes          *string  `json:"notes"`
}

func (h *Handler) HandleCreateSupplier(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req createSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	if req.NIT == "" {
		response.Error(c, http.StatusBadRequest, "El NIT es obligatorio", nil)
		return
	}
	if req.PhoneNumber == "" {
		response.Error(c, http.StatusBadRequest, "El teléfono del proveedor es obligatorio", nil)
		return
	}
	supplier, err := h.service.CreateSupplier(c.Request.Context(), ctx.OrganizationID, services.CreateSupplierInput{
		NIT:            req.NIT,
		PhoneNumber:    req.PhoneNumber,
		DisplayName:    req.DisplayName,
		DeliveryDays:   req.DeliveryDays,
		MinOrderAmount: req.MinOrderAmount,
		Notes:          req.Notes,
	}, ctx.Identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSupplierAlreadyExists):
			response.Error(c, http.StatusBadRequest, "Ya existe un proveedor con este NIT", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al crear proveedor", err)
		}
		return
	}
	response.Success(c, http.StatusCreated, supplier)
}

type updateSupplierRequest struct {
	DeliveryDays   *int32   `json:"delivery_days"`
	MinOrderAmount *float64 `json:"min_order_amount"`
	Notes          *string  `json:"notes"`
	IsActive       *bool    `json:"is_active"`
}

func (h *Handler) HandleUpdateSupplier(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	var req updateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	supplier, err := h.service.UpdateSupplier(c.Request.Context(), ctx.OrganizationID, services.UpdateSupplierInput{
		ID:             id,
		DeliveryDays:   req.DeliveryDays,
		MinOrderAmount: req.MinOrderAmount,
		Notes:          req.Notes,
		IsActive:       isActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSupplierNotFound):
			response.Error(c, http.StatusNotFound, "Proveedor no encontrado", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al actualizar proveedor", err)
		}
		return
	}
	response.Success(c, http.StatusOK, supplier)
}

// ---------- Products ----------

func (h *Handler) HandleListProducts(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	products, err := h.service.ListProducts(c.Request.Context(), ctx.OrganizationID, 100, 0)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar productos", err)
		return
	}
	response.Success(c, http.StatusOK, products)
}

type createProductRequest struct {
	Name string `json:"name"`
	SKU  string `json:"sku"`
	Unit string `json:"unit"`
}

func (h *Handler) HandleCreateProduct(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	if req.Name == "" || req.SKU == "" {
		response.Error(c, http.StatusBadRequest, "El nombre y el SKU son obligatorios", nil)
		return
	}
	product, err := h.service.CreateProduct(c.Request.Context(), ctx.OrganizationID, services.CreateProductInput{
		Name: req.Name,
		SKU:  req.SKU,
		Unit: req.Unit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al crear producto", err)
		return
	}
	response.Success(c, http.StatusCreated, product)
}

type updateProductRequest struct {
	Name     string `json:"name"`
	SKU      string `json:"sku"`
	Unit     string `json:"unit"`
	IsActive *bool  `json:"is_active"`
}

func (h *Handler) HandleUpdateProduct(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	id := parseID(c)
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	product, err := h.service.UpdateProduct(c.Request.Context(), ctx.OrganizationID, services.UpdateProductInput{
		ID:       id,
		Name:     req.Name,
		SKU:      req.SKU,
		Unit:     req.Unit,
		IsActive: isActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProductNotFound):
			response.Error(c, http.StatusNotFound, "Producto no encontrado", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al actualizar producto", err)
		}
		return
	}
	response.Success(c, http.StatusOK, product)
}

// ---------- Runs ----------

func (h *Handler) HandleListRuns(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	runs, err := h.service.ListRuns(c.Request.Context(), ctx.OrganizationID, 50, 0)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar ejecuciones", err)
		return
	}
	response.Success(c, http.StatusOK, runs)
}

type runProductRequest struct {
	ProductID int32   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
}

type createRunRequest struct {
	SupplierIDs []int32             `json:"supplier_ids"`
	Products    []runProductRequest `json:"products"`
	Nota        *string             `json:"nota"`
}

func (h *Handler) HandleCreateRun(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	var req createRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	if len(req.SupplierIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "Selecciona al menos un proveedor", nil)
		return
	}
	if len(req.Products) == 0 {
		response.Error(c, http.StatusBadRequest, "Selecciona al menos un producto", nil)
		return
	}
	products := make([]services.RunProduct, 0, len(req.Products))
	for _, p := range req.Products {
		products = append(products, services.RunProduct{ProductID: p.ProductID, Quantity: p.Quantity})
	}
	run, err := h.service.CreateRun(c.Request.Context(), ctx.OrganizationID, ctx.Identity.UserID, services.CreateRunInput{
		SupplierIDs: req.SupplierIDs,
		Products:    products,
		Nota:        req.Nota,
	})
	if err != nil {
		// Drafting failures escalate the run (with audit); the escalated run
		// is returned so the wizard can surface the status.
		if run != nil {
			response.Success(c, http.StatusOK, run)
			return
		}
		switch {
		case errors.Is(err, domain.ErrNoSuppliers):
			response.Error(c, http.StatusBadRequest, "Selecciona al menos un proveedor", err)
		case errors.Is(err, domain.ErrNoProducts):
			response.Error(c, http.StatusBadRequest, "Selecciona al menos un producto", err)
		case errors.Is(err, domain.ErrSupplierNotFound):
			response.Error(c, http.StatusBadRequest, "Uno de los proveedores no existe", err)
		case errors.Is(err, domain.ErrProductNotFound):
			response.Error(c, http.StatusBadRequest, "Uno de los productos no existe", err)
		case errors.Is(err, domain.ErrSupplierInactive):
			response.Error(c, http.StatusBadRequest, "Uno de los proveedores está inactivo", err)
		case errors.Is(err, domain.ErrCreditsExhausted):
			response.Error(c, http.StatusPaymentRequired, "Has agotado tus créditos de IA para este periodo", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al crear la ejecución", err)
		}
		return
	}
	response.Success(c, http.StatusCreated, run)
}

func (h *Handler) HandleSendRun(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	runID := parseID(c)
	run, err := h.service.SendRun(c.Request.Context(), ctx.OrganizationID, runID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRunNotFound):
			response.Error(c, http.StatusNotFound, "Ejecución no encontrada", err)
		case errors.Is(err, domain.ErrRunNotDraft):
			response.Error(c, http.StatusBadRequest, "La ejecución ya fue enviada", err)
		case errors.Is(err, domain.ErrNoDraftedMessages):
			response.Error(c, http.StatusBadRequest, "La ejecución no tiene mensajes redactados", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al enviar la ejecución", err)
		}
		return
	}
	response.Success(c, http.StatusOK, run)
}

func (h *Handler) HandleGetRunBoard(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	runID := parseID(c)
	board, err := h.service.GetBoard(c.Request.Context(), ctx.OrganizationID, runID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRunNotFound):
			response.Error(c, http.StatusNotFound, "Ejecución no encontrada", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al consultar la cotización", err)
		}
		return
	}
	response.Success(c, http.StatusOK, board)
}

// ---------- Orders ----------

type orderItemRequest struct {
	ProductID int32   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
}

type placeOrderRequest struct {
	SupplierID int32               `json:"supplier_id"`
	Items      []orderItemRequest  `json:"items"`
	Notes      *string             `json:"notes"`
	Override   bool                `json:"override"`
}

func (h *Handler) HandlePlaceOrder(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	runID := parseID(c)
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	if req.SupplierID == 0 {
		response.Error(c, http.StatusBadRequest, "Selecciona un proveedor", nil)
		return
	}
	if len(req.Items) == 0 {
		response.Error(c, http.StatusBadRequest, "El pedido debe incluir al menos un producto", nil)
		return
	}
	items := make([]domain.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, domain.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	order, err := h.service.PlaceOrder(c.Request.Context(), ctx.OrganizationID, ctx.Identity.UserID, services.PlaceOrderInput{
		RunID:      runID,
		SupplierID: req.SupplierID,
		Items:      items,
		Notes:      req.Notes,
		Override:   req.Override,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRunNotFound):
			response.Error(c, http.StatusNotFound, "Ejecución no encontrada", err)
		case errors.Is(err, domain.ErrInvalidRunStatus):
			response.Error(c, http.StatusBadRequest, "La ejecución no está en un estado válido para pedidos", err)
		case errors.Is(err, domain.ErrResponseNotAnswered):
			response.Error(c, http.StatusBadRequest, "El proveedor no tiene una respuesta confirmada", err)
		case errors.Is(err, domain.ErrResponseRequiresHuman):
			response.Error(c, http.StatusBadRequest, "La respuesta del proveedor requiere revisión humana (usa la anulación explícita si aplica)", err)
		case errors.Is(err, domain.ErrKillSwitchEnabled), errors.Is(err, domain.ErrConsentWithdrawn):
			response.Error(c, http.StatusBadRequest, "El envío de confirmación está bloqueado; el pedido quedó registrado", err)
		case errors.Is(err, domain.ErrDefaultPipelineMissing):
			response.Error(c, http.StatusBadRequest, "La organización no tiene un pipeline por defecto", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al registrar el pedido", err)
		}
		return
	}
	response.Success(c, http.StatusCreated, order)
}

func (h *Handler) HandleListRunOrders(c *gin.Context) {
	ctx := authcontext.GetRequestContext(c)
	runID := parseID(c)
	orders, err := h.service.ListRunOrders(c.Request.Context(), ctx.OrganizationID, runID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar pedidos", err)
		return
	}
	response.Success(c, http.StatusOK, orders)
}

func parseID(c *gin.Context) int32 {
	var i int32
	for _, ch := range c.Param("id") {
		if ch >= '0' && ch <= '9' {
			i = i*10 + int32(ch-'0')
		}
	}
	return i
}
