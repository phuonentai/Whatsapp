package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ErrRateLimited is returned when the per-org send bucket is empty; the
// outbox dispatcher treats it as transient and retries with backoff.
var ErrRateLimited = errors.New("procurement: rate limit (10 msgs / 10s) exceeded")

// Pacer enforces the per-organization WhatsApp send pacing (10 msgs / 10s,
// D16). One token bucket per organization is shared by all dispatcher
// workers, so the burst is never exceeded regardless of concurrency.
type Pacer interface {
	Allow(orgID int32) bool
}

type tokenBucketPacer struct {
	mu      sync.Mutex
	limiters map[int32]*rate.Limiter
}

// NewTokenBucketPacer creates the shared per-org pacer.
func NewTokenBucketPacer() Pacer {
	return &tokenBucketPacer{limiters: map[int32]*rate.Limiter{}}
}

func (p *tokenBucketPacer) limiter(orgID int32) *rate.Limiter {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l, ok := p.limiters[orgID]; ok {
		return l
	}
	// rate.Every(1s) with burst 10 == 10 messages per 10 seconds.
	l := rate.NewLimiter(rate.Every(1*time.Second), 10)
	p.limiters[orgID] = l
	return l
}

func (p *tokenBucketPacer) Allow(orgID int32) bool {
	return p.limiter(orgID).Allow()
}

// waitPacer blocks until a token is available (or ctx is done). Used by tests
// to verify burst limits deterministically.
type waitPacer interface {
	Wait(ctx context.Context, orgID int32) error
}
