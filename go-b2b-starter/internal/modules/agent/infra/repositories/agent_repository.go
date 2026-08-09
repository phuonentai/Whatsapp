package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
)

// agentRepository implements domain.AgentRepository on top of sqlc.Store.
type agentRepository struct {
	store sqlc.Store
}

// NewAgentRepository creates the agent repository.
func NewAgentRepository(store sqlc.Store) domain.AgentRepository {
	return &agentRepository{store: store}
}

// ---------- Flows ----------

func (r *agentRepository) CreateFlow(ctx context.Context, orgID, conversationID, contactID int32) (*domain.ConversationFlow, error) {
	row, err := r.store.CreateConversationFlow(ctx, sqlc.CreateConversationFlowParams{
		OrganizationID: orgID,
		ConversationID: conversationID,
		ContactID:      contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation flow: %w", err)
	}
	return mapFlow(&row), nil
}

func (r *agentRepository) GetFlow(ctx context.Context, orgID, flowID int32) (*domain.ConversationFlow, error) {
	row, err := r.store.GetConversationFlow(ctx, sqlc.GetConversationFlowParams{
		ID:             flowID,
		OrganizationID: orgID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrFlowNotFound
		}
		return nil, fmt.Errorf("failed to get conversation flow: %w", err)
	}
	return mapFlow(&row), nil
}

func (r *agentRepository) GetActiveFlowByConversation(ctx context.Context, orgID, conversationID int32) (*domain.ConversationFlow, error) {
	row, err := r.store.GetActiveFlowByConversation(ctx, sqlc.GetActiveFlowByConversationParams{
		OrganizationID: orgID,
		ConversationID: conversationID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrFlowNotFound
		}
		return nil, fmt.Errorf("failed to get active flow: %w", err)
	}
	return mapFlow(&row), nil
}

func (r *agentRepository) UpdateFlowStatus(ctx context.Context, orgID, flowID int32, status domain.FlowStatus) (*domain.ConversationFlow, error) {
	row, err := r.store.UpdateFlowStatus(ctx, sqlc.UpdateFlowStatusParams{
		ID:             flowID,
		OrganizationID: orgID,
		Status:         string(status),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrFlowNotFound
		}
		return nil, fmt.Errorf("failed to update flow status: %w", err)
	}
	return mapFlow(&row), nil
}

// ---------- Settings ----------

func (r *agentRepository) GetSettings(ctx context.Context, orgID int32) (*domain.AgentSettings, error) {
	row, err := r.store.GetAgentSettings(ctx, orgID)
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSettingsNotFound
		}
		return nil, fmt.Errorf("failed to get agent settings: %w", err)
	}
	return mapSettings(&row), nil
}

func (r *agentRepository) UpsertSettings(ctx context.Context, s *domain.AgentSettings) (*domain.AgentSettings, error) {
	guardrails, err := json.Marshal(s.Guardrails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal guardrails: %w", err)
	}
	row, err := r.store.UpsertAgentSettings(ctx, sqlc.UpsertAgentSettingsParams{
		OrganizationID:   s.OrganizationID,
		Mode:             string(s.Mode),
		Tone:             string(s.Tone),
		BrandVoice:       helpers.ToPgText(s.BrandVoice),
		AutopilotStart:   timeToPgTime(s.AutopilotStart),
		AutopilotEnd:     timeToPgTime(s.AutopilotEnd),
		Timezone:         s.Timezone,
		KillSwitch:       s.KillSwitch,
		MaxDailyMessages: s.MaxDailyMessages,
		ConsentRequired:  s.ConsentRequired,
		ConsentTemplate:  helpers.ToPgText(s.ConsentTemplate),
		Guardrails:       guardrails,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert agent settings: %w", err)
	}
	return mapSettings(&row), nil
}

// ---------- Suggestions ----------

func (r *agentRepository) InsertSuggestion(ctx context.Context, s *domain.Suggestion) (*domain.Suggestion, error) {
	metadata, err := json.Marshal(s.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal suggestion metadata: %w", err)
	}
	row, err := r.store.InsertSuggestion(ctx, sqlc.InsertSuggestionParams{
		OrganizationID:     s.OrganizationID,
		ConversationID:     s.ConversationID,
		ContactID:          s.ContactID,
		FlowID:             helpers.ToPgInt4Ptr(s.FlowID),
		Type:               string(s.Type),
		Body:               helpers.ToPgText(s.Body),
		Metadata:           metadata,
		Source:             string(s.Source),
		ApprovedByMemberID: helpers.ToPgText(s.ApprovedByMemberID),
		WhatsappMessageID:  helpers.ToPgText(s.WhatsAppMessageID),
		RequestID:          helpers.ToPgText(s.RequestID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert suggestion: %w", err)
	}
	return mapSuggestion(&row), nil
}

func (r *agentRepository) ListSuggestions(ctx context.Context, orgID int32, status domain.SuggestionStatus, limit, offset int32) ([]*domain.Suggestion, error) {
	rows, err := r.store.ListSuggestionsByOrgStatus(ctx, sqlc.ListSuggestionsByOrgStatusParams{
		OrganizationID: orgID,
		Status:         string(status),
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list suggestions: %w", err)
	}
	out := make([]*domain.Suggestion, 0, len(rows))
	for i := range rows {
		out = append(out, mapSuggestion(&rows[i]))
	}
	return out, nil
}

func (r *agentRepository) GetSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error) {
	row, err := r.store.GetSuggestionByID(ctx, sqlc.GetSuggestionByIDParams{
		ID:             suggestionID,
		OrganizationID: orgID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSuggestionNotFound
		}
		return nil, fmt.Errorf("failed to get suggestion: %w", err)
	}
	return mapSuggestion(&row), nil
}

func (r *agentRepository) ApproveSuggestion(ctx context.Context, orgID, suggestionID int32, approvedByMember string) (*domain.Suggestion, error) {
	row, err := r.store.ApproveSuggestion(ctx, sqlc.ApproveSuggestionParams{
		ID:                 suggestionID,
		OrganizationID:     orgID,
		ApprovedByMemberID: helpers.ToPgText(approvedByMember),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSuggestionNotFound
		}
		return nil, fmt.Errorf("failed to approve suggestion: %w", err)
	}
	return mapSuggestion(&row), nil
}

func (r *agentRepository) RejectSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error) {
	row, err := r.store.RejectSuggestion(ctx, sqlc.RejectSuggestionParams{
		ID:             suggestionID,
		OrganizationID: orgID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSuggestionNotFound
		}
		return nil, fmt.Errorf("failed to reject suggestion: %w", err)
	}
	return mapSuggestion(&row), nil
}

func (r *agentRepository) SupersedePendingReplies(ctx context.Context, orgID, conversationID int32) error {
	if err := r.store.SupersedePendingSuggestionsForConversation(ctx, sqlc.SupersedePendingSuggestionsForConversationParams{
		OrganizationID: orgID,
		ConversationID: conversationID,
	}); err != nil {
		return fmt.Errorf("failed to supersede pending suggestions: %w", err)
	}
	return nil
}

func (r *agentRepository) GetPendingSuggestionByMessage(ctx context.Context, orgID int32, whatsappMessageID string) (*domain.Suggestion, error) {
	row, err := r.store.GetPendingSuggestionByWhatsAppMessage(ctx, sqlc.GetPendingSuggestionByWhatsAppMessageParams{
		OrganizationID:    orgID,
		WhatsappMessageID: helpers.ToPgText(whatsappMessageID),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSuggestionNotFound
		}
		return nil, fmt.Errorf("failed to get pending suggestion by message: %w", err)
	}
	return mapSuggestion(&row), nil
}

// ---------- Audit ----------

func (r *agentRepository) InsertAction(ctx context.Context, a *domain.AgentAction) (*domain.AgentAction, error) {
	policyInput, err := json.Marshal(a.PolicyInput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy input: %w", err)
	}
	reasons, err := json.Marshal(a.Reasons)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reasons: %w", err)
	}
	row, err := r.store.InsertAgentAction(ctx, sqlc.InsertAgentActionParams{
		OrganizationID:     a.OrganizationID,
		FlowID:             helpers.ToPgInt4Ptr(a.FlowID),
		Action:             a.Action,
		Decision:           string(a.Decision),
		PolicyInput:        policyInput,
		Reasons:            reasons,
		ApprovedByMemberID: helpers.ToPgText(a.ApprovedByMemberID),
		WhatsappMessageID:  helpers.ToPgText(a.WhatsAppMessageID),
		RequestID:          helpers.ToPgText(a.RequestID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert agent action: %w", err)
	}
	return mapAction(&row), nil
}

// ---------- Usage ----------

func (r *agentRepository) CountMessagesSentToday(ctx context.Context, orgID int32, since time.Time) (int64, error) {
	count, err := r.store.CountSentTodayByOrganization(ctx, sqlc.CountSentTodayByOrganizationParams{
		OrganizationID: orgID,
		CreatedAt:      helpers.ToPgTimestamp(since),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count sent today: %w", err)
	}
	return count, nil
}

// ---------- Contact / conversation resolution ----------

func (r *agentRepository) ResolveContact(ctx context.Context, orgID int32, phoneNumber, displayName string, lastMessageAt time.Time) (*domain.ContactRef, error) {
	row, err := r.store.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgID,
		PhoneNumber:    phoneNumber,
		DisplayName:    helpers.ToPgText(displayName),
		AvatarUrl:      pgtype.Text{},
		Metadata:       []byte(`{}`),
		LastMessageAt:  helpers.ToPgTimestamp(lastMessageAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve contact: %w", err)
	}
	return mapContactRef(&row), nil
}

func (r *agentRepository) GetContactRef(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	row, err := r.store.GetContactByID(ctx, sqlc.GetContactByIDParams{
		ID:             contactID,
		OrganizationID: orgID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrContactNotFound
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return mapContactRef(&row), nil
}

func (r *agentRepository) ResolveConversation(ctx context.Context, orgID, contactID int32, lastMessageAt time.Time) (*domain.ConversationRef, error) {
	// Idempotent insert of an active conversation; on conflict, fall back to
	// the existing active conversation (same pattern as CRMService).
	row, err := r.store.InsertActiveConversationIdempotent(ctx, sqlc.InsertActiveConversationIdempotentParams{
		OrganizationID: orgID,
		ContactID:      contactID,
		LastMessageAt:  helpers.ToPgTimestamp(lastMessageAt),
		Metadata:       []byte(`{}`),
	})
	if err == nil {
		return mapConversationRef(&row), nil
	}
	existing, err2 := r.store.GetActiveConversationByContact(ctx, sqlc.GetActiveConversationByContactParams{
		ContactID:      contactID,
		OrganizationID: orgID,
	})
	if err2 != nil {
		return nil, fmt.Errorf("failed to resolve conversation: %w", err)
	}
	return mapConversationRef(&existing), nil
}

func (r *agentRepository) GetConversationRef(ctx context.Context, orgID, conversationID int32) (*domain.ConversationRef, error) {
	row, err := r.store.GetConversationByID(ctx, sqlc.GetConversationByIDParams{
		ID:             conversationID,
		OrganizationID: orgID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	return mapConversationRef(&row), nil
}

func (r *agentRepository) ListConversationsByContact(ctx context.Context, orgID, contactID int32) ([]*domain.ConversationRef, error) {
	rows, err := r.store.ListConversationsByContact(ctx, sqlc.ListConversationsByContactParams{
		OrganizationID: orgID,
		ContactID:      contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations by contact: %w", err)
	}
	out := make([]*domain.ConversationRef, 0, len(rows))
	for i := range rows {
		out = append(out, mapConversationRef(&rows[i]))
	}
	return out, nil
}

func (r *agentRepository) ListMessagesByConversation(ctx context.Context, orgID, conversationID int32, limit, offset int32) ([]*domain.MessageRef, error) {
	rows, err := r.store.ListMessagesByConversation(ctx, sqlc.ListMessagesByConversationParams{
		OrganizationID: orgID,
		ConversationID: conversationID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list messages by conversation: %w", err)
	}
	out := make([]*domain.MessageRef, 0, len(rows))
	for i := range rows {
		out = append(out, mapMessageRef(&rows[i]))
	}
	return out, nil
}

// ---------- Compliance ----------

func (r *agentRepository) UpdateContactConsent(ctx context.Context, orgID, contactID int32, status domain.ConsentStatus, consentedAt *time.Time) (*domain.ContactRef, error) {
	row, err := r.store.UpdateContactConsent(ctx, sqlc.UpdateContactConsentParams{
		ID:             contactID,
		OrganizationID: orgID,
		ConsentStatus:  string(status),
		ConsentedAt:    helpers.ToPgTimestampPtr(consentedAt),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrContactNotFound
		}
		return nil, fmt.Errorf("failed to update contact consent: %w", err)
	}
	return mapContactRef(&row), nil
}

func (r *agentRepository) AnonymizeContact(ctx context.Context, orgID, contactID int32) error {
	if err := r.store.AnonymizeContact(ctx, sqlc.AnonymizeContactParams{
		ID:             contactID,
		OrganizationID: orgID,
	}); err != nil {
		return fmt.Errorf("failed to anonymize contact: %w", err)
	}
	return nil
}

// ---------- mapping helpers ----------

func mapFlow(row *sqlc.AgentConversationFlow) *domain.ConversationFlow {
	return &domain.ConversationFlow{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ConversationID: row.ConversationID,
		ContactID:      row.ContactID,
		Status:         domain.FlowStatus(row.Status),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

func mapSettings(row *sqlc.AgentAgentSetting) *domain.AgentSettings {
	guardrails := domain.DefaultGuardrails()
	if len(row.Guardrails) > 0 {
		_ = json.Unmarshal(row.Guardrails, &guardrails)
	}
	return &domain.AgentSettings{
		ID:               row.ID,
		OrganizationID:   row.OrganizationID,
		Mode:             domain.Mode(row.Mode),
		Tone:             domain.Tone(row.Tone),
		BrandVoice:       helpers.FromPgText(row.BrandVoice),
		AutopilotStart:   pgTimeToTime(row.AutopilotStart),
		AutopilotEnd:     pgTimeToTime(row.AutopilotEnd),
		Timezone:         row.Timezone,
		KillSwitch:       row.KillSwitch,
		MaxDailyMessages: row.MaxDailyMessages,
		ConsentRequired:  row.ConsentRequired,
		ConsentTemplate:  helpers.FromPgText(row.ConsentTemplate),
		Guardrails:       guardrails,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func mapSuggestion(row *sqlc.AgentAgentSuggestion) *domain.Suggestion {
	metadata := map[string]any{}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	flowID := helpers.FromPgInt4Ptr(row.FlowID)
	if row.FlowID.Valid && row.FlowID.Int32 == 0 {
		flowID = nil
	}
	return &domain.Suggestion{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		ConversationID:     row.ConversationID,
		ContactID:          row.ContactID,
		FlowID:             flowID,
		Type:               domain.SuggestionType(row.Type),
		Body:               helpers.FromPgText(row.Body),
		Metadata:           metadata,
		Status:             domain.SuggestionStatus(row.Status),
		Source:             domain.SuggestionSource(row.Source),
		ApprovedByMemberID: helpers.FromPgText(row.ApprovedByMemberID),
		WhatsAppMessageID:  helpers.FromPgText(row.WhatsappMessageID),
		RequestID:          helpers.FromPgText(row.RequestID),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func mapAction(row *sqlc.AgentAgentAction) *domain.AgentAction {
	policyInput := map[string]any{}
	if len(row.PolicyInput) > 0 {
		_ = json.Unmarshal(row.PolicyInput, &policyInput)
	}
	reasons := []string{}
	if len(row.Reasons) > 0 {
		_ = json.Unmarshal(row.Reasons, &reasons)
	}
	return &domain.AgentAction{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		FlowID:             helpers.FromPgInt4Ptr(row.FlowID),
		Action:             row.Action,
		Decision:           domain.AgentDecision(row.Decision),
		PolicyInput:        policyInput,
		Reasons:            reasons,
		ApprovedByMemberID: helpers.FromPgText(row.ApprovedByMemberID),
		WhatsAppMessageID:  helpers.FromPgText(row.WhatsappMessageID),
		RequestID:          helpers.FromPgText(row.RequestID),
		CreatedAt:          row.CreatedAt.Time,
	}
}

func mapContactRef(row *sqlc.CrmContact) *domain.ContactRef {
	return &domain.ContactRef{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		PhoneNumber:     row.PhoneNumber,
		DisplayName:     helpers.FromPgText(row.DisplayName),
		Email:           helpers.FromPgText(row.Email),
		TipoDocumento:   helpers.FromPgText(row.TipoDocumento),
		NumeroDocumento: helpers.FromPgText(row.NumeroDocumento),
		ConsentStatus:   domain.ConsentStatus(row.ConsentStatus),
		ConsentedAt:     helpers.FromPgTimestampPtr(row.ConsentedAt),
	}
}

func mapConversationRef(row *sqlc.CrmConversation) *domain.ConversationRef {
	return &domain.ConversationRef{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ContactID:      row.ContactID,
		Status:         row.Status,
		LastMessageAt:  helpers.FromPgTimestampPtr(row.LastMessageAt),
	}
}

func mapMessageRef(row *sqlc.CrmMessage) *domain.MessageRef {
	return &domain.MessageRef{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		ConversationID:    row.ConversationID,
		ContactID:         row.ContactID,
		Direction:         row.Direction,
		MessageType:       row.MessageType,
		Content:           row.Content.String,
		Status:            row.Status,
		WhatsAppMessageID: row.WhatsappMessageID.String,
		CreatedAt:         row.CreatedAt.Time,
	}
}

// timeToPgTime converts "HH:MM" to pgtype.Time (24h).
func timeToPgTime(v string) pgtype.Time {
	if v == "" {
		return pgtype.Time{}
	}
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return pgtype.Time{}
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return pgtype.Time{}
	}
	return pgtype.Time{
		Microseconds: int64(h*3600+m*60) * 1_000_000,
		Valid:        true,
	}
}

// pgTimeToTime converts pgtype.Time to "HH:MM".
func pgTimeToTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	totalSeconds := t.Microseconds / 1_000_000
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func isNoRows(err error) bool {
	return err != nil && (err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows"))
}
