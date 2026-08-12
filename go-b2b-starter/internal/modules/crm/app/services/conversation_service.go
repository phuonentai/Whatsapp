package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
)

type ConversationService interface {
	ListConversations(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string, view conversationscope.ViewScope, scope conversationscope.Scope) ([]*domain.ConversationWithContact, error)
	GetConversation(ctx context.Context, orgID, convID int32, scope conversationscope.Scope) (*domain.ConversationWithContact, error)
	UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus, scope conversationscope.Scope) (*domain.Conversation, error)
	ListMessages(ctx context.Context, orgID, convID int32, limit, offset int32, scope conversationscope.Scope) ([]*domain.Message, error)
	// UpdateAssignee re-asigna una conversación (permiso inbox:reassign).
	// Devuelve (nil, nil) cuando la conversación está fuera de scope o no
	// existe — el handler responde 404 (sin filtrar existencia).
	UpdateAssignee(ctx context.Context, orgID, convID int32, assignee *string, actorID string, scope conversationscope.Scope) (*domain.Conversation, error)
}

type conversationService struct {
	convRepo    domain.ConversationRepository
	msgRepo     domain.MessageRepository
	contactRepo domain.ContactRepository
}

func NewConversationService(convRepo domain.ConversationRepository, msgRepo domain.MessageRepository, contactRepo domain.ContactRepository) ConversationService {
	return &conversationService{convRepo: convRepo, msgRepo: msgRepo, contactRepo: contactRepo}
}

func (s *conversationService) ListConversations(ctx context.Context, orgID int32, limit, offset int32, statusFilter, channelFilter string, view conversationscope.ViewScope, scope conversationscope.Scope) ([]*domain.ConversationWithContact, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if channelFilter != domain.ChannelWhatsapp && channelFilter != domain.ChannelInstagram {
		channelFilter = ""
	}
	return s.convRepo.ListByOrganization(ctx, orgID, limit, offset, statusFilter, channelFilter, view, scope)
}

func (s *conversationService) GetConversation(ctx context.Context, orgID, convID int32, scope conversationscope.Scope) (*domain.ConversationWithContact, error) {
	conv, err := s.convRepo.GetByID(ctx, orgID, convID, scope)
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

func (s *conversationService) UpdateStatus(ctx context.Context, orgID, convID int32, status domain.ConversationStatus, scope conversationscope.Scope) (*domain.Conversation, error) {
	valid := map[domain.ConversationStatus]bool{
		domain.ConversationStatusActive:   true,
		domain.ConversationStatusClosed:   true,
		domain.ConversationStatusArchived: true,
	}
	if !valid[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID, scope)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	return s.convRepo.UpdateStatus(ctx, orgID, conv.ID, status, scope)
}

func (s *conversationService) ListMessages(ctx context.Context, orgID, convID int32, limit, offset int32, scope conversationscope.Scope) ([]*domain.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID, scope)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	return s.msgRepo.ListByConversation(ctx, orgID, conv.ID, limit, offset)
}

func (s *conversationService) UpdateAssignee(ctx context.Context, orgID, convID int32, assignee *string, actorID string, scope conversationscope.Scope) (*domain.Conversation, error) {
	// Visibilidad previa: fuera de scope o inexistente → error de no
	// encontrado (404 sin filtrar existencia). El UPDATE de assignee queda
	// acotado a la fila ya verificada visible.
	conv, err := s.convRepo.GetByID(ctx, orgID, convID, scope)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	previous := conv.AssigneeStytchMemberID
	updated, err := s.convRepo.UpdateAssignee(ctx, orgID, conv.ID, assignee)
	if err != nil {
		return nil, fmt.Errorf("failed to reassign conversation: %w", err)
	}

	// Audit ledger append-only (actor, origen, destino).
	event := &domain.ConversationEvent{
		OrganizationID:      orgID,
		ConversationID:      conv.ID,
		EventType:           "assigned",
		ActorStytchMemberID: &actorID,
		Payload: map[string]interface{}{
			"from": nullableString(previous),
			"to":   nullableString(assignee),
		},
	}
	if assignee == nil {
		event.EventType = "unassigned"
	}
	if err := s.convRepo.InsertEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to audit reassignment: %w", err)
	}

	return updated, nil
}

func nullableString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
