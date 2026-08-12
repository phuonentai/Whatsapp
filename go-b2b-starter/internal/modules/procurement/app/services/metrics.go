package services

import (
	"sync"
)

// MetricsSink records observability counters for procurement (D16).
// The production wiring may swap this for the platform metrics surface; the
// default implementation is an in-memory counter map (assertable in tests).
type MetricsSink interface {
	Inc(name string, labels map[string]string)
}

// CounterSink is a thread-safe in-memory counter map.
type CounterSink struct {
	mu       sync.Mutex
	counters map[string]int64
}

// NewCounterSink creates an empty counter sink.
func NewCounterSink() *CounterSink {
	return &CounterSink{counters: map[string]int64{}}
}

// Inc increments the named counter, folding label pairs into the key.
func (c *CounterSink) Inc(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[Key(name, labels)]++
}

// Get returns the counter value for a keyed name (0 when absent).
func (c *CounterSink) Get(key string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counters[key]
}

// Key folds a counter name and label map into a stable map key.
func Key(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	out := name + "{"
	first := true
	// deterministic ordering for stable keys
	for _, k := range sortedKeys(labels) {
		if !first {
			out += ","
		}
		out += k + "=" + labels[k]
		first = false
	}
	return out + "}"
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// Metric names (D16).
const (
	MetricDraftAttempt        = "procurement.draft_attempt"
	MetricExtractionAttempt   = "procurement.extraction_attempt"
	MetricSummaryAttempt      = "procurement.summary_attempt"
	MetricExtractionEscalated = "procurement.extraction_escalation"
	MetricSendSuccess         = "procurement.send_success"
	MetricSendRetry           = "procurement.send_retry"
	MetricSendDeadLetter      = "procurement.send_deadletter"
	MetricOrderPlaced         = "procurement.order_placed"
	MetricBlock               = "procurement.block"
)

// noopMetrics is the default sink when none is provided.
type noopMetrics struct{}

func (noopMetrics) Inc(string, map[string]string) {}
