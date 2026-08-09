package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
)

const ticketsModuleKey = "tickets"

// TicketService implements the tickets helpdesk capability.
type TicketService interface {
	Create(ctx context.Context, orgID int32, req *CreateTicketRequest, actorStytchMember string) (*domain.Ticket, error)
	Get(ctx context.Context, orgID, ticketID int32) (*domain.Ticket, error)
	List(ctx context.Context, orgID int32, status, assignee string, limit, offset int32) ([]*domain.Ticket, error)
	Transition(ctx context.Context, orgID, ticketID int32, to domain.TicketStatus, actorStytchMember string) (*domain.Ticket, error)
	Assign(ctx context.Context, orgID, ticketID int32, assignee string, actorStytchMember string) (*domain.Ticket, error)
	SetPriority(ctx context.Context, orgID, ticketID int32, priority domain.TicketPriority, actorStytchMember string) (*domain.Ticket, error)
	SetTags(ctx context.Context, orgID, ticketID int32, tags []string, actorStytchMember string) (*domain.Ticket, error)
	AddInternalNote(ctx context.Context, orgID, ticketID int32, body string, actorStytchMember string) (*domain.TicketEvent, error)
	ListEvents(ctx context.Context, ticketID int32) ([]*domain.TicketEvent, error)
}

type CreateTicketRequest struct {
	ContactID      *int32
	ConversationID *int32
	Title          string
	Description    string
	Priority       domain.TicketPriority
	Tags           []string
}

type ticketService struct {
	repo          domain.TicketRepository
	moduleService registryServices.ModuleService
	now           func() time.Time
}

func NewTicketService(repo domain.TicketRepository, moduleService registryServices.ModuleService) TicketService {
	return &ticketService{repo: repo, moduleService: moduleService, now: time.Now}
}

// moduleConfig returns the org's tickets module config merged with defaults.
func (s *ticketService) moduleConfig(ctx context.Context, orgID int32) map[string]any {
	modules, orgMods, err := s.moduleService.ListOrgModules(ctx, orgID)
	if err != nil {
		return map[string]any{}
	}
	for i, m := range modules {
		if m.Key != ticketsModuleKey {
			continue
		}
		if i < len(orgMods) && orgMods[i].Config != nil {
			return orgMods[i].Config
		}
	}
	return map[string]any{}
}

// slaFor returns the SLA due-at for a priority based on module config.
func slaFor(config map[string]any, priority domain.TicketPriority, now time.Time) *time.Time {
	seconds, ok := slaSeconds(config, priority)
	if !ok {
		return nil
	}
	due := now.Add(time.Duration(seconds) * time.Second)
	return &due
}

func slaSeconds(config map[string]any, priority domain.TicketPriority) (int64, bool) {
	if raw, ok := config["sla_hours"]; ok {
		if sla, ok := raw.(map[string]any); ok {
			if v, ok := sla[string(priority)]; ok {
				switch n := v.(type) {
				case float64:
					return int64(n) * 3600, true
				case int:
					return int64(n) * 3600, true
				}
			}
		}
	}
	if seconds, ok := domain.DefaultSLASeconds[priority]; ok {
		return seconds, true
	}
	return 0, false
}

func (s *ticketService) Create(ctx context.Context, orgID int32, req *CreateTicketRequest, actor string) (*domain.Ticket, error) {
	config := s.moduleConfig(ctx, orgID)

	priority := req.Priority
	if priority == "" {
		priority = domain.PriorityNormal
	}
	if !priority.IsValid() {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidPriority, priority)
	}
	if !s.priorityAllowed(config, priority) {
		return nil, fmt.Errorf("%w: %s (no permitida en config)", domain.ErrInvalidPriority, priority)
	}

	now := s.now()
	ticket := &domain.Ticket{
		OrganizationID: orgID,
		ContactID:      req.ContactID,
		ConversationID: req.ConversationID,
		Title:          req.Title,
		Description:    req.Description,
		Status:         domain.StatusOpen,
		Priority:       priority,
		Tags:           req.Tags,
		SLADueAt:       slaFor(config, priority, now),
	}
	created, err := s.repo.Create(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	_, err = s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          created.ID,
		EventType:         domain.EventCreated,
		ActorStytchMember: actor,
		Payload: map[string]any{
			"title":     created.Title,
			"priority":  string(created.Priority),
			"sla_due_at": created.SLADueAt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("record ticket created event: %w", err)
	}
	return created, nil
}

func (s *ticketService) Get(ctx context.Context, orgID, ticketID int32) (*domain.Ticket, error) {
	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *ticketService) List(ctx context.Context, orgID int32, status, assignee string, limit, offset int32) ([]*domain.Ticket, error) {
	return s.repo.List(ctx, orgID, status, assignee, limit, offset)
}

func (s *ticketService) Transition(ctx context.Context, orgID, ticketID int32, to domain.TicketStatus, actor string) (*domain.Ticket, error) {
	if !to.IsValid() {
		return nil, fmt.Errorf("invalid ticket status: %s", to)
	}
	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(ticket.Status, to) {
		return nil, fmt.Errorf("%w: %s -> %s", domain.ErrInvalidTransition, ticket.Status, to)
	}
	updated, err := s.repo.UpdateStatus(ctx, orgID, ticketID, to)
	if err != nil {
		return nil, fmt.Errorf("transition ticket: %w", err)
	}
	_, err = s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          ticketID,
		EventType:         domain.EventStatusChanged,
		ActorStytchMember: actor,
		Payload:           map[string]any{"from": string(ticket.Status), "to": string(to)},
	})
	if err != nil {
		return nil, fmt.Errorf("record status change event: %w", err)
	}
	return updated, nil
}

func (s *ticketService) Assign(ctx context.Context, orgID, ticketID int32, assignee string, actor string) (*domain.Ticket, error) {
	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateAssignee(ctx, orgID, ticketID, assignee)
	if err != nil {
		return nil, fmt.Errorf("assign ticket: %w", err)
	}
	eventType := domain.EventAssigned
	if assignee == "" {
		eventType = domain.EventUnassigned
	}
	_, err = s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          ticketID,
		EventType:         eventType,
		ActorStytchMember: actor,
		Payload:           map[string]any{"assignee": assignee, "previous": ticket.AssigneeStytchMember},
	})
	if err != nil {
		return nil, fmt.Errorf("record assignment event: %w", err)
	}
	return updated, nil
}

func (s *ticketService) SetPriority(ctx context.Context, orgID, ticketID int32, priority domain.TicketPriority, actor string) (*domain.Ticket, error) {
	if !priority.IsValid() {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidPriority, priority)
	}
	config := s.moduleConfig(ctx, orgID)
	if !s.priorityAllowed(config, priority) {
		return nil, fmt.Errorf("%w: %s (no permitida en config)", domain.ErrInvalidPriority, priority)
	}
	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	slaDueAt := slaFor(config, priority, s.now())
	updated, err := s.repo.UpdatePriority(ctx, orgID, ticketID, priority, slaDueAt)
	if err != nil {
		return nil, fmt.Errorf("set ticket priority: %w", err)
	}
	_, err = s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          ticketID,
		EventType:         domain.EventPriorityChange,
		ActorStytchMember: actor,
		Payload: map[string]any{
			"from":      string(ticket.Priority),
			"to":        string(priority),
			"sla_due_at": slaDueAt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("record priority change event: %w", err)
	}
	return updated, nil
}

func (s *ticketService) SetTags(ctx context.Context, orgID, ticketID int32, tags []string, actor string) (*domain.Ticket, error) {
	config := s.moduleConfig(ctx, orgID)
	if !s.tagsAllowed(config, tags) {
		return nil, errors.New("ticket tag no permitido en config")
	}
	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateTags(ctx, orgID, ticketID, tags)
	if err != nil {
		return nil, fmt.Errorf("set ticket tags: %w", err)
	}
	_, err = s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          ticketID,
		EventType:         domain.EventTagsChanged,
		ActorStytchMember: actor,
		Payload:           map[string]any{"from": ticket.Tags, "to": tags},
	})
	if err != nil {
		return nil, fmt.Errorf("record tags change event: %w", err)
	}
	return updated, nil
}

func (s *ticketService) AddInternalNote(ctx context.Context, orgID, ticketID int32, body string, actor string) (*domain.TicketEvent, error) {
	if _, err := s.repo.GetByID(ctx, orgID, ticketID); err != nil {
		return nil, err
	}
	event, err := s.repo.InsertEvent(ctx, &domain.TicketEvent{
		TicketID:          ticketID,
		EventType:         domain.EventNoteInternal,
		ActorStytchMember: actor,
		Payload:           map[string]any{"body": body},
	})
	if err != nil {
		return nil, fmt.Errorf("add internal note: %w", err)
	}
	return event, nil
}

func (s *ticketService) ListEvents(ctx context.Context, ticketID int32) ([]*domain.TicketEvent, error) {
	return s.repo.ListEvents(ctx, ticketID)
}

func (s *ticketService) priorityAllowed(config map[string]any, priority domain.TicketPriority) bool {
	raw, ok := config["priorities"]
	if !ok {
		return true
	}
	list, ok := raw.([]any)
	if !ok {
		return true
	}
	for _, item := range list {
		if str, ok := item.(string); ok && domain.TicketPriority(str) == priority {
			return true
		}
	}
	return false
}

func (s *ticketService) tagsAllowed(config map[string]any, tags []string) bool {
	raw, ok := config["tags"]
	if !ok {
		return true
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return true
	}
	allowed := make(map[string]bool, len(list))
	for _, item := range list {
		if str, ok := item.(string); ok {
			allowed[str] = true
		}
	}
	for _, tag := range tags {
		if !allowed[tag] {
			return false
		}
	}
	return true
}
