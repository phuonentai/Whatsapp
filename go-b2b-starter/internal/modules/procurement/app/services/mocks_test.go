package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ---------- logger stub ----------

type stubLogger struct{}

func (stubLogger) Debug(string, ...logger.Fields)        {}
func (stubLogger) Info(string, ...logger.Fields)         {}
func (stubLogger) Warn(string, ...logger.Fields)         {}
func (stubLogger) Error(string, ...logger.Fields)        {}
func (stubLogger) Fatal(string, ...logger.Fields)        {}
func (stubLogger) WithFields(logger.Fields) logger.Logger { return stubLogger{} }

// ---------- LLM mock ----------

type mockLLM struct {
	mu        sync.Mutex
	responses []string
	next      int
	err       error
	calls     int
	models    []string
	prompts   []string
}

func (m *mockLLM) addResponse(text string) { m.responses = append(m.responses, text) }

func (m *mockLLM) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.prompts = append(m.prompts, req.Prompt)
	if m.err != nil {
		return nil, m.err
	}
	if m.next >= len(m.responses) {
		return &llmdomain.CompletionResponse{Text: `{"message":"respuesta"}`}, nil
	}
	text := m.responses[m.next]
	m.next++
	return &llmdomain.CompletionResponse{Text: text, TokensUsed: 10, Model: "gpt-4o-mini"}, nil
}

func (m *mockLLM) CompleteStream(ctx context.Context, req llmdomain.CompletionRequest, cb func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	return m.Complete(ctx, req)
}

func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return nil, 0, nil
}

// ctxCapturingLLM records the context seen by the provider call.
type ctxCapturingLLM struct {
	lastCtx context.Context
}

func (c *ctxCapturingLLM) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	c.lastCtx = ctx
	return &llmdomain.CompletionResponse{Text: `{"message":"hola"}`, TokensUsed: 1, Model: "gpt-4o-mini"}, nil
}

func (c *ctxCapturingLLM) CompleteStream(ctx context.Context, req llmdomain.CompletionRequest, cb func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	return c.Complete(ctx, req)
}

func (c *ctxCapturingLLM) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return nil, 0, nil
}

// ---------- billing mock (embed to satisfy the huge interface) ----------

type mockBilling struct {
	billingServices.BillingService
	exhausted bool
	status    *billingDomain.AiUsageStatus
}

func (m *mockBilling) GetAiUsageStatus(ctx context.Context, orgID int32) (*billingDomain.AiUsageStatus, error) {
	if m.status != nil {
		return m.status, nil
	}
	if m.exhausted {
		return &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}, nil
	}
	return &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}, nil
}

// ---------- run repo mock ----------

type mockRunRepo struct {
	runs        map[int32]*domain.InquiryRun
	recipients  map[int32]*domain.InquiryRecipient
	responses   map[int32]*domain.InquiryResponse
	byPhone     map[string][]*domain.InquiryRecipient
	suppliers   map[int32]*domain.Supplier
	displayName map[int32]string
	nextRunID   int32
	nextRecID   int32
	nextRespID  int32
	mu          sync.Mutex
	transitions []string
}

func newMockRunRepo() *mockRunRepo {
	return &mockRunRepo{
		runs:        map[int32]*domain.InquiryRun{},
		recipients:  map[int32]*domain.InquiryRecipient{},
		responses:   map[int32]*domain.InquiryResponse{},
		byPhone:     map[string][]*domain.InquiryRecipient{},
		suppliers:   map[int32]*domain.Supplier{},
		displayName: map[int32]string{},
	}
}

// seedSupplier registers an active supplier for run-creation tests.
func (m *mockRunRepo) seedSupplier(id, contactID int32, nit, display string) {
	m.suppliers[id] = &domain.Supplier{ID: id, OrganizationID: 42, ContactID: contactID, NIT: nit, IsActive: true}
	m.displayName[id] = display
}

func (m *mockRunRepo) CreateRun(ctx context.Context, orgID int32, nota *string, memberID string) (*domain.InquiryRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRunID++
	now := time.Now()
	run := &domain.InquiryRun{
		ID: m.nextRunID, OrganizationID: orgID, Status: domain.RunDraft, Source: "manual",
		Nota: nota, CreatedByMemberID: &memberID, CreatedAt: now, UpdatedAt: now,
	}
	m.runs[run.ID] = run
	return run, nil
}

func (m *mockRunRepo) GetRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error) {
	run, ok := m.runs[runID]
	if !ok || run.OrganizationID != orgID {
		return nil, domain.ErrRunNotFound
	}
	return run, nil
}

func (m *mockRunRepo) ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error) {
	return nil, nil
}

func (m *mockRunRepo) TransitionRun(ctx context.Context, orgID, runID int32, from, to domain.RunStatus) (*domain.InquiryRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[runID]
	if !ok || run.OrganizationID != orgID {
		return nil, domain.ErrRunNotFound
	}
	if run.Status != from {
		return nil, domain.ErrInvalidTransition
	}
	if !run.CanTransition(to) {
		return nil, domain.ErrInvalidTransition
	}
	run.Status = to
	now := time.Now()
	if to == domain.RunSending && run.SentAt == nil {
		run.SentAt = &now
	}
	if to == domain.RunCompleted || to == domain.RunPartiallyAnswered || to == domain.RunFailed || to == domain.RunCancelled {
		run.CompletedAt = &now
	}
	m.transitions = append(m.transitions, fmt.Sprintf("%s->%s", from, to))
	return run, nil
}

func (m *mockRunRepo) CreateRecipient(ctx context.Context, orgID, runID, supplierID, contactID int32, drafted *string) (*domain.InquiryRecipient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRecID++
	now := time.Now()
	r := &domain.InquiryRecipient{
		ID: m.nextRecID, OrganizationID: orgID, RunID: runID, SupplierID: supplierID,
		ContactID: contactID, DraftedMessage: drafted, Status: domain.RecipientPending,
		CreatedAt: now, UpdatedAt: now,
	}
	m.recipients[r.ID] = r
	return r, nil
}

func (m *mockRunRepo) GetRecipient(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	r, ok := m.recipients[recipientID]
	if !ok || r.OrganizationID != orgID {
		return nil, domain.ErrRecipientNotFound
	}
	return r, nil
}

func (m *mockRunRepo) ListRunRecipients(ctx context.Context, orgID, runID int32) ([]*domain.InquiryRecipient, error) {
	var out []*domain.InquiryRecipient
	for _, r := range m.recipients {
		if r.RunID == runID && r.OrganizationID == orgID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRunRepo) ListSuppliersWithDisplay(ctx context.Context, orgID int32, ids []int32) ([]domain.SupplierWithDisplay, error) {
	out := make([]domain.SupplierWithDisplay, 0, len(ids))
	for _, id := range ids {
		sup, ok := m.suppliers[id]
		if !ok || sup.OrganizationID != orgID {
			continue
		}
		out = append(out, domain.SupplierWithDisplay{Supplier: sup, DisplayName: m.displayName[id]})
	}
	return out, nil
}

func (m *mockRunRepo) ListRunRecipientsWithPhone(ctx context.Context, orgID, runID int32) ([]domain.RecipientWithPhone, error) {
	recs, err := m.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RecipientWithPhone, 0, len(recs))
	for _, r := range recs {
		out = append(out, domain.RecipientWithPhone{Recipient: r, ContactPhone: "+57300123456" + fmt.Sprint(r.ID%10)})
	}
	return out, nil
}

func (m *mockRunRepo) MarkRecipientSent(ctx context.Context, orgID, recipientID int32, providerID string) (*domain.InquiryRecipient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.recipients[recipientID]
	if !ok || r.OrganizationID != orgID {
		return nil, domain.ErrRecipientNotFound
	}
	if r.Status != domain.RecipientPending {
		return nil, domain.ErrRecipientNotFound
	}
	r.Status = domain.RecipientSent
	r.ProviderMessageID = &providerID
	now := time.Now()
	r.SentAt = &now
	return r, nil
}

func (m *mockRunRepo) MarkRecipientAnswered(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.recipients[recipientID]
	if !ok || r.OrganizationID != orgID {
		return nil, domain.ErrRecipientNotFound
	}
	if r.Status != domain.RecipientPending && r.Status != domain.RecipientSent {
		return nil, domain.ErrRecipientNotFound
	}
	r.Status = domain.RecipientAnswered
	now := time.Now()
	r.AnsweredAt = &now
	return r, nil
}

func (m *mockRunRepo) MarkRecipientTimedOut(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.recipients[recipientID]
	if !ok || r.OrganizationID != orgID {
		return nil, domain.ErrRecipientNotFound
	}
	if r.Status != domain.RecipientSent {
		return nil, domain.ErrRecipientNotFound
	}
	r.Status = domain.RecipientTimedOut
	return r, nil
}

func (m *mockRunRepo) MarkRecipientFailed(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.recipients[recipientID]
	if !ok || r.OrganizationID != orgID {
		return nil, domain.ErrRecipientNotFound
	}
	if r.Status != domain.RecipientPending && r.Status != domain.RecipientSent {
		return nil, domain.ErrRecipientNotFound
	}
	r.Status = domain.RecipientFailed
	return r, nil
}

func (m *mockRunRepo) ListActiveRecipientsByPhone(ctx context.Context, orgID int32, phone string) ([]*domain.InquiryRecipient, error) {
	recs := m.byPhone[phone]
	var out []*domain.InquiryRecipient
	for _, r := range recs {
		if r.OrganizationID == orgID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRunRepo) ListAwaitingRecipients(ctx context.Context, orgID, runID int32) ([]*domain.InquiryRecipient, error) {
	return m.ListRunRecipients(ctx, orgID, runID)
}

func (m *mockRunRepo) ListExpiredSentRecipients(ctx context.Context, orgID, runID int32, window int32) ([]*domain.InquiryRecipient, error) {
	var out []*domain.InquiryRecipient
	for _, r := range m.recipients {
		if r.RunID == runID && r.OrganizationID == orgID && r.Status == domain.RecipientSent {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRunRepo) SaveResponse(ctx context.Context, resp *domain.InquiryResponse) (*domain.InquiryResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.responses {
		if existing.RecipientID == resp.RecipientID && existing.RawMessageID == resp.RawMessageID {
			return nil, domain.ErrDuplicateResponse
		}
	}
	m.nextRespID++
	resp.ID = m.nextRespID
	m.responses[resp.ID] = resp
	return resp, nil
}

func (m *mockRunRepo) GetResponseByRecipientMessage(ctx context.Context, recipientID int32, rawMsgID string) (*domain.InquiryResponse, error) {
	for _, resp := range m.responses {
		if resp.RecipientID == recipientID && resp.RawMessageID == rawMsgID {
			return resp, nil
		}
	}
	return nil, domain.ErrResponseNotFound
}

func (m *mockRunRepo) ListRunResponses(ctx context.Context, orgID, runID int32) ([]*domain.InquiryResponse, error) {
	var out []*domain.InquiryResponse
	for _, resp := range m.responses {
		r := m.recipients[resp.RecipientID]
		if r != nil && r.RunID == runID && r.OrganizationID == orgID {
			out = append(out, resp)
		}
	}
	return out, nil
}

func (m *mockRunRepo) RunBoardRows(ctx context.Context, orgID, runID int32) ([]domain.BoardRow, error) {
	recs, err := m.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	rows := make([]domain.BoardRow, 0, len(recs))
	for _, r := range recs {
		row := domain.BoardRow{
			RecipientID: r.ID, RecipientStatus: r.Status, SupplierID: r.SupplierID,
			ContactID: r.ContactID, DisplayName: fmt.Sprintf("Prov%d", r.SupplierID), PhoneNumber: "+573001234567",
		}
		var latest *domain.InquiryResponse
		for _, resp := range m.responses {
			if resp.RecipientID == r.ID && (latest == nil || resp.ID > latest.ID) {
				latest = resp
			}
		}
		if latest != nil {
			row.Response = latest
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (m *mockRunRepo) SendFanOut(ctx context.Context, orgID, runID int32, events []domain.OutboxEventInput) (*domain.InquiryRun, error) {
	if len(events) == 0 {
		return nil, domain.ErrNoDraftedMessages
	}
	return m.TransitionRun(ctx, orgID, runID, domain.RunDraft, domain.RunSending)
}

// ---------- order repo mock ----------

type mockOrderRepo struct {
	orders map[int32]*domain.Order
	nextID int32
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{orders: map[int32]*domain.Order{}}
}

func (m *mockOrderRepo) PlaceOrderTx(ctx context.Context, in domain.PlaceOrderTxParams) (*domain.Order, error) {
	return m.create(in.Order)
}

func (m *mockOrderRepo) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	return m.create(order)
}

func (m *mockOrderRepo) create(order *domain.Order) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.RunID == order.RunID && o.SupplierID == order.SupplierID {
			return nil, domain.ErrOrderAlreadyPlaced
		}
	}
	m.nextID++
	order.ID = m.nextID
	cp := *order
	m.orders[cp.ID] = &cp
	return &cp, nil
}

func (m *mockOrderRepo) GetOrderByRunSupplier(ctx context.Context, runID, supplierID int32) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.RunID == runID && o.SupplierID == supplierID {
			return o, nil
		}
	}
	return nil, domain.ErrOrderNotFound
}

func (m *mockOrderRepo) GetOrder(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	o, ok := m.orders[orderID]
	if !ok || o.OrganizationID != orgID {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) MarkOrderConfirmSent(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	o, err := m.GetOrder(ctx, orgID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.OrderPlaced {
		return nil, domain.ErrOrderNotFound
	}
	o.Status = domain.OrderConfirmSent
	return o, nil
}

func (m *mockOrderRepo) MarkOrderSendBlocked(ctx context.Context, orgID, orderID int32, reason string) (*domain.Order, error) {
	o, err := m.GetOrder(ctx, orgID, orderID)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderSendBlocked
	o.BlockedReason = &reason
	return o, nil
}

func (m *mockOrderRepo) MarkOrderConfirmFailed(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	o, err := m.GetOrder(ctx, orgID, orderID)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderConfirmFailed
	return o, nil
}

func (m *mockOrderRepo) ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error) {
	return nil, nil
}

// ---------- audit repo mock ----------

type mockAuditRepo struct {
	mu     sync.Mutex
	events []domain.AuditEntry
}

func (m *mockAuditRepo) Record(ctx context.Context, entry domain.AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, entry)
	return nil
}

func (m *mockAuditRepo) hasAction(action string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e.Action == action {
			return true
		}
	}
	return false
}

// ---------- contact lookup mock ----------

type mockContacts struct {
	byID map[int32]*domain.ContactRef
}

func newMockContacts() *mockContacts {
	return &mockContacts{byID: map[int32]*domain.ContactRef{}}
}

func (m *mockContacts) ContactByID(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	c, ok := m.byID[contactID]
	if !ok || c.ID != contactID {
		return nil, domain.ErrContactNotFound
	}
	return c, nil
}

// ---------- kill switch mock ----------

type mockKillSwitch struct {
	on bool
}

func (m *mockKillSwitch) GetAgentKillSwitch(ctx context.Context, orgID int32) (bool, error) {
	return m.on, nil
}

// ---------- whatsapp config mock ----------

type mockConfigRepo struct {
	whatsappDomain.ConfigRepository
	config *whatsappDomain.WhatsAppConfig
	err    error
}

func (m *mockConfigRepo) GetByOrganizationID(ctx context.Context, orgID int32) (*whatsappDomain.WhatsAppConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.config == nil {
		return nil, errors.New("whatsapp_not_configured: no config")
	}
	return m.config, nil
}

// ---------- sender mock ----------

type mockSender struct {
	msgID string
	err   error
	calls int
	last  struct {
		to   string
		body string
	}
}

func (m *mockSender) SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error) {
	m.calls++
	m.last.to = to
	m.last.body = body
	if m.err != nil {
		return "", m.err
	}
	return m.msgID, nil
}

// ---------- pacer fake ----------

type fakePacer struct {
	allowed bool
}

func (f *fakePacer) Allow(orgID int32) bool { return f.allowed }

// ---------- supplier/product mocks ----------

type mockSupplierRepo struct {
	domain.SupplierRepository
	suppliers []*domain.Supplier
	err       error
}

func (m *mockSupplierRepo) ListActive(ctx context.Context, orgID int32) ([]*domain.Supplier, error) {
	return m.suppliers, m.err
}

type mockProductRepo struct {
	products []*domain.Product
	err      error
}

func (m *mockProductRepo) ListByIDs(ctx context.Context, orgID int32, ids []int32) ([]*domain.Product, error) {
	wanted := map[int32]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	out := []*domain.Product{}
	for _, p := range m.products {
		if wanted[p.ID] {
			out = append(out, p)
		}
	}
	return out, m.err
}

func (m *mockProductRepo) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Product, error) {
	return m.products, m.err
}

func (m *mockProductRepo) GetByID(ctx context.Context, orgID, id int32) (*domain.Product, error) {
	for _, p := range m.products {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, domain.ErrProductNotFound
}

func (m *mockProductRepo) Create(ctx context.Context, orgID int32, p *domain.Product) (*domain.Product, error) {
	m.products = append(m.products, p)
	return p, nil
}

func (m *mockProductRepo) Update(ctx context.Context, orgID int32, p *domain.Product) (*domain.Product, error) {
	return p, nil
}
