package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type conversationRepository struct {
	store sqlc.Store
}

func NewConversationRepository(store sqlc.Store) domain.ConversationRepository {
	return &conversationRepository{store: store}
}

func (r *conversationRepository) GetByID(ctx context.Context, orgID, convID int32) (*domain.Conversation, error) {
	params := sqlc.GetConversationByIDParams{
		ID:             convID,
		OrganizationID: orgID,
	}

	result, err := r.store.GetConversationByID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) GetActiveByContact(ctx context.Context, orgID, contactID int32) (*domain.Conversation, error) {
	params := sqlc.GetActiveConversationByContactParams{
		ContactID:      contactID,
		OrganizationID: orgID,
	}

	result, err := r.store.GetActiveConversationByContact(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get active conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) Create(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	params := sqlc.CreateConversationParams{
		OrganizationID: conv.OrganizationID,
		ContactID:      conv.ContactID,
		Status:         string(conv.Status),
		LastMessageAt:  helpers.ToPgTimestampPtr(conv.LastMessageAt),
		Metadata:       helpers.ToJSONB(conv.Metadata),
	}

	result, err := r.store.CreateConversation(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) EnsureActive(ctx context.Context, conv *domain.Conversation) (*domain.Conversation, error) {
	params := sqlc.InsertActiveConversationIdempotentParams{
		OrganizationID: conv.OrganizationID,
		ContactID:      conv.ContactID,
		LastMessageAt:  helpers.ToPgTimestampPtr(conv.LastMessageAt),
		Metadata:       helpers.ToJSONB(conv.Metadata),
	}

	result, err := r.store.InsertActiveConversationIdempotent(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := r.GetActiveByContact(ctx, conv.OrganizationID, conv.ContactID)
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

	result, err := r.store.UpdateConversationLastMessageAt(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation last_message_at: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus) (*domain.Conversation, error) {
	params := sqlc.UpdateConversationStatusParams{
		ID:             convID,
		OrganizationID: orgID,
		Status:         string(status),
	}

	result, err := r.store.UpdateConversationStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation status: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *conversationRepository) ListByOrganization(ctx context.Context, orgID int32, limit, offset int32, statusFilter string) ([]*domain.ConversationWithContact, error) {
	params := sqlc.ListConversationsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
		Column4:        statusFilter,
	}

	results, err := r.store.ListConversationsByOrganization(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	convs := make([]*domain.ConversationWithContact, len(results))
	for i, row := range results {
		convs[i] = &domain.ConversationWithContact{
			Conversation: domain.Conversation{
				ID:             row.ID,
				OrganizationID: row.OrganizationID,
				ContactID:      row.ContactID,
				Status:         domain.ConversationStatus(row.Status),
				LastMessageAt:  helpers.FromPgTimestampPtr(row.LastMessageAt),
				Metadata:       helpers.FromJSONB(row.Metadata),
				CreatedAt:      row.CreatedAt.Time,
				UpdatedAt:      row.UpdatedAt.Time,
			},
			ContactPhone:       helpers.FromPgText(row.ContactPhone),
			ContactDisplayName: helpers.FromPgText(row.ContactDisplayName),
		}
	}

	return convs, nil
}

func (r *conversationRepository) mapToDomain(c *sqlc.CrmConversation) *domain.Conversation {
	return &domain.Conversation{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		ContactID:      c.ContactID,
		Status:         domain.ConversationStatus(c.Status),
		LastMessageAt:  helpers.FromPgTimestampPtr(c.LastMessageAt),
		Metadata:       helpers.FromJSONB(c.Metadata),
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
}
