package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type ConversationService interface {
	ListConversations(ctx context.Context, orgID int32, limit, offset int32, statusFilter string) ([]*domain.ConversationWithContact, error)
	GetConversation(ctx context.Context, orgID, convID int32) (*domain.ConversationWithContact, error)
	UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus) (*domain.Conversation, error)
	ListMessages(ctx context.Context, orgID, convID int32, limit, offset int32) ([]*domain.Message, error)
}

type conversationService struct {
	convRepo domain.ConversationRepository
	msgRepo  domain.MessageRepository
	contactRepo domain.ContactRepository
}

func NewConversationService(convRepo domain.ConversationRepository, msgRepo domain.MessageRepository, contactRepo domain.ContactRepository) ConversationService {
	return &conversationService{convRepo: convRepo, msgRepo: msgRepo, contactRepo: contactRepo}
}

func (s *conversationService) ListConversations(ctx context.Context, orgID int32, limit, offset int32, statusFilter string) ([]*domain.ConversationWithContact, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.convRepo.ListByOrganization(ctx, orgID, limit, offset, statusFilter)
}

func (s *conversationService) GetConversation(ctx context.Context, orgID, convID int32) (*domain.ConversationWithContact, error) {
	conv, err := s.convRepo.GetByID(ctx, orgID, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	contact, err := s.contactRepo.GetByID(ctx, orgID, conv.ContactID)
	if err != nil {
		_ = err
	}

	result := &domain.ConversationWithContact{
		Conversation: *conv,
	}
	if contact != nil {
		result.ContactPhone = contact.PhoneNumber
		result.ContactDisplayName = contact.DisplayName
	}
	return result, nil
}

func (s *conversationService) UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus) (*domain.Conversation, error) {
	valid := map[domain.ConversationStatus]bool{
		domain.ConversationStatusActive:   true,
		domain.ConversationStatusClosed:   true,
		domain.ConversationStatusArchived: true,
	}
	if !valid[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	return s.convRepo.UpdateStatus(ctx, orgID, conv.ID, status)
}

func (s *conversationService) ListMessages(ctx context.Context, orgID, convID int32, limit, offset int32) ([]*domain.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	return s.msgRepo.ListByConversation(ctx, orgID, conv.ID, limit, offset)
}
