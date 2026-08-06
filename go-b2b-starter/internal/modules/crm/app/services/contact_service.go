package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type ContactService interface {
	Create(ctx context.Context, orgID int32, req *CreateContactRequest) (*domain.Contact, error)
	GetByID(ctx context.Context, orgID, contactID int32) (*domain.Contact, error)
	List(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*domain.Contact, error)
	Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.Contact, error)
	Update(ctx context.Context, orgID int32, req *UpdateContactRequest) (*domain.Contact, error)
	Delete(ctx context.Context, orgID, contactID int32) error
}

type CreateContactRequest struct {
	OrganizationID  int32
	PhoneNumber     string
	DisplayName     string
	Email           string
	CompanyID       *int32
	Source          domain.ContactSource
	LeadStatus      domain.LeadStatus
	JobTitle        string
	AssignedTo      *int32
	TipoDocumento   domain.TipoDocumento
	NumeroDocumento string
}

type UpdateContactRequest struct {
	ID              int32
	OrganizationID  int32
	DisplayName     string
	Email           string
	CompanyID       *int32
	Source          string
	LeadStatus      string
	JobTitle        string
	AssignedTo      *int32
	TipoDocumento   string
	NumeroDocumento string
	AvatarURL       string
}

type contactService struct {
	contactRepo     domain.ContactRepository
	featureProvider features.FeatureProvider
}

func NewContactService(contactRepo domain.ContactRepository, featureProvider features.FeatureProvider) ContactService {
	return &contactService{contactRepo: contactRepo, featureProvider: featureProvider}
}

func (s *contactService) Create(ctx context.Context, orgID int32, req *CreateContactRequest) (*domain.Contact, error) {
	contact := &domain.Contact{
		OrganizationID:  orgID,
		PhoneNumber:     req.PhoneNumber,
		DisplayName:     req.DisplayName,
		Email:           req.Email,
		CompanyID:       req.CompanyID,
		Source:          req.Source,
		LeadStatus:      req.LeadStatus,
		JobTitle:        req.JobTitle,
		AssignedTo:      req.AssignedTo,
		TipoDocumento:   req.TipoDocumento,
		NumeroDocumento: req.NumeroDocumento,
	}
	created, err := s.contactRepo.UpsertByPhone(ctx, contact)
	if err != nil {
		return nil, err
	}
	return created, nil
}
func (s *contactService) GetByID(ctx context.Context, orgID, contactID int32) (*domain.Contact, error) {
	return s.contactRepo.GetByID(ctx, orgID, contactID)
}
func (s *contactService) List(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*domain.Contact, error) {
	return s.contactRepo.ListFiltered(ctx, orgID, source, leadStatus, companyID, assignedTo, limit, offset)
}
func (s *contactService) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.Contact, error) {
	return s.contactRepo.Search(ctx, orgID, query, limit, offset)
}
func (s *contactService) Update(ctx context.Context, orgID int32, req *UpdateContactRequest) (*domain.Contact, error) {
	contact, err := s.contactRepo.GetByID(ctx, orgID, req.ID)
	if err != nil {
		return nil, fmt.Errorf("contacto no encontrado: %w", err)
	}
	if req.DisplayName != "" { contact.DisplayName = req.DisplayName }
	if req.Email != "" { contact.Email = req.Email }
	if req.CompanyID != nil { contact.CompanyID = req.CompanyID }
	if req.Source != "" { contact.Source = domain.ContactSource(req.Source) }
	if req.LeadStatus != "" { contact.LeadStatus = domain.LeadStatus(req.LeadStatus) }
	if req.JobTitle != "" { contact.JobTitle = req.JobTitle }
	if req.AssignedTo != nil { contact.AssignedTo = req.AssignedTo }
	if req.TipoDocumento != "" { contact.TipoDocumento = domain.TipoDocumento(req.TipoDocumento) }
	if req.NumeroDocumento != "" { contact.NumeroDocumento = req.NumeroDocumento }
	if req.AvatarURL != "" { contact.AvatarURL = req.AvatarURL }
	return s.contactRepo.Update(ctx, contact)
}
func (s *contactService) Delete(ctx context.Context, orgID, contactID int32) error {
	return s.contactRepo.Delete(ctx, orgID, contactID)
}
