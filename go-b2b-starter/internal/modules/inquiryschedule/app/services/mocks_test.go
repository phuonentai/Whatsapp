package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ---------- logger stub ----------

type stubLogger struct{}

func (stubLogger) Debug(string, ...logger.Fields)         {}
func (stubLogger) Info(string, ...logger.Fields)          {}
func (stubLogger) Warn(string, ...logger.Fields)          {}
func (stubLogger) Error(string, ...logger.Fields)         {}
func (stubLogger) Fatal(string, ...logger.Fields)         {}
func (stubLogger) WithFields(logger.Fields) logger.Logger { return stubLogger{} }

func testLogger() logger.Logger { return stubLogger{} }

// ---------- KillSwitchReader mock ----------

type mockKillSwitch struct {
	mu       sync.Mutex
	enabled  bool
	checked  int
}

func (m *mockKillSwitch) IsKillSwitchEnabled(ctx context.Context, orgID int32) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checked++
	return m.enabled, nil
}

// ---------- AuditLogWriter mock ----------

type mockAudit struct {
	mu      sync.Mutex
	entries []domain.AuditEvent
}

func (m *mockAudit) Record(ctx context.Context, event domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, event)
	return nil
}

func (m *mockAudit) actions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, e := range m.entries {
		out = append(out, e.Action)
	}
	return out
}

func (m *mockAudit) skips() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, e := range m.entries {
		if e.Action == "skip" && e.Reason != nil {
			out = append(out, *e.Reason)
		}
	}
	return out
}

// ---------- InquiryRunCreator mock ----------

type mockCreator struct {
	mu         sync.Mutex
	calls      int
	result     *domain.ScheduledRunResult
	err        error
	lastInput  domain.CreateScheduledRunInput
	lastRunIDs []int32
}

func (m *mockCreator) CreateScheduledRun(ctx context.Context, in domain.CreateScheduledRunInput) (*domain.ScheduledRunResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastInput = in
	if m.err != nil {
		return nil, m.err
	}
	res := m.result
	if res == nil {
		res = &domain.ScheduledRunResult{RunID: 42, DraftedCount: 2}
	}
	return res, nil
}

// ---------- RecipientStateReader mock ----------

type mockRecipientReader struct {
	mu         sync.Mutex
	candidates []*domain.FollowUpCandidate
	targets    map[int32]*domain.FollowUpCandidate
	active     []domain.RecipientRef
}

func (m *mockRecipientReader) ListFollowUpCandidates(ctx context.Context, orgID int32, limit int32) ([]*domain.FollowUpCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*domain.FollowUpCandidate(nil), m.candidates...), nil
}

func (m *mockRecipientReader) ListOverdueRecipientsForRun(ctx context.Context, orgID, runID int32) ([]*domain.FollowUpCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*domain.FollowUpCandidate
	for _, c := range m.candidates {
		if c.RunID == runID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *mockRecipientReader) ListOverdueRecipientsForContact(ctx context.Context, orgID, contactID int32) ([]*domain.FollowUpCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*domain.FollowUpCandidate
	for _, c := range m.candidates {
		if c.ContactID == contactID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *mockRecipientReader) ActiveRecipientsByPhone(ctx context.Context, orgID int32, phoneNumber string) ([]domain.RecipientRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.RecipientRef(nil), m.active...), nil
}

func (m *mockRecipientReader) GetFollowUpTarget(ctx context.Context, orgID, recipientID int32) (*domain.FollowUpCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.targets[recipientID]; ok {
		return t, nil
	}
	return nil, domain.ErrRecipientNotFound
}

// ---------- FollowUpEnqueuer mock (atomic guard) ----------

type mockEnqueuer struct {
	mu       sync.Mutex
	events   []domain.OutboxEventInput
	limitHit map[int32]int // recipient -> nudge budget remaining
	guard    func(orgID, recipientID int32) (bool, error)
}

func (m *mockEnqueuer) EnqueueNudge(ctx context.Context, orgID, recipientID int32, maxNudges int32, event domain.OutboxEventInput) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.guard != nil {
		ok, err := m.guard(orgID, recipientID)
		if err != nil {
			return false, err
		}
		if ok {
			m.events = append(m.events, event)
		}
		return ok, nil
	}
	m.events = append(m.events, event)
	return true, nil
}

func (m *mockEnqueuer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func (m *mockEnqueuer) messages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, e := range m.events {
		var ev inqEvents.FollowupSend
		if err := json.Unmarshal(e.Payload, &ev); err == nil {
			out = append(out, ev.Message)
		}
	}
	return out
}

// ---------- FollowUpSender mock ----------

type mockSender struct {
	mu      sync.Mutex
	sent    []*domain.FollowUpSend
	err     error
}

func (m *mockSender) SendFollowUp(ctx context.Context, orgID int32, send *domain.FollowUpSend) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	m.sent = append(m.sent, send)
	return "wamid.mock", nil
}

// ---------- ScheduleRepository mock (ticker tests) ----------

type mockScheduleRepo struct {
	mu         sync.Mutex
	due        []*domain.Schedule
	events     []inqEvents.InquiryRunScheduled
	claimErr   error
	advanceErr error
}

func (m *mockScheduleRepo) setDue(schedules ...*domain.Schedule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.due = append([]*domain.Schedule(nil), schedules...)
}

// ClaimAndAdvanceAndEnqueue emulates the FOR UPDATE SKIP LOCKED claim: each
// call claims up to the limit due schedules and records one event each;
// concurrent callers see disjoint due sets.
func (m *mockScheduleRepo) ClaimAndAdvanceAndEnqueue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.claimErr != nil {
		return nil, m.claimErr
	}
	n := int(limit)
	if n > len(m.due) {
		n = len(m.due)
	}
	claimed := m.due[:n]
	m.due = m.due[n:]
	for _, s := range claimed {
		m.events = append(m.events, *inqEvents.NewInquiryRunScheduled(
			s.ID, s.OrganizationID, s.ProductIDs, s.SupplierIDs, s.Note, time.Now()))
	}
	return claimed, nil
}

func (m *mockScheduleRepo) eventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// The mock only needs the claim method for ticker tests; the remaining
// interface methods are no-ops/panics that tests never call.
func (m *mockScheduleRepo) Create(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	return s, nil
}
func (m *mockScheduleRepo) Get(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return nil, domain.ErrScheduleNotFound
}
func (m *mockScheduleRepo) GetForUpdate(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return m.Get(ctx, orgID, id)
}
func (m *mockScheduleRepo) List(ctx context.Context, orgID int32) ([]*domain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) Update(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	return s, nil
}
func (m *mockScheduleRepo) Delete(ctx context.Context, orgID, id int32) error { return nil }
func (m *mockScheduleRepo) Pause(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) Resume(ctx context.Context, orgID, id int32, next time.Time) (*domain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) ClaimDue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*domain.Schedule(nil), m.due...), nil
}
func (m *mockScheduleRepo) MarkFiredOccurrence(ctx context.Context, orgID, id int32, occurrence time.Time) (*domain.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) ListWithStatus(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error) {
	return nil, nil
}
func (m *mockScheduleRepo) RecentRuns(ctx context.Context, orgID, id int32, limit int32) ([]*domain.ScheduledRun, error) {
	return nil, nil
}
func (m *mockScheduleRepo) CountOverdueRecipients(ctx context.Context, orgID, id int32) (int32, error) {
	return 0, nil
}
