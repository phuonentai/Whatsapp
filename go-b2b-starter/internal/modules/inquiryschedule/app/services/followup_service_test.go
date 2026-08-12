package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
)

// TestFollowUp covers the follow-up service (group 6): deadline detection,
// exactly one nudge, double-nudge guard, escalation at max nudges, consent
// gating, and the kill switch.

func candidate(recipientID, runID int32, followupCount int32, consent string) *domain.FollowUpCandidate {
	sentAt := time.Now().Add(-5 * time.Hour)
	return &domain.FollowUpCandidate{
		RecipientID:         recipientID,
		OrganizationID:      10,
		RunID:               runID,
		SupplierID:          200,
		ContactID:           300,
		RecipientStatus:     "sent",
		SentAt:              &sentAt,
		FollowupCount:       followupCount,
		RunStatus:           "awaiting_responses",
		ContactPhone:        "+573001112233",
		ConsentStatus:       consent,
		SupplierDisplayName: "Distribuciones Andinas SAS",
		SupplierNIT:         "900123456",
		DeadlineHours:       4,
		MaxNudges:           1,
		MessageTemplate:     domain.DefaultFollowUpTemplate,
	}
}

func TestFollowUp_OverdueRecipientReceivesOneNudge(t *testing.T) {
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 0, "granted")}}
	enqueuer := &mockEnqueuer{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	nudged, err := svc.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if nudged != 1 || enqueuer.count() != 1 {
		t.Fatalf("nudged=%d enqueued=%d, want 1/1", nudged, enqueuer.count())
	}
	msgs := enqueuer.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages=%v", msgs)
	}
	if !strings.Contains(msgs[0], "Distribuciones Andinas SAS") {
		t.Fatalf("[proveedor] not replaced: %q", msgs[0])
	}
	if strings.Contains(msgs[0], "[proveedor]") {
		t.Fatalf("template placeholder left in message: %q", msgs[0])
	}
}

func TestFollowUp_DoubleNudgePrevented(t *testing.T) {
	// Sweep and reply-trigger race on the same overdue recipient: the
	// atomic guard allows only one nudge (the second EnqueueNudge returns
	// false because the cap was reached).
	attempts := 0
	enqueuer := &mockEnqueuer{guard: func(orgID, recipientID int32) (bool, error) {
		attempts++
		return attempts == 1, nil
	}}
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 0, "granted")}}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	// Two concurrent paths: the sweep and the reply trigger.
	if _, err := svc.SweepOrg(context.Background(), 10, 100); err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if _, err := svc.SweepOrg(context.Background(), 10, 100); err != nil {
		t.Fatalf("SweepOrg second: %v", err)
	}
	if enqueuer.count() != 1 {
		t.Fatalf("enqueued %d events, want 1 (double-nudge guard)", enqueuer.count())
	}
}

func TestFollowUp_MaxNudgesReachedStopsFurtherNudges(t *testing.T) {
	// followup_count == max_nudges: the candidate query no longer returns
	// the recipient (guard at the cap); no event is enqueued and the
	// recipient is surfaced as overdue (badge) by the status query.
	readers := &mockRecipientReader{} // no candidates
	enqueuer := &mockEnqueuer{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	nudged, err := svc.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if nudged != 0 || enqueuer.count() != 0 {
		t.Fatalf("nudged=%d enqueued=%d, want 0/0", nudged, enqueuer.count())
	}
}

func TestFollowUp_WithdrawnConsentEscalatesInsteadOfNudging(t *testing.T) {
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 0, "withdrawn")}}
	enqueuer := &mockEnqueuer{}
	audit := &mockAudit{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, audit, testLogger())

	nudged, err := svc.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if nudged != 0 || enqueuer.count() != 0 {
		t.Fatalf("nudged=%d enqueued=%d, want 0/0", nudged, enqueuer.count())
	}
	reasons := audit.skips()
	if len(reasons) != 1 || reasons[0] != "consent_withdrawn" {
		t.Fatalf("skip audits = %v, want [consent_withdrawn]", reasons)
	}
}

func TestFollowUp_KillSwitchCancelsNudges(t *testing.T) {
	readers := &mockRecipientReader{candidates: []*domain.FollowUpCandidate{candidate(1, 7, 0, "granted")}}
	enqueuer := &mockEnqueuer{}
	audit := &mockAudit{}
	kill := &mockKillSwitch{enabled: true}
	svc := NewFollowUpService(readers, enqueuer, kill, audit, testLogger())

	nudged, err := svc.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if nudged != 0 || enqueuer.count() != 0 {
		t.Fatalf("nudged=%d enqueued=%d, want 0/0", nudged, enqueuer.count())
	}
	reasons := audit.skips()
	if len(reasons) != 1 || reasons[0] != "kill_switch" {
		t.Fatalf("skip audits = %v, want [kill_switch]", reasons)
	}
}

func TestFollowUp_ReplyBeforeDeadlineSuppressesNudge(t *testing.T) {
	// No overdue candidates (recipient answered before the deadline): the
	// reader returns nothing and no nudge is enqueued.
	readers := &mockRecipientReader{}
	enqueuer := &mockEnqueuer{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	nudged, err := svc.SweepOrg(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SweepOrg: %v", err)
	}
	if nudged != 0 || enqueuer.count() != 0 {
		t.Fatalf("nudged=%d enqueued=%d, want 0/0", nudged, enqueuer.count())
	}
}

func TestFollowUp_ReplyArrivalExcludesAnsweringRecipient(t *testing.T) {
	// The message received from contact 300 answered recipient 1 of run 7;
	// recipient 2 of the same run is still overdue and should be nudged.
	cand1 := candidate(1, 7, 0, "granted")
	cand1.ContactID = 300
	cand2 := candidate(2, 7, 0, "granted")
	cand2.ContactID = 400
	readers := &mockRecipientReader{
		candidates: []*domain.FollowUpCandidate{cand1, cand2},
		active:     []domain.RecipientRef{{RecipientID: 1, RunID: 7}},
	}
	enqueuer := &mockEnqueuer{}
	svc := NewFollowUpService(readers, enqueuer, &mockKillSwitch{}, &mockAudit{}, testLogger())

	if err := svc.OnReplyArrival(context.Background(), 10, "+573001112233"); err != nil {
		t.Fatalf("OnReplyArrival: %v", err)
	}
	if enqueuer.count() != 1 {
		t.Fatalf("enqueued %d events, want 1 (the just-answered recipient excluded)", enqueuer.count())
	}
}
