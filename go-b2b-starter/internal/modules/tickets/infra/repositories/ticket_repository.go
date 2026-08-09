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
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
)

type ticketRepository struct {
	store sqlc.Store
}

func NewTicketRepository(store sqlc.Store) domain.TicketRepository {
	return &ticketRepository{store: store}
}

func (r *ticketRepository) Create(ctx context.Context, ticket *domain.Ticket) (*domain.Ticket, error) {
	row, err := r.store.CreateTicket(ctx, sqlc.CreateTicketParams{
		OrganizationID:       ticket.OrganizationID,
		ContactID:            helpers.ToPgInt4Ptr(ticket.ContactID),
		ConversationID:       helpers.ToPgInt4Ptr(ticket.ConversationID),
		Title:                ticket.Title,
		Description:          helpers.ToPgText(ticket.Description),
		Status:               string(ticket.Status),
		Priority:             string(ticket.Priority),
		Tags:                 toJSONB(ticket.Tags),
		AssigneeStytchMemberID: helpers.ToPgText(ticket.AssigneeStytchMember),
		SlaDueAt:               helpers.ToPgTimestamptzPtr(ticket.SLADueAt),
	})
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) GetByID(ctx context.Context, orgID, ticketID int32) (*domain.Ticket, error) {
	row, err := r.store.GetTicketByID(ctx, sqlc.GetTicketByIDParams{ID: ticketID, OrganizationID: orgID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) List(ctx context.Context, orgID int32, status, assignee string, limit, offset int32) ([]*domain.Ticket, error) {
	rows, err := r.store.ListTicketsByOrg(ctx, sqlc.ListTicketsByOrgParams{
		OrganizationID: orgID,
		Column2:        status,
		Column3:        assignee,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	tickets := make([]*domain.Ticket, len(rows))
	for i := range rows {
		tickets[i] = mapTicket(&rows[i])
	}
	return tickets, nil
}

func (r *ticketRepository) UpdateStatus(ctx context.Context, orgID, ticketID int32, status domain.TicketStatus) (*domain.Ticket, error) {
	row, err := r.store.UpdateTicketStatus(ctx, sqlc.UpdateTicketStatusParams{
		ID:             ticketID,
		OrganizationID: orgID,
		Status:         string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("update ticket status: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) UpdateAssignee(ctx context.Context, orgID, ticketID int32, assignee string) (*domain.Ticket, error) {
	row, err := r.store.UpdateTicketAssignee(ctx, sqlc.UpdateTicketAssigneeParams{
		ID:                     ticketID,
		OrganizationID:         orgID,
		AssigneeStytchMemberID: helpers.ToPgText(assignee),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("update ticket assignee: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) UpdatePriority(ctx context.Context, orgID, ticketID int32, priority domain.TicketPriority, slaDueAt *time.Time) (*domain.Ticket, error) {
	row, err := r.store.UpdateTicketPriority(ctx, sqlc.UpdateTicketPriorityParams{
		ID:             ticketID,
		OrganizationID: orgID,
		Priority:       string(priority),
		SlaDueAt:       helpers.ToPgTimestamptzPtr(slaDueAt),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("update ticket priority: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) UpdateTags(ctx context.Context, orgID, ticketID int32, tags []string) (*domain.Ticket, error) {
	row, err := r.store.UpdateTicketTags(ctx, sqlc.UpdateTicketTagsParams{
		ID:             ticketID,
		OrganizationID: orgID,
		Tags:           toJSONB(tags),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("update ticket tags: %w", err)
	}
	return mapTicket(&row), nil
}

func (r *ticketRepository) InsertEvent(ctx context.Context, event *domain.TicketEvent) (*domain.TicketEvent, error) {
	row, err := r.store.InsertTicketEvent(ctx, sqlc.InsertTicketEventParams{
		TicketID:            event.TicketID,
		EventType:           string(event.EventType),
		ActorStytchMemberID: helpers.ToPgText(event.ActorStytchMember),
		Payload:             helpers.ToJSONB(event.Payload),
	})
	if err != nil {
		return nil, fmt.Errorf("insert ticket event: %w", err)
	}
	return mapTicketEvent(&row), nil
}

func (r *ticketRepository) ListEvents(ctx context.Context, ticketID int32) ([]*domain.TicketEvent, error) {
	rows, err := r.store.ListTicketEvents(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list ticket events: %w", err)
	}
	events := make([]*domain.TicketEvent, len(rows))
	for i := range rows {
		events[i] = mapTicketEvent(&rows[i])
	}
	return events, nil
}

func mapTicket(t *sqlc.CrmTicket) *domain.Ticket {
	return &domain.Ticket{
		ID:                   t.ID,
		OrganizationID:       t.OrganizationID,
		ContactID:            helpers.FromPgInt4Ptr(t.ContactID),
		ConversationID:       helpers.FromPgInt4Ptr(t.ConversationID),
		Title:                t.Title,
		Description:          helpers.FromPgText(t.Description),
		Status:               domain.TicketStatus(t.Status),
		Priority:             domain.TicketPriority(t.Priority),
		Tags:                 fromJSONBStringSlice(t.Tags),
		AssigneeStytchMember: helpers.FromPgText(t.AssigneeStytchMemberID),
		SLADueAt:             helpers.FromPgTimestamptzPtr(t.SlaDueAt),
		CreatedAt:            t.CreatedAt.Time,
		UpdatedAt:            t.UpdatedAt.Time,
	}
}

func mapTicketEvent(e *sqlc.CrmTicketEvent) *domain.TicketEvent {
	return &domain.TicketEvent{
		ID:                e.ID,
		TicketID:          e.TicketID,
		EventType:         domain.TicketEventType(e.EventType),
		ActorStytchMember: helpers.FromPgText(e.ActorStytchMemberID),
		Payload:           helpers.FromJSONB(e.Payload),
		CreatedAt:         e.CreatedAt.Time,
	}
}

func toJSONB(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func fromJSONBStringSlice(b []byte) []string {
	var result []string
	_ = json.Unmarshal(b, &result)
	return result
}
