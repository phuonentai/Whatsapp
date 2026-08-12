package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

// procurementService is the handler-facing façade composed from the run,
// board, and order services.
type procurementService struct {
	suppliers domain.SupplierRepository
	products  domain.ProductRepository
	runs      *runService
	board     *boardService
	orders    *orderService
}

// NewProcurementService builds the composed façade.
func NewProcurementService(
	suppliers domain.SupplierRepository,
	products domain.ProductRepository,
	runs *runService,
	board *boardService,
	orders *orderService,
) ProcurementService {
	return &procurementService{suppliers: suppliers, products: products, runs: runs, board: board, orders: orders}
}

func (s *procurementService) ListSuppliers(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Supplier, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.suppliers.List(ctx, orgID, limit, offset)
}

func (s *procurementService) CreateSupplier(ctx context.Context, orgID int32, in CreateSupplierInput, memberID string) (*domain.Supplier, error) {
	if in.NIT == "" || in.PhoneNumber == "" {
		return nil, domain.ErrSupplierAlreadyExists // validated upstream; keep repo pure
	}
	return s.suppliers.Create(ctx, orgID, &domain.Supplier{
		OrganizationID: orgID,
		NIT:            in.NIT,
		DeliveryDays:   in.DeliveryDays,
		MinOrderAmount: in.MinOrderAmount,
		Notes:          in.Notes,
	}, domain.ContactInput{
		PhoneNumber: in.PhoneNumber,
		DisplayName: in.DisplayName,
		NIT:         in.NIT,
	}, memberID)
}

func (s *procurementService) UpdateSupplier(ctx context.Context, orgID int32, in UpdateSupplierInput) (*domain.Supplier, error) {
	return s.suppliers.Update(ctx, orgID, &domain.Supplier{
		ID:             in.ID,
		OrganizationID: orgID,
		DeliveryDays:   in.DeliveryDays,
		MinOrderAmount: in.MinOrderAmount,
		Notes:          in.Notes,
		IsActive:       in.IsActive,
	})
}

func (s *procurementService) ListProducts(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.products.List(ctx, orgID, limit, offset)
}

func (s *procurementService) CreateProduct(ctx context.Context, orgID int32, in CreateProductInput) (*domain.Product, error) {
	unit := in.Unit
	if unit == "" {
		unit = "und"
	}
	return s.products.Create(ctx, orgID, &domain.Product{
		OrganizationID: orgID,
		Name:           in.Name,
		SKU:            in.SKU,
		Unit:           unit,
	})
}

func (s *procurementService) UpdateProduct(ctx context.Context, orgID int32, in UpdateProductInput) (*domain.Product, error) {
	return s.products.Update(ctx, orgID, &domain.Product{
		ID:             in.ID,
		OrganizationID: orgID,
		Name:           in.Name,
		SKU:            in.SKU,
		Unit:           in.Unit,
		IsActive:       in.IsActive,
	})
}

func (s *procurementService) ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error) {
	return s.runs.ListRuns(ctx, orgID, limit, offset)
}

func (s *procurementService) CreateRun(ctx context.Context, orgID int32, memberID string, in CreateRunInput) (*domain.InquiryRun, error) {
	return s.runs.CreateRun(ctx, orgID, memberID, in)
}

func (s *procurementService) SendRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error) {
	return s.runs.SendRun(ctx, orgID, runID)
}

func (s *procurementService) GetBoard(ctx context.Context, orgID, runID int32) (*domain.Board, error) {
	return s.board.GetBoard(ctx, orgID, runID)
}

func (s *procurementService) PlaceOrder(ctx context.Context, orgID int32, memberID string, in PlaceOrderInput) (*domain.Order, error) {
	return s.orders.PlaceOrder(ctx, orgID, memberID, in)
}

func (s *procurementService) ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error) {
	return s.orders.ListRunOrders(ctx, orgID, runID)
}
