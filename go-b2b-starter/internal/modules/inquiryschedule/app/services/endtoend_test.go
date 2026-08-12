package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
)

// TestScheduledInquiryEndToEnd (group 9.4, mock fallback execution per
// governance): schedule -> tick (claim+advance+enqueue) -> run creation ->
// overdue -> one nudge -> escalation. The mock outbox is the repository
// enqueue; the mock send is the FollowUpSender stub.
func TestScheduledInquiryEndToEnd(t *testing.T) {
	// --- 1. A due schedule exists (Monday–Friday 08:00, next_run_at past) ---
	repo := &mockScheduleRepo{}
	repo.setDue(&domain.Schedule{
		ID: 1, OrganizationID: 10, Name: "Matinal", RunTime: "08:00",
		DaysOfWeek:  domain.DaysOfWeek{domain.Monday, domain.Tuesday, domain.Wednesday, domain.Thursday, domain.Friday},
		ProductIDs:  []int32{11, 12},
		SupplierIDs: []int32{21, 22},
		IsActive:    true,
		NextRunAt:   time.Now().Add(-time.Minute),
	})

	// --- 2. Tick: exactly one inquiry_run.scheduled event is enqueued ---
	claimed, err := repo.ClaimAndAdvanceAndEnqueue(context.Background(), 50)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 || repo.eventCount() != 1 {
		t.Fatalf("claimed=%d events=%d, want 1/1", len(claimed), repo.eventCount())
	}

	// --- 3. Run creation handler consumes the event (deduped) ---
	creator := &mockCreator{result: &domain.ScheduledRunResult{RunID: 500, DraftedCount: 2}}
	audit := &mockAudit{}
	runHandler := NewScheduledRunHandler(&mockKillSwitch{}, creator, audit, testLogger())

	// Reconstruct the outbox event from the claim (the real dispatcher does
	// this via the registry codec).
	ev := &inqEvents.InquiryRunScheduled{}
	payload, _ := json.Marshal(inqEvents.NewInquiryRunScheduled(1, 10, []int32{11, 12}, []int32{21, 22}, "nota", time.Now()))
	_ = json.Unmarshal(payload, ev)
	if err := runHandler.HandleScheduledRun(context.Background(), ev); err != nil {
		t.Fatalf("run creation: %v", err)
	}
	if creator.calls != 1 {
		t.Fatalf("creator called %d times, want 1", creator.calls)
	}

	// --- 4. Recipients sent; one overdue unanswered (deadline passed) ---
	sentAt := time.Now().Add(-5 * time.Hour)
	overdue := candidate(1, 500, 0, "granted")
	overdue.SentAt = &sentAt
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{overdue}}
	enqueuer := &mockEnqueuer{}
	followUp := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, audit, testLogger())

	nudged, err := followUp.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if nudged != 1 || enqueuer.count() != 1 {
		t.Fatalf("nudged=%d enqueued=%d, want 1/1", nudged, enqueuer.count())
	}

	// --- 5. The follow-up send handler dispatches the nudge ---
	var followupEvent *inqEvents.FollowupSend
	for _, e := range enqueuer.events {
		if err := json.Unmarshal(e.Payload, &followupEvent); err == nil {
			break
		}
	}
	if followupEvent == nil {
		t.Fatal("no followup_send payload captured")
	}
	sender := &mockSender{}
	sendHandler := NewFollowUpSendHandler(
		&mockRecipientReader{targets: map[int32]*domain.FollowUpCandidate{1: overdue}},
		&mockKillSwitch{}, sender, audit, testLogger())
	if err := sendHandler.HandleFollowUpSend(context.Background(), followupEvent); err != nil {
		t.Fatalf("followup send: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d, want 1", len(sender.sent))
	}

	// --- 6. Escalation: max_nudges reached -> no further automatic nudges ---
	escalated := candidate(1, 500, 1, "granted") // followup_count == max_nudges(1)
	escalated.SentAt = &sentAt
	readers.candidates = []*domain.FollowUpCandidate{escalated}
	nudged2, err := followUp.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	if nudged2 != 0 {
		t.Fatalf("nudged %d at cap, want 0 (human escalation)", nudged2)
	}
	if enqueuer.count() != 1 {
		t.Fatalf("total enqueued %d, want 1 (at most max_nudges sends)", enqueuer.count())
	}
}
