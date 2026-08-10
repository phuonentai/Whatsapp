package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"payment_approved","id":424242,"data":{}}`))
	require.NoError(t, err)
	require.Len(t, handler.calls, 1)
	assert.Equal(t, "payment_approved", handler.calls[0].eventType)
	assert.Equal(t, "424242", handler.calls[0].paymentID)
}

func TestProcessMPWebhookEvent_PaymentEventsDispatchAllTypes(t *testing.T) {
	handler := &fakePaymentEventHandler{}
	svc := &billingService{logger: fakeGrantLogger{}, paymentEventHandler: handler}

	for _, typ := range []string{"payment_created", "payment_updated", "payment_approved"} {
		payload := []byte(`{"type":"` + typ + `","id":7,"data":{}}`)
		require.NoError(t, svc.ProcessMPWebhookEvent(context.Background(), payload))
	}
	require.Len(t, handler.calls, 3)
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

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{"type":"payment_approved","id":1,"data":{}}`))
	require.NoError(t, err)
}
