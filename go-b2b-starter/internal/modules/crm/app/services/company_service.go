package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type CompanyService interface {
	Create(ctx context.Context, orgID int32, req *CreateCompanyRequest) (*domain.Company, error)
	GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error)
	List(ctx context.Context, orgID int32, limit, offset int32) (ListResult[*domain.CompanyWithCounts], error)
	Search(ctx context.Context, orgID int32, query string, limit, offset int32) (ListResult[*domain.CompanyWithCounts], error)
	Update(ctx context.Context, orgID int32, req *UpdateCompanyRequest) (*domain.Company, error)
	Delete(ctx context.Context, orgID, companyID int32) error
}

type CreateCompanyRequest struct {
	Nombre         string `json:"name"`
	Nit            string `json:"nit"`
	TipoEmpresa    string `json:"tipo_empresa"`
	Sector         string `json:"sector"`
	Ciudad         string `json:"ciudad"`
	Departamento   string `json:"departamento"`
	Website        string `json:"website"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	Notes          string `json:"notes"`
	OwnerAccountID *int32 `json:"owner_account_id"`
}

type UpdateCompanyRequest struct {
	ID             int32  `json:"id"`
	Nombre         string `json:"name"`
	Nit            string `json:"nit"`
	TipoEmpresa    string `json:"tipo_empresa"`
	Sector         string `json:"sector"`
	Ciudad         string `json:"ciudad"`
	Departamento   string `json:"departamento"`
	Website        string `json:"website"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	Notes          string `json:"notes"`
	OwnerAccountID *int32 `json:"owner_account_id"`
}

type companyService struct {
	companyRepo     domain.CompanyRepository
	featureProvider features.FeatureProvider
}

func NewCompanyService(companyRepo domain.CompanyRepository, featureProvider features.FeatureProvider) CompanyService {
	return &companyService{companyRepo: companyRepo, featureProvider: featureProvider}
}

func (s *companyService) Create(ctx context.Context, orgID int32, req *CreateCompanyRequest) (*domain.Company, error) {
	company := &domain.Company{
		OrganizationID: orgID, Name: req.Nombre, Nit: req.Nit,
		TipoEmpresa: req.TipoEmpresa, Sector: req.Sector, Ciudad: req.Ciudad,
		Departamento: req.Departamento, Website: req.Website, Phone: req.Phone,
		Address: req.Address, Notes: req.Notes, OwnerAccountID: req.OwnerAccountID,
	}
	if err := company.Validate(); err != nil { return nil, err }
	created, err := s.companyRepo.Create(ctx, company)
	if err != nil {
		if isUniqueViolationOn(err, "companies_organization_id_name_key") {
			return nil, domain.ErrCompanyDuplicateName
		}
		return nil, err
	}
	return created, nil
}
func (s *companyService) GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error) {
	return s.companyRepo.GetByID(ctx, orgID, companyID)
}
func (s *companyService) List(ctx context.Context, orgID int32, limit, offset int32) (ListResult[*domain.CompanyWithCounts], error) {
	items, err := s.companyRepo.List(ctx, orgID, limit, offset)
	if err != nil {
		return ListResult[*domain.CompanyWithCounts]{}, err
	}
	total, err := s.companyRepo.CountList(ctx, orgID)
	if err != nil {
		return ListResult[*domain.CompanyWithCounts]{}, err
	}
	return ListResult[*domain.CompanyWithCounts]{Items: items, Total: total}, nil
}
func (s *companyService) Search(ctx context.Context, orgID int32, query string, limit, offset int32) (ListResult[*domain.CompanyWithCounts], error) {
	items, err := s.companyRepo.Search(ctx, orgID, query, limit, offset)
	if err != nil {
		return ListResult[*domain.CompanyWithCounts]{}, err
	}
	total, err := s.companyRepo.CountSearch(ctx, orgID, query)
	if err != nil {
		return ListResult[*domain.CompanyWithCounts]{}, err
	}
	return ListResult[*domain.CompanyWithCounts]{Items: items, Total: total}, nil
}
func (s *companyService) Update(ctx context.Context, orgID int32, req *UpdateCompanyRequest) (*domain.Company, error) {
	company := &domain.Company{ID: req.ID, OrganizationID: orgID, Name: req.Nombre, Nit: req.Nit,
		TipoEmpresa: req.TipoEmpresa, Sector: req.Sector, Ciudad: req.Ciudad,
		Departamento: req.Departamento, Website: req.Website, Phone: req.Phone,
		Address: req.Address, Notes: req.Notes, OwnerAccountID: req.OwnerAccountID}
	updated, err := s.companyRepo.Update(ctx, company)
	if err != nil {
		if isUniqueViolationOn(err, "companies_organization_id_name_key") {
			return nil, domain.ErrCompanyDuplicateName
		}
		return nil, err
	}
	return updated, nil
}
func (s *companyService) Delete(ctx context.Context, orgID, companyID int32) error {
	return s.companyRepo.Delete(ctx, orgID, companyID)
}
