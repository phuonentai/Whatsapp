package outbox

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// Registry maps event types to payload codecs. Modules register their own
// event types so the platform package stays free of app-level imports.
type Registry struct {
	mu     sync.RWMutex
	codecs map[string]EventCodec
}

// NewRegistry creates an empty event codec registry.
func NewRegistry() *Registry {
	return &Registry{codecs: make(map[string]EventCodec)}
}

// Register binds a codec to an event type.
func (r *Registry) Register(eventType string, codec EventCodec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codecs[eventType] = codec
}

// Codec returns the codec registered for an event type.
func (r *Registry) Codec(eventType string) (EventCodec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	codec, ok := r.codecs[eventType]
	return codec, ok
}

// Dispatcher polls the outbox and delivers pending events through the
// platform event bus with retry and backoff, dead-lettering on exhaustion.
type Dispatcher struct {
	repo     Repository
	bus      eventbus.EventBus
	registry *Registry
	logger   loggerdomain.Logger
	cfg      Config
	lastPrune time.Time
}

// NewDispatcher builds a dispatcher over the outbox repository.
func NewDispatcher(repo Repository, bus eventbus.EventBus, registry *Registry, log loggerdomain.Logger, cfg Config) *Dispatcher {
	return &Dispatcher{
		repo:      repo,
		bus:       bus,
		registry:  registry,
		logger:    log,
		cfg:       cfg,
		lastPrune: time.Now(),
	}
}

// Start runs the dispatch loop in a background goroutine until ctx is done.
// Pending events committed before a crash are resumed on the next boot.
func (d *Dispatcher) Start(ctx context.Context) {
	if !d.cfg.Enabled {
		d.logger.Info("outbox dispatcher disabled", nil)
		return
	}
	go d.run(ctx)
}

func (d *Dispatcher) run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("outbox dispatcher stopped", nil)
			return
		case <-ticker.C:
			if err := d.dispatchOnce(ctx); err != nil {
				d.logger.Error("outbox dispatch cycle failed", loggerdomain.Fields{"error": err.Error()})
			}
			d.maybePrune(ctx)
		}
	}
}

func (d *Dispatcher) dispatchOnce(ctx context.Context) error {
	events, err := d.repo.ClaimPending(ctx, d.cfg.BatchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		if err := d.dispatchOne(ctx, event); err != nil {
			d.logger.Error("outbox event dispatch failed", loggerdomain.Fields{
				"event_id":   event.ID,
				"event_type": event.EventType,
				"error":      err.Error(),
			})
		}
	}
	return nil
}

func (d *Dispatcher) dispatchOne(ctx context.Context, event *OutboxEvent) error {
	codec, ok := d.registry.Codec(event.EventType)
	if !ok {
		err := fmt.Errorf("no codec registered for event type %q", event.EventType)
		if dlErr := d.repo.DeadLetter(ctx, event.ID, err.Error()); dlErr != nil {
			return fmt.Errorf("%v; dead-letter failed: %w", err, dlErr)
		}
		return err
	}

	domainEvent, err := codec(event.Payload)
	if err != nil {
		return fmt.Errorf("decode event %d: %w", event.ID, err)
	}

	if err := d.bus.Publish(ctx, domainEvent); err != nil {
		return d.handleFailure(ctx, event, err)
	}

	if err := d.repo.MarkDispatched(ctx, event.ID); err != nil {
		return err
	}
	return nil
}

func (d *Dispatcher) handleFailure(ctx context.Context, event *OutboxEvent, cause error) error {
	if event.Attempts+1 >= d.cfg.MaxAttempts {
		err := fmt.Errorf("outbox event %d exhausted %d attempts: %w", event.ID, d.cfg.MaxAttempts, cause)
		if dlErr := d.repo.DeadLetter(ctx, event.ID, err.Error()); dlErr != nil {
			return fmt.Errorf("%v; dead-letter failed: %w", err, dlErr)
		}
		d.logger.Error("outbox event dead-lettered", loggerdomain.Fields{
			"event_id":   event.ID,
			"event_type": event.EventType,
			"error":      err.Error(),
		})
		return err
	}

	next := d.backoff(event.Attempts)
	if err := d.repo.Retry(ctx, event.ID, time.Now().Add(next), cause.Error()); err != nil {
		return err
	}
	return cause
}

// backoff computes the exponential backoff with jitter for the given attempt.
func (d *Dispatcher) backoff(attempt int) time.Duration {
	base := d.cfg.BackoffBase
	if base <= 0 {
		base = time.Second
	}
	max := d.cfg.BackoffMax
	if max <= 0 {
		max = 60 * time.Second
	}

	dur := base
	for i := 0; i < attempt && dur < max; i++ {
		dur *= 2
	}
	if dur > max {
		dur = max
	}

	jitter := time.Duration(rand.Int63n(int64(dur)/5+1)) - dur/10
	result := dur + jitter
	if result > max {
		result = max
	}
	if result < time.Millisecond {
		result = time.Millisecond
	}
	return result
}

func (d *Dispatcher) maybePrune(ctx context.Context) {
	if d.cfg.RetentionDays <= 0 {
		return
	}
	if time.Since(d.lastPrune) < 24*time.Hour {
		return
	}
	d.lastPrune = time.Now()
	cutoff := time.Now().Add(-time.Duration(d.cfg.RetentionDays) * 24 * time.Hour)
	if err := d.repo.Prune(ctx, cutoff); err != nil {
		d.logger.Error("outbox prune failed", loggerdomain.Fields{"error": err.Error()})
	}
}
