package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type CompanyService interface {
	Create(ctx context.Context, orgID int32, req *CreateCompanyRequest) (*domain.Company, error)
	GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error)
	List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.CompanyWithCounts, error)
	Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.CompanyWithCounts, error)
	Update(ctx context.Context, orgID int32, req *UpdateCompanyRequest) (*domain.Company, error)
	Delete(ctx context.Context, orgID, companyID int32) error
}

type CreateCompanyRequest struct {
	Nombre         string
	Nit            string
	TipoEmpresa    string
	Sector         string
	Ciudad         string
	Departamento   string
	Website        string
	Phone          string
	Address        string
	Notes          string
	OwnerAccountID *int32
}

type UpdateCompanyRequest struct {
	ID             int32
	Nombre         string
	Nit            string
	TipoEmpresa    string
	Sector         string
	Ciudad         string
	Departamento   string
	Website        string
	Phone          string
	Address        string
	Notes          string
	OwnerAccountID *int32
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
	return s.companyRepo.Create(ctx, company)
}
func (s *companyService) GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error) {
	return s.companyRepo.GetByID(ctx, orgID, companyID)
}
func (s *companyService) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	return s.companyRepo.List(ctx, orgID, limit, offset)
}
func (s *companyService) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	return s.companyRepo.Search(ctx, orgID, query, limit, offset)
}
func (s *companyService) Update(ctx context.Context, orgID int32, req *UpdateCompanyRequest) (*domain.Company, error) {
	company := &domain.Company{ID: req.ID, OrganizationID: orgID, Name: req.Nombre, Nit: req.Nit,
		TipoEmpresa: req.TipoEmpresa, Sector: req.Sector, Ciudad: req.Ciudad,
		Departamento: req.Departamento, Website: req.Website, Phone: req.Phone,
		Address: req.Address, Notes: req.Notes, OwnerAccountID: req.OwnerAccountID}
	return s.companyRepo.Update(ctx, company)
}
func (s *companyService) Delete(ctx context.Context, orgID, companyID int32) error {
	return s.companyRepo.Delete(ctx, orgID, companyID)
}
