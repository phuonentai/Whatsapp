package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// --- fakes ---

type fakePaymentRepo struct {
	byID            map[int64]*domain.ClientPayment
	byPreference    map[string]*domain.ClientPayment
	byPayment       map[string]*domain.ClientPayment
	nextID          int64
	transitionCalls []domain.PaymentStatus
}

func newFakePaymentRepo() *fakePaymentRepo {
	return &fakePaymentRepo{
		byID:         map[int64]*domain.ClientPayment{},
		byPreference: map[string]*domain.ClientPayment{},
		byPayment:    map[string]*domain.ClientPayment{},
	}
}

func (r *fakePaymentRepo) Create(ctx context.Context, p *domain.ClientPayment) (*domain.ClientPayment, error) {
	r.nextID++
	p.ID = r.nextID
	r.byID[p.ID] = p
	r.byPreference[p.MPPreferenceID] = p
	if p.MPPaymentID != "" {
		r.byPayment[p.MPPaymentID] = p
	}
	return p, nil
}

func (r *fakePaymentRepo) GetByPreferenceID(ctx context.Context, preferenceID string) (*domain.ClientPayment, error) {
	if p, ok := r.byPreference[preferenceID]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (r *fakePaymentRepo) GetByPaymentID(ctx context.Context, paymentID string) (*domain.ClientPayment, error) {
	if p, ok := r.byPayment[paymentID]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (r *fakePaymentRepo) AttachPaymentID(ctx context.Context, id int64, mpPaymentID string) (*domain.ClientPayment, error) {
	p := r.byID[id]
	if p == nil || p.Status.IsTerminal() {
		return nil, domain.ErrPaymentTerminal
	}
	p.MPPaymentID = mpPaymentID
	r.byPayment[mpPaymentID] = p
	return p, nil
}

func (r *fakePaymentRepo) Transition(ctx context.Context, id int64, status domain.PaymentStatus, mpPaymentID string, paidAt *time.Time) (*domain.ClientPayment, error) {
	p := r.byID[id]
	if p == nil || p.Status.IsTerminal() {
		return nil, domain.ErrPaymentTerminal
	}
	r.transitionCalls = append(r.transitionCalls, status)
	p.Status = status
	p.MPPaymentID = mpPaymentID
	p.PaidAt = paidAt
	if status == domain.PaymentStatusPaid {
		r.byPayment[mpPaymentID] = p
	}
	return p, nil
}

type fakePaymentRail struct {
	initPoint      string
	preferenceID   string
	createErr      error
	detail         *domain.PaymentDetail
	verifyErr      error
	createdPrices  []int64
	createdRefs    []string
}

func (r *fakePaymentRail) CreatePreference(ctx context.Context, orgID, dealID int32, unitPriceCOP int64, currency string) (string, string, error) {
	r.createdPrices = append(r.createdPrices, unitPriceCOP)
	r.createdRefs = append(r.createdRefs, "")
	return r.initPoint, r.preferenceID, r.createErr
}

func (r *fakePaymentRail) VerifyPayment(ctx context.Context, paymentID string) (*domain.PaymentDetail, error) {
	return r.detail, r.verifyErr
}

type fakeDealRepo struct {
	deals map[int32]*crmdomain.DealWithRefs
}

func (m *fakeDealRepo) Create(ctx context.Context, deal *crmdomain.Deal) (*crmdomain.Deal, error) { return deal, nil }
func (m *fakeDealRepo) GetByID(ctx context.Context, orgID, dealID int32) (*crmdomain.DealWithRefs, error) {
	if d, ok := m.deals[dealID]; ok {
		return d, nil
	}
	return nil, errors.New("deal not found")
}
func (m *fakeDealRepo) List(ctx context.Context, orgID int32, pipelineID, stageID int32, status string, contactID, limit, offset int32) ([]*crmdomain.DealWithRefs, error) {
	return nil, nil
}
func (m *fakeDealRepo) Update(ctx context.Context, deal *crmdomain.Deal) (*crmdomain.Deal, error) { return deal, nil }
func (m *fakeDealRepo) UpdateStage(ctx context.Context, orgID, dealID, stageID int32) (*crmdomain.Deal, error) {
	return nil, nil
}
func (m *fakeDealRepo) Delete(ctx context.Context, orgID, dealID int32) error { return nil }

type fakeConvRepo struct {
	byContact map[int32]*crmdomain.Conversation
}

func (m *fakeConvRepo) GetByID(ctx context.Context, orgID, convID int32) (*crmdomain.Conversation, error) {
	return nil, nil
}
func (m *fakeConvRepo) GetActiveByContact(ctx context.Context, orgID, contactID int32) (*crmdomain.Conversation, error) {
	if c, ok := m.byContact[contactID]; ok {
		return c, nil
	}
	return nil, errors.New("conversation not found")
}
func (m *fakeConvRepo) GetActiveByContactChannel(ctx context.Context, orgID, contactID int32, channel string) (*crmdomain.Conversation, error) {
	return m.GetActiveByContact(ctx, orgID, contactID)
}
func (m *fakeConvRepo) Create(ctx context.Context, conv *crmdomain.Conversation) (*crmdomain.Conversation, error) {
	return conv, nil
}
func (m *fakeConvRepo) EnsureActive(ctx context.Context, conv *crmdomain.Conversation) (*crmdomain.Conversation, error) {
	return conv, nil
}
func (m *fakeConvRepo) UpdateLastMessageAt(ctx context.Context, orgID, convID int32, lastMessageAt *time.Time) (*crmdomain.Conversation, error) {
	return nil, nil
}
func (m *fakeConvRepo) UpdateStatus(ctx context.Context, orgID, convID int32, status crmdomain.ConversationStatus) (*crmdomain.Conversation, error) {
	return nil, nil
}
func (m *fakeConvRepo) ListByOrganization(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string) ([]*crmdomain.ConversationWithContact, error) {
	return nil, nil
}

type fakeActivitySvc struct {
	created []*crmServices.CreateActivityRequest
}

func (m *fakeActivitySvc) Create(ctx context.Context, orgID int32, req *crmServices.CreateActivityRequest) (*crmdomain.Activity, error) {
	m.created = append(m.created, req)
	return &crmdomain.Activity{}, nil
}
func (m *fakeActivitySvc) ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) (crmServices.ListResult[*crmdomain.ActivityWithActor], error) {
	return crmServices.ListResult[*crmdomain.ActivityWithActor]{}, nil
}
func (m *fakeActivitySvc) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) (crmServices.ListResult[*crmdomain.ActivityWithActor], error) {
	return crmServices.ListResult[*crmdomain.ActivityWithActor]{}, nil
}
func (m *fakeActivitySvc) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) (crmServices.ListResult[*crmdomain.ActivityWithActor], error) {
	return crmServices.ListResult[*crmdomain.ActivityWithActor]{}, nil
}
func (m *fakeActivitySvc) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) (crmServices.ListResult[*crmdomain.ActivityWithActor], error) {
	return crmServices.ListResult[*crmdomain.ActivityWithActor]{}, nil
}

type fakeOutbound struct {
	sent []string
}

func (m *fakeOutbound) SendMessage(ctx context.Context, orgID, convID int32, content string) (*crmdomain.Message, error) {
	m.sent = append(m.sent, content)
	return &crmdomain.Message{}, nil
}

type fakeLogger struct{}

func (fakeLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (fakeLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (fakeLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (fakeLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (fakeLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (fakeLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger { return fakeLogger{} }

func newTestSvc(repo *fakePaymentRepo, rail *fakePaymentRail, rate float64) (*paymentsService, *fakeDealRepo, *fakeConvRepo, *fakeActivitySvc, *fakeOutbound) {
	deals := &fakeDealRepo{deals: map[int32]*crmdomain.DealWithRefs{}}
	convs := &fakeConvRepo{byContact: map[int32]*crmdomain.Conversation{}}
	acts := &fakeActivitySvc{}
	out := &fakeOutbound{}
	svc := &paymentsService{
		repo: repo, rail: rail,
		dealRepo: deals, convRepo: convs,
		activitySvc: acts, outbound: out, logger: fakeLogger{},
		commissionRate: rate,
	}
	return svc, deals, convs, acts, out
}

// --- CreateLink ---

func TestCreateLink_AppliesCommission(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{initPoint: "https://checkout.mercadopago.com/x", preferenceID: "pref-1"}
	svc, _, _, _, _ := newTestSvc(repo, rail, 0.025)

	link, err := svc.CreateLink(context.Background(), 7, 99, nil, 100000)
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.mercadopago.com/x", link)
	require.Len(t, rail.createdPrices, 1)
	assert.Equal(t, int64(102500), rail.createdPrices[0])

	require.Len(t, repo.byID, 1)
	p := repo.byPreference["pref-1"]
	require.NotNil(t, p)
	assert.Equal(t, int64(100000), p.AmountCOP)
	assert.Equal(t, int64(2500), p.CommissionCOP)
	assert.Equal(t, domain.PaymentStatusPending, p.Status)
}

func TestCreateLink_ZeroCommissionPricesExactAmount(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{initPoint: "https://checkout.mercadopago.com/x", preferenceID: "pref-1"}
	svc, _, _, _, _ := newTestSvc(repo, rail, 0.0)

	_, err := svc.CreateLink(context.Background(), 7, 99, nil, 50000)
	require.NoError(t, err)
	require.Len(t, rail.createdPrices, 1)
	assert.Equal(t, int64(50000), rail.createdPrices[0])
	assert.Equal(t, int64(0), repo.byPreference["pref-1"].CommissionCOP)
}

func TestCreateLink_NonPositiveAmountRejected(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{}
	svc, _, _, _, _ := newTestSvc(repo, rail, 0.0)

	_, err := svc.CreateLink(context.Background(), 7, 99, nil, 0)
	require.Error(t, err)
	assert.Empty(t, rail.createdPrices)
	assert.Empty(t, repo.byID)
}

func TestCreateLink_RailFailurePersistsNothing(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{createErr: errors.New("mp down")}
	svc, _, _, _, _ := newTestSvc(repo, rail, 0.0)

	_, err := svc.CreateLink(context.Background(), 7, 99, nil, 1000)
	require.Error(t, err)
	assert.Empty(t, repo.byID)
}

// --- HandlePaymentEvent ---

func TestHandlePaymentEvent_ApprovedMarksPaidAndConfirms(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{detail: &domain.PaymentDetail{Status: domain.PaymentStatusPaid}}
	svc, deals, convs, acts, out := newTestSvc(repo, rail, 0.0)

	created, err := repo.Create(context.Background(), &domain.ClientPayment{
		OrganizationID: 7, DealID: 99, AmountCOP: 100000, Currency: "COP",
		Status: domain.PaymentStatusPending, MPPreferenceID: "pref-1",
	})
	require.NoError(t, err)
	_ = created
	deals.deals[99] = &crmdomain.DealWithRefs{}
	contactID := int32(5)
	deals.deals[99].ContactID = &contactID
	convs.byContact[5] = &crmdomain.Conversation{}

	// link the payment id as if payment_created had attached it
	_, err = repo.AttachPaymentID(context.Background(), created.ID, "pay-111")
	require.NoError(t, err)

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-111"))

	assert.Equal(t, domain.PaymentStatusPaid, repo.byID[created.ID].Status)
	require.Len(t, out.sent, 1)
	assert.Contains(t, out.sent[0], "Pago confirmado")
	require.Len(t, acts.created, 1)
	assert.Equal(t, "Pago recibido", acts.created[0].Asunto)
}

func TestHandlePaymentEvent_DuplicateEventTransitionsOnce(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{detail: &domain.PaymentDetail{Status: domain.PaymentStatusPaid}}
	svc, deals, convs, _, out := newTestSvc(repo, rail, 0.0)

	created, _ := repo.Create(context.Background(), &domain.ClientPayment{
		OrganizationID: 7, DealID: 99, AmountCOP: 100000, Currency: "COP",
		Status: domain.PaymentStatusPending, MPPreferenceID: "pref-1",
	})
	_, _ = repo.AttachPaymentID(context.Background(), created.ID, "pay-222")
	deals.deals[99] = &crmdomain.DealWithRefs{}
	contactID := int32(5)
	deals.deals[99].ContactID = &contactID
	convs.byContact[5] = &crmdomain.Conversation{}

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-222"))
	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-222"))

	require.Len(t, out.sent, 1, "confirmation must fire exactly once")
	paidCalls := 0
	for _, s := range repo.transitionCalls {
		if s == domain.PaymentStatusPaid {
			paidCalls++
		}
	}
	assert.Equal(t, 1, paidCalls)
}

func TestHandlePaymentEvent_VerificationFailureLeavesPending(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{verifyErr: errors.New("mp api down")}
	svc, deals, convs, _, out := newTestSvc(repo, rail, 0.0)

	created, _ := repo.Create(context.Background(), &domain.ClientPayment{
		OrganizationID: 7, DealID: 99, AmountCOP: 100000, Currency: "COP",
		Status: domain.PaymentStatusPending, MPPreferenceID: "pref-1",
	})
	_, _ = repo.AttachPaymentID(context.Background(), created.ID, "pay-333")
	deals.deals[99] = &crmdomain.DealWithRefs{}
	contactID := int32(5)
	deals.deals[99].ContactID = &contactID
	convs.byContact[5] = &crmdomain.Conversation{}

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-333"))
	assert.Equal(t, domain.PaymentStatusPending, repo.byID[created.ID].Status)
	assert.Empty(t, out.sent)
}

func TestHandlePaymentEvent_UntrackedPaymentAcknowledged(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{detail: &domain.PaymentDetail{Status: domain.PaymentStatusPaid}}
	svc, _, _, _, out := newTestSvc(repo, rail, 0.0)

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-999"))
	assert.Empty(t, out.sent)
	assert.Empty(t, repo.transitionCalls)
}

func TestHandlePaymentEvent_CorrelatesViaPreferenceFromProviderDetail(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{detail: &domain.PaymentDetail{
		Status: domain.PaymentStatusPaid, PreferenceID: "pref-7",
	}}
	svc, deals, convs, _, out := newTestSvc(repo, rail, 0.0)

	created, _ := repo.Create(context.Background(), &domain.ClientPayment{
		OrganizationID: 7, DealID: 99, AmountCOP: 100000, Currency: "COP",
		Status: domain.PaymentStatusPending, MPPreferenceID: "pref-7",
	})
	deals.deals[99] = &crmdomain.DealWithRefs{}
	contactID := int32(5)
	deals.deals[99].ContactID = &contactID
	convs.byContact[5] = &crmdomain.Conversation{}

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-777"))
	assert.Equal(t, domain.PaymentStatusPaid, repo.byID[created.ID].Status)
	assert.Equal(t, "pay-777", repo.byID[created.ID].MPPaymentID)
	require.Len(t, out.sent, 1)
}

func TestHandlePaymentEvent_WithdrawnContactStillConfirmed(t *testing.T) {
	repo := newFakePaymentRepo()
	rail := &fakePaymentRail{detail: &domain.PaymentDetail{Status: domain.PaymentStatusPaid}}
	svc, deals, convs, _, out := newTestSvc(repo, rail, 0.0)

	created, _ := repo.Create(context.Background(), &domain.ClientPayment{
		OrganizationID: 7, DealID: 99, AmountCOP: 100000, Currency: "COP",
		Status: domain.PaymentStatusPending, MPPreferenceID: "pref-1",
	})
	_, _ = repo.AttachPaymentID(context.Background(), created.ID, "pay-444")
	deals.deals[99] = &crmdomain.DealWithRefs{}
	contactID := int32(5)
	deals.deals[99].ContactID = &contactID
	convs.byContact[5] = &crmdomain.Conversation{}

	require.NoError(t, svc.HandlePaymentEvent(context.Background(), "payment_approved", "pay-444"))
	require.Len(t, out.sent, 1)
	assert.NotContains(t, out.sent[0], "oferta")
	assert.NotContains(t, out.sent[0], "promo")
}
