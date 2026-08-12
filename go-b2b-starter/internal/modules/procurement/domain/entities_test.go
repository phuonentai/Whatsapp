package domain

import "testing"

func TestInquiryRunCanTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     RunStatus
		to       RunStatus
		expected bool
	}{
		{"draft to sending", RunDraft, RunSending, true},
		{"draft to escalated", RunDraft, RunEscalated, true},
		{"draft to cancelled", RunDraft, RunCancelled, true},
		{"draft to completed invalid", RunDraft, RunCompleted, false},
		{"draft to awaiting invalid", RunDraft, RunAwaitingResponses, false},

		{"sending to awaiting", RunSending, RunAwaitingResponses, true},
		{"sending to failed", RunSending, RunFailed, true},
		{"sending to escalated", RunSending, RunEscalated, true},
		{"sending to cancelled", RunSending, RunCancelled, true},
		{"sending to completed invalid", RunSending, RunCompleted, false},

		{"awaiting to completed", RunAwaitingResponses, RunCompleted, true},
		{"awaiting to partially answered", RunAwaitingResponses, RunPartiallyAnswered, true},
		{"awaiting to escalated", RunAwaitingResponses, RunEscalated, true},
		{"awaiting to cancelled", RunAwaitingResponses, RunCancelled, true},
		{"awaiting to failed invalid", RunAwaitingResponses, RunFailed, false},

		{"completed is terminal", RunCompleted, RunCancelled, false},
		{"failed is terminal", RunFailed, RunEscalated, false},
		{"escalated is terminal", RunEscalated, RunCompleted, false},
		{"cancelled is terminal", RunCancelled, RunEscalated, false},
		{"unknown status", RunStatus("bogus"), RunCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &InquiryRun{Status: tt.from}
			if got := run.CanTransition(tt.to); got != tt.expected {
				t.Fatalf("CanTransition(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.expected)
			}
		})
	}
}

func TestInquiryRunEscalatable(t *testing.T) {
	for _, s := range []RunStatus{RunDraft, RunSending, RunAwaitingResponses} {
		run := &InquiryRun{Status: s}
		if !run.Escalatable() {
			t.Fatalf("expected %s to be escalatable (escalation always allowed)", s)
		}
	}
	for _, s := range []RunStatus{RunCompleted, RunPartiallyAnswered, RunFailed, RunEscalated, RunCancelled} {
		run := &InquiryRun{Status: s}
		if run.Escalatable() {
			t.Fatalf("expected %s to be terminal (not escalatable)", s)
		}
	}
}

// TestEscalationAllowedUnderKillSwitch documents that escalation is allowed
// regardless of any org setting (mirrors agent-governance: escalate_human is
// non-governable). The domain carries no settings; the service layer applies
// the kill switch only to SEND guards.
func TestEscalationAllowedUnderKillSwitch(t *testing.T) {
	run := &InquiryRun{Status: RunSending}
	if !run.CanTransition(RunEscalated) {
		t.Fatalf("escalation must be allowed from sending under any settings")
	}
}

func TestValidRunStatus(t *testing.T) {
	valid := []RunStatus{RunDraft, RunSending, RunAwaitingResponses, RunCompleted, RunPartiallyAnswered, RunFailed, RunEscalated, RunCancelled}
	for _, s := range valid {
		if !ValidRunStatus(s) {
			t.Fatalf("expected %s to be valid", s)
		}
	}
	if ValidRunStatus(RunStatus("nope")) {
		t.Fatalf("expected invalid status to be rejected")
	}
}
