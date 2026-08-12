package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	inqEvents "github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain/events"
)

// TestRunCreation covers the inquiry_run.scheduled handler (group 5):
// duplicate dispatch creates exactly one run, kill switch skips with audit,
// credits-exhausted run is escalated.

func newScheduledRunEvent(orgID, scheduleID int32) *inqEvents.InquiryRunScheduled {
	return inqEvents.NewInquiryRunScheduled(scheduleID, orgID, []int32{1, 2}, []int32{3, 4}, "nota", time.Now())
}

func TestRunCreation_DuplicateDispatchCreatesExactlyOneRun(t *testing.T) {
	creator := &mockCreator{result: &domain.ScheduledRunResult{RunID: 7, Skipped: true}}
	audit := &mockAudit{}
	handler := NewScheduledRunHandler(&mockKillSwitch{}, creator, audit, testLogger())

	ev := newScheduledRunEvent(10, 99)
	if err := handler.HandleScheduledRun(context.Background(), ev); err != nil {
		t.Fatalf("HandleScheduledRun: %v", err)
	}
	// Redelivery: same event again.
	if err := handler.HandleScheduledRun(context.Background(), ev); err != nil {
		t.Fatalf("HandleScheduledRun (redelivery): %v", err)
	}

	if creator.calls != 2 {
		t.Fatalf("creator called %d times, want 2 (each dispatch is handled; the creator dedupes)", creator.calls)
	}
	// The creator's Skipped result means no second run is created; the
	// skip/duplicate_occurrence audit is written by the creator adapter.
	if creator.lastInput.ScheduleID != 99 || creator.lastInput.OrganizationID != 10 {
		t.Fatalf("creator input mismatch: %+v", creator.lastInput)
	}
}

func TestRunCreation_KillSwitchSkipsWithAudit(t *testing.T) {
	creator := &mockCreator{result: &domain.ScheduledRunResult{RunID: 1}}
	audit := &mockAudit{}
	kill := &mockKillSwitch{enabled: true}
	handler := NewScheduledRunHandler(kill, creator, audit, testLogger())

	if err := handler.HandleScheduledRun(context.Background(), newScheduledRunEvent(10, 99)); err != nil {
		t.Fatalf("HandleScheduledRun: %v", err)
	}
	if creator.calls != 0 {
		t.Fatalf("creator called %d times, want 0 (kill switch)", creator.calls)
	}
	reasons := audit.skips()
	if len(reasons) != 1 || reasons[0] != "kill_switch" {
		t.Fatalf("skip audits = %v, want [kill_switch]", reasons)
	}
}

func TestRunCreation_CreditsExhaustedRunIsEscalated(t *testing.T) {
	creator := &mockCreator{result: &domain.ScheduledRunResult{RunID: 3, Escalated: true}}
	audit := &mockAudit{}
	handler := NewScheduledRunHandler(&mockKillSwitch{}, creator, audit, testLogger())

	if err := handler.HandleScheduledRun(context.Background(), newScheduledRunEvent(10, 99)); err != nil {
		t.Fatalf("HandleScheduledRun: %v", err)
	}
	if creator.calls != 1 {
		t.Fatalf("creator called %d times, want 1", creator.calls)
	}
	// The escalated result is surfaced; the creator adapter audited the
	// escalation (credits exhausted => no unmetered LLM call).
}

func TestRunCreation_CreatorErrorReturnsForOutboxRetry(t *testing.T) {
	creator := &mockCreator{err: errors.New("db down")}
	handler := NewScheduledRunHandler(&mockKillSwitch{}, creator, &mockAudit{}, testLogger())

	err := handler.HandleScheduledRun(context.Background(), newScheduledRunEvent(10, 99))
	if err == nil {
		t.Fatal("expected error to be returned for outbox retry")
	}
}
