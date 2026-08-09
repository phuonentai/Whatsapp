package stytch

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the Stytch circuit breaker is open and a
// request is rejected without reaching the Stytch API.
var ErrCircuitOpen = errors.New("stytch circuit breaker open: request blocked")

// CircuitState describes the current state of the Stytch circuit breaker.
type CircuitState int

const (
	// CircuitClosed allows requests to flow to the Stytch API.
	CircuitClosed CircuitState = iota
	// CircuitOpen rejects requests without calling Stytch.
	CircuitOpen
	// CircuitHalfOpen allows a limited number of probe requests during recovery.
	CircuitHalfOpen
)

// CircuitBreaker guards outbound Stytch B2B API calls. It trips open after
// `threshold` consecutive failures and stays open for `cooldown`, then lets a
// limited number of half-open probes through to test recovery.
type CircuitBreaker struct {
	mu             sync.Mutex
	state          CircuitState
	failures       int
	lastFailureAt  time.Time
	threshold      int
	cooldown       time.Duration
	halfOpenProbes int
	halfOpenMax    int
}

// NewCircuitBreaker builds a two-tier circuit breaker.
func NewCircuitBreaker(threshold int, cooldown time.Duration, halfOpenProbes int) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Second
	}
	if halfOpenProbes <= 0 {
		halfOpenProbes = 2
	}
	return &CircuitBreaker{
		state:          CircuitClosed,
		threshold:      threshold,
		cooldown:       cooldown,
		halfOpenMax:    halfOpenProbes,
		halfOpenProbes: halfOpenProbes,
	}
}

// Allow reports whether a call may proceed; it also advances breaker state.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureAt) > cb.cooldown {
			cb.state = CircuitHalfOpen
			cb.halfOpenProbes = cb.halfOpenMax
		} else {
			return false
		}
		fallthrough
	case CircuitHalfOpen:
		if cb.halfOpenProbes <= 0 {
			cb.state = CircuitOpen
			cb.lastFailureAt = time.Now()
			return false
		}
		cb.halfOpenProbes--
		if cb.halfOpenProbes <= 0 {
			cb.state = CircuitOpen
			cb.lastFailureAt = time.Now()
		}
		return true
	default:
		return true
	}
}

// Success records a successful call and resets the breaker to closed.
func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}

// Failure records a failed call, tripping the breaker open at threshold.
func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureAt = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// State returns the current breaker state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Run guards a single outbound Stytch call. When the breaker is open it fails
// fast with ErrCircuitOpen without invoking fn; otherwise it invokes fn and
// records success/failure.
func (cb *CircuitBreaker) Run(ctx context.Context, fn func() error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if !cb.Allow() {
		return ErrCircuitOpen
	}
	if err := fn(); err != nil {
		cb.Failure()
		return err
	}
	cb.Success()
	return nil
}
