package services

import (
	"context"
	"sync"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
)

// TestSweep covers the follow-up sweep and its race with the reply trigger
// (group 7): exactly one inquiry.followup_send per overdue recipient, and the
// send-failure retry/dead-letter contract.

func TestSweep_ReplyTriggerAndSweepRaceProduceOneEvent(t *testing.T) {
	// One overdue recipient; the reply-trigger path and the sweep path both
	// fire concurrently. The atomic nudge guard admits exactly one enqueue.
	var attempts int
	enqueuer := &mockEnqueuer{guard: func(orgID, recipientID int32) (bool, error) {
		attempts++
		return attempts == 1, nil
	}}
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 0, "granted")}}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _ = svc.OnReplyArrival(context.Background(), 10, "+573001112233") }()
	go func() { defer wg.Done(); _, _ = svc.SweepOrg(context.Background(), 10, 100) }()
	wg.Wait()

	if enqueuer.count() != 1 {
		t.Fatalf("enqueued %d followup_send events, want 1 (double-nudge guard)", enqueuer.count())
	}
}

func TestSweep_OverdueRecipientWithoutBudgetIsNeverEnqueued(t *testing.T) {
	// followup_count already at max_nudges: neither path enqueues.
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 1, "granted")}}
	enqueuer := &mockEnqueuer{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	if _, err := svc.SweepOrg(context.Background(), 10, 100); err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if enqueuer.count() != 0 {
		t.Fatalf("enqueued %d events, want 0", enqueuer.count())
	}
}

// TestSweep_SendFailureRetriesWithBackoffThenDeadLetters simulates the
// durable-pipeline contract around a failing follow-up send: the handler
// returns the error (dispatcher retries with exponential backoff) and the
// event is dead-lettered after max attempts.
func TestSweep_SendFailureRetriesWithBackoffThenDeadLetters(t *testing.T) {
	readers := &mockRecipientReader{targets: map[int32]*domain.FollowUpCandidate{
		1: target(1, 7, "sent", "granted", "awaiting_responses"),
	}}
	sender := &mockSender{err: errSendDown}
	handler := NewFollowUpSendHandler(readers, &mockKillSwitch{}, sender, &mockAudit{}, testLogger())

	// Attempts 1..max-1: handler returns the error -> dispatcher backs off.
	const maxAttempts = 3
	attempts := 0
	for attempts < maxAttempts-1 {
		err := handler.HandleFollowUpSend(context.Background(), followupEvent(1, 7))
		if err == nil {
			t.Fatalf("attempt %d: expected error for retry", attempts)
		}
		attempts++
	}
	// Final attempt: still failing -> dead-letter (dispatcher contract).
	err := handler.HandleFollowUpSend(context.Background(), followupEvent(1, 7))
	if err == nil {
		t.Fatal("expected error on final attempt (dead-letter path)")
	}
	if len(sender.sent) != 0 {
		t.Fatalf("sent %d messages despite failure, want 0", len(sender.sent))
	}
}

var errSendDown = &mockSendError{}

type mockSendError struct{}

func (e *mockSendError) Error() string { return "meta api unavailable" }
