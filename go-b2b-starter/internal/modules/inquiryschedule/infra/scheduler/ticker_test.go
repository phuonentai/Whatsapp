package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// TestScheduler covers the claim ticker (group 4): concurrent tickers never
// double-claim a due schedule, and a crash before commit re-fires on the next
// tick (at-least-once).

type stubLogger struct{}

func (stubLogger) Debug(string, ...logger.Fields)         {}
func (stubLogger) Info(string, ...logger.Fields)          {}
func (stubLogger) Warn(string, ...logger.Fields)          {}
func (stubLogger) Error(string, ...logger.Fields)         {}
func (stubLogger) Fatal(string, ...logger.Fields)         {}
func (stubLogger) WithFields(logger.Fields) logger.Logger { return stubLogger{} }

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

// inMemoryRepo emulates the FOR UPDATE SKIP LOCKED claim over a slice of due
// schedules; every successful claim records one inquiry_run.scheduled event
// and advances the schedule.
type inMemoryRepo struct {
	mu        sync.Mutex
	due       map[int32]*domain.Schedule
	events    int
	crashOnce bool // simulate crash-before-commit on the first claim
	failOnce  bool
}

func newInMemoryRepo(schedules ...*domain.Schedule) *inMemoryRepo {
	r := &inMemoryRepo{due: map[int32]*domain.Schedule{}}
	for _, s := range schedules {
		r.due[s.ID] = s
	}
	return r
}

func (r *inMemoryRepo) ClaimAndAdvanceAndEnqueue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failOnce {
		r.failOnce = false
		return nil, nil // crash-before-commit: no event, no advance
	}
	var claimed []*domain.Schedule
	for id, s := range r.due {
		if int32(len(claimed)) >= limit {
			break
		}
		if !s.IsActive || s.NextRunAt.After(time.Now()) {
			continue
		}
		r.events++
		s.NextRunAt = s.NextRunAt.Add(24 * time.Hour) // advance
		claimed = append(claimed, s)
		delete(r.due, id)
	}
	return claimed, nil
}

func (r *inMemoryRepo) eventCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.events
}

// The mock implements the rest of the interface as no-ops (not exercised).
func (r *inMemoryRepo) Create(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	return s, nil
}
func (r *inMemoryRepo) Get(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return nil, domain.ErrScheduleNotFound
}
func (r *inMemoryRepo) GetForUpdate(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return r.Get(ctx, orgID, id)
}
func (r *inMemoryRepo) List(ctx context.Context, orgID int32) ([]*domain.Schedule, error) { return nil, nil }
func (r *inMemoryRepo) Update(ctx context.Context, orgID int32, s *domain.Schedule) (*domain.Schedule, error) {
	return s, nil
}
func (r *inMemoryRepo) Delete(ctx context.Context, orgID, id int32) error { return nil }
func (r *inMemoryRepo) Pause(ctx context.Context, orgID, id int32) (*domain.Schedule, error) {
	return nil, nil
}
func (r *inMemoryRepo) Resume(ctx context.Context, orgID, id int32, next time.Time) (*domain.Schedule, error) {
	return nil, nil
}
func (r *inMemoryRepo) ClaimDue(ctx context.Context, limit int32) ([]*domain.Schedule, error) {
	return nil, nil
}
func (r *inMemoryRepo) MarkFiredOccurrence(ctx context.Context, orgID, id int32, occurrence time.Time) (*domain.Schedule, error) {
	return nil, nil
}
func (r *inMemoryRepo) ListWithStatus(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error) {
	return nil, nil
}
func (r *inMemoryRepo) RecentRuns(ctx context.Context, orgID, id int32, limit int32) ([]*domain.ScheduledRun, error) {
	return nil, nil
}
func (r *inMemoryRepo) CountOverdueRecipients(ctx context.Context, orgID, id int32) (int32, error) {
	return 0, nil
}

func dueSchedule(id, orgID int32) *domain.Schedule {
	return &domain.Schedule{
		ID:             id,
		OrganizationID: orgID,
		Name:           "Matinal",
		RunTime:        "08:00",
		DaysOfWeek:     domain.DaysOfWeek{domain.Monday, domain.Tuesday, domain.Wednesday, domain.Thursday, domain.Friday},
		ProductIDs:     []int32{1},
		SupplierIDs:    []int32{2},
		IsActive:       true,
		NextRunAt:      time.Now().Add(-time.Minute),
	}
}

func TestScheduler_ConcurrentTickersClaimExactlyOnce(t *testing.T) {
	repo := newInMemoryRepo(dueSchedule(1, 10))
	ticker := NewScheduleTicker(repo, fakeClock{now: time.Now()}, stubLogger{})

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ticker.TickOnce(context.Background())
		}()
	}
	wg.Wait()

	if got := repo.eventCount(); got != 1 {
		t.Fatalf("events = %d, want 1 (concurrent tickers must not double-claim)", got)
	}
}

func TestScheduler_CrashBeforeCommitRefiresOnNextTick(t *testing.T) {
	repo := newInMemoryRepo(dueSchedule(1, 10))
	repo.failOnce = true // first tick crashes before commit (no event)
	ticker := NewScheduleTicker(repo, fakeClock{now: time.Now()}, stubLogger{})

	if err := ticker.TickOnce(context.Background()); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if got := repo.eventCount(); got != 0 {
		t.Fatalf("events after crashed tick = %d, want 0", got)
	}
	if err := ticker.TickOnce(context.Background()); err != nil {
		t.Fatalf("tick 2: %v", err)
	}
	if got := repo.eventCount(); got != 1 {
		t.Fatalf("events after re-fire = %d, want 1 (at-least-once)", got)
	}
}

func TestScheduler_PausedOrNotDueSchedulesNeverClaimed(t *testing.T) {
	paused := dueSchedule(1, 10)
	paused.IsActive = false
	notDue := dueSchedule(2, 10)
	notDue.NextRunAt = time.Now().Add(time.Hour)
	repo := newInMemoryRepo(paused, notDue)
	ticker := NewScheduleTicker(repo, fakeClock{now: time.Now()}, stubLogger{})

	if err := ticker.TickOnce(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if got := repo.eventCount(); got != 0 {
		t.Fatalf("events = %d, want 0 (paused/not-due schedules skipped)", got)
	}
}
