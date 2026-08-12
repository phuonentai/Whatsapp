package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	ticketsDomain "github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ---- fakes ----

type noopLogger struct{}

func (noopLogger) Debug(string, ...loggerdomain.Fields) {}
func (noopLogger) Info(string, ...loggerdomain.Fields)  {}
func (noopLogger) Warn(string, ...loggerdomain.Fields)  {}
func (noopLogger) Error(string, ...loggerdomain.Fields) {}
func (noopLogger) Fatal(string, ...loggerdomain.Fields) {}
func (noopLogger) WithFields(fields loggerdomain.Fields) loggerdomain.Logger {
	return noopLogger{}
}

type fakeGraphClient struct {
	exchangeCodeFn       func(code string) (string, error)
	resolveBusinessFn    func() (string, error)
	resolveWABAFn       func() ([]graphapi.WABAInfo, error)
	createSystemUserFn  func() (string, error)
	subscribeWABAFn     func() error
	registerSubsFn      func() error
	sendTestMessageFn   func() error
	submitTemplateFn    func(name, language, category, body string) (string, error)
	getTemplateStatusFn func(metaTemplateID string) (string, error)
}

func (f *fakeGraphClient) ExchangeCode(_ context.Context, code string) (string, error) {
	if f.exchangeCodeFn != nil {
		return f.exchangeCodeFn(code)
	}
	return "user-token", nil
}
func (f *fakeGraphClient) ResolveBusiness(context.Context, string) (string, error) {
	if f.resolveBusinessFn != nil {
		return f.resolveBusinessFn()
	}
	return "business-1", nil
}
func (f *fakeGraphClient) ResolveWABAAndNumbers(context.Context, string, string) ([]graphapi.WABAInfo, error) {
	if f.resolveWABAFn != nil {
		return f.resolveWABAFn()
	}
	return []graphapi.WABAInfo{{ID: "waba-1", DisplayName: "Acme", PhoneNumbers: []graphapi.WABAPhoneNumber{{ID: "phone-1", DisplayPhoneNumber: "+573001234567"}}}}, nil
}
func (f *fakeGraphClient) CreateSystemUser(context.Context, string, string, string) (string, error) {
	if f.createSystemUserFn != nil {
		return f.createSystemUserFn()
	}
	return "system-token", nil
}
func (f *fakeGraphClient) SubscribeWABA(context.Context, string, string) error {
	if f.subscribeWABAFn != nil {
		return f.subscribeWABAFn()
	}
	return nil
}
func (f *fakeGraphClient) RegisterAppSubscriptions(context.Context, string, string, string, string) error {
	if f.registerSubsFn != nil {
		return f.registerSubsFn()
	}
	return nil
}
func (f *fakeGraphClient) SendTestMessage(context.Context, string, string, string, string, string) error {
	if f.sendTestMessageFn != nil {
		return f.sendTestMessageFn()
	}
	return nil
}
func (f *fakeGraphClient) SubmitTemplate(_ context.Context, _, _, _, _, name, language, category, body string) (string, error) {
	if f.submitTemplateFn != nil {
		return f.submitTemplateFn(name, language, category, body)
	}
	return "meta-1", nil
}
func (f *fakeGraphClient) GetTemplateStatus(_ context.Context, _, _, _, _, metaTemplateID string) (string, error) {
	if f.getTemplateStatusFn != nil {
		return f.getTemplateStatusFn(metaTemplateID)
	}
	return "APPROVED", nil
}

type fakeConfigRepo struct {
	configs map[int32]*whatsappDomain.WhatsAppConfig
}

func newFakeConfigRepo() *fakeConfigRepo {
	return &fakeConfigRepo{configs: map[int32]*whatsappDomain.WhatsAppConfig{}}
}
func (f *fakeConfigRepo) GetByPhoneNumberID(_ context.Context, _ string) (*whatsappDomain.WhatsAppConfig, error) {
	return nil, errors.New("not found")
}
func (f *fakeConfigRepo) GetByOrganizationID(_ context.Context, orgID int32) (*whatsappDomain.WhatsAppConfig, error) {
	if c, ok := f.configs[orgID]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeConfigRepo) GetByVerifyToken(_ context.Context, _ string) (*whatsappDomain.WhatsAppConfig, error) {
	return nil, errors.New("not found")
}
func (f *fakeConfigRepo) GetByWABAID(_ context.Context, _ string) (*whatsappDomain.WhatsAppConfig, error) {
	return nil, errors.New("not found")
}
func (f *fakeConfigRepo) clone(c *whatsappDomain.WhatsAppConfig) *whatsappDomain.WhatsAppConfig {
	cp := *c
	cp.Metadata = map[string]interface{}{}
	for k, v := range c.Metadata {
		cp.Metadata[k] = v
	}
	return &cp
}
func (f *fakeConfigRepo) Create(_ context.Context, config *whatsappDomain.WhatsAppConfig) (*whatsappDomain.WhatsAppConfig, error) {
	config.ID = int32(len(f.configs) + 1)
	stored := f.clone(config)
	f.configs[config.OrganizationID] = stored
	return f.clone(stored), nil
}
func (f *fakeConfigRepo) Update(_ context.Context, config *whatsappDomain.WhatsAppConfig) (*whatsappDomain.WhatsAppConfig, error) {
	stored := f.clone(config)
	f.configs[config.OrganizationID] = stored
	return f.clone(stored), nil
}

type fakeSignupRepo struct {
	flows map[int32]*whatsappDomain.SignupFlow
}

func newFakeSignupRepo() *fakeSignupRepo {
	return &fakeSignupRepo{flows: map[int32]*whatsappDomain.SignupFlow{}}
}
func (f *fakeSignupRepo) Upsert(_ context.Context, flow *whatsappDomain.SignupFlow) (*whatsappDomain.SignupFlow, error) {
	flow.ID = int64(len(f.flows) + 1)
	if flow.Metadata == nil {
		flow.Metadata = map[string]interface{}{}
	}
	f.flows[flow.OrganizationID] = flow
	return flow, nil
}
func (f *fakeSignupRepo) GetByOrganization(_ context.Context, orgID int32) (*whatsappDomain.SignupFlow, error) {
	if fl, ok := f.flows[orgID]; ok {
		return fl, nil
	}
	return nil, fmt.Errorf("%w: no flow", whatsappDomain.ErrSignupNotFound)
}
func (f *fakeSignupRepo) UpdateStatus(_ context.Context, orgID int32, status whatsappDomain.SignupStatus, step, errorCode string, retryCount int, metadata map[string]interface{}) (*whatsappDomain.SignupFlow, error) {
	fl := f.flows[orgID]
	if fl == nil {
		return nil, errors.New("signup flow not found")
	}
	fl.Status = status
	fl.Step = step
	fl.ErrorCode = errorCode
	fl.RetryCount = retryCount
	fl.Metadata = metadata
	return fl, nil
}

type fakeTicketService struct {
	ticketsServices.TicketService
	createFn func(ctx context.Context, orgID int32, req *ticketsServices.CreateTicketRequest, actor string) (*ticketsDomain.Ticket, error)
	created  []*ticketsServices.CreateTicketRequest
}

func (f *fakeTicketService) Create(ctx context.Context, orgID int32, req *ticketsServices.CreateTicketRequest, actor string) (*ticketsDomain.Ticket, error) {
	f.created = append(f.created, req)
	if f.createFn != nil {
		return f.createFn(ctx, orgID, req, actor)
	}
	return &ticketsDomain.Ticket{ID: 42, OrganizationID: orgID, Title: req.Title, Priority: req.Priority}, nil
}

// ---- helpers ----

func newTestSignupService(graph *fakeGraphClient, cfgRepo *fakeConfigRepo, signupRepo *fakeSignupRepo, tickets *fakeTicketService) SignupService {
	svc := &signupService{
		configRepo:    cfgRepo,
		signupRepo:    signupRepo,
		graphClient:   graph,
		metaCfg:       graphapi.MetaConfig{AppID: "app-1", ConfigID: "cfg-1", RedirectURI: ""},
		appID:         "app-1",
		callbackURL:   "https://platform.example.com/api/v1/webhooks/whatsapp",
		ticketService: tickets,
		logger:        noopLogger{},
		backoff:       func(int) time.Duration { return time.Millisecond },
	}
	return svc
}

// ---- tests ----

func TestExchange_Success_ConnectedWithGeneratedSecrets(t *testing.T) {
	cfgRepo := newFakeConfigRepo()
	svc := newTestSignupService(&fakeGraphClient{}, cfgRepo, newFakeSignupRepo(), &fakeTicketService{})

	result, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != whatsappDomain.SignupStatusConnected {
		t.Fatalf("expected connected, got %s", result.Status)
	}
	if result.Config == nil {
		t.Fatal("expected config in result")
	}
	if !strings.HasPrefix(result.Config.WebhookSecret, "whsec_") {
		t.Fatalf("expected webhook secret with prefix, got %q", result.Config.WebhookSecret)
	}
	for _, v := range []string{result.Config.WebhookSecret, result.Config.VerifyToken, result.Config.AccessToken} {
		if !strings.Contains(v, "****") {
			t.Fatalf("expected masked secret in result, got %q", v)
		}
	}
	stored := cfgRepo.configs[7]
	if stored == nil {
		t.Fatal("expected config persisted")
	}
	if len(stored.WebhookSecret) <= 40 || !strings.HasPrefix(stored.WebhookSecret, "whsec_") {
		t.Fatalf("expected raw generated webhook secret stored, got %q", stored.WebhookSecret)
	}
	if len(stored.VerifyToken) <= 40 || !strings.HasPrefix(stored.VerifyToken, "verify_") {
		t.Fatalf("expected raw generated verify token stored, got %q", stored.VerifyToken)
	}
	if stored.AccessToken != "system-token" {
		t.Fatalf("expected system user token stored, got %q", stored.AccessToken)
	}
	if result.Config.PhoneNumberID != "phone-1" || result.Config.WABAID != "waba-1" {
		t.Fatalf("unexpected config values: %+v", result.Config)
	}
	if result.Config.Metadata["coexistence"] != false {
		t.Fatalf("expected coexistence=false (no certificate), got %v", result.Config.Metadata["coexistence"])
	}
	if !result.Config.IsActive {
		t.Fatal("expected config active")
	}
}

func TestExchange_Success_CoexistenceDetected(t *testing.T) {
	graph := &fakeGraphClient{resolveWABAFn: func() ([]graphapi.WABAInfo, error) {
		return []graphapi.WABAInfo{{ID: "waba-1", PhoneNumbers: []graphapi.WABAPhoneNumber{{ID: "phone-1", DisplayPhoneNumber: "+573001234567", Certificate: "cert-abc"}}}}, nil
	}}
	svc := newTestSignupService(graph, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})

	result, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Config.Metadata["coexistence"] != true {
		t.Fatalf("expected coexistence=true with certificate, got %v", result.Config.Metadata["coexistence"])
	}
}

func TestExchange_MissingCode(t *testing.T) {
	svc := newTestSignupService(&fakeGraphClient{}, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})

	_, err := svc.Exchange(context.Background(), 7, "", "member-1")
	if !errors.Is(err, whatsappDomain.ErrSignupCodeRequired) {
		t.Fatalf("expected ErrSignupCodeRequired, got %v", err)
	}
}

func TestExchange_AlreadyConnected(t *testing.T) {
	signupRepo := newFakeSignupRepo()
	signupRepo.flows[7] = &whatsappDomain.SignupFlow{OrganizationID: 7, Status: whatsappDomain.SignupStatusConnected}
	svc := newTestSignupService(&fakeGraphClient{}, newFakeConfigRepo(), signupRepo, &fakeTicketService{})

	_, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	if !errors.Is(err, whatsappDomain.ErrSignupAlreadyConnected) {
		t.Fatalf("expected ErrSignupAlreadyConnected, got %v", err)
	}
}

func TestExchange_InProgress(t *testing.T) {
	signupRepo := newFakeSignupRepo()
	signupRepo.flows[7] = &whatsappDomain.SignupFlow{OrganizationID: 7, Status: whatsappDomain.SignupStatusRegistering}
	svc := newTestSignupService(&fakeGraphClient{}, newFakeConfigRepo(), signupRepo, &fakeTicketService{})

	_, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	if !errors.Is(err, whatsappDomain.ErrSignupInProgress) {
		t.Fatalf("expected ErrSignupInProgress, got %v", err)
	}
}

func TestExchange_TokenExchangeFailure_CreatesTicket(t *testing.T) {
	graph := &fakeGraphClient{exchangeCodeFn: func(code string) (string, error) {
		return "", fmt.Errorf("oauth rejected")
	}}
	signupRepo := newFakeSignupRepo()
	tickets := &fakeTicketService{}
	svc := newTestSignupService(graph, newFakeConfigRepo(), signupRepo, tickets)

	_, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	var failedErr *whatsappDomain.SignupFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected SignupFailedError, got %v", err)
	}
	if failedErr.Code != "token_exchange_failed" {
		t.Fatalf("expected token_exchange_failed, got %s", failedErr.Code)
	}

	flow := signupRepo.flows[7]
	if flow.Status != whatsappDomain.SignupStatusFailed {
		t.Fatalf("expected flow failed, got %s", flow.Status)
	}
	if flow.RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", flow.RetryCount)
	}
	if flow.ErrorCode != "token_exchange_failed" {
		t.Fatalf("expected error_code recorded, got %s", flow.ErrorCode)
	}
	if len(tickets.created) != 1 {
		t.Fatalf("expected 1 HITL ticket, got %d", len(tickets.created))
	}
	if tickets.created[0].Priority != ticketsDomain.PriorityHigh {
		t.Fatalf("expected high priority ticket, got %s", tickets.created[0].Priority)
	}
	if flow.Metadata["ticket_id"] != int32(42) {
		t.Fatalf("expected ticket_id recorded, got %v", flow.Metadata["ticket_id"])
	}
}

func TestExchange_TicketServiceFailure_FallsBackToLogging(t *testing.T) {
	graph := &fakeGraphClient{exchangeCodeFn: func(code string) (string, error) {
		return "", fmt.Errorf("oauth rejected")
	}}
	signupRepo := newFakeSignupRepo()
	tickets := &fakeTicketService{createFn: func(ctx context.Context, orgID int32, req *ticketsServices.CreateTicketRequest, actor string) (*ticketsDomain.Ticket, error) {
		return nil, fmt.Errorf("tickets module disabled")
	}}
	svc := newTestSignupService(graph, newFakeConfigRepo(), signupRepo, tickets)

	_, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	var failedErr *whatsappDomain.SignupFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected SignupFailedError even when tickets disabled, got %v", err)
	}
	if failedErr.Code != "token_exchange_failed" {
		t.Fatalf("expected token_exchange_failed, got %s", failedErr.Code)
	}
	flow := signupRepo.flows[7]
	if flow.Metadata["ticket_id"] != nil {
		t.Fatalf("expected no ticket_id when tickets module disabled, got %v", flow.Metadata["ticket_id"])
	}
}

func TestExchange_RetriesResolveBusiness(t *testing.T) {
	calls := 0
	graph := &fakeGraphClient{resolveBusinessFn: func() (string, error) {
		calls++
		if calls < 3 {
			return "", fmt.Errorf("transient failure %d", calls)
		}
		return "business-1", nil
	}}
	svc := newTestSignupService(graph, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})

	result, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if result.Status != whatsappDomain.SignupStatusConnected {
		t.Fatalf("expected connected, got %s", result.Status)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
}

func TestExchange_ResolveWABAFailure_ExhaustsRetries(t *testing.T) {
	graph := &fakeGraphClient{resolveWABAFn: func() ([]graphapi.WABAInfo, error) {
		return nil, fmt.Errorf("graph down")
	}}
	svc := newTestSignupService(graph, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})

	_, err := svc.Exchange(context.Background(), 7, "auth-code", "member-1")
	var failedErr *whatsappDomain.SignupFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected SignupFailedError, got %v", err)
	}
	if failedErr.Code != "waba_resolution_failed" {
		t.Fatalf("expected waba_resolution_failed, got %s", failedErr.Code)
	}
}

func TestMetaConfig_ReturnsBootstrap(t *testing.T) {
	svc := newTestSignupService(&fakeGraphClient{}, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})
	meta, err := svc.MetaConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.AppID != "app-1" || meta.ConfigID != "cfg-1" {
		t.Fatalf("unexpected meta config: %+v", meta)
	}
}

func TestStatus_NotFound(t *testing.T) {
	svc := newTestSignupService(&fakeGraphClient{}, newFakeConfigRepo(), newFakeSignupRepo(), &fakeTicketService{})
	_, err := svc.Status(context.Background(), 7)
	if !errors.Is(err, whatsappDomain.ErrSignupNotFound) {
		t.Fatalf("expected ErrSignupNotFound, got %v", err)
	}
}
