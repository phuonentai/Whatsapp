package services

import (
	"sync"
	"testing"
)

// TestTokenBucketPacerBurstNeverExceedsLimit verifies the per-org pacer
// allows a burst of exactly 10 tokens and then denies until refill — the
// 10 msgs / 10s WhatsApp cap (D16).
func TestTokenBucketPacerBurstNeverExceedsLimit(t *testing.T) {
	pacer := NewTokenBucketPacer()

	// Burst: 10 allowed immediately.
	for i := 0; i < 10; i++ {
		if !pacer.Allow(42) {
			t.Fatalf("expected token %d of burst to be allowed", i)
		}
	}
	if pacer.Allow(42) {
		t.Fatalf("expected the 11th call in the burst window to be denied")
	}

	// Different orgs are independent.
	if !pacer.Allow(7) {
		t.Fatalf("expected another org's bucket to be independent")
	}
}

// TestTokenBucketPacerConcurrentWorkers verifies concurrent dispatcher
// workers cannot exceed the burst (shared bucket).
func TestTokenBucketPacerConcurrentWorkers(t *testing.T) {
	pacer := NewTokenBucketPacer()
	const workers = 8
	const perWorker = 8

	var wg sync.WaitGroup
	allowed := make(chan bool, workers*perWorker)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				allowed <- pacer.Allow(99)
			}
		}()
	}
	wg.Wait()
	close(allowed)

	yes := 0
	for v := range allowed {
		if v {
			yes++
		}
	}
	if yes > 10 {
		t.Fatalf("concurrent workers exceeded the 10/10s burst: %d allowed", yes)
	}
}
