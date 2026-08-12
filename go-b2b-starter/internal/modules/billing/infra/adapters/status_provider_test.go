package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moasq/go-b2b-starter/internal/modules/paywall"
)

func TestParseStatusFromReason_KnownStatuses(t *testing.T) {
	cases := map[string]string{
		"subscription status: past_due": paywall.StatusPastDue,
		"subscription status: canceled": paywall.StatusUnpaid,
		"subscription status: unpaid":   paywall.StatusUnpaid,
		"subscription status: trialing": paywall.StatusTrialing,
	}
	for reason, want := range cases {
		assert.Equal(t, want, parseStatusFromReason(reason), "reason %q", reason)
	}
}

func TestParseStatusFromReason_UnknownStatusesAreDistinctNotNone(t *testing.T) {
	// Unknown/unmapped provider statuses (Polar revoked/incomplete, MP raw
	// paused/in_process) must resolve to a distinct inactive state so the
	// paywall lazy guard attempts a provider refresh — never "none", which
	// would look like "no subscription" and skip the refresh.
	for _, reason := range []string{
		"subscription status: revoked",
		"subscription status: incomplete",
		"subscription status: paused",
		"subscription status: in_process",
		"subscription status: whatever",
	} {
		got := parseStatusFromReason(reason)
		assert.Equal(t, paywall.StatusUnknown, got, "reason %q", reason)
		assert.NotEqual(t, paywall.StatusNone, got, "reason %q", reason)
	}
}

func TestParseStatusFromReason_MPPendingResolvesThroughUnknownPath(t *testing.T) {
	// MP "pending" is a valid trial/billing state, not "no subscription": it
	// must resolve to the unknown/inactive path (distinct from "none") so the
	// paywall lazy guard refreshes before denying.
	got := parseStatusFromReason("subscription status: pending")
	assert.Equal(t, paywall.StatusUnknown, got)
	assert.NotEqual(t, paywall.StatusNone, got)
	assert.NotEqual(t, paywall.StatusActive, got)
}
