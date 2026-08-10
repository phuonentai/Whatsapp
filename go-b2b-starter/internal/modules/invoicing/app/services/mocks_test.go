package services

import (
	"context"
	"errors"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	invdomain "github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type mockInvoiceRepo struct {
	byDeal map[int32]*invdomain.Invoice
	byExt  map[string]*invdomain.Invoice
	byID   map[int64]*invdomain.Invoice
	pending []*invdomain.Invoice
}

func newMockInvoiceRepo() *mockInvoiceRepo {
	return &mockInvoiceRepo{
		byDeal: map[int32]*invdomain.Invoice{},
		byExt:  map[string]*invdomain.Invoice{},
		byID:   map[int64]*invdomain.Invoice{},
	}
}

func (m *mockInvoiceRepo) GetByDeal(ctx context.Context, orgID, dealID int32) (*invdomain.Invoice, error) {
	if inv, ok := m.byDeal[dealID]; ok {
		return inv, nil
	}
	return nil, invdomain.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) GetByExternalID(ctx context.Context, externalID string) (*invdomain.Invoice, error) {
	if inv, ok := m.byExt[externalID]; ok {
		return inv, nil
	}
	return nil, invdomain.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) Insert(ctx context.Context, inv *invdomain.Invoice) (*invdomain.Invoice, error) {
	if _, exists := m.byDeal[inv.DealID]; exists {
		return nil, invdomain.ErrInvoiceExists
	}
	inv.ID = int64(len(m.byID) + 1)
	m.byDeal[inv.DealID] = inv
	m.byID[inv.ID] = inv
	m.byExt[inv.ExternalID] = inv
	return inv, nil
}

func (m *mockInvoiceRepo) UpdateStatus(ctx context.Context, id int64, status invdomain.InvoiceStatus, cufe, pdfURL string) (*invdomain.Invoice, error) {
	inv := m.byID[id]
	if inv == nil {
		return nil, invdomain.ErrInvoiceNotFound
	}
	inv.Status = status
	if cufe != "" {
		inv.Cufe = cufe
	}
	if pdfURL != "" {
		inv.PdfURL = pdfURL
	}
	return inv, nil
}

func (m *mockInvoiceRepo) MarkNotified(ctx context.Context, id int64, status invdomain.InvoiceStatus) (*invdomain.Invoice, error) {
	inv := m.byID[id]
	if inv == nil {
		return nil, invdomain.ErrInvoiceNotFound
	}
	inv.NotifiedStatus = status
	return inv, nil
}

func (m *mockInvoiceRepo) ListByStatus(ctx context.Context, status invdomain.InvoiceStatus, limit int32) ([]*invdomain.Invoice, error) {
	return m.pending, nil
}

type mockProvider struct {
	createCalls int
	created     *invdomain.Invoice
	customer    *invdomain.CustomerRef
	status      *invdomain.Invoice
}

func (p *mockProvider) CreateInvoice(ctx context.Context, orgID int32, req *invdomain.InvoiceRequest) (*invdomain.Invoice, error) {
	p.createCalls++
	return p.created, nil
}

func (p *mockProvider) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*invdomain.Invoice, error) {
	return p.status, nil
}

func (p *mockProvider) UpsertCustomer(ctx context.Context, orgID int32, customer invdomain.CustomerInfo) (*invdomain.CustomerRef, error) {
	return p.customer, nil
}

type mockDealRepo struct {
	deals map[int32]*domain.DealWithRefs
}

func (m *mockDealRepo) Create(ctx context.Context, deal *domain.Deal) (*domain.Deal, error) { return deal, nil }
func (m *mockDealRepo) GetByID(ctx context.Context, orgID, dealID int32) (*domain.DealWithRefs, error) {
	if d, ok := m.deals[dealID]; ok {
		return d, nil
	}
	return nil, errors.New("deal not found")
}
func (m *mockDealRepo) List(ctx context.Context, orgID int32, pipelineID, stageID int32, status string, contactID, limit, offset int32) ([]*domain.DealWithRefs, error) {
	return nil, nil
}
func (m *mockDealRepo) Update(ctx context.Context, deal *domain.Deal) (*domain.Deal, error)      { return deal, nil }
func (m *mockDealRepo) UpdateStage(ctx context.Context, orgID, dealID, stageID int32) (*domain.Deal, error) {
	return nil, nil
}
func (m *mockDealRepo) Delete(ctx context.Context, orgID, dealID int32) error { return nil }

type mockContactRepo struct {
	contacts map[int32]*domain.Contact
}

func (m *mockContactRepo) UpsertByPhone(ctx context.Context, c *domain.Contact) (*domain.Contact, error) {
	return c, nil
}
func (m *mockContactRepo) GetByID(ctx context.Context, orgID, contactID int32) (*domain.Contact, error) {
	if c, ok := m.contacts[contactID]; ok {
		return c, nil
	}
	return nil, errors.New("contact not found")
}
func (m *mockContactRepo) GetByPhone(ctx context.Context, orgID int32, phone string) (*domain.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) ListFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*domain.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) Update(ctx context.Context, c *domain.Contact) (*domain.Contact, error) { return c, nil }
func (m *mockContactRepo) Delete(ctx context.Context, orgID, contactID int32) error              { return nil }

type mockCompanyRepo struct {
	companies map[int32]*domain.CompanyWithCounts
}

func (m *mockCompanyRepo) Create(ctx context.Context, c *domain.Company) (*domain.Company, error) { return c, nil }
func (m *mockCompanyRepo) GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error) {
	if c, ok := m.companies[companyID]; ok {
		return c, nil
	}
	return nil, errors.New("company not found")
}
func (m *mockCompanyRepo) GetByNit(ctx context.Context, orgID int32, nit string) (*domain.Company, error) {
	return nil, errors.New("company not found")
}
func (m *mockCompanyRepo) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	return nil, nil
}
func (m *mockCompanyRepo) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	return nil, nil
}
func (m *mockCompanyRepo) Update(ctx context.Context, c *domain.Company) (*domain.Company, error) { return c, nil }
func (m *mockCompanyRepo) Delete(ctx context.Context, orgID, companyID int32) error              { return nil }

type mockConvRepo struct {
	byContact map[int32]*domain.Conversation
}

func (m *mockConvRepo) GetByID(ctx context.Context, orgID, convID int32) (*domain.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) GetActiveByContact(ctx context.Context, orgID, contactID int32) (*domain.Conversation, error) {
	if c, ok := m.byContact[contactID]; ok {
		return c, nil
	}
	return nil, errors.New("conversation not found")
}
func (m *mockConvRepo) Create(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	return conv, nil
}
func (m *mockConvRepo) EnsureActive(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	return conv, nil
}
func (m *mockConvRepo) UpdateLastMessageAt(ctx context.Context, orgID, convID int32, lastMessageAt *time.Time) (*domain.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus) (*domain.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) ListByOrganization(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string) ([]*domain.ConversationWithContact, error) {
	return nil, nil
}

type mockActivitySvc struct{}

func (m *mockActivitySvc) Create(ctx context.Context, orgID int32, req *crmServices.CreateActivityRequest) (*domain.Activity, error) {
	return nil, nil
}
func (m *mockActivitySvc) ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}
func (m *mockActivitySvc) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}
func (m *mockActivitySvc) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}
func (m *mockActivitySvc) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}

type mockOutbound struct {
	sent []string
}

func (m *mockOutbound) SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error) {
	m.sent = append(m.sent, content)
	return &domain.Message{}, nil
}

type nopLogger struct{}

func (nopLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (nopLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (nopLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return nopLogger{}
}

func newTestService(repo *mockInvoiceRepo, provider *mockProvider) (*invoicingService, *mockOutbound) {
	out := &mockOutbound{}
	svc := &invoicingService{
		repo:      repo,
		provider:  provider,
		dealRepo:  &mockDealRepo{},
		companyRepo: &mockCompanyRepo{},
		contactRepo: &mockContactRepo{},
		convRepo:   &mockConvRepo{},
		activitySvc: &mockActivitySvc{},
		outbound:   out,
		logger:     nopLogger{},
	}
	return svc, out
}

func (m *mockContactRepo) UpsertByIGUser(ctx context.Context, c *domain.Contact) (*domain.Contact, error) {
	return c, nil
}

func (m *mockContactRepo) GetByIGUser(ctx context.Context, orgID int32, igUserID string) (*domain.Contact, error) {
	return nil, domain.ErrContactNotFound
}

func (m *mockContactRepo) UpdateInstagramProfile(ctx context.Context, orgID, contactID int32, username, avatarURL, displayName string) (*domain.Contact, error) {
	return nil, domain.ErrContactNotFound
}

func (m *mockConvRepo) GetActiveByContactChannel(ctx context.Context, orgID, contactID int32, channel string) (*domain.Conversation, error) {
	return m.GetActiveByContact(ctx, orgID, contactID)
}
