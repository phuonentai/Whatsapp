package repositories

import (
	"context"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func mapProduct(row sqlc.ProcurementProduct) *domain.Product {
	return &domain.Product{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		SKU:            row.Sku,
		Unit:           row.Unit,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

type productRepository struct {
	store sqlc.Store
}

// NewProductRepository builds the product repository.
func NewProductRepository(store sqlc.Store) domain.ProductRepository {
	return &productRepository{store: store}
}

func (r *productRepository) Create(ctx context.Context, orgID int32, product *domain.Product) (*domain.Product, error) {
	row, err := r.store.CreateProduct(ctx, sqlc.CreateProductParams{
		OrganizationID: orgID,
		Name:           product.Name,
		Sku:            product.SKU,
		Unit:           product.Unit,
	})
	if err != nil {
		return nil, err
	}
	return mapProduct(row), nil
}

func (r *productRepository) GetByID(ctx context.Context, orgID, id int32) (*domain.Product, error) {
	row, err := r.store.GetProduct(ctx, sqlc.GetProductParams{ID: id, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapProduct(row), nil
}

func (r *productRepository) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Product, error) {
	rows, err := r.store.ListProductsByOrganization(ctx, sqlc.ListProductsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Product, 0, len(rows))
	for i := range rows {
		out = append(out, mapProduct(rows[i]))
	}
	return out, nil
}

func (r *productRepository) ListByIDs(ctx context.Context, orgID int32, ids []int32) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return []*domain.Product{}, nil
	}
	rows, err := r.store.ListProductsByIDs(ctx, sqlc.ListProductsByIDsParams{
		OrganizationID: orgID,
		Column2:        ids,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Product, 0, len(rows))
	for i := range rows {
		out = append(out, mapProduct(rows[i]))
	}
	return out, nil
}

func (r *productRepository) Update(ctx context.Context, orgID int32, product *domain.Product) (*domain.Product, error) {
	row, err := r.store.UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:             product.ID,
		OrganizationID: orgID,
		Name:           product.Name,
		Sku:            product.SKU,
		Unit:           product.Unit,
		IsActive:       product.IsActive,
	})
	if isNoRows(err) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapProduct(row), nil
}
