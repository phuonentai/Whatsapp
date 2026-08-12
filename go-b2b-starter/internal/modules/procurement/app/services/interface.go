// Package services implements the procurement application layer: drafting,
// reply extraction, run lifecycle/fan-out, the aggregation board, and order
// placement. All state transitions go through transaction-isolated repository
// guards so outbox/webhook redelivery is idempotent.
package services

import (
	"context"

	procurementEvents "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

// CreateSupplierInput is the POST /api/procurement/suppliers payload.
type CreateSupplierInput struct {
	NIT            string
	PhoneNumber    string
	DisplayName    string
	DeliveryDays   *int32
	MinOrderAmount *float64
	Notes          *string
}

// UpdateSupplierInput carries editable supplier fields (is_active always set).
type UpdateSupplierInput struct {
	ID             int32
	DeliveryDays   *int32
	MinOrderAmount *float64
	Notes          *string
	IsActive       bool
}

// CreateProductInput is the POST /api/procurement/products payload.
type CreateProductInput struct {
	Name string
	SKU  string
	Unit string
}

// UpdateProductInput carries editable product fields.
type UpdateProductInput struct {
	ID       int32
	Name     string
	SKU      string
	Unit     string
	IsActive bool
}

// RunProduct is one product × quantity line of a run request (name × quantity
// goes into the drafting prompt).
type RunProduct struct {
	ProductID int32
	Quantity  float64
}

// CreateRunInput is the POST /api/procurement/runs payload.
type CreateRunInput struct {
	SupplierIDs []int32
	Products    []RunProduct
	Nota        *string
}

// PlaceOrderInput is the POST /api/procurement/runs/:id/orders payload.
type PlaceOrderInput struct {
	RunID      int32
	SupplierID int32
	Items      []domain.OrderItem
	Notes      *string
	Override   bool
}

// ProcurementService is the handler-facing façade for the module.
type ProcurementService interface {
	ListSuppliers(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Supplier, error)
	CreateSupplier(ctx context.Context, orgID int32, in CreateSupplierInput, memberID string) (*domain.Supplier, error)
	UpdateSupplier(ctx context.Context, orgID int32, in UpdateSupplierInput) (*domain.Supplier, error)

	ListProducts(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Product, error)
	CreateProduct(ctx context.Context, orgID int32, in CreateProductInput) (*domain.Product, error)
	UpdateProduct(ctx context.Context, orgID int32, in UpdateProductInput) (*domain.Product, error)

	ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error)
	CreateRun(ctx context.Context, orgID int32, memberID string, in CreateRunInput) (*domain.InquiryRun, error)
	SendRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error)
	GetBoard(ctx context.Context, orgID, runID int32) (*domain.Board, error)

	PlaceOrder(ctx context.Context, orgID int32, memberID string, in PlaceOrderInput) (*domain.Order, error)
	ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error)
}

// DraftingService drafts one personalized Spanish message per supplier
// through the metered LLM client (D3/D11). displayName is the supplier's
// business display name (NIT persona jurídica) used for the greeting.
type DraftingService interface {
	DraftForSupplier(ctx context.Context, orgID int32, supplier *domain.Supplier, displayName string, products []*domain.Product, quantities map[int32]float64) (string, error)
}

// ExtractionService runs exactly one metered extraction call per eligible
// reply (D4) with the structured quote contract.
type ExtractionService interface {
	ExtractReply(ctx context.Context, orgID int32, content string) (*domain.ExtractionResult, error)
}

// SendHandler processes procurement durable outbox events with dispatch-time
// state re-validation (D14).
type SendHandler interface {
	HandleInquirySend(ctx context.Context, e *procurementEvents.InquirySend) error
	HandleOrderConfirmSend(ctx context.Context, e *procurementEvents.OrderConfirmSend) error
}

// KillSwitchReader is the tenant-scoped kill-switch seam (agent_settings).
type KillSwitchReader interface {
	GetAgentKillSwitch(ctx context.Context, organizationID int32) (bool, error)
}

// ActiveRecipientChecker is the tenant-scoped "is this sender an active run
// recipient?" seam consumed by the agent skip check (task 10).
type ActiveRecipientChecker interface {
	IsActiveRecipientByPhone(ctx context.Context, orgID int32, phoneNumber string) (bool, error)
}
