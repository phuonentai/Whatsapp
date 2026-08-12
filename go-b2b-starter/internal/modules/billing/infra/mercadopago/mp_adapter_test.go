package mercadopago

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	mp "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
)

type noopAdapterLogger struct{}

func (noopAdapterLogger) Debug(msg string, fields ...logger.Fields)                    {}
func (noopAdapterLogger) Info(msg string, fields ...logger.Fields)                     {}
func (noopAdapterLogger) Warn(msg string, fields ...logger.Fields)                     {}
func (noopAdapterLogger) Error(msg string, fields ...logger.Fields)                    {}
func (noopAdapterLogger) Fatal(msg string, fields ...logger.Fields)                    {}
func (noopAdapterLogger) WithFields(fields logger.Fields) logger.Logger                 { return noopAdapterLogger{} }

// newCaptureServer records request bodies per path and serves canned
// responses for the preapproval creation and search endpoints.
func newCaptureServer(t *testing.T, preapprovalResp, searchResp string) (*httptest.Server, *sync.Map) {
	t.Helper()
	var mu sync.Mutex
	bodies := &sync.Map{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies.Store(r.URL.Path, json.RawMessage(raw))
		mu.Unlock()

		switch r.URL.Path {
		case "/preapproval":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(preapprovalResp))
		case "/preapproval/search":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(searchResp))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server, bodies
}

func newTestAdapter(t *testing.T, server *httptest.Server, cfg *mp.Config) *mpAdapter {
	t.Helper()
	client, err := mp.NewClient(&mp.Config{
		AccessToken: "TEST-token",
		BaseURL:     server.URL,
	})
	require.NoError(t, err)
	return &mpAdapter{client: client, logger: noopAdapterLogger{}, cfg: cfg}
}

func capturedBody(t *testing.T, bodies *sync.Map, path string) map[string]any {
	t.Helper()
	raw, ok := bodies.Load(path)
	require.True(t, ok, "no request captured for %s", path)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw.(json.RawMessage), &body))
	return body
}

func TestCreateCheckoutSession_AttachesPlanQuotaMetadata(t *testing.T) {
	cfg := &mp.Config{
		CheckoutPlanID:       "plan-checkout",
		CheckoutInvoiceCount: 25,
		BusinessPlanID:       "plan-business",
		BusinessInvoiceCount: 100,
		BackURL:              "https://app.example/dashboard",
	}
	server, bodies := newCaptureServer(t, `{"id":"pre-1","status":"authorized","init_point":"https://checkout.example/x"}`, `{}`)
	adapter := newTestAdapter(t, server, cfg)

	_, err := adapter.CreateCheckoutSession(context.Background(), "plan-checkout", "org_1")
	require.NoError(t, err)

	body := capturedBody(t, bodies, "/preapproval")
	metadata, ok := body["metadata"].(map[string]any)
	require.True(t, ok, "preapproval body must carry metadata")
	assert.Equal(t, float64(25), metadata["invoice_count_max"])
}

func TestCreateCheckoutSession_BusinessPlanQuotaMetadata(t *testing.T) {
	cfg := &mp.Config{
		CheckoutPlanID:       "plan-checkout",
		CheckoutInvoiceCount: 25,
		BusinessPlanID:       "plan-business",
		BusinessInvoiceCount: 100,
	}
	server, bodies := newCaptureServer(t, `{"id":"pre-2","status":"authorized","init_point":"https://checkout.example/x"}`, `{}`)
	adapter := newTestAdapter(t, server, cfg)

	_, err := adapter.CreateCheckoutSession(context.Background(), "plan-business", "org_1")
	require.NoError(t, err)

	body := capturedBody(t, bodies, "/preapproval")
	metadata, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(100), metadata["invoice_count_max"])
}

func TestCreateCheckoutSession_UnknownPlanCarriesNoMetadata(t *testing.T) {
	cfg := &mp.Config{
		CheckoutPlanID:       "plan-checkout",
		CheckoutInvoiceCount: 25,
		BusinessPlanID:       "plan-business",
		BusinessInvoiceCount: 100,
	}
	server, bodies := newCaptureServer(t, `{"id":"pre-3","status":"authorized","init_point":"https://checkout.example/x"}`, `{}`)
	adapter := newTestAdapter(t, server, cfg)

	_, err := adapter.CreateCheckoutSession(context.Background(), "plan-unknown", "org_1")
	require.NoError(t, err)

	body := capturedBody(t, bodies, "/preapproval")
	_, ok := body["metadata"]
	assert.False(t, ok, "unknown plan must not attach quota metadata")
}

func TestGetSubscription_MapsStatusAndParsesQuotaMetadata(t *testing.T) {
	searchResp := `{"results":[{"id":"pre-1","external_reference":"org_1","status":"authorized","date_created":"2026-01-01T00:00:00Z","metadata":{"invoice_count_max":50}}],"paging":{"total":1}}`
	server, _ := newCaptureServer(t, `{}`, searchResp)
	adapter := newTestAdapter(t, server, &mp.Config{})

	sub, err := adapter.GetSubscription(context.Background(), "org_1")
	require.NoError(t, err)
	// Raw MP status "authorized" must never reach callers: mapped to "active".
	assert.Equal(t, "active", sub.SubscriptionStatus)
	assert.Equal(t, int32(50), sub.Metadata["invoice_count_max"])
}

func TestGetSubscription_StringQuotaMetadata(t *testing.T) {
	searchResp := `{"results":[{"id":"pre-1","external_reference":"org_1","status":"paused","metadata":{"invoice_count_max":"50"}}],"paging":{"total":1}}`
	server, _ := newCaptureServer(t, `{}`, searchResp)
	adapter := newTestAdapter(t, server, &mp.Config{})

	sub, err := adapter.GetSubscription(context.Background(), "org_1")
	require.NoError(t, err)
	assert.Equal(t, "past_due", sub.SubscriptionStatus)
	assert.Equal(t, int32(50), sub.Metadata["invoice_count_max"])
}

func TestGetSubscription_CancelledStatusMapped(t *testing.T) {
	searchResp := `{"results":[{"id":"pre-1","external_reference":"org_1","status":"cancelled","metadata":{}}],"paging":{"total":1}}`
	server, _ := newCaptureServer(t, `{}`, searchResp)
	adapter := newTestAdapter(t, server, &mp.Config{})

	sub, err := adapter.GetSubscription(context.Background(), "org_1")
	require.NoError(t, err)
	assert.Equal(t, "canceled", sub.SubscriptionStatus)
	_, ok := sub.Metadata["invoice_count_max"]
	assert.False(t, ok, "absent quota metadata must not be materialized")
}
