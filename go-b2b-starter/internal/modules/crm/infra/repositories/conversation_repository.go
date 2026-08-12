package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	"github.com/moasq/go-b2b-starter/internal/platform/dbctx"
)

type conversationRepository struct {
	store sqlc.Store
}

func NewConversationRepository(store sqlc.Store) domain.ConversationRepository {
	return &conversationRepository{store: store}
}

// storeOf prefiere la store transaccional del request (RLS con session vars
// seteadas por el middleware de scope); si no hay, usa la store del pool
// (enforcement primario: query layer con predicado de scope).
func (r *conversationRepository) storeOf(ctx context.Context) sqlc.Store {
	return dbctx.StoreFrom(ctx, r.store)
}

// scopeParams construye los parámetros de scope de las queries sqlc desde el
// Scope del miembro (flag + permisos). Convención compartida con el resolver
// domain y la política RLS.
func scopeParams(s conversationscope.Scope) (enabled, viewAll, unassigned bool, member string) {
	return s.FlagEnabled, s.ViewAll, s.ViewUnassigned, s.MemberID
}

func (r *conversationRepository) GetByID(ctx context.Context, orgID, convID int32, scope conversationscope.Scope) (*domain.Conversation, error) {
	enabled, viewAll, unassigned, member := scopeParams(scope)
	params := sqlc.GetConversationByIDParams{
		ID:              convID,
		OrganizationID:  orgID,
		ScopeEnabled:    enabled,
		ScopeViewAll:    viewAll,
		ScopeMember:     member,
		ScopeUnassigned: unassigned,
	}

	result, err := r.storeOf(ctx).GetConversationByID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) GetActiveByContact(ctx context.Context, orgID, contactID int32) (*domain.Conversation, error) {
	return r.GetActiveByContactChannel(ctx, orgID, contactID, domain.ChannelWhatsapp)
}

func (r *conversationRepository) GetActiveByContactChannel(ctx context.Context, orgID, contactID int32, channel string) (*domain.Conversation, error) {
	params := sqlc.GetActiveConversationByContactParams{
		ContactID:      contactID,
		OrganizationID: orgID,
		Channel:        channel,
	}

	result, err := r.storeOf(ctx).GetActiveConversationByContact(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get active conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) Create(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	params := sqlc.CreateConversationParams{
		OrganizationID:         conv.OrganizationID,
		ContactID:              conv.ContactID,
		Channel:                channelOrDefault(conv.Channel),
		Status:                 string(conv.Status),
		LastMessageAt:          helpers.ToPgTimestampPtr(conv.LastMessageAt),
		Metadata:               helpers.ToJSONB(conv.Metadata),
		AssigneeStytchMemberID: helpers.ToPgTextPtr(conv.AssigneeStytchMemberID),
	}

	result, err := r.storeOf(ctx).CreateConversation(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) EnsureActive(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	params := sqlc.InsertActiveConversationIdempotentParams{
		OrganizationID:         conv.OrganizationID,
		ContactID:              conv.ContactID,
		Channel:                channelOrDefault(conv.Channel),
		LastMessageAt:          helpers.ToPgTimestampPtr(conv.LastMessageAt),
		Metadata:               helpers.ToJSONB(conv.Metadata),
		AssigneeStytchMemberID: helpers.ToPgTextPtr(conv.AssigneeStytchMemberID),
	}

	result, err := r.storeOf(ctx).InsertActiveConversationIdempotent(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := r.GetActiveByContactChannel(ctx, conv.OrganizationID, conv.ContactID, channelOrDefault(conv.Channel))
			if getErr != nil {
				return nil, fmt.Errorf("failed to fetch active conversation after idempotent insert: %w", getErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to ensure active conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) UpdateLastMessageAt(ctx context.Context, orgID, convID int32, lastMessageAt *time.Time) (*domain.Conversation, error) {
	params := sqlc.UpdateConversationLastMessageAtParams{
		ID:             convID,
		OrganizationID: orgID,
		LastMessageAt:  helpers.ToPgTimestampPtr(lastMessageAt),
	}

	result, err := r.storeOf(ctx).UpdateConversationLastMessageAt(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation last_message_at: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus, scope conversationscope.Scope) (*domain.Conversation, error) {
	enabled, viewAll, unassigned, member := scopeParams(scope)
	params := sqlc.UpdateConversationStatusParams{
		ID:              convID,
		OrganizationID:  orgID,
		Status:          string(status),
		ScopeEnabled:    enabled,
		ScopeViewAll:    viewAll,
		ScopeMember:     member,
		ScopeUnassigned: unassigned,
	}

	result, err := r.storeOf(ctx).UpdateConversationStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation status: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) ListByOrganization(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string, view conversationscope.ViewScope, scope conversationscope.Scope) ([]*domain.ConversationWithContact, error) {
	enabled, viewAll, unassigned, member := scopeParams(scope)
	params := sqlc.ListConversationsByOrganizationParams{
		OrganizationID:  orgID,
		StatusFilter:    statusFilter,
		ChannelFilter:   channelFilter,
		ScopeEnabled:    enabled,
		ScopeViewAll:    viewAll,
		ScopeMember:     member,
		ScopeUnassigned: unassigned,
		ScopeView:       string(view),
		PageOffset:      offset,
		PageLimit:       limit,
	}

	results, err := r.storeOf(ctx).ListConversationsByOrganization(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	convs := make([]*domain.ConversationWithContact, len(results))
	for i, row := range results {
		convs[i] = &domain.ConversationWithContact{
			Conversation: domain.Conversation{
				ID:                      row.ID,
				OrganizationID:          row.OrganizationID,
				ContactID:               row.ContactID,
				Channel:                 row.Channel,
				Status:                  domain.ConversationStatus(row.Status),
				LastMessageAt:           helpers.FromPgTimestampPtr(row.LastMessageAt),
				Metadata:                helpers.FromJSONB(row.Metadata),
				CreatedAt:               row.CreatedAt.Time,
				UpdatedAt:               row.UpdatedAt.Time,
				AssigneeStytchMemberID:  helpers.FromPgTextPtr(row.AssigneeStytchMemberID),
			},
			ContactPhone:             helpers.FromPgText(row.ContactPhone),
			ContactDisplayName:       helpers.FromPgText(row.ContactDisplayName),
			ContactInstagramUsername: helpers.FromPgText(row.ContactInstagramUsername),
			ContactAvatarURL:         helpers.FromPgText(row.ContactAvatarUrl),
		}
	}

	return convs, nil
}

func (r *conversationRepository) UpdateAssignee(ctx context.Context, orgID, convID int32, assignee *string) (*domain.Conversation, error) {
	params := sqlc.UpdateConversationAssigneeParams{
		ID:                     convID,
		OrganizationID:         orgID,
		AssigneeStytchMemberID: helpers.ToPgTextPtr(assignee),
	}

	result, err := r.storeOf(ctx).UpdateConversationAssignee(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation assignee: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) InsertEvent(ctx context.Context, event *domain.ConversationEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation event payload: %w", err)
	}
	params := sqlc.InsertConversationEventParams{
		OrganizationID:      event.OrganizationID,
		ConversationID:      event.ConversationID,
		EventType:           event.EventType,
		ActorStytchMemberID: helpers.ToPgTextPtr(event.ActorStytchMemberID),
		Payload:             payload,
	}
	_, err = r.storeOf(ctx).InsertConversationEvent(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to insert conversation event: %w", err)
	}
	return nil
}

func (r *conversationRepository) ResolveContactAssignee(ctx context.Context, orgID, contactID int32) (*string, error) {
	params := sqlc.ResolveContactAssigneeParams{
		ID:             contactID,
		OrganizationID: orgID,
	}
	row, err := r.storeOf(ctx).ResolveContactAssignee(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve contact assignee: %w", err)
	}
	return helpers.FromPgTextPtr(row), nil
}

func (r *conversationRepository) ResolveCompanyOwnerMemberByPhone(ctx context.Context, orgID int32, phone string) (*string, error) {
	params := sqlc.GetCompanyOwnerMemberByPhoneParams{
		OrganizationID: orgID,
		PhoneNumber:    helpers.ToPgText(phone),
	}
	row, err := r.storeOf(ctx).GetCompanyOwnerMemberByPhone(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve company owner by phone: %w", err)
	}
	return helpers.FromPgTextPtr(row), nil
}

func (r *conversationRepository) mapToDomain(c *sqlc.CrmConversation) *domain.Conversation {
	return &domain.Conversation{
		ID:                     c.ID,
		OrganizationID:         c.OrganizationID,
		ContactID:              c.ContactID,
		Channel:                c.Channel,
		Status:                 domain.ConversationStatus(c.Status),
		LastMessageAt:          helpers.FromPgTimestampPtr(c.LastMessageAt),
		Metadata:               helpers.FromJSONB(c.Metadata),
		CreatedAt:              c.CreatedAt.Time,
		UpdatedAt:              c.UpdatedAt.Time,
		AssigneeStytchMemberID: helpers.FromPgTextPtr(c.AssigneeStytchMemberID),
	}
}

func channelOrDefault(channel string) string {
	if channel == "" {
		return domain.ChannelWhatsapp
	}
	return channel
}
