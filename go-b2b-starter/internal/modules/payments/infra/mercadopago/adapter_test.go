package mercadopago

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	mp "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (noopLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (noopLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return noopLogger{}
}

func newTestRail(t *testing.T, server *httptest.Server) domain.PaymentRail {
	t.Helper()
	client, err := mp.NewClient(&mp.Config{
		AccessToken: "TEST-token",
		BaseURL:     server.URL,
	})
	require.NoError(t, err)
	return NewPaymentRail(client, noopLogger{}, "http://localhost:3000/dashboard")
}

func TestCreatePreference_ReturnsInitPoint(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/checkout/preferences", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pref-42","init_point":"https://checkout.mercadopago.com/pay"}`))
	}))
	defer server.Close()

	rail := newTestRail(t, server)
	link, prefID, err := rail.CreatePreference(context.Background(), 7, 99, 102500, "COP")
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.mercadopago.com/pay", link)
	assert.Equal(t, "pref-42", prefID)

	items, ok := gotBody["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, float64(102500), item["unit_price"])
	assert.Equal(t, "deal:99", gotBody["external_reference"])
	assert.Equal(t, "http://localhost:3000/dashboard", gotBody["back_urls"].(map[string]any)["success"])
}

func TestCreatePreference_Non201ResponseWrappedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer server.Close()

	rail := newTestRail(t, server)
	_, _, err := rail.CreatePreference(context.Background(), 7, 99, 1000, "COP")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}

func TestVerifyPayment_MapsStatusAndCorrelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/payments/pay-1", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1,"status":"approved","external_reference":"deal:99","preference_id":"pref-42","transaction_amount":102500}`))
	}))
	defer server.Close()

	rail := newTestRail(t, server)
	detail, err := rail.VerifyPayment(context.Background(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusPaid, detail.Status)
	assert.Equal(t, "pref-42", detail.PreferenceID)
	assert.Equal(t, "deal:99", detail.ExternalRef)
	assert.Equal(t, int64(102500), detail.TransactionAmount)
}

func TestVerifyPayment_NonApprovedMapsPendingOrFailed(t *testing.T) {
	cases := []struct {
		mpStatus string
		want     domain.PaymentStatus
	}{
		{"in_process", domain.PaymentStatusPending},
		{"pending", domain.PaymentStatusPending},
		{"rejected", domain.PaymentStatusFailed},
		{"cancelled", domain.PaymentStatusFailed},
		{"refunded", domain.PaymentStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.mpStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := json.Marshal(map[string]any{"id": 1, "status": tc.mpStatus})
				_, _ = w.Write(body)
			}))
			defer server.Close()

			rail := newTestRail(t, server)
			detail, err := rail.VerifyPayment(context.Background(), "pay-1")
			require.NoError(t, err)
			assert.Equal(t, tc.want, detail.Status)
		})
	}
}

func TestVerifyPayment_Non200WrappedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer server.Close()

	rail := newTestRail(t, server)
	_, err := rail.VerifyPayment(context.Background(), "pay-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}
