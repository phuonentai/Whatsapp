package services

import (
	"testing"
)

func TestCounterSinkIncrements(t *testing.T) {
	sink := NewCounterSink()
	sink.Inc(MetricDraftAttempt, map[string]string{"org": "1"})
	sink.Inc(MetricDraftAttempt, map[string]string{"org": "1"})
	sink.Inc(MetricDraftAttempt, map[string]string{"org": "2"})

	if sink.Get(Key(MetricDraftAttempt, map[string]string{"org": "1"})) != 2 {
		t.Fatalf("expected 2 drafts for org 1")
	}
	if sink.Get(Key(MetricDraftAttempt, map[string]string{"org": "2"})) != 1 {
		t.Fatalf("expected 1 draft for org 2")
	}
}

func TestCounterSinkLabelOrdering(t *testing.T) {
	sink := NewCounterSink()
	sink.Inc(MetricBlock, map[string]string{"org": "1", "reason": "kill_switch"})
	if sink.Get(MetricBlock) != 0 {
		t.Fatalf("unlabeled key must not match labeled increments")
	}
	if sink.Get(Key(MetricBlock, map[string]string{"reason": "kill_switch", "org": "1"})) != 1 {
		t.Fatalf("key folding must be order-independent")
	}
}
