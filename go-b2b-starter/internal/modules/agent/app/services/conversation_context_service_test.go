package services

import (
	"context"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ctxMockRepo is a focused AgentRepository mock for conversation-context
// flows: cached context, meta, recent messages, settings, contact, and the
// conversation ownership lookup. It embeds the shared mockRepo so the full
// AgentRepository interface is satisfied; only context-relevant methods are
// overridden below.
type ctxMockRepo struct {
	*mockRepo
	meta        *domain.ConversationContextMeta
	cached      *domain.ConversationContext
	recent      []*domain.MessageRef
	upserted    []*domain.ConversationContext
	convMissing bool
}

func newCtxMockRepo() *ctxMockRepo {
	base := newMockRepo()
	base.settings = &domain.AgentSettings{
		OrganizationID: 42, Mode: domain.ModeCopilot, Tone: domain.ToneFormal,
		ConsentRequired: false, Timezone: "America/Bogota",
	}
	base.contact = &domain.ContactRef{ID: 1, OrganizationID: 42, PhoneNumber: "+573001234567", DisplayName: "Ana", ConsentStatus: domain.ConsentGranted}
	return &ctxMockRepo{
		mockRepo: base,
		meta: &domain.ConversationContextMeta{
			Channel: domain.ChannelWhatsapp, MessageCount: 2, LatestMessageID: 5,
		},
		recent: []*domain.MessageRef{
			{ID: 4, Direction: "inbound", Content: "hola, quiero cotizar", CreatedAt: time.Now()},
			{ID: 5, Direction: "outbound", Content: "claro, con gusto", CreatedAt: time.Now()},
		},
	}
}

func (m *ctxMockRepo) GetConversationRef(ctx context.Context, orgID, conversationID int32, scope conversationscope.Scope) (*domain.ConversationRef, error) {
	if m.convMissing {
		return nil, domain.ErrConversationNotFound
	}
	return &domain.ConversationRef{ID: conversationID, OrganizationID: orgID, ContactID: 1}, nil
}
func (m *ctxMockRepo) GetContactRef(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	return m.contact, nil
}
func (m *ctxMockRepo) GetConversationContext(ctx context.Context, orgID, conversationID int32) (*domain.ConversationContext, error) {
	if m.cached == nil {
		return nil, domain.ErrContextNotFound
	}
	return m.cached, nil
}
func (m *ctxMockRepo) UpsertConversationContext(ctx context.Context, orgID int32, c *domain.ConversationContext) (*domain.ConversationContext, error) {
	c.UpdatedAt = time.Now()
	m.upserted = append(m.upserted, c)
	return c, nil
}
func (m *ctxMockRepo) GetConversationContextMeta(ctx context.Context, orgID, conversationID int32, scope conversationscope.Scope) (*domain.ConversationContextMeta, error) {
	return m.meta, nil
}
func (m *ctxMockRepo) ListRecentConversationMessages(ctx context.Context, orgID, conversationID int32, limit int32, scope conversationscope.Scope) ([]*domain.MessageRef, error) {
	return m.recent, nil
}

// mockCtxLLM is an LLMClient stub that records the org-tagged context and
// PII facts attached to the generation call.
type mockCtxLLM struct {
	text string
	err  error
}

func (m *mockCtxLLM) Complete(ctx context.Context, request llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 10, Model: "gpt-test"}, nil
}
func (m *mockCtxLLM) CompleteStream(ctx context.Context, request llmdomain.CompletionRequest, callback func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	return nil, errNotSupported
}
func (m *mockCtxLLM) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return nil, 0, errNotSupported
}

var errNotSupported = context.DeadlineExceeded // reused sentinel, never matched in these tests

func newCtxService(repo domain.AgentRepository, llm llmdomain.LLMClient, billing *mockBilling) domain.ConversationContextService {
	if billing == nil {
		billing = &mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 100}}
	}
	return NewConversationContextService(repo, llm, billing, noopLogger{})
}

const contextJSON = `{"summary": "El cliente pidió una cotización.", "detected_intent": "cotización", "key_facts": ["Pidió cotización", "Cliente de Bogotá"]}`

func TestGetContext_GeneratesAndMeters(t *testing.T) {
	repo := newCtxMockRepo()
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Status != domain.ContextStatusAvailable {
		t.Fatalf("expected available, got %s", ctx.Status)
	}
	if ctx.Summary == "" || len(ctx.KeyFacts) == 0 || ctx.DetectedIntent == "" {
		t.Fatalf("expected populated AI fields, got %+v", ctx)
	}
	if ctx.SourceCursor != 5 {
		t.Fatalf("expected cursor 5, got %d", ctx.SourceCursor)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("expected one upsert, got %d", len(repo.upserted))
	}
}

func TestGetContext_CreditsExhaustedReturnsUnavailable(t *testing.T) {
	repo := newCtxMockRepo()
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, &mockBilling{
		status: &billingDomain.AiUsageStatus{CreditsMax: 10, CreditsRemaining: 0},
	})

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Status != domain.ContextStatusUnavailable {
		t.Fatalf("expected unavailable, got %s", ctx.Status)
	}
	if ctx.Summary != "" {
		t.Fatalf("expected no unmetered fallback summary, got %q", ctx.Summary)
	}
}

func TestGetContext_ConsentNotGrantedReturnsStructuralOnly(t *testing.T) {
	repo := newCtxMockRepo()
	repo.settings.ConsentRequired = true
	repo.contact.ConsentStatus = domain.ConsentNone
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Status != domain.ContextStatusStructural {
		t.Fatalf("expected structural, got %s", ctx.Status)
	}
	if !ctx.ConsentGated {
		t.Fatal("expected consent_gated=true")
	}
	if ctx.MessageCount != 2 || ctx.Channel != domain.ChannelWhatsapp {
		t.Fatalf("expected structural projection populated, got %+v", ctx)
	}
}

func TestGetContext_WithdrawnConsentMasksRead(t *testing.T) {
	repo := newCtxMockRepo()
	repo.contact.ConsentStatus = domain.ConsentWithdrawn
	repo.cached = &domain.ConversationContext{
		ConversationID: 7, Summary: "contexto viejo", SourceCursor: 5,
		GeneratedAt: time.Now(),
	}
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Status != domain.ContextStatusStructural {
		t.Fatalf("expected structural, got %s", ctx.Status)
	}
	if ctx.Summary != "" {
		t.Fatalf("expected masked read (no cached facts), got %q", ctx.Summary)
	}
}

func TestGetContext_StaleCursorRegenerates(t *testing.T) {
	repo := newCtxMockRepo()
	repo.cached = &domain.ConversationContext{
		ConversationID: 7, Summary: "viejo", SourceCursor: 3, GeneratedAt: time.Now(),
	}
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Summary == "viejo" || ctx.SourceCursor != 5 {
		t.Fatalf("expected regeneration with new cursor, got %+v", ctx)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("expected regeneration upsert, got %d", len(repo.upserted))
	}
}

func TestGetContext_FreshCacheServedWithoutLLM(t *testing.T) {
	repo := newCtxMockRepo()
	repo.cached = &domain.ConversationContext{
		ConversationID: 7, Summary: "contexto fresco", SourceCursor: 5,
		GeneratedAt: time.Now(),
	}
	llm := &mockCtxLLM{text: contextJSON}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Summary != "contexto fresco" {
		t.Fatalf("expected cached summary, got %q", ctx.Summary)
	}
	if len(repo.upserted) != 0 {
		t.Fatalf("expected no upsert on cache hit, got %d", len(repo.upserted))
	}
}

func TestGetContext_ConversationMissing(t *testing.T) {
	repo := newCtxMockRepo()
	repo.convMissing = true
	svc := newCtxService(repo, &mockCtxLLM{text: contextJSON}, nil)

	_, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != domain.ErrConversationNotFound {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestGetContext_LLMFailureReturnsUnavailable(t *testing.T) {
	repo := newCtxMockRepo()
	llm := &mockCtxLLM{err: errNotSupported}
	svc := newCtxService(repo, llm, nil)

	ctx, err := svc.GetContext(context.Background(), 42, 7, conversationscope.Scope{MemberID: "member-1", FlagEnabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Status != domain.ContextStatusUnavailable {
		t.Fatalf("expected unavailable on LLM failure, got %s", ctx.Status)
	}
}
