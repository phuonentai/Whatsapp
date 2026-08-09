package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// DenialError carries the guardrail reasons for a denied action.
type DenialError struct {
	Reasons []string
}

func (e *DenialError) Error() string {
	return "agent action denied by guardrails: " + strings.Join(e.Reasons, ", ")
}

// agentService implements AgentService: linear pipeline
// analysis -> decide -> notify/send (design D2).
type agentService struct {
	repo       domain.AgentRepository
	guardrails domain.GuardrailService
	llm        llmdomain.LLMClient
	billing    billingServices.BillingService
	outbound   crmServices.OutboundService
	log        logger.Logger
}

// NewAgentService creates the agent pipeline service.
func NewAgentService(
	repo domain.AgentRepository,
	guardrails domain.GuardrailService,
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	outbound crmServices.OutboundService,
	log logger.Logger,
) AgentService {
	return &agentService{repo: repo, guardrails: guardrails, llm: llm, billing: billing, outbound: outbound, log: log}
}

// affirmativeTerms are deterministic consent-acceptance signals (Ley 1581).
var affirmativeTerms = []string{"sí", "si ", "acepto", "autorizo", "ok", "okay", "claro", "dale", "me parece bien"}

func (s *agentService) HandleMessageReceived(ctx context.Context, event *whatsappEvents.MessageReceived) error {
	orgID := event.OrganizationID
	phone, err := whatsapp.CanonicalizeE164(event.From)
	if err != nil {
		return fmt.Errorf("invalid sender phone: %w", err)
	}

	contact, err := s.repo.ResolveContact(ctx, orgID, phone, event.From, event.WhatsAppTimestamp)
	if err != nil {
		return err
	}
	conv, err := s.repo.ResolveConversation(ctx, orgID, contact.ID, event.WhatsAppTimestamp)
	if err != nil {
		return err
	}

	settings, err := s.getSettingsOrDefault(ctx, orgID)
	if err != nil {
		return err
	}

	flow, err := s.repo.GetActiveFlowByConversation(ctx, orgID, conv.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrFlowNotFound) {
			return err
		}
		flow, err = s.repo.CreateFlow(ctx, orgID, conv.ID, contact.ID)
		if err != nil {
			return err
		}
	}

	if settings.KillSwitch {
		if _, err := s.repo.UpdateFlowStatus(ctx, orgID, flow.ID, domain.FlowStatusCancelled); err != nil {
			return err
		}
		return s.audit(ctx, orgID, flow, domain.GuardrailActionSendMessage, domain.DecisionSkip,
			[]string{"kill_switch"}, "", event.MessageSID, contact)
	}

	// Dedupe: a retried webhook must not run the pipeline twice for one message.
	if _, err := s.repo.GetPendingSuggestionByMessage(ctx, orgID, event.MessageSID); err == nil {
		return nil
	}

	// Consent state machine (Ley 1581) — runs before analysis so the consent
	// template is the first message a new contact receives.
	switch contact.ConsentStatus {
	case domain.ConsentNone:
		if settings.ConsentRequired {
			if err := s.sendConsentRequest(ctx, orgID, conv.ID, settings, flow, event.MessageSID, contact); err != nil {
				return err
			}
			return nil
		}
	case domain.ConsentRequested:
		if isAffirmative(event.Content) {
			now := time.Now()
			contact, err = s.repo.UpdateContactConsent(ctx, orgID, contact.ID, domain.ConsentGranted, &now)
			if err != nil {
				return err
			}
			if err := s.audit(ctx, orgID, flow, "consent_grant", domain.DecisionAllow, nil, "", event.MessageSID, contact); err != nil {
				return err
			}
			s.log.Info("contact consent granted", loggerdomain.Fields{"org_id": orgID, "contact_id": contact.ID})
		}
	}

	// Analysis (metered, credit-gated).
	draft, skipReason, err := s.analyze(ctx, orgID, settings, event, contact)
	if err != nil {
		return err
	}
	if skipReason != "" {
		if err := s.escalate(ctx, orgID, flow, event.MessageSID, contact, skipReason); err != nil {
			return err
		}
		return nil
	}

	// Decide + act.
	if settings.Mode == domain.ModeAutopilot {
		return s.tryAutopilotSend(ctx, orgID, conv.ID, flow, settings, contact, draft, event.MessageSID)
	}

	return s.createReplySuggestion(ctx, orgID, conv.ID, contact.ID, flow, draft, event.MessageSID, domain.SuggestionSourceCopilot)
}

// analyze runs the consolidated metered LLM call. Returns the suggested reply,
// a skip reason when credits are exhausted (empty otherwise), and an error.
func (s *agentService) analyze(ctx context.Context, orgID int32, settings *domain.AgentSettings, event *whatsappEvents.MessageReceived, contact *domain.ContactRef) (string, string, error) {
	status, err := s.billing.GetAiUsageStatus(ctx, orgID)
	if err != nil {
		s.log.Warn("ai usage status unavailable, proceeding fail-open", loggerdomain.Fields{"org_id": orgID, "error": err.Error()})
	} else if status != nil && status.CreditsMax > 0 && status.CreditsRemaining <= 0 {
		return "", "ai_credits_exhausted", nil
	}

	systemPrompt := analysisSystemPrompt(settings)
	userPrompt := fmt.Sprintf(
		"Mensaje entrante de %s (%s):\n%s",
		contact.DisplayName, contact.PhoneNumber, event.Content,
	)

	ctx = llmdomain.WithOrgID(ctx, orgID)
	ctx = llmdomain.WithPiiFacts(ctx, llmdomain.PiiFacts{
		PhoneNumber:     contact.PhoneNumber,
		DisplayName:     contact.DisplayName,
		NumeroDocumento: contact.NumeroDocumento,
	})

	resp, err := s.llm.Complete(ctx, llmdomain.CompletionRequest{
		Prompt: systemPrompt + "\n\n" + userPrompt,
	})
	if err != nil {
		return "", "", fmt.Errorf("agent analysis failed: %w", err)
	}

	draft, err := extractSuggestedReply(resp.Text)
	if err != nil {
		s.log.Warn("agent analysis response unparsable, using raw text", loggerdomain.Fields{"org_id": orgID, "error": err.Error()})
		draft = strings.TrimSpace(resp.Text)
	}
	if strings.TrimSpace(draft) == "" {
		return "", "empty_draft", nil
	}
	return draft, "", nil
}

// tryAutopilotSend evaluates the send guardrails and sends or falls back.
func (s *agentService) tryAutopilotSend(ctx context.Context, orgID, convID int32, flow *domain.ConversationFlow, settings *domain.AgentSettings, contact *domain.ContactRef, draft, messageID string) error {
	sentToday, err := s.repo.CountMessagesSentToday(ctx, orgID, startOfDayUTC())
	if err != nil {
		s.log.Warn("sent-today count unavailable, treating as zero", loggerdomain.Fields{"org_id": orgID, "error": err.Error()})
	}

	dec, err := s.guardrails.Evaluate(ctx, orgID, domain.GuardrailInput{
		Action:      domain.GuardrailActionSendMessage,
		Draft:       draft,
		Settings:    *settings,
		Contact:     toContactFacts(contact),
		SentToday:   sentToday,
		Autonomous:  true,
		Now:         time.Now(),
	})
	if err != nil {
		dec = domain.GuardrailDecision{Allowed: false, Reasons: []string{"guardrail_error"}}
	}

	if err := s.audit(ctx, orgID, flow, domain.GuardrailActionSendMessage, decisionOf(dec), dec.Reasons, "", messageID, contact); err != nil {
		return err
	}

	if dec.Allowed {
		if _, err := s.outbound.SendMessage(ctx, orgID, convID, draft); err != nil {
			return fmt.Errorf("autopilot send failed: %w", err)
		}
		if _, err := s.repo.UpdateFlowStatus(ctx, orgID, flow.ID, domain.FlowStatusSucceeded); err != nil {
			return err
		}
		s.log.Info("autopilot message sent", loggerdomain.Fields{"org_id": orgID, "conv_id": convID})
		return nil
	}

	if containsReason(dec.Reasons, "escalation_match") {
		return s.escalate(ctx, orgID, flow, messageID, contact, "escalation_match")
	}
	return s.createReplySuggestion(ctx, orgID, convID, contact.ID, flow, draft, messageID, domain.SuggestionSourceAutopilotFallback)
}

// createReplySuggestion supersedes stale pending replies and inserts a draft.
func (s *agentService) createReplySuggestion(ctx context.Context, orgID, convID, contactID int32, flow *domain.ConversationFlow, draft, messageID string, source domain.SuggestionSource) error {
	if err := s.repo.SupersedePendingReplies(ctx, orgID, convID); err != nil {
		return err
	}
	if _, err := s.repo.InsertSuggestion(ctx, &domain.Suggestion{
		OrganizationID:    orgID,
		ConversationID:    convID,
		ContactID:         contactID,
		FlowID:            &flow.ID,
		Type:              domain.SuggestionReply,
		Body:              draft,
		Metadata:          map[string]any{},
		Source:            source,
		WhatsAppMessageID: messageID,
	}); err != nil {
		return err
	}
	return nil
}

// escalate marks the flow awaiting_human and surfaces an escalation suggestion.
func (s *agentService) escalate(ctx context.Context, orgID int32, flow *domain.ConversationFlow, messageID string, contact *domain.ContactRef, reason string) error {
	if _, err := s.repo.UpdateFlowStatus(ctx, orgID, flow.ID, domain.FlowStatusAwaitingHuman); err != nil {
		return err
	}
	if _, err := s.repo.InsertSuggestion(ctx, &domain.Suggestion{
		OrganizationID:    orgID,
		ConversationID:    flow.ConversationID,
		ContactID:         flow.ContactID,
		FlowID:            &flow.ID,
		Type:              domain.SuggestionEscalation,
		Body:              "Requiere atención humana: " + reason,
		Metadata:          map[string]any{"reason": reason},
		Source:            domain.SuggestionSourceEscalation,
		WhatsAppMessageID: messageID,
	}); err != nil {
		return err
	}
	return s.audit(ctx, orgID, flow, domain.GuardrailActionEscalate, domain.DecisionAllow, []string{reason}, "", messageID, contact)
}

// sendConsentRequest sends the Ley 1581 template and marks consent requested.
func (s *agentService) sendConsentRequest(ctx context.Context, orgID, convID int32, settings *domain.AgentSettings, flow *domain.ConversationFlow, messageID string, contact *domain.ContactRef) error {
	template := settings.ConsentTemplate
	if strings.TrimSpace(template) == "" {
		template = "Hola, para atenderte necesitamos tu autorización para el tratamiento de tus datos personales conforme a la Ley 1581. ¿Nos autorizas? (Responde sí o acepto)."
	}
	if _, err := s.outbound.SendMessage(ctx, orgID, convID, template); err != nil {
		return fmt.Errorf("consent template send failed: %w", err)
	}
	if _, err := s.repo.UpdateContactConsent(ctx, orgID, contact.ID, domain.ConsentRequested, nil); err != nil {
		return err
	}
	if err := s.audit(ctx, orgID, flow, "consent_request", domain.DecisionAllow, nil, "", messageID, contact); err != nil {
		return err
	}
	s.log.Info("consent template sent", loggerdomain.Fields{"org_id": orgID, "contact_id": contact.ID})
	return nil
}

func (s *agentService) getSettingsOrDefault(ctx context.Context, orgID int32) (*domain.AgentSettings, error) {
	settings, err := s.repo.GetSettings(ctx, orgID)
	if err == nil {
		return settings, nil
	}
	if errors.Is(err, domain.ErrSettingsNotFound) {
		def := domain.DefaultSettings(orgID)
		return &def, nil
	}
	return nil, err
}

// audit appends one immutable governance row.
func (s *agentService) audit(ctx context.Context, orgID int32, flow *domain.ConversationFlow, action string, decision domain.AgentDecision, reasons []string, approver, messageID string, contact *domain.ContactRef) error {
	flowID := flow.ID
	_, err := s.repo.InsertAction(ctx, &domain.AgentAction{
		OrganizationID:    orgID,
		FlowID:            &flowID,
		Action:            action,
		Decision:          decision,
		PolicyInput: map[string]any{
			"action":        action,
			"flow_status":   string(flow.Status),
			"contact_phone": contact.PhoneNumber,
			"consent":       string(contact.ConsentStatus),
			"approver":      approver,
			"now":           time.Now().UTC().Format(time.RFC3339),
		},
		Reasons:           reasons,
		ApprovedByMemberID: approver,
		WhatsAppMessageID:  messageID,
	})
	if err != nil {
		return fmt.Errorf("failed to audit agent action: %w", err)
	}
	return nil
}

// ---------- Approval flow (copilot) ----------

func (s *agentService) ApproveSuggestion(ctx context.Context, orgID, suggestionID int32, editedBody, memberID string) (*domain.Suggestion, error) {
	suggestion, err := s.repo.GetSuggestion(ctx, orgID, suggestionID)
	if err != nil {
		return nil, err
	}
	if suggestion.Status != domain.SuggestionPending {
		return nil, domain.ErrSuggestionNotFound
	}

	settings, err := s.getSettingsOrDefault(ctx, orgID)
	if err != nil {
		return nil, err
	}
	contact, err := s.repo.GetContactRef(ctx, orgID, suggestion.ContactID)
	if err != nil {
		return nil, err
	}

	body := suggestion.Body
	if strings.TrimSpace(editedBody) != "" {
		body = editedBody
	}

	flow, err := s.repo.GetFlow(ctx, orgID, *suggestion.FlowID)
	if err != nil {
		return nil, err
	}

	dec, err := s.guardrails.Evaluate(ctx, orgID, domain.GuardrailInput{
		Action:      domain.GuardrailActionSendMessage,
		Draft:       body,
		Settings:    *settings,
		Contact:     toContactFacts(contact),
		Autonomous:  false,
		Approver:    memberID,
		Now:         time.Now(),
	})
	if err != nil {
		dec = domain.GuardrailDecision{Allowed: false, Reasons: []string{"guardrail_error"}}
	}

	if err := s.audit(ctx, orgID, flow, domain.GuardrailActionSendMessage, decisionOf(dec), dec.Reasons, memberID, suggestion.WhatsAppMessageID, contact); err != nil {
		return nil, err
	}

	if !dec.Allowed {
		rejected, rejErr := s.repo.RejectSuggestion(ctx, orgID, suggestionID)
		if rejErr != nil {
			return nil, rejErr
		}
		return rejected, &DenialError{Reasons: dec.Reasons}
	}

	if _, err := s.outbound.SendMessage(ctx, orgID, suggestion.ConversationID, body); err != nil {
		return nil, fmt.Errorf("approved send failed: %w", err)
	}
	approved, err := s.repo.ApproveSuggestion(ctx, orgID, suggestionID, memberID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.UpdateFlowStatus(ctx, orgID, flow.ID, domain.FlowStatusSucceeded); err != nil {
		return nil, err
	}
	return approved, nil
}

func (s *agentService) RejectSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error) {
	suggestion, err := s.repo.GetSuggestion(ctx, orgID, suggestionID)
	if err != nil {
		return nil, err
	}
	if suggestion.Status != domain.SuggestionPending {
		return nil, domain.ErrSuggestionNotFound
	}
	rejected, err := s.repo.RejectSuggestion(ctx, orgID, suggestionID)
	if err != nil {
		return nil, err
	}
	contact, err := s.repo.GetContactRef(ctx, orgID, suggestion.ContactID)
	if err != nil {
		return nil, err
	}
	if flow, err := s.repo.GetFlow(ctx, orgID, *suggestion.FlowID); err == nil {
		if err := s.audit(ctx, orgID, flow, domain.GuardrailActionSendMessage, domain.DecisionDeny, []string{"human_rejection"}, "", suggestion.WhatsAppMessageID, contact); err != nil {
			return nil, err
		}
	}
	return rejected, nil
}

func (s *agentService) ListPendingSuggestions(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Suggestion, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListSuggestions(ctx, orgID, domain.SuggestionPending, limit, offset)
}

func (s *agentService) GetFlowDebug(ctx context.Context, orgID, conversationID int32) (*FlowDebug, error) {
	flow, err := s.repo.GetActiveFlowByConversation(ctx, orgID, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrFlowNotFound) {
			return &FlowDebug{Flow: nil, Suggestions: []*domain.Suggestion{}}, nil
		}
		return nil, err
	}
	suggestions, err := s.repo.ListSuggestions(ctx, orgID, domain.SuggestionPending, 20, 0)
	if err != nil {
		return nil, err
	}
	return &FlowDebug{Flow: flow, Suggestions: suggestions}, nil
}

func (s *agentService) GetSettings(ctx context.Context, orgID int32) (*domain.AgentSettings, error) {
	return s.getSettingsOrDefault(ctx, orgID)
}

func (s *agentService) UpdateSettings(ctx context.Context, orgID int32, settings *domain.AgentSettings) (*domain.AgentSettings, error) {
	if settings.Mode != domain.ModeCopilot && settings.Mode != domain.ModeAutopilot {
		return nil, fmt.Errorf("invalid mode %q", settings.Mode)
	}
	if settings.Tone != domain.ToneFormal && settings.Tone != domain.ToneCasual {
		return nil, fmt.Errorf("invalid tone %q", settings.Tone)
	}
	if settings.MaxDailyMessages < 0 {
		return nil, fmt.Errorf("max_daily_messages must be >= 0")
	}
	merged := domain.DefaultSettings(orgID)
	merged.Mode = settings.Mode
	merged.Tone = settings.Tone
	merged.BrandVoice = settings.BrandVoice
	merged.AutopilotStart = settings.AutopilotStart
	merged.AutopilotEnd = settings.AutopilotEnd
	merged.Timezone = settings.Timezone
	if merged.Timezone == "" {
		merged.Timezone = "America/Bogota"
	}
	merged.KillSwitch = settings.KillSwitch
	merged.MaxDailyMessages = settings.MaxDailyMessages
	merged.ConsentRequired = settings.ConsentRequired
	merged.ConsentTemplate = settings.ConsentTemplate
	if settings.Guardrails.Never != nil {
		merged.Guardrails.Never = settings.Guardrails.Never
	}
	if settings.Guardrails.Escalate != nil {
		merged.Guardrails.Escalate = settings.Guardrails.Escalate
	}
	return s.repo.UpsertSettings(ctx, &merged)
}

// ---------- helpers ----------

// analysisSystemPrompt builds the persona instructions from tone + brand voice.
func analysisSystemPrompt(settings *domain.AgentSettings) string {
	var toneInstruction string
	switch settings.Tone {
	case domain.ToneCasual:
		toneInstruction = "Responde en un registro informal colombiano: usa 'tú' o 'vos'. Sé cercano y amigable."
	default:
		toneInstruction = "Responde en un registro formal colombiano: usa 'usted'. Sé cortés y profesional."
	}
	prompt := "Eres un asistente comercial de WhatsApp para una empresa colombiana.\n" +
		toneInstruction + "\n" +
		"Analiza el mensaje entrante y responde ÚNICAMENTE con JSON válido con esta forma exacta: " +
		`{"intent": "texto", "sentiment": "texto", "suggested_reply": "tu borrador de respuesta"}. ` +
		"El borrador debe estar en español y listo para enviar al cliente."
	if strings.TrimSpace(settings.BrandVoice) != "" {
		prompt += "\nVoz de la marca: " + settings.BrandVoice
	}
	return prompt
}

// extractSuggestedReply parses the JSON analysis response.
func extractSuggestedReply(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	start, end := strings.Index(trimmed, "{"), strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("no JSON object in response")
	}
	var parsed struct {
		Intent        string `json:"intent"`
		Sentiment     string `json:"sentiment"`
		SuggestedReply string `json:"suggested_reply"`
	}
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &parsed); err != nil {
		return "", err
	}
	return parsed.SuggestedReply, nil
}

// isAffirmative detects a consent acceptance signal.
func isAffirmative(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	for _, term := range affirmativeTerms {
		if strings.Contains(c, term) {
			return true
		}
	}
	return false
}

func toContactFacts(c *domain.ContactRef) domain.ContactFacts {
	return domain.ContactFacts{
		PhoneNumber:     c.PhoneNumber,
		DisplayName:     c.DisplayName,
		NumeroDocumento: c.NumeroDocumento,
		ConsentStatus:   c.ConsentStatus,
	}
}

func decisionOf(dec domain.GuardrailDecision) domain.AgentDecision {
	if dec.Allowed {
		return domain.DecisionAllow
	}
	return domain.DecisionDeny
}

func containsReason(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

func startOfDayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
