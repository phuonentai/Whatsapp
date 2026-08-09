package graphapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// stubTransport returns canned responses per request, simulating Meta.
type stubTransport struct {
	status int
	body   string
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Request:    req,
	}, nil
}

func testClient(t *testing.T, status int, body string, breaker *whatsapp.CircuitBreaker) Client {
	t.Helper()
	if breaker == nil {
		breaker = whatsapp.NewCircuitBreaker(5, 60*time.Second, 2)
	}
	return newClient(ClientConfig{
		AppID:      "app-1",
		AppSecret:  "secret-1",
		APIBase:    "https://graph.facebook.com",
		APIVersion: "v21.0",
	}, &http.Client{Transport: &stubTransport{status: status, body: body}}, breaker)
}

func TestExchangeCode_Success(t *testing.T) {
	c := testClient(t, http.StatusOK, `{"access_token":"EAATestToken","token_type":"bearer","expires_in":3600}`, nil)

	token, err := c.ExchangeCode(context.Background(), "the-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "EAATestToken" {
		t.Fatalf("expected EAATestToken, got %s", token)
	}
}

func TestExchangeCode_GraphErrorMapped(t *testing.T) {
	c := testClient(t, http.StatusBadRequest, `{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190,"error_subcode":460}}`, nil)

	_, err := c.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error")
	}
	var gErr *GraphError
	if !errors.As(err, &gErr) {
		t.Fatalf("expected *GraphError, got %T", err)
	}
	if gErr.Code != 190 || gErr.Subcode != 460 {
		t.Fatalf("expected code 190 subcode 460, got code=%d subcode=%d", gErr.Code, gErr.Subcode)
	}
	if !strings.Contains(gErr.Error(), "Invalid OAuth access token") {
		t.Fatalf("expected message in error, got: %s", gErr.Error())
	}
}

func TestSendTestMessage_Success(t *testing.T) {
	c := testClient(t, http.StatusOK, `{"messaging_product":"whatsapp","contacts":[{"input":"+573001234567","wa_id":"573001234567"}],"messages":[{"id":"wamid.abc"}]}`, nil)

	err := c.SendTestMessage(context.Background(), "token", "https://graph.facebook.com", "v21.0", "12345", "+573001234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCircuitBreaker_TripAndHalfOpen(t *testing.T) {
	breaker := whatsapp.NewCircuitBreaker(5, 40*time.Millisecond, 2)
	c := testClient(t, http.StatusInternalServerError, `{"error":{"message":"boom","type":"OAuthException","code":1}}`, breaker)

	for i := 0; i < 5; i++ {
		if _, err := c.ExchangeCode(context.Background(), "code"); err == nil {
			t.Fatalf("attempt %d expected failure", i+1)
		}
	}

	// 6th call must be blocked by the open breaker without hitting the transport.
	_, err := c.ExchangeCode(context.Background(), "code")
	if err == nil || !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("expected circuit breaker open error, got: %v", err)
	}

	// After cooldown the breaker half-opens and lets a probe through to the transport.
	time.Sleep(60 * time.Millisecond)
	probeErr := func() error {
		_, e := c.ExchangeCode(context.Background(), "code")
		return e
	}()
	if probeErr == nil {
		t.Fatal("expected transport error on half-open probe")
	}
	if strings.Contains(probeErr.Error(), "circuit breaker open") {
		t.Fatalf("expected transport failure during half-open probe, got breaker block: %v", probeErr)
	}
}

func TestRegisterAppSubscriptions_PassesVerifyToken(t *testing.T) {
	var gotURL string
	var gotForm string
	tr := &recordingTransport{fn: func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		body, _ := io.ReadAll(req.Body)
		gotForm = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
			Request:    req,
		}, nil
	}}

	c := newClient(ClientConfig{
		AppID:      "app-1",
		AppSecret:  "secret-1",
		APIBase:    "https://graph.facebook.com",
		APIVersion: "v21.0",
	}, &http.Client{Transport: tr}, whatsapp.NewCircuitBreaker(5, time.Minute, 2))

	if err := c.RegisterAppSubscriptions(context.Background(), "user-token", "app-1", "https://platform.example.com/api/v1/webhooks/whatsapp", "verify-abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(gotURL, "/v21.0/app-1/subscriptions") {
		t.Fatalf("expected subscriptions path in URL, got %s", gotURL)
	}
	for _, want := range []string{"object=whatsapp_business_account", "callback_url=https%3A%2F%2Fplatform.example.com%2Fapi%2Fv1%2Fwebhooks%2Fwhatsapp", "verify_token=verify-abc", "fields=messages%2Cstatuses"} {
		if !strings.Contains(gotForm, want) {
			t.Fatalf("expected %q in form body, got %s", want, gotForm)
		}
	}
}

type recordingTransport struct {
	fn func(*http.Request) (*http.Response, error)
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.fn(req)
}
