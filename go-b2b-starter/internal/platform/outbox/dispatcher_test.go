package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...loggerdomain.Fields)            {}
func (noopLogger) Info(string, ...loggerdomain.Fields)             {}
func (noopLogger) Warn(string, ...loggerdomain.Fields)             {}
func (noopLogger) Error(string, ...loggerdomain.Fields)            {}
func (noopLogger) Fatal(string, ...loggerdomain.Fields)            {}
func (noopLogger) WithFields(loggerdomain.Fields) loggerdomain.Logger {
	return noopLogger{}
}

type fakeRepo struct {
	mu            sync.Mutex
	claimQueue    []*OutboxEvent
	dispatched    []int64
	retried       []int64
	deadLettered  []int64
	failNextTimes map[int64]int
}

func (f *fakeRepo) Insert(context.Context, string, json.RawMessage, *int32) (*OutboxEvent, error) {
	return nil, nil
}
func (f *fakeRepo) ClaimPending(_ context.Context, limit int32) ([]*OutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.claimQueue) == 0 {
		return nil, nil
	}
	if int32(len(f.claimQueue)) > limit {
		f.claimQueue = f.claimQueue[limit:]
	}
	out := f.claimQueue
	f.claimQueue = nil
	return out, nil
}
func (f *fakeRepo) MarkDispatched(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dispatched = append(f.dispatched, id)
	return nil
}
func (f *fakeRepo) Retry(_ context.Context, id int64, _ time.Time, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retried = append(f.retried, id)
	return nil
}
func (f *fakeRepo) DeadLetter(_ context.Context, id int64, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deadLettered = append(f.deadLettered, id)
	return nil
}
func (f *fakeRepo) ListDeadLetter(context.Context, int32) ([]*OutboxEvent, error) {
	return nil, nil
}
func (f *fakeRepo) Requeue(context.Context, int64) error { return nil }
func (f *fakeRepo) Prune(context.Context, time.Time) error {
	return nil
}

type failingBus struct {
	failures map[string]int
}

func (b *failingBus) Publish(_ context.Context, e eventbus.Event) error {
	if n := b.failures[e.EventName()]; n > 0 {
		b.failures[e.EventName()] = n - 1
		return errors.New("handler failed")
	}
	return nil
}
func (b *failingBus) Subscribe(string, eventbus.EventHandler[eventbus.Event]) error {
	return nil
}
func (b *failingBus) Unsubscribe(string, eventbus.EventHandler[eventbus.Event]) error {
	return nil
}
func (b *failingBus) Close() error { return nil }

type probeEvent struct {
	eventbus.BaseEvent
}

func newProbeEvent(name string) *probeEvent {
	return &probeEvent{BaseEvent: eventbus.BaseEvent{
		ID:   name + "-id",
		Name: name,
	}}
}

func newDispatcherForTest(t *testing.T, repo *fakeRepo, bus eventbus.EventBus, cfg Config) *Dispatcher {
	t.Helper()
	reg := NewRegistry()
	reg.Register("probe.event", func(payload json.RawMessage) (eventbus.Event, error) {
		var e probeEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}
		return &e, nil
	})
	return NewDispatcher(repo, bus, reg, noopLogger{}, cfg)
}

func testConfig() Config {
	return Config{
		PollInterval:  time.Millisecond,
		MaxAttempts:   3,
		BatchSize:     10,
		Enabled:       true,
		BackoffBase:   time.Millisecond,
		BackoffMax:    10 * time.Millisecond,
		RetentionDays: 14,
	}
}

func eventFixture(id int64, attempts int) *OutboxEvent {
	payload, _ := json.Marshal(newProbeEvent("probe.event"))
	return &OutboxEvent{
		ID:        id,
		EventType: "probe.event",
		Payload:   payload,
		Status:    StatusPending,
		Attempts:  attempts,
	}
}

func TestDispatcher_SuccessMarksDispatched(t *testing.T) {
	repo := &fakeRepo{claimQueue: []*OutboxEvent{eventFixture(1, 0)}}
	d := newDispatcherForTest(t, repo, &failingBus{}, testConfig())

	if err := d.dispatchOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.dispatched) != 1 || repo.dispatched[0] != 1 {
		t.Fatalf("expected event 1 dispatched, got %v", repo.dispatched)
	}
	if len(repo.retried) != 0 || len(repo.deadLettered) != 0 {
		t.Fatalf("unexpected retry/dead-letter state: %v %v", repo.retried, repo.deadLettered)
	}
}

func TestDispatcher_RetriesUntilMaxThenDeadLetters(t *testing.T) {
	repo := &fakeRepo{}
	bus := &failingBus{failures: map[string]int{"probe.event": 100}}
	d := newDispatcherForTest(t, repo, bus, testConfig())

	for i := 0; i < 3; i++ {
		repo.claimQueue = append(repo.claimQueue, eventFixture(1, i))
		if err := d.dispatchOnce(context.Background()); err != nil {
			t.Fatalf("dispatch cycle failed: %v", err)
		}
	}

	// Attempts 0,1 -> retried; attempt 2 (attempts=2, 2+1>=3) -> dead-letter.
	if len(repo.retried) != 2 {
		t.Fatalf("expected 2 retries, got %d", len(repo.retried))
	}
	if len(repo.deadLettered) != 1 || repo.deadLettered[0] != 1 {
		t.Fatalf("expected event 1 dead-lettered after max attempts, got %v", repo.deadLettered)
	}
}

func TestDispatcher_RecoversAfterTransientFailures(t *testing.T) {
	repo := &fakeRepo{claimQueue: []*OutboxEvent{eventFixture(1, 0)}}
	bus := &failingBus{failures: map[string]int{"probe.event": 1}}
	d := newDispatcherForTest(t, repo, bus, testConfig())

	if err := d.dispatchOnce(context.Background()); err != nil {
		t.Fatalf("dispatch cycle failed: %v", err)
	}
	if len(repo.retried) != 1 {
		t.Fatalf("expected 1 retry after failed attempt, got %d", len(repo.retried))
	}
	repo.claimQueue = append(repo.claimQueue, eventFixture(1, 1))
	if err := d.dispatchOnce(context.Background()); err != nil {
		t.Fatalf("expected second attempt to succeed: %v", err)
	}
	if len(repo.dispatched) != 1 {
		t.Fatalf("expected event dispatched after recovery, got %v", repo.dispatched)
	}
}

func TestDispatcher_UnknownEventTypeDeadLetters(t *testing.T) {
	payload, _ := json.Marshal(newProbeEvent("probe.event"))
	unknown := &OutboxEvent{ID: 7, EventType: "no.such.event", Payload: payload, Status: StatusPending}
	repo := &fakeRepo{claimQueue: []*OutboxEvent{unknown}}
	d := newDispatcherForTest(t, repo, &failingBus{}, testConfig())

	if err := d.dispatchOnce(context.Background()); err != nil {
		t.Fatalf("dispatch cycle failed: %v", err)
	}
	if len(repo.deadLettered) != 1 || repo.deadLettered[0] != 7 {
		t.Fatalf("expected unknown event dead-lettered, got %v", repo.deadLettered)
	}
}

func TestDispatcher_DisabledDoesNotDispatch(t *testing.T) {
	repo := &fakeRepo{claimQueue: []*OutboxEvent{eventFixture(1, 0)}}
	cfg := testConfig()
	cfg.Enabled = false
	d := newDispatcherForTest(t, repo, &failingBus{}, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	d.Start(ctx)
	time.Sleep(5 * time.Millisecond)

	if len(repo.dispatched) != 0 {
		t.Fatalf("disabled dispatcher must not dispatch, got %v", repo.dispatched)
	}
}

func TestDispatcher_BackoffGrowsAndCaps(t *testing.T) {
	d := newDispatcherForTest(t, &fakeRepo{}, &failingBus{}, testConfig())

	b0 := d.backoff(0)
	b1 := d.backoff(1)
	b2 := d.backoff(2)
	b5 := d.backoff(5)

	if b0 > b1 || b1 > b2 {
		t.Fatalf("expected growing backoff, got %v %v %v", b0, b1, b2)
	}
	if b5 > d.cfg.BackoffMax {
		t.Fatalf("expected backoff capped at max, got %v (max %v)", b5, d.cfg.BackoffMax)
	}
}

func TestDispatcher_ResumesPendingAfterRestart(t *testing.T) {
	// Pending events committed before a crash are claimed again on boot.
	repo := &fakeRepo{claimQueue: []*OutboxEvent{eventFixture(42, 0)}}
	d := newDispatcherForTest(t, repo, &failingBus{}, testConfig())

	if err := d.dispatchOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.dispatched) != 1 || repo.dispatched[0] != 42 {
		t.Fatalf("expected pending event 42 resumed and dispatched, got %v", repo.dispatched)
	}
}
