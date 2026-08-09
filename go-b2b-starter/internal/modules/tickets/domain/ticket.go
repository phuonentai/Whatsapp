package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTicketNotFound       = errors.New("ticket not found")
	ErrInvalidTransition    = errors.New("invalid ticket status transition")
	ErrInvalidPriority      = errors.New("invalid ticket priority")
	ErrTicketModuleDisabled = errors.New("tickets module disabled")
)

type TicketStatus string

const (
	StatusOpen            TicketStatus = "open"
	StatusInProgress      TicketStatus = "in_progress"
	StatusWaitingCustomer TicketStatus = "waiting_customer"
	StatusResolved        TicketStatus = "resolved"
	StatusCancelled       TicketStatus = "cancelled"
)

type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityNormal TicketPriority = "normal"
	PriorityHigh   TicketPriority = "high"
)

// DefaultSLASeconds maps priority to SLA hours (fallback when module config
// does not define sla_hours).
var DefaultSLASeconds = map[TicketPriority]int64{
	PriorityLow:    48 * 3600,
	PriorityNormal: 24 * 3600,
	PriorityHigh:   8 * 3600,
}

// DefaultPriorities is the fallback priority set when module config
// does not define priorities.
var DefaultPriorities = []TicketPriority{PriorityLow, PriorityNormal, PriorityHigh}

// validTransitions defines the ticket state machine.
var validTransitions = map[TicketStatus]map[TicketStatus]bool{
	StatusOpen: {
		StatusInProgress:      true,
		StatusWaitingCustomer: true,
		StatusResolved:        true,
		StatusCancelled:       true,
	},
	StatusInProgress: {
		StatusWaitingCustomer: true,
		StatusResolved:        true,
		StatusCancelled:       true,
	},
	StatusWaitingCustomer: {
		StatusInProgress: true,
		StatusResolved:   true,
		StatusCancelled:  true,
	},
	// resolved and cancelled are terminal.
}

func (s TicketStatus) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusWaitingCustomer, StatusResolved, StatusCancelled:
		return true
	}
	return false
}

func (p TicketPriority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityNormal, PriorityHigh:
		return true
	}
	return false
}

// CanTransition reports whether from -> to is a valid state-machine transition.
func CanTransition(from, to TicketStatus) bool {
	if from == to {
		return true
	}
	if targets, ok := validTransitions[from]; ok {
		return targets[to]
	}
	return false
}

type TicketEventType string

const (
	EventCreated        TicketEventType = "created"
	EventStatusChanged  TicketEventType = "status_changed"
	EventAssigned       TicketEventType = "assigned"
	EventUnassigned     TicketEventType = "unassigned"
	EventPriorityChange TicketEventType = "priority_changed"
	EventNoteInternal   TicketEventType = "note_internal"
	EventTagsChanged    TicketEventType = "tags_changed"
)

type Ticket struct {
	ID                    int32
	OrganizationID        int32
	ContactID             *int32
	ConversationID        *int32
	Title                 string
	Description           string
	Status                TicketStatus
	Priority              TicketPriority
	Tags                  []string
	AssigneeStytchMember  string
	SLADueAt              *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IsOverdue reports whether the ticket missed its SLA deadline and is not
// resolved or cancelled.
func (t *Ticket) IsOverdue(now time.Time) bool {
	if t.SLADueAt == nil {
		return false
	}
	if t.Status == StatusResolved || t.Status == StatusCancelled {
		return false
	}
	return now.After(*t.SLADueAt)
}

func (t *Ticket) String() string {
	return fmt.Sprintf("Ticket{id=%d, org=%d, status=%s, priority=%s}", t.ID, t.OrganizationID, t.Status, t.Priority)
}

type TicketEvent struct {
	ID                int32
	TicketID          int32
	EventType         TicketEventType
	ActorStytchMember string
	Payload           map[string]any
	CreatedAt         time.Time
}

type TicketRepository interface {
	Create(ctx context.Context, ticket *Ticket) (*Ticket, error)
	GetByID(ctx context.Context, orgID, ticketID int32) (*Ticket, error)
	List(ctx context.Context, orgID int32, status, assignee string, limit, offset int32) ([]*Ticket, error)
	UpdateStatus(ctx context.Context, orgID, ticketID int32, status TicketStatus) (*Ticket, error)
	UpdateAssignee(ctx context.Context, orgID, ticketID int32, assignee string) (*Ticket, error)
	UpdatePriority(ctx context.Context, orgID, ticketID int32, priority TicketPriority, slaDueAt *time.Time) (*Ticket, error)
	UpdateTags(ctx context.Context, orgID, ticketID int32, tags []string) (*Ticket, error)
	InsertEvent(ctx context.Context, event *TicketEvent) (*TicketEvent, error)
	ListEvents(ctx context.Context, ticketID int32) ([]*TicketEvent, error)
}
