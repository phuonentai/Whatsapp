package stytch

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerRunBlocksWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute, 1)

	called := 0
	attempt := func() error {
		return cb.Run(context.Background(), func() error {
			called++
			return errors.New("boom")
		})
	}

	// Fail twice -> breaker opens.
	for i := 0; i < 2; i++ {
		if err := attempt(); err == nil {
			t.Fatalf("expected failure, got nil")
		}
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state, got %v", cb.State())
	}

	// Open breaker must block before invoking fn.
	if err := attempt(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if called != 2 {
		t.Fatalf("expected fn to be skipped while open, called %d times", called)
	}
}

func TestCircuitBreakerRunHalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Millisecond, 1)

	for i := 0; i < 2; i++ {
		_ = cb.Run(context.Background(), func() error { return errors.New("boom") })
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state, got %v", cb.State())
	}

	// After cooldown the breaker half-opens and lets one probe through.
	time.Sleep(2 * time.Millisecond)
	if err := cb.Run(context.Background(), func() error { return nil }); err != nil {
		t.Fatalf("expected half-open probe to succeed, got %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after success, got %v", cb.State())
	}
}

func TestCircuitBreakerRunContextCancelled(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Minute, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Run(ctx, func() error { return nil })
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
}

func TestClientRunUsesSharedBreaker(t *testing.T) {
	client := &Client{
		breaker: NewCircuitBreaker(1, time.Minute, 1),
	}

	if err := client.Run(context.Background(), func() error {
		return errors.New("boom")
	}); err == nil {
		t.Fatalf("expected failure, got nil")
	}
	if client.BreakerState() != CircuitOpen {
		t.Fatalf("expected open breaker, got %v", client.BreakerState())
	}

	err := client.Run(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestNewCircuitBreakerDefaults(t *testing.T) {
	cb := NewCircuitBreaker(0, 0, 0)
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed breaker, got %v", cb.State())
	}
	// Threshold 0 normalizes to 5; a single failure must not open it.
	if err := cb.Run(context.Background(), func() error { return errors.New("boom") }); err == nil {
		t.Fatalf("expected failure, got nil")
	}
	if cb.State() == CircuitOpen {
		t.Fatalf("expected breaker to stay closed after one failure")
	}
}
