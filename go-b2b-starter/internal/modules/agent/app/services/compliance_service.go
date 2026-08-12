package services

import (
	"context"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
)

// complianceService implements ComplianceService (Ley 1581 export/forget).
type complianceService struct {
	repo domain.AgentRepository
}

// NewComplianceService creates the compliance service.
func NewComplianceService(repo domain.AgentRepository) ComplianceService {
	return &complianceService{repo: repo}
}

func (s *complianceService) ExportContact(ctx context.Context, orgID, contactID int32) (*ExportBundle, error) {
	contact, err := s.repo.GetContactRef(ctx, orgID, contactID)
	if err != nil {
		return nil, err
	}

	bundle := &ExportBundle{
		Contact:       mapContactExport(contact, contact.ConsentStatus == domain.ConsentWithdrawn),
		Conversations: []*ConversationExport{},
	}

	// Compliance export/forget es org:manage (admin) → scope org-wide.
	conversations, err := s.repo.ListConversationsByContact(ctx, orgID, contactID, conversationscope.Scope{ViewAll: true})
	if err != nil {
		return nil, err
	}

	for _, conv := range conversations {
		conversationExport := &ConversationExport{
			ID:       conv.ID,
			Status:   conv.Status,
			Messages: []*MessageExport{},
		}
		messages, err := s.repo.ListMessagesByConversation(ctx, orgID, conv.ID, 500, 0)
		if err != nil {
			return nil, err
		}
		for _, msg := range messages {
			conversationExport.Messages = append(conversationExport.Messages, &MessageExport{
				Direction: msg.Direction,
				Type:      msg.MessageType,
				Content:   msg.Content,
				Status:    msg.Status,
				CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		bundle.Conversations = append(bundle.Conversations, conversationExport)
	}

	return bundle, nil
}

func (s *complianceService) ForgetContact(ctx context.Context, orgID, contactID int32) error {
	// Idempotent: re-running anonymization changes nothing further.
	return s.repo.AnonymizeContact(ctx, orgID, contactID)
}

// mapContactExport masks PII when consent is withdrawn.
func mapContactExport(c *domain.ContactRef, mask bool) *ContactExport {
	export := &ContactExport{
		PhoneNumber:     c.PhoneNumber,
		DisplayName:     c.DisplayName,
		Email:           c.Email,
		TipoDocumento:   c.TipoDocumento,
		NumeroDocumento: c.NumeroDocumento,
		ConsentStatus:   string(c.ConsentStatus),
	}
	if c.ConsentedAt != nil {
		export.ConsentedAt = c.ConsentedAt.UTC().Format(time.RFC3339)
	}
	if mask {
		export.PhoneNumber = "[ELIMINADO]"
		export.DisplayName = "[ANONIMIZADO]"
		export.Email = ""
		export.TipoDocumento = ""
		export.NumeroDocumento = ""
	}
	if export.DisplayName == "" {
		export.DisplayName = export.PhoneNumber
	}
	if export.PhoneNumber == "" {
		export.PhoneNumber = "[SIN_DATO]"
	}
	return export
}
