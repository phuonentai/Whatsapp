package graphapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// templateCaptureTransport records the last request and returns a canned body.
type templateCaptureTransport struct {
	status int
	body   string
	reqURL atomic.Value // stores *url.URL of the last request
	reqAuth atomic.Value
	reqBody atomic.Value
}

func (s *templateCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.reqURL.Store(req.URL)
	s.reqAuth.Store(req.Header.Get("Authorization"))
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		s.reqBody.Store(string(raw))
	}
	return &http.Response{
		StatusCode: s.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Request:    req,
	}, nil
}

func templateClient(t *testing.T, status int, body string, breaker *whatsapp.CircuitBreaker) (Client, *templateCaptureTransport) {
	t.Helper()
	tr := &templateCaptureTransport{status: status, body: body}
	if breaker == nil {
		breaker = whatsapp.NewCircuitBreaker(5, 60*time.Second, 2)
	}
	c := newClient(ClientConfig{
		AppID:      "app-1",
		AppSecret:  "secret-1",
		APIBase:    "https://graph.facebook.com",
		APIVersion: "v21.0",
	}, &http.Client{Transport: tr}, breaker)
	return c, tr
}

func TestSubmitTemplate_Success(t *testing.T) {
	c, tr := templateClient(t, http.StatusOK, `{"id":"1045559864261146"}`, nil)

	id, err := c.SubmitTemplate(context.Background(),
		"EAAToken", "https://graph.facebook.com", "v21.0", "12345",
		"confirmacion_pedido", "es", "UTILITY", "Hola {{1}}, tu pedido {{2}} fue confirmado.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "1045559864261146" {
		t.Fatalf("expected template id 1045559864261146, got %s", id)
	}

	u := tr.reqURL.Load().(*url.URL)
	if !strings.HasSuffix(u.Path, "/v21.0/12345/message_templates") {
		t.Fatalf("expected message_templates POST path, got %s", u.Path)
	}
	if auth := tr.reqAuth.Load().(string); auth != "Bearer EAAToken" {
		t.Fatalf("expected Bearer token, got %q", auth)
	}

	rawBody := tr.reqBody.Load().(string)
	form, err := url.ParseQuery(rawBody)
	if err != nil {
		t.Fatalf("expected form-encoded body, got %q: %v", rawBody, err)
	}
	if form.Get("name") != "confirmacion_pedido" || form.Get("language") != "es" || form.Get("category") != "UTILITY" {
		t.Fatalf("unexpected form values: %v", form)
	}
	var components []map[string]any
	if err := json.Unmarshal([]byte(form.Get("components")), &components); err != nil {
		t.Fatalf("components must be JSON: %v", err)
	}
	if len(components) != 1 || components[0]["type"] != "BODY" {
		t.Fatalf("expected single BODY component, got %v", components)
	}
}

func TestSubmitTemplate_GraphErrorPropagated(t *testing.T) {
	c, _ := templateClient(t, http.StatusBadRequest, `{"error":{"message":"Invalid name","type":"GraphMethodException","code":100}}`, nil)

	_, err := c.SubmitTemplate(context.Background(),
		"EAAToken", "https://graph.facebook.com", "v21.0", "12345",
		"", "es", "UTILITY", "body")
	if err == nil {
		t.Fatal("expected error")
	}
	var gErr *GraphError
	if !errors.As(err, &gErr) {
		t.Fatalf("expected *GraphError, got %T", err)
	}
	if gErr.Code != 100 {
		t.Fatalf("expected code 100, got %d", gErr.Code)
	}
}

func TestGetTemplateStatus_Success(t *testing.T) {
	c, tr := templateClient(t, http.StatusOK, `{"status":"APPROVED"}`, nil)

	status, err := c.GetTemplateStatus(context.Background(),
		"EAAToken", "https://graph.facebook.com", "v21.0", "12345", "1045559864261146")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "APPROVED" {
		t.Fatalf("expected APPROVED, got %s", status)
	}

	u := tr.reqURL.Load().(*url.URL)
	if !strings.HasSuffix(u.Path, "/v21.0/1045559864261146") {
		t.Fatalf("expected template GET path, got %s", u.Path)
	}
	if !strings.Contains(u.RawQuery, "fields=status") {
		t.Fatalf("expected fields=status, got %s", u.RawQuery)
	}
}

func TestTemplateClient_CircuitBreakerOpensAfterFiveFifths(t *testing.T) {
	breaker := whatsapp.NewCircuitBreaker(5, 10*time.Millisecond, 2)
	c, _ := templateClient(t, http.StatusInternalServerError, `{"error":{"message":"boom","type":"Server","code":1}}`, breaker)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := c.SubmitTemplate(ctx, "t", "https://graph.facebook.com", "v21.0", "12345", "n", "es", "UTILITY", "b"); err == nil {
			t.Fatalf("iteration %d: expected 5xx error", i)
		}
	}
	if breaker.State() != whatsapp.CircuitOpen {
		t.Fatalf("expected circuit open after 5 consecutive 5xx, got state %d", breaker.State())
	}

	// Subsequent calls fail fast without reaching the transport.
	if _, err := c.SubmitTemplate(ctx, "t", "https://graph.facebook.com", "v21.0", "12345", "n", "es", "UTILITY", "b"); err == nil {
		t.Fatal("expected circuit-open error")
	} else if !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("expected circuit breaker open message, got %v", err)
	}
}
