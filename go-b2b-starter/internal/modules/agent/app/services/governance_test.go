package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/infra/guardrails"
	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
)

// mockGovernanceRepo embeds the AgentRepository interface and overrides only
// the methods GovernManualSend needs.
type mockGovernanceRepo struct {
	domain.AgentRepository
	settings    *domain.AgentSettings
	sentToday   int64
	audited     []*domain.AgentAction
	contact     *domain.ContactRef
	conversation *domain.ConversationRef
}

func (m *mockGovernanceRepo) GetConversationRef(ctx context.Context, orgID, conversationID int32, scope conversationscope.Scope) (*domain.ConversationRef, error) {
	return m.conversation, nil
}

func (m *mockGovernanceRepo) GetContactRef(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	return m.contact, nil
}

func (m *mockGovernanceRepo) GetSettings(ctx context.Context, orgID int32) (*domain.AgentSettings, error) {
	return m.settings, nil
}

func (m *mockGovernanceRepo) CountMessagesSentToday(ctx context.Context, orgID int32, since time.Time) (int64, error) {
	return m.sentToday, nil
}

func (m *mockGovernanceRepo) InsertAction(ctx context.Context, record *domain.AgentAction) (*domain.AgentAction, error) {
	m.audited = append(m.audited, record)
	return record, nil
}

// fakeOutbound records send attempts and never touches the network.
type fakeOutbound struct {
	crmServices.OutboundService
	sent []string
}

func (f *fakeOutbound) SendMessage(ctx context.Context, orgID, convID int32, content string) (*crmdomain.Message, error) {
	f.sent = append(f.sent, content)
	return &crmdomain.Message{
		ID:             1,
		OrganizationID: orgID,
		ConversationID: convID,
		Direction:      crmdomain.MessageDirectionOutbound,
		Content:        content,
	}, nil
}

func forbiddenTermSettings(terms []string) *domain.AgentSettings {
	never := &domain.NeverRules{ForbiddenTerms: terms}
	return &domain.AgentSettings{
		Guardrails: domain.Guardrails{Never: never},
	}
}

func newGovernanceHarness(settings *domain.AgentSettings) (*mockGovernanceRepo, *fakeOutbound, AgentService) {
	repo := &mockGovernanceRepo{
		settings: settings,
		contact: &domain.ContactRef{
			PhoneNumber:   "+573001234567",
			DisplayName:   "Cliente Prueba",
			ConsentStatus: domain.ConsentGranted,
		},
		conversation: &domain.ConversationRef{ID: 42, ContactID: 7},
	}
	outbound := &fakeOutbound{}
	svc := NewAgentService(
		repo,
		guardrails.NewGuardrailService(),
		nil, // llm client unused
		nil, // billing unused
		outbound,
		noopLogger{},
	)
	return repo, outbound, svc
}
func TestGovernManualSendDeniesForbiddenTermAndAudits(t *testing.T) {
	repo, outbound, svc := newGovernanceHarness(forbiddenTermSettings([]string{"precio especial"}))

	reasons, msg, err := svc.GovernManualSend(context.Background(), 1, 42, "Te ofrezco un precio especial hoy", "member-1")
	require.NoError(t, err)
	require.Nil(t, msg)
	require.Contains(t, reasons, "forbidden_term")

	// The denial is audited with the actor; nothing was sent.
	require.Len(t, repo.audited, 1)
	assert.Equal(t, domain.DecisionDeny, repo.audited[0].Decision)
	assert.Contains(t, repo.audited[0].Reasons, "forbidden_term")
	assert.Equal(t, "member-1", repo.audited[0].ApprovedByMemberID)
	assert.Empty(t, outbound.sent)
}

func TestGovernManualSendAllowsCompliantMemberMessage(t *testing.T) {
	repo, outbound, svc := newGovernanceHarness(forbiddenTermSettings([]string{"precio especial"}))

	reasons, msg, err := svc.GovernManualSend(context.Background(), 1, 42, "Hola, confirmamos tu pedido", "member-1")
	require.NoError(t, err)
	require.Nil(t, reasons)
	require.NotNil(t, msg)
	assert.Equal(t, "Hola, confirmamos tu pedido", msg.Content)

	// Allowed decision audited; exactly one send happened.
	require.Len(t, repo.audited, 1)
	assert.Equal(t, domain.DecisionAllow, repo.audited[0].Decision)
	assert.Equal(t, []string{"Hola, confirmamos tu pedido"}, outbound.sent)
}

func TestGovernManualSendKillSwitchBlocksEveryone(t *testing.T) {
	repo, outbound, svc := newGovernanceHarness(&domain.AgentSettings{KillSwitch: true})

	// Even an admin actor is blocked by the kill switch (hard rule).
	reasons, msg, err := svc.GovernManualSend(context.Background(), 1, 42, "Hola", "admin-1")
	require.NoError(t, err)
	require.Nil(t, msg)
	require.Contains(t, reasons, "kill_switch")
	require.Len(t, repo.audited, 1)
	assert.Equal(t, domain.DecisionDeny, repo.audited[0].Decision)
	assert.Empty(t, outbound.sent)
}
