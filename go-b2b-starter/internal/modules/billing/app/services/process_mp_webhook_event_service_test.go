package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// errOrgAdapter fails org mapping so subscription processing errors early,
// isolating the dispatch assertion.
type errOrgAdapter struct {
	domain.OrganizationAdapter
}

func (errOrgAdapter) GetOrganizationIDByStytchOrgID(ctx context.Context, stytchOrgID string) (int32, error) {
	return 0, errors.New("no such org")
}

// fakePaymentEventHandler captures dispatched MercadoPago payment events.
type fakePaymentEventHandler struct {
	calls []struct{ eventType, paymentID string }
}

func (f *fakePaymentEventHandler) HandlePaymentEvent(ctx context.Context, eventType, paymentID string) error {
	f.calls = append(f.calls, struct{ eventType, paymentID string }{eventType, paymentID})
	return nil
}

func TestProcessMPWebhookEvent_PaymentEventsDispatchedToClientPayments(t *testing.T) {
	handler := &fakePaymentEventHandler{}
	svc := &billingService{logger: fakeGrantLogger{}, paymentEventHandler: handler}

	// The notification id (424242) must NOT be dispatched: the client-payments
	// handler correlates on the payment id carried in data.id.
	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"payment_approved","id":424242,"data":{"id":999999}}`))
	require.NoError(t, err)
	require.Len(t, handler.calls, 1)
	assert.Equal(t, "payment_approved", handler.calls[0].eventType)
	assert.Equal(t, "999999", handler.calls[0].paymentID)
}

func TestProcessMPWebhookEvent_PaymentEventsDispatchAllTypes(t *testing.T) {
	handler := &fakePaymentEventHandler{}
	svc := &billingService{logger: fakeGrantLogger{}, paymentEventHandler: handler}

	for i, typ := range []string{"payment_created", "payment_updated", "payment_approved"} {
		payload := []byte(`{"type":"` + typ + `","id":7,"data":{"id":` + fmt.Sprintf("%d", i+1) + `}}`)
		require.NoError(t, svc.ProcessMPWebhookEvent(context.Background(), payload))
	}
	require.Len(t, handler.calls, 3)
	for i, typ := range []string{"payment_created", "payment_updated", "payment_approved"} {
		assert.Equal(t, typ, handler.calls[i].eventType)
		assert.Equal(t, fmt.Sprintf("%d", i+1), handler.calls[i].paymentID)
	}
}

func TestProcessMPWebhookEvent_PaymentEventsDispatchStringPaymentIDs(t *testing.T) {
	handler := &fakePaymentEventHandler{}
	svc := &billingService{logger: fakeGrantLogger{}, paymentEventHandler: handler}

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"payment_approved","id":7,"data":{"id":"pay-abc"}}`))
	require.NoError(t, err)
	require.Len(t, handler.calls, 1)
	assert.Equal(t, "pay-abc", handler.calls[0].paymentID)
}

func TestProcessMPWebhookEvent_SubscriptionEventsDoNotReachHandler(t *testing.T) {
	handler := &fakePaymentEventHandler{}
	svc := &billingService{logger: fakeGrantLogger{}, paymentEventHandler: handler, orgAdapter: errOrgAdapter{}}

	payload := []byte(`{"type":"subscription_authorized","id":9,"data":{"id":"preapproval-1","external_reference":"org_1","status":"authorized"}}`)
	err := svc.ProcessMPWebhookEvent(context.Background(), payload)
	require.Error(t, err)
	assert.Empty(t, handler.calls)
}

func TestProcessMPWebhookEvent_PaymentEventWithoutHandlerIsAcked(t *testing.T) {
	svc := &billingService{logger: fakeGrantLogger{}}

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"payment_approved","id":1,"data":{"id":5}}`))
	require.NoError(t, err)
}

func TestProcessMPWebhookEvent_SubscriptionCancelledWithoutDatesKeepsPriorBounds(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeUpsertRepo{stored: &domain.Subscription{
		OrganizationID:     42,
		ExternalCustomerID: "org_stytch_1",
		SubscriptionID:     "pre-1",
		SubscriptionStatus: "active",
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
	}}
	svc := &billingService{
		repo:       repo,
		orgAdapter: &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}},
		logger:     fakeGrantLogger{},
	}

	// The event carries no end_date/next_payment_date; processing must not
	// error (NOT NULL violation) and must retain the prior period bounds.
	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"subscription_cancelled","id":9,"data":{"id":"pre-1","external_reference":"org_stytch_1","status":"cancelled"}}`))
	require.NoError(t, err)

	require.NotNil(t, repo.stored)
	assert.Equal(t, "canceled", repo.stored.SubscriptionStatus)
	assert.Equal(t, start, repo.stored.CurrentPeriodStart)
	assert.Equal(t, end, repo.stored.CurrentPeriodEnd)
}

func TestProcessMPWebhookEvent_SubscriptionCancelledWithoutDatesAndWithoutPriorRow(t *testing.T) {
	repo := &fakeUpsertRepo{}
	svc := &billingService{
		repo:       repo,
		orgAdapter: &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}},
		logger:     fakeGrantLogger{},
	}

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"subscription_cancelled","id":9,"data":{"id":"pre-1","external_reference":"org_stytch_1","status":"cancelled"}}`))
	require.NoError(t, err)

	require.NotNil(t, repo.stored)
	assert.Equal(t, "canceled", repo.stored.SubscriptionStatus)
	assert.False(t, repo.stored.CurrentPeriodStart.IsZero())
	assert.False(t, repo.stored.CurrentPeriodEnd.IsZero())
}
