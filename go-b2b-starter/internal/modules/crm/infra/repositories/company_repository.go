package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type companyRepository struct{ store sqlc.Store }

func NewCompanyRepository(store sqlc.Store) domain.CompanyRepository {
	return &companyRepository{store: store}
}

func (r *companyRepository) Create(ctx context.Context, company *domain.Company) (*domain.Company, error) {
	result, err := r.store.CreateCompany(ctx, sqlc.CreateCompanyParams{
		OrganizationID: company.OrganizationID, Name: company.Name,
		Nit: helpers.ToPgText(company.Nit), TipoEmpresa: helpers.ToPgText(company.TipoEmpresa),
		Sector: helpers.ToPgText(company.Sector), Ciudad: helpers.ToPgText(company.Ciudad),
		Departamento: helpers.ToPgText(company.Departamento), Website: helpers.ToPgText(company.Website),
		Phone: helpers.ToPgText(company.Phone), Address: helpers.ToPgText(company.Address),
		Notes: helpers.ToPgText(company.Notes), Metadata: helpers.ToJSONB(company.Metadata),
		OwnerAccountID: helpers.ToPgInt4Ptr(company.OwnerAccountID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}
	return mapCrmCompany(&result), nil
}

func (r *companyRepository) GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error) {
	result, err := r.store.GetCompanyByID(ctx, sqlc.GetCompanyByIDParams{ID: companyID, OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return &domain.CompanyWithCounts{Company: *mapCompanyByIDRow(&result), TotalContactos: int32(result.TotalContactos), TotalNegocios: int32(result.TotalNegocios)}, nil
}

func (r *companyRepository) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	results, err := r.store.ListCompaniesByOrganization(ctx, sqlc.ListCompaniesByOrganizationParams{OrganizationID: orgID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	companies := make([]*domain.CompanyWithCounts, len(results))
	for i := range results {
		companies[i] = &domain.CompanyWithCounts{Company: *mapCompanyListRow(&results[i]), TotalContactos: int32(results[i].TotalContactos), TotalNegocios: int32(results[i].TotalNegocios)}
	}
	return companies, nil
}

func (r *companyRepository) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	results, err := r.store.SearchCompanies(ctx, sqlc.SearchCompaniesParams{OrganizationID: orgID, Column2: helpers.ToPgText(query), Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("failed to search companies: %w", err)
	}
	companies := make([]*domain.CompanyWithCounts, len(results))
	for i := range results {
		companies[i] = &domain.CompanyWithCounts{Company: *mapCompanySearchRow(&results[i]), TotalContactos: int32(results[i].TotalContactos), TotalNegocios: int32(results[i].TotalNegocios)}
	}
	return companies, nil
}

func (r *companyRepository) GetByNit(ctx context.Context, orgID int32, nit string) (*domain.Company, error) {
	result, err := r.store.GetCompanyByNit(ctx, sqlc.GetCompanyByNitParams{
		OrganizationID: orgID,
		Nit:            helpers.ToPgText(nit),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCompanyNotFound
		}
		return nil, fmt.Errorf("failed to get company by nit: %w", err)
	}
	return mapCrmCompany(&result), nil
}

func (r *companyRepository) Update(ctx context.Context, company *domain.Company) (*domain.Company, error) {
	result, err := r.store.UpdateCompany(ctx, sqlc.UpdateCompanyParams{
		ID: company.ID, OrganizationID: company.OrganizationID,
		Column3: helpers.ToPgText(company.Name), Column4: helpers.ToPgText(company.Nit),
		Column5: helpers.ToPgText(company.TipoEmpresa), Column6: helpers.ToPgText(company.Sector),
		Column7: helpers.ToPgText(company.Ciudad), Column8: helpers.ToPgText(company.Departamento),
		Column9: helpers.ToPgText(company.Website), Column10: helpers.ToPgText(company.Phone),
		Column11: helpers.ToPgText(company.Address), Column12: helpers.ToPgText(company.Notes),
		Column13: helpers.ToJSONB(company.Metadata), OwnerAccountID: helpers.ToPgInt4Ptr(company.OwnerAccountID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update company: %w", err)
	}
	return mapCrmCompany(&result), nil
}

func (r *companyRepository) Delete(ctx context.Context, orgID, companyID int32) error {
	return r.store.DeleteCompany(ctx, sqlc.DeleteCompanyParams{ID: companyID, OrganizationID: orgID})
}

func mapCrmCompany(c *sqlc.CrmCompany) *domain.Company {
	return &domain.Company{
		ID: c.ID, OrganizationID: c.OrganizationID, Name: c.Name,
		Nit: helpers.FromPgText(c.Nit), TipoEmpresa: helpers.FromPgText(c.TipoEmpresa),
		Sector: helpers.FromPgText(c.Sector), Ciudad: helpers.FromPgText(c.Ciudad),
		Departamento: helpers.FromPgText(c.Departamento), Website: helpers.FromPgText(c.Website),
		Phone: helpers.FromPgText(c.Phone), Address: helpers.FromPgText(c.Address),
		Notes: helpers.FromPgText(c.Notes), Metadata: helpers.FromJSONB(c.Metadata),
		OwnerAccountID: helpers.FromPgInt4Ptr(c.OwnerAccountID),
		CreatedAt:      c.CreatedAt.Time, UpdatedAt: c.UpdatedAt.Time,
	}
}

func mapCompanyByIDRow(r *sqlc.GetCompanyByIDRow) *domain.Company {
	return &domain.Company{
		ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
		Nit: helpers.FromPgText(r.Nit), TipoEmpresa: helpers.FromPgText(r.TipoEmpresa),
		Sector: helpers.FromPgText(r.Sector), Ciudad: helpers.FromPgText(r.Ciudad),
		Departamento: helpers.FromPgText(r.Departamento), Website: helpers.FromPgText(r.Website),
		Phone: helpers.FromPgText(r.Phone), Address: helpers.FromPgText(r.Address),
		Notes: helpers.FromPgText(r.Notes), Metadata: helpers.FromJSONB(r.Metadata),
		OwnerAccountID: helpers.FromPgInt4Ptr(r.OwnerAccountID),
		CreatedAt:      r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}
}

func mapCompanyListRow(r *sqlc.ListCompaniesByOrganizationRow) *domain.Company {
	return &domain.Company{
		ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
		Nit: helpers.FromPgText(r.Nit), TipoEmpresa: helpers.FromPgText(r.TipoEmpresa),
		Sector: helpers.FromPgText(r.Sector), Ciudad: helpers.FromPgText(r.Ciudad),
		Departamento: helpers.FromPgText(r.Departamento), Website: helpers.FromPgText(r.Website),
		Phone: helpers.FromPgText(r.Phone), Address: helpers.FromPgText(r.Address),
		Notes: helpers.FromPgText(r.Notes), Metadata: helpers.FromJSONB(r.Metadata),
		OwnerAccountID: helpers.FromPgInt4Ptr(r.OwnerAccountID),
		CreatedAt:      r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}
}

func mapCompanySearchRow(r *sqlc.SearchCompaniesRow) *domain.Company {
	return &domain.Company{
		ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
		Nit: helpers.FromPgText(r.Nit), TipoEmpresa: helpers.FromPgText(r.TipoEmpresa),
		Sector: helpers.FromPgText(r.Sector), Ciudad: helpers.FromPgText(r.Ciudad),
		Departamento: helpers.FromPgText(r.Departamento), Website: helpers.FromPgText(r.Website),
		Phone: helpers.FromPgText(r.Phone), Address: helpers.FromPgText(r.Address),
		Notes: helpers.FromPgText(r.Notes), Metadata: helpers.FromJSONB(r.Metadata),
		OwnerAccountID: helpers.FromPgInt4Ptr(r.OwnerAccountID),
		CreatedAt:      r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}
}
