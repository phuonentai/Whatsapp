package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	registryDomain "github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type fakeOrgModuleRepo struct {
	orgMods []*registryDomain.OrganizationModule
}

func (f *fakeOrgModuleRepo) ListByOrg(ctx context.Context, orgID int32) ([]*registryDomain.OrganizationModule, error) {
	return f.orgMods, nil
}
func (f *fakeOrgModuleRepo) GetByKey(ctx context.Context, orgID int32, moduleKey string) (*registryDomain.OrganizationModule, error) {
	for _, om := range f.orgMods {
		if om.ModuleKey == moduleKey {
			return om, nil
		}
	}
	return nil, nil
}
func (f *fakeOrgModuleRepo) UpsertConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*registryDomain.OrganizationModule, error) {
	return &registryDomain.OrganizationModule{OrganizationID: orgID, ModuleKey: moduleKey, Config: config}, nil
}
func (f *fakeOrgModuleRepo) Delete(ctx context.Context, orgID int32, moduleKey string) error { return nil }

type fakeModuleRepo struct {
	modules []*registryDomain.Module
}

func (f *fakeModuleRepo) ListActive(ctx context.Context) ([]*registryDomain.Module, error) { return f.modules, nil }
func (f *fakeModuleRepo) GetByKey(ctx context.Context, key string) (*registryDomain.Module, error) {
	for _, m := range f.modules {
		if m.Key == key {
			return m, nil
		}
	}
	return nil, domain.ErrTicketNotFound
}

type fakeTicketRepo struct {
	tickets map[int32]*domain.Ticket
	events  map[int32][]*domain.TicketEvent
	nextID  int32
}

func newFakeTicketRepo() *fakeTicketRepo {
	return &fakeTicketRepo{tickets: map[int32]*domain.Ticket{}, events: map[int32][]*domain.TicketEvent{}, nextID: 1}
}

func (f *fakeTicketRepo) Create(ctx context.Context, t *domain.Ticket) (*domain.Ticket, error) {
	t.ID = f.nextID
	f.nextID++
	f.tickets[t.ID] = t
	return t, nil
}
func (f *fakeTicketRepo) GetByID(ctx context.Context, orgID, id int32) (*domain.Ticket, error) {
	t, ok := f.tickets[id]
	if !ok || t.OrganizationID != orgID {
		return nil, domain.ErrTicketNotFound
	}
	return t, nil
}
func (f *fakeTicketRepo) List(ctx context.Context, orgID int32, status, assignee string, limit, offset int32) ([]*domain.Ticket, error) {
	return nil, nil
}
func (f *fakeTicketRepo) UpdateStatus(ctx context.Context, orgID, id int32, status domain.TicketStatus) (*domain.Ticket, error) {
	t, err := f.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	t.Status = status
	return t, nil
}
func (f *fakeTicketRepo) UpdateAssignee(ctx context.Context, orgID, id int32, assignee string) (*domain.Ticket, error) {
	t, err := f.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	t.AssigneeStytchMember = assignee
	return t, nil
}
func (f *fakeTicketRepo) UpdatePriority(ctx context.Context, orgID, id int32, priority domain.TicketPriority, sla *time.Time) (*domain.Ticket, error) {
	t, err := f.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	t.Priority = priority
	t.SLADueAt = sla
	return t, nil
}
func (f *fakeTicketRepo) UpdateTags(ctx context.Context, orgID, id int32, tags []string) (*domain.Ticket, error) {
	t, err := f.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	t.Tags = tags
	return t, nil
}
func (f *fakeTicketRepo) InsertEvent(ctx context.Context, e *domain.TicketEvent) (*domain.TicketEvent, error) {
	f.events[e.TicketID] = append(f.events[e.TicketID], e)
	return e, nil
}
func (f *fakeTicketRepo) ListEvents(ctx context.Context, ticketID int32) ([]*domain.TicketEvent, error) {
	return f.events[ticketID], nil
}

type nilLogger struct{}

func (n nilLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (n nilLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (n nilLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (n nilLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (n nilLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (n nilLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return n
}

func newTestService(t *testing.T, config map[string]any) (TicketService, *fakeTicketRepo, *fakeModuleRepo) {
	t.Helper()
	fakeRepo := newFakeTicketRepo()
	fakeMod := &fakeModuleRepo{modules: []*registryDomain.Module{{
		Key: "tickets", Name: "Tickets", GrantedFeatures: []string{"tickets_module"},
	}}}
	orgRepo := &fakeOrgModuleRepo{}
	if config != nil {
		orgRepo.orgMods = []*registryDomain.OrganizationModule{{OrganizationID: 1, ModuleKey: "tickets", Config: config}}
	}
	ms := registryServices.NewModuleService(fakeMod, orgRepo, nilLogger{})
	svc := NewTicketService(fakeRepo, ms)
	return svc, fakeRepo, fakeMod
}

func TestCreateTicket_RecordsCreatedEvent(t *testing.T) {
	svc, repo, _ := newTestService(t, nil)
	contact := int32(5)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{
		ContactID: &contact, Title: "Problema con factura", Priority: domain.PriorityHigh,
	}, "member-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusOpen, ticket.Status)
	assert.Equal(t, domain.PriorityHigh, ticket.Priority)
	require.NotNil(t, ticket.SLADueAt) // default SLA high=8h
	assert.WithinDuration(t, time.Now().Add(8*time.Hour), *ticket.SLADueAt, time.Minute)

	events := repo.events[ticket.ID]
	require.Len(t, events, 1)
	assert.Equal(t, domain.EventCreated, events[0].EventType)
	assert.Equal(t, "member-1", events[0].ActorStytchMember)
}

func TestTransition_ValidAndInvalid(t *testing.T) {
	svc, _, _ := newTestService(t, nil)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T"}, "member-1")
	require.NoError(t, err)

	// open -> in_progress is valid.
	updated, err := svc.Transition(context.Background(), 1, ticket.ID, domain.StatusInProgress, "member-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusInProgress, updated.Status)

	// resolved is terminal; resolved -> open must be rejected.
	_, err = svc.Transition(context.Background(), 1, ticket.ID, domain.StatusResolved, "member-1")
	require.NoError(t, err)
	_, err = svc.Transition(context.Background(), 1, ticket.ID, domain.StatusOpen, "member-1")
	require.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestTransition_RecordsEvent(t *testing.T) {
	svc, repo, _ := newTestService(t, nil)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T"}, "member-1")
	require.NoError(t, err)
	_, err = svc.Transition(context.Background(), 1, ticket.ID, domain.StatusWaitingCustomer, "member-1")
	require.NoError(t, err)

	var found bool
	for _, e := range repo.events[ticket.ID] {
		if e.EventType == domain.EventStatusChanged && e.Payload["to"] == "waiting_customer" {
			found = true
		}
	}
	assert.True(t, found, "expected status_changed event")
}

func TestSetPriority_ReArmsSLAFromConfig(t *testing.T) {
	config := map[string]any{"sla_hours": map[string]any{"high": float64(2)}}
	svc, _, _ := newTestService(t, config)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T", Priority: domain.PriorityHigh}, "m")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(2*time.Hour), *ticket.SLADueAt, time.Minute)
}

func TestSetPriority_RejectsUnconfiguredPriority(t *testing.T) {
	config := map[string]any{"priorities": []any{"low", "normal"}}
	svc, _, _ := newTestService(t, config)
	_, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T", Priority: domain.PriorityHigh}, "m")
	require.ErrorIs(t, err, domain.ErrInvalidPriority)
}

func TestAssign_RecordsAssignmentEvent(t *testing.T) {
	svc, repo, _ := newTestService(t, nil)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T"}, "m")
	require.NoError(t, err)
	_, err = svc.Assign(context.Background(), 1, ticket.ID, "member-42", "m")
	require.NoError(t, err)

	events := repo.events[ticket.ID]
	require.Len(t, events, 2)
	assert.Equal(t, domain.EventAssigned, events[1].EventType)
	assert.Equal(t, "member-42", events[1].Payload["assignee"])
}

func TestAddInternalNote_IsTeamOnlyEvent(t *testing.T) {
	svc, repo, _ := newTestService(t, nil)
	ticket, err := svc.Create(context.Background(), 1, &CreateTicketRequest{Title: "T"}, "m")
	require.NoError(t, err)
	event, err := svc.AddInternalNote(context.Background(), 1, ticket.ID, "nota interna", "m")
	require.NoError(t, err)
	assert.Equal(t, domain.EventNoteInternal, event.EventType)
	assert.Equal(t, "nota interna", event.Payload["body"])
	assert.Len(t, repo.events[ticket.ID], 2) // created + note
}
