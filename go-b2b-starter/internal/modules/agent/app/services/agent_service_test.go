package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ---------- mocks ----------

type mockRepo struct {
	settings   *domain.AgentSettings
	flow       *domain.ConversationFlow
	contact    *domain.ContactRef
	conv       *domain.ConversationRef
	suggestions []*domain.Suggestion
	actions    []*domain.AgentAction
	sentToday  int64
	consentCalls []domain.ConsentStatus
	superseded bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		settings:    nil,
		contact:     &domain.ContactRef{ID: 1, OrganizationID: 42, PhoneNumber: "+573001234567", DisplayName: "Ana", ConsentStatus: domain.ConsentGranted},
		conv:        &domain.ConversationRef{ID: 7, OrganizationID: 42, ContactID: 1},
		flow:        &domain.ConversationFlow{ID: 3, OrganizationID: 42, ConversationID: 7, ContactID: 1, Status: domain.FlowStatusRunning},
		suggestions: []*domain.Suggestion{},
		actions:     []*domain.AgentAction{},
	}
}

func (m *mockRepo) CreateFlow(ctx context.Context, orgID, conversationID, contactID int32) (*domain.ConversationFlow, error) {
	return m.flow, nil
}
func (m *mockRepo) GetFlow(ctx context.Context, orgID, flowID int32) (*domain.ConversationFlow, error) {
	if m.flow == nil {
		return nil, domain.ErrFlowNotFound
	}
	return m.flow, nil
}
func (m *mockRepo) GetActiveFlowByConversation(ctx context.Context, orgID, conversationID int32) (*domain.ConversationFlow, error) {
	if m.flow == nil {
		return nil, domain.ErrFlowNotFound
	}
	return m.flow, nil
}
func (m *mockRepo) UpdateFlowStatus(ctx context.Context, orgID, flowID int32, status domain.FlowStatus) (*domain.ConversationFlow, error) {
	m.flow.Status = status
	return m.flow, nil
}
func (m *mockRepo) GetSettings(ctx context.Context, orgID int32) (*domain.AgentSettings, error) {
	if m.settings == nil {
		return nil, domain.ErrSettingsNotFound
	}
	return m.settings, nil
}
func (m *mockRepo) UpsertSettings(ctx context.Context, s *domain.AgentSettings) (*domain.AgentSettings, error) {
	m.settings = s
	return s, nil
}
func (m *mockRepo) InsertSuggestion(ctx context.Context, s *domain.Suggestion) (*domain.Suggestion, error) {
	s.ID = int32(len(m.suggestions) + 1)
	s.Status = domain.SuggestionPending
	m.suggestions = append(m.suggestions, s)
	return s, nil
}
func (m *mockRepo) ListSuggestions(ctx context.Context, orgID int32, status domain.SuggestionStatus, limit, offset int32) ([]*domain.Suggestion, error) {
	var out []*domain.Suggestion
	for _, s := range m.suggestions {
		if s.Status == status {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockRepo) GetSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error) {
	for _, s := range m.suggestions {
		if s.ID == suggestionID {
			return s, nil
		}
	}
	return nil, domain.ErrSuggestionNotFound
}
func (m *mockRepo) ApproveSuggestion(ctx context.Context, orgID, suggestionID int32, approvedByMember string) (*domain.Suggestion, error) {
	for _, s := range m.suggestions {
		if s.ID == suggestionID {
			s.Status = domain.SuggestionApproved
			s.ApprovedByMemberID = approvedByMember
			return s, nil
		}
	}
	return nil, domain.ErrSuggestionNotFound
}
func (m *mockRepo) RejectSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error) {
	for _, s := range m.suggestions {
		if s.ID == suggestionID {
			s.Status = domain.SuggestionRejected
			return s, nil
		}
	}
	return nil, domain.ErrSuggestionNotFound
}
func (m *mockRepo) SupersedePendingReplies(ctx context.Context, orgID, conversationID int32) error {
	m.superseded = true
	return nil
}
func (m *mockRepo) GetPendingSuggestionByMessage(ctx context.Context, orgID int32, whatsappMessageID string) (*domain.Suggestion, error) {
	return nil, domain.ErrSuggestionNotFound
}
func (m *mockRepo) InsertAction(ctx context.Context, a *domain.AgentAction) (*domain.AgentAction, error) {
	a.ID = int32(len(m.actions) + 1)
	m.actions = append(m.actions, a)
	return a, nil
}
func (m *mockRepo) CountMessagesSentToday(ctx context.Context, orgID int32, since time.Time) (int64, error) {
	return m.sentToday, nil
}
func (m *mockRepo) ResolveContact(ctx context.Context, orgID int32, phoneNumber, displayName string, lastMessageAt time.Time) (*domain.ContactRef, error) {
	return m.contact, nil
}
func (m *mockRepo) GetContactRef(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	if m.contact == nil {
		return nil, domain.ErrContactNotFound
	}
	return m.contact, nil
}
func (m *mockRepo) ResolveConversation(ctx context.Context, orgID, contactID int32, lastMessageAt time.Time) (*domain.ConversationRef, error) {
	return m.conv, nil
}
func (m *mockRepo) GetConversationRef(ctx context.Context, orgID, conversationID int32) (*domain.ConversationRef, error) {
	return m.conv, nil
}
func (m *mockRepo) ListConversationsByContact(ctx context.Context, orgID, contactID int32) ([]*domain.ConversationRef, error) {
	return []*domain.ConversationRef{m.conv}, nil
}
func (m *mockRepo) ListMessagesByConversation(ctx context.Context, orgID, conversationID int32, limit, offset int32) ([]*domain.MessageRef, error) {
	return []*domain.MessageRef{
		{ID: 1, OrganizationID: orgID, ConversationID: conversationID, Direction: "inbound", Content: "hola", Status: "received", CreatedAt: time.Now()},
	}, nil
}
func (m *mockRepo) UpdateContactConsent(ctx context.Context, orgID, contactID int32, status domain.ConsentStatus, consentedAt *time.Time) (*domain.ContactRef, error) {
	m.consentCalls = append(m.consentCalls, status)
	m.contact.ConsentStatus = status
	return m.contact, nil
}
func (m *mockRepo) AnonymizeContact(ctx context.Context, orgID, contactID int32) error {
	m.contact.ConsentStatus = domain.ConsentWithdrawn
	return nil
}

type mockOutbound struct {
	sent [][]any // [orgID, convID, content]
	err  error
}

func (m *mockOutbound) SendMessage(ctx context.Context, orgID, convID int32, content string) (*crmDomain.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.sent = append(m.sent, []any{orgID, convID, content})
	return &crmDomain.Message{ID: 9}, nil
}

type mockGuardrails struct {
	decisions []domain.GuardrailDecision
	err       error
}

func (m *mockGuardrails) Evaluate(ctx context.Context, orgID int32, input domain.GuardrailInput) (domain.GuardrailDecision, error) {
	if m.err != nil {
		return domain.GuardrailDecision{}, m.err
	}
	if len(m.decisions) > 0 {
		dec := m.decisions[0]
		m.decisions = m.decisions[1:]
		return dec, nil
	}
	return domain.GuardrailDecision{Allowed: true}, nil
}

type mockLLM struct {
	text string
	err  error
}

func (m *mockLLM) Complete(ctx context.Context, request llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 10, Model: "gpt-test"}, nil
}
func (m *mockLLM) CompleteStream(ctx context.Context, request llmdomain.CompletionRequest, callback func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	return nil, errors.New("not used")
}
func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return nil, 0, errors.New("not used")
}

type mockBilling struct {
	status *billingDomain.AiUsageStatus
	err    error
}

func (m *mockBilling) ProcessWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error {
	return nil
}
func (m *mockBilling) GetBillingStatus(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) CheckQuotaAvailability(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) ConsumeInvoiceQuota(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) VerifyAndConsumeQuota(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) SyncSubscriptionFromPolar(ctx context.Context, organizationID int32) error {
	return nil
}
func (m *mockBilling) VerifyPaymentFromCheckout(ctx context.Context, sessionID string) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) RefreshSubscriptionStatus(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) CreateMPCheckout(ctx context.Context, planID string) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) VerifyMPPayment(ctx context.Context, paymentID string) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) ProcessMPWebhookEvent(ctx context.Context, rawPayload json.RawMessage) error {
	return nil
}
func (m *mockBilling) CancelMPSubscription(ctx context.Context, subscriptionID string) (*billingDomain.BillingStatus, error) {
	return &billingDomain.BillingStatus{}, nil
}
func (m *mockBilling) GetAiUsageStatus(ctx context.Context, organizationID int32) (*billingDomain.AiUsageStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.status, nil
}

func newTestService(repo *mockRepo, g domain.GuardrailService, llm llmdomain.LLMClient, bill billingDomainMock, out *mockOutbound) AgentService {
	return NewAgentService(repo, g, llm, bill, out, noopLogger{})
}

// billingDomainMock avoids an import cycle in the mock struct name.
type billingDomainMock = billingInterface

type billingInterface interface {
	ProcessWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error
	GetBillingStatus(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error)
	CheckQuotaAvailability(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error)
	ConsumeInvoiceQuota(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error)
	VerifyAndConsumeQuota(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error)
	SyncSubscriptionFromPolar(ctx context.Context, organizationID int32) error
	VerifyPaymentFromCheckout(ctx context.Context, sessionID string) (*billingDomain.BillingStatus, error)
	RefreshSubscriptionStatus(ctx context.Context, organizationID int32) (*billingDomain.BillingStatus, error)
	CreateMPCheckout(ctx context.Context, planID string) (*billingDomain.BillingStatus, error)
	VerifyMPPayment(ctx context.Context, paymentID string) (*billingDomain.BillingStatus, error)
	ProcessMPWebhookEvent(ctx context.Context, rawPayload json.RawMessage) error
	CancelMPSubscription(ctx context.Context, subscriptionID string) (*billingDomain.BillingStatus, error)
	GetAiUsageStatus(ctx context.Context, organizationID int32) (*billingDomain.AiUsageStatus, error)
}

func inEvent(content string) *whatsappEvents.MessageReceived {
	return &whatsappEvents.MessageReceived{
		OrganizationID: 42,
		MessageSID:     "wamid-1",
		From:           "+573001234567",
		Content:        content,
		WhatsAppTimestamp: time.Now(),
	}
}

// ---------- copilot flow ----------

func TestCopilotCreatesPendingSuggestion(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false}
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{text: `{"intent":"compra","sentiment":"positivo","suggested_reply":"Claro, con gusto."}`}, &mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}}, &mockOutbound{})

	if err := svc.HandleMessageReceived(context.Background(), inEvent("¿Tienen el plan pro?")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(repo.suggestions))
	}
	s := repo.suggestions[0]
	if s.Status != domain.SuggestionPending || s.Source != domain.SuggestionSourceCopilot {
		t.Fatalf("unexpected suggestion state: %+v", s)
	}
	if s.Body != "Claro, con gusto." {
		t.Fatalf("unexpected draft: %q", s.Body)
	}
}

// ---------- approval flow (6.1) ----------

func TestApproveSuggestionSendsOnlyOnAllow(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false}
	created, _ := repo.InsertSuggestion(context.Background(), &domain.Suggestion{
		OrganizationID: 42, ConversationID: 7, ContactID: 1, FlowID: &[]int32{3}[0],
		Type: domain.SuggestionReply, Body: "Borrador", Source: domain.SuggestionSourceCopilot,
	})
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: true}}}, &mockLLM{}, &mockBilling{}, out)

	approved, err := svc.ApproveSuggestion(context.Background(), 42, created.ID, "", "stytch_member_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved.Status != domain.SuggestionApproved {
		t.Fatalf("expected approved, got %s", approved.Status)
	}
	if len(out.sent) != 1 {
		t.Fatalf("expected 1 send, got %d", len(out.sent))
	}
	if out.sent[0][2] != "Borrador" {
		t.Fatalf("expected original draft sent, got %v", out.sent[0][2])
	}
	if len(repo.actions) != 1 || repo.actions[0].Decision != domain.DecisionAllow {
		t.Fatalf("expected one allow audit row, got %+v", repo.actions)
	}
	if repo.actions[0].ApprovedByMemberID != "stytch_member_1" {
		t.Fatalf("approval anchor missing: %+v", repo.actions[0])
	}
}

func TestApproveSuggestionEditedBodySent(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false}
	created, _ := repo.InsertSuggestion(context.Background(), &domain.Suggestion{
		OrganizationID: 42, ConversationID: 7, ContactID: 1, FlowID: &[]int32{3}[0],
		Type: domain.SuggestionReply, Body: "Borrador original", Source: domain.SuggestionSourceCopilot,
	})
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: true}}}, &mockLLM{}, &mockBilling{}, out)

	approved, err := svc.ApproveSuggestion(context.Background(), 42, created.ID, "Versión editada", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved.Status != domain.SuggestionApproved {
		t.Fatalf("expected approved")
	}
	if out.sent[0][2] != "Versión editada" {
		t.Fatalf("expected edited body sent, got %v", out.sent[0][2])
	}
}

func TestApproveSuggestionDeniedSendsNothingAndAudits(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false}
	created, _ := repo.InsertSuggestion(context.Background(), &domain.Suggestion{
		OrganizationID: 42, ConversationID: 7, ContactID: 1, FlowID: &[]int32{3}[0],
		Type: domain.SuggestionReply, Body: "10% de descuento", Source: domain.SuggestionSourceCopilot,
	})
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: false, Reasons: []string{"discount_exceeds_cap"}}}}, &mockLLM{}, &mockBilling{}, out)

	_, err := svc.ApproveSuggestion(context.Background(), 42, created.ID, "", "m1")
	var denial *DenialError
	if !errors.As(err, &denial) {
		t.Fatalf("expected DenialError, got %v", err)
	}
	if len(out.sent) != 0 {
		t.Fatalf("denied action must not send, got %d sends", len(out.sent))
	}
	if len(repo.actions) != 1 || repo.actions[0].Decision != domain.DecisionDeny {
		t.Fatalf("expected one deny audit row, got %+v", repo.actions)
	}
	if repo.suggestions[0].Status != domain.SuggestionRejected {
		t.Fatalf("denied suggestion should be rejected, got %s", repo.suggestions[0].Status)
	}
}

// ---------- autopilot path (7.1) ----------

func TestAutopilotSendsWhenAllowed(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{
		OrganizationID: 42, Mode: domain.ModeAutopilot, ConsentRequired: false,
		AutopilotStart: "00:00", AutopilotEnd: "23:59", Timezone: "UTC",
	}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: true}}}, &mockLLM{text: `{"suggested_reply":"Respuesta autónoma"}`}, &mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("Hola")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.sent) != 1 {
		t.Fatalf("autopilot should send, got %d", len(out.sent))
	}
	if len(repo.suggestions) != 0 {
		t.Fatalf("no fallback suggestion expected, got %d", len(repo.suggestions))
	}
	if repo.flow.Status != domain.FlowStatusSucceeded {
		t.Fatalf("flow should be succeeded, got %s", repo.flow.Status)
	}
}

func TestAutopilotDenialFallsBackToDraft(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{
		OrganizationID: 42, Mode: domain.ModeAutopilot, ConsentRequired: false,
		AutopilotStart: "00:00", AutopilotEnd: "23:59", Timezone: "UTC",
	}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: false, Reasons: []string{"daily_limit_reached"}}}}, &mockLLM{text: `{"suggested_reply":"Draft"}`}, &mockBilling{}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("Hola")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.sent) != 0 {
		t.Fatalf("denied autopilot must not send")
	}
	if len(repo.suggestions) != 1 || repo.suggestions[0].Source != domain.SuggestionSourceAutopilotFallback {
		t.Fatalf("expected fallback draft, got %+v", repo.suggestions)
	}
	if len(repo.actions) != 1 || repo.actions[0].Decision != domain.DecisionDeny {
		t.Fatalf("expected deny audit row, got %+v", repo.actions)
	}
}

func TestAutopilotEscalationMatchEscalates(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{
		OrganizationID: 42, Mode: domain.ModeAutopilot, ConsentRequired: false,
		AutopilotStart: "00:00", AutopilotEnd: "23:59", Timezone: "UTC",
	}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{decisions: []domain.GuardrailDecision{{Allowed: false, Reasons: []string{"escalation_match"}}}}, &mockLLM{text: `{"suggested_reply":"Draft"}`}, &mockBilling{}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("abogado")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.sent) != 0 {
		t.Fatalf("escalation must not send")
	}
	if len(repo.suggestions) != 1 || repo.suggestions[0].Type != domain.SuggestionEscalation {
		t.Fatalf("expected escalation suggestion, got %+v", repo.suggestions)
	}
	if repo.flow.Status != domain.FlowStatusAwaitingHuman {
		t.Fatalf("flow should await human, got %s", repo.flow.Status)
	}
}

func TestCreditsExhaustedSkipsAnalysisAndEscalates(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{}, &mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("Hola")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.sent) != 0 || len(repo.suggestions) != 1 || repo.suggestions[0].Type != domain.SuggestionEscalation {
		t.Fatalf("expected escalation, got sends=%d suggestions=%+v", len(out.sent), repo.suggestions)
	}
}

func TestKillSwitchCancelsFlow(t *testing.T) {
	repo := newMockRepo()
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: false, KillSwitch: true}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{}, &mockBilling{}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("Hola")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.flow.Status != domain.FlowStatusCancelled {
		t.Fatalf("flow should be cancelled, got %s", repo.flow.Status)
	}
	if len(out.sent) != 0 {
		t.Fatalf("kill switch must block sends")
	}
}

// ---------- consent (8.1/8.2) ----------

func TestConsentNoneSendsTemplateAndRequestsConsent(t *testing.T) {
	repo := newMockRepo()
	repo.contact.ConsentStatus = domain.ConsentNone
	repo.settings = &domain.AgentSettings{
		OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: true,
		ConsentTemplate: "¿Autorizas el tratamiento de tus datos? (sí/acepto)",
	}
	out := &mockOutbound{}
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{}, &mockBilling{}, out)

	if err := svc.HandleMessageReceived(context.Background(), inEvent("Hola")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.sent) != 1 || out.sent[0][2] != "¿Autorizas el tratamiento de tus datos? (sí/acepto)" {
		t.Fatalf("expected exactly one consent template send, got %+v", out.sent)
	}
	if len(repo.consentCalls) != 1 || repo.consentCalls[0] != domain.ConsentRequested {
		t.Fatalf("expected consent requested, got %v", repo.consentCalls)
	}
	if len(repo.suggestions) != 0 {
		t.Fatalf("no other autonomous reply expected, got %+v", repo.suggestions)
	}
}

func TestAffirmativeReplyGrantsConsent(t *testing.T) {
	repo := newMockRepo()
	repo.contact.ConsentStatus = domain.ConsentRequested
	repo.settings = &domain.AgentSettings{OrganizationID: 42, Mode: domain.ModeCopilot, ConsentRequired: true}
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{text: `{"suggested_reply":"Draft"}`}, &mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}}, &mockOutbound{})

	if err := svc.HandleMessageReceived(context.Background(), inEvent("sí, acepto")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.consentCalls) != 1 || repo.consentCalls[0] != domain.ConsentGranted {
		t.Fatalf("expected consent granted, got %v", repo.consentCalls)
	}
	if repo.contact.ConsentStatus != domain.ConsentGranted {
		t.Fatalf("contact should be granted, got %s", repo.contact.ConsentStatus)
	}
}

// ---------- settings (10.1) ----------

func TestUpdateSettingsValidation(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo, &mockGuardrails{}, &mockLLM{}, &mockBilling{}, &mockOutbound{})

	_, err := svc.UpdateSettings(context.Background(), 42, &domain.AgentSettings{Mode: "rogue"})
	if err == nil {
		t.Fatal("invalid mode must be rejected")
	}
	updated, err := svc.UpdateSettings(context.Background(), 42, &domain.AgentSettings{
		Mode: domain.ModeAutopilot, Tone: domain.ToneCasual, MaxDailyMessages: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Mode != domain.ModeAutopilot || updated.Tone != domain.ToneCasual {
		t.Fatalf("unexpected settings: %+v", updated)
	}
	if updated.Timezone != "America/Bogota" {
		t.Fatalf("default timezone expected, got %q", updated.Timezone)
	}
	if updated.Guardrails.Never == nil || updated.Guardrails.Never.MaxDiscountPercent == nil {
		t.Fatalf("default guardrails expected, got %+v", updated.Guardrails)
	}
}

// ---------- compliance export/forget (9.1/9.2) ----------

func TestExportContactMasksWhenWithdrawn(t *testing.T) {
	repo := newMockRepo()
	repo.contact.ConsentStatus = domain.ConsentWithdrawn
	svc := NewComplianceService(repo)

	bundle, err := svc.ExportContact(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bundle.Contact.PhoneNumber != "[ELIMINADO]" || bundle.Contact.NumeroDocumento != "" {
		t.Fatalf("withdrawn export must mask PII, got %+v", bundle.Contact)
	}
	if len(bundle.Conversations) != 1 || len(bundle.Conversations[0].Messages) != 1 {
		t.Fatalf("export bundle incomplete: %+v", bundle)
	}
}

func TestForgetContactAnonymizesAndIsIdempotent(t *testing.T) {
	repo := newMockRepo()
	svc := NewComplianceService(repo)

	if err := svc.ForgetContact(context.Background(), 42, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.contact.ConsentStatus != domain.ConsentWithdrawn {
		t.Fatalf("contact should be withdrawn after forget")
	}
	if err := svc.ForgetContact(context.Background(), 42, 1); err != nil {
		t.Fatalf("forget must be idempotent, got %v", err)
	}
}
