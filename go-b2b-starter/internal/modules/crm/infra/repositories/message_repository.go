package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type messageRepository struct {
	store sqlc.Store
}

func NewMessageRepository(store sqlc.Store) domain.MessageRepository {
	return &messageRepository{store: store}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	params := sqlc.CreateMessageParams{
		OrganizationID:    msg.OrganizationID,
		ConversationID:    msg.ConversationID,
		ContactID:         msg.ContactID,
		Channel:           channelOrDefault(msg.Channel),
		ProviderMessageID: helpers.ToPgText(msg.ProviderMessageID),
		Direction:         string(msg.Direction),
		MessageType:       string(msg.MessageType),
		Content:           helpers.ToPgText(msg.Content),
		Status:            msg.Status,
		MessageData:       helpers.ToJSONB(msg.MessageData),
		ChatTimestamp:     helpers.ToPgTimestampPtr(msg.ChatTimestamp),
	}

	result, err := r.store.CreateMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *messageRepository) InsertIdempotent(ctx context.Context, msg *domain.Message) (*domain.Message, bool, error) {
	params := sqlc.InsertMessageIdempotentParams{
		OrganizationID:    msg.OrganizationID,
		ConversationID:    msg.ConversationID,
		ContactID:         msg.ContactID,
		Channel:           channelOrDefault(msg.Channel),
		ProviderMessageID: helpers.ToPgText(msg.ProviderMessageID),
		Direction:         string(msg.Direction),
		MessageType:       string(msg.MessageType),
		Content:           helpers.ToPgText(msg.Content),
		Status:            msg.Status,
		MessageData:       helpers.ToJSONB(msg.MessageData),
		ChatTimestamp:     helpers.ToPgTimestampPtr(msg.ChatTimestamp),
	}

	result, err := r.store.InsertMessageIdempotent(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) && msg.ProviderMessageID != "" {
			existing, getErr := r.GetByProviderID(ctx, msg.OrganizationID, channelOrDefault(msg.Channel), msg.ProviderMessageID)
			if getErr != nil {
				return nil, false, fmt.Errorf("failed to fetch message after idempotent insert: %w", getErr)
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("failed to insert message idempotently: %w", err)
	}

	return r.mapToDomain(&result), true, nil
}

func (r *messageRepository) GetByProviderID(ctx context.Context, orgID int32, channel, providerMessageID string) (*domain.Message, error) {
	params := sqlc.GetMessageByProviderIDParams{
		OrganizationID:    orgID,
		Channel:           channel,
		ProviderMessageID: helpers.ToPgText(providerMessageID),
	}

	result, err := r.store.GetMessageByProviderID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get message by provider id: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *messageRepository) UpdateStatus(ctx context.Context, id int32, status, providerMessageID string) (*domain.Message, error) {
	result, err := r.store.UpdateMessageStatus(ctx, sqlc.UpdateMessageStatusParams{
		ID:                id,
		Status:            status,
		ProviderMessageID: helpers.ToPgText(providerMessageID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to update message status: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *messageRepository) ListByConversation(ctx context.Context, orgID, convID int32, limit, offset int32) ([]*domain.Message, error) {
	params := sqlc.ListMessagesByConversationParams{
		ConversationID: convID,
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	}

	results, err := r.store.ListMessagesByConversation(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]*domain.Message, len(results))
	for i := range results {
		messages[i] = r.mapToDomain(&results[i])
	}

	return messages, nil
}

func (r *messageRepository) mapToDomain(m *sqlc.CrmMessage) *domain.Message {
	return &domain.Message{
		ID:                m.ID,
		OrganizationID:    m.OrganizationID,
		ConversationID:    m.ConversationID,
		ContactID:         m.ContactID,
		Channel:           m.Channel,
		ProviderMessageID: helpers.FromPgText(m.ProviderMessageID),
		Direction:         domain.MessageDirection(m.Direction),
		MessageType:       domain.MessageType(m.MessageType),
		Content:           helpers.FromPgText(m.Content),
		Status:            m.Status,
		MessageData:       helpers.FromJSONB(m.MessageData),
		ChatTimestamp:     helpers.FromPgTimestampPtr(m.ChatTimestamp),
		CreatedAt:         m.CreatedAt.Time,
		UpdatedAt:         m.UpdatedAt.Time,
	}
}
