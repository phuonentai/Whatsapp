package siigo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

func newTestConfig(tokenURL, baseURL string) *Config {
	return &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		BaseURL:      baseURL,
		TokenURL:     tokenURL,
		Sandbox:      true,
	}
}

// mockRoundTripper counts token requests to assert single-flight refresh.
func TestTokenCache_RefreshesAndCaches(t *testing.T) {
	var mu sync.Mutex
	refreshes := 0
	cache := newTokenCache(func() (string, error) {
		mu.Lock()
		refreshes++
		mu.Unlock()
		return "token-1", nil
	})

	first, err := cache.Get()
	if err != nil || first != "token-1" {
		t.Fatalf("first Get failed: %v, token=%q", err, first)
	}
	// Second Get within TTL must NOT refresh.
	second, err := cache.Get()
	if err != nil || second != "token-1" {
		t.Fatalf("second Get failed: %v, token=%q", err, second)
	}
	if refreshes != 1 {
		t.Fatalf("expected 1 refresh, got %d", refreshes)
	}

	// Expire the token and invalidate; next Get must refresh.
	cache.invalidate()
	third, err := cache.Get()
	if err != nil || third != "token-1" {
		t.Fatalf("refreshed Get failed: %v, token=%q", err, third)
	}
	if refreshes != 2 {
		t.Fatalf("expected 2 refreshes after invalidation, got %d", refreshes)
	}
}

func TestAdapter_401RetriesWithFreshToken(t *testing.T) {
	var mu sync.Mutex
	firstCall := true
	tokenRequests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenRequests++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"token-1"}`)
			return
		}
		if r.URL.Path == "/v1/customers" {
			mu.Lock()
			isFirst := firstCall
			firstCall = false
			mu.Unlock()
			if isFirst {
				w.WriteHeader(http.StatusUnauthorized) // expired token → adapter retries
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cust-1","name":"Test"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil)
	customer, err := adapter.UpsertCustomer(context.Background(), 1, domain.CustomerInfo{Name: "Test", Identification: "900123"})
	if err != nil {
		t.Fatalf("UpsertCustomer failed: %v", err)
	}
	if customer.ExternalID != "cust-1" {
		t.Fatalf("unexpected customer id %q", customer.ExternalID)
	}
	if tokenRequests != 2 {
		t.Fatalf("expected 2 token fetches (initial + post-401 refresh), got %d", tokenRequests)
	}
}

func TestAdapter_CreateInvoiceMapsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"token-1"}`)
			return
		}
		if r.URL.Path == "/v1/invoices" {
			// Assert request carries bearer token + JSON body shape.
			if r.Header.Get("Authorization") != "Bearer token-1" {
				t.Errorf("missing bearer token on invoice request")
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("bad request body: %v", err)
			}
			items, ok := body["items"].([]any)
			if !ok || len(items) != 1 {
				t.Fatalf("expected 1 item, got %v", body["items"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"inv-1","number":"FV1","cufe":"CUFE123","status":"valid","pdf_url":"https://pdf"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	amount := 150.0
	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil)
	inv, err := adapter.CreateInvoice(context.Background(), 7, &domain.InvoiceRequest{
		OrganizationID: 7,
		DealID:         99,
		Customer:       domain.CustomerInfo{Name: "Cliente", Identification: "C.C. 1"},
		Amount:         &amount,
		Currency:       "COP",
		Description:    "Negocio de prueba",
	})
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}
	if inv.ExternalID != "inv-1" || inv.Cufe != "CUFE123" || inv.Status != domain.InvoiceStatusValid {
		t.Fatalf("unexpected invoice mapping: %+v", inv)
	}
}

func TestAdapter_GetInvoiceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
			return
		}
		if r.URL.Path == "/v1/invoices/inv-9" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"inv-9","cufe":"C","status":"errored","pdf_url":"x"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil)
	inv, err := adapter.GetInvoiceStatus(context.Background(), 7, "inv-9")
	if err != nil {
		t.Fatalf("GetInvoiceStatus failed: %v", err)
	}
	if inv.Status != domain.InvoiceStatusErrored {
		t.Fatalf("expected errored status, got %q", inv.Status)
	}
}
