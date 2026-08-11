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
	cache := newTokenCache(func(orgID int32) (string, error) {
		mu.Lock()
		refreshes++
		mu.Unlock()
		return "token-1", nil
	})

	first, err := cache.Get(1)
	if err != nil || first != "token-1" {
		t.Fatalf("first Get failed: %v, token=%q", err, first)
	}
	// Second Get within TTL must NOT refresh.
	second, err := cache.Get(1)
	if err != nil || second != "token-1" {
		t.Fatalf("second Get failed: %v, token=%q", err, second)
	}
	if refreshes != 1 {
		t.Fatalf("expected 1 refresh, got %d", refreshes)
	}

	// Expire the token and invalidate; next Get must refresh.
	cache.invalidate(1)
	third, err := cache.Get(1)
	if err != nil || third != "token-1" {
		t.Fatalf("refreshed Get failed: %v, token=%q", err, third)
	}
	if refreshes != 2 {
		t.Fatalf("expected 2 refreshes after invalidation, got %d", refreshes)
	}
}

// TestTokenCache_IsolatesOrganizations asserts tokens are cached per org.
func TestTokenCache_IsolatesOrganizations(t *testing.T) {
	var mu sync.Mutex
	refreshes := 0
	cache := newTokenCache(func(orgID int32) (string, error) {
		mu.Lock()
		refreshes++
		mu.Unlock()
		return "token-for-" + fmt.Sprint(orgID), nil
	})

	tok1, err := cache.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := cache.Get(2)
	if err != nil {
		t.Fatal(err)
	}
	if tok1 == tok2 {
		t.Fatalf("orgs must not share tokens: %q", tok1)
	}
	if refreshes != 2 {
		t.Fatalf("expected 2 refreshes for 2 orgs, got %d", refreshes)
	}
	// Cached lookups do not refresh.
	_, _ = cache.Get(1)
	_, _ = cache.Get(2)
	if refreshes != 2 {
		t.Fatalf("expected no extra refreshes, got %d", refreshes)
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

	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil, nil)
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
	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil, nil)
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

	adapter := NewAdapter(newTestConfig(srv.URL+"/token", srv.URL), nil, nil)
	inv, err := adapter.GetInvoiceStatus(context.Background(), 7, "inv-9")
	if err != nil {
		t.Fatalf("GetInvoiceStatus failed: %v", err)
	}
	if inv.Status != domain.InvoiceStatusErrored {
		t.Fatalf("expected errored status, got %q", inv.Status)
	}
}

func TestAdapter_GetNumeration_AutoMode(t *testing.T) {
	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", NumberingMode: "auto"}, nil, nil)
	info, err := adapter.GetNumeration(context.Background(), 1)
	if err != nil {
		t.Fatalf("auto numeration must not error: %v", err)
	}
	if info.Mode != domain.NumerationAuto {
		t.Fatalf("expected auto mode, got %s", info.Mode)
	}
}

func TestAdapter_GetNumeration_ManualReadsLastInvoice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
		case r.URL.Path == "/v1/invoices":
			if r.URL.Query().Get("page") != "0" {
				t.Errorf("expected page=0, got %v", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id":"i1","number":"FAC1-00123"},{"id":"i2","number":"FAC1-00122"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token", NumberingMode: "manual"}, nil, nil)
	info, err := adapter.GetNumeration(context.Background(), 1)
	if err != nil {
		t.Fatalf("manual numeration failed: %v", err)
	}
	if info.Mode != domain.NumerationManual || info.NextNumber != "FAC1-00124" {
		t.Fatalf("unexpected numeration: %+v", info)
	}
}

func TestAdapter_ListCustomers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
		case r.URL.Path == "/v1/customers":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id":"c1","name":"Cliente Uno","identification":"900111222","identification_type":"NIT","email":"a@b.co","phone":"+573001"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	customers, err := adapter.ListCustomers(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("ListCustomers failed: %v", err)
	}
	if len(customers) != 1 || customers[0].Name != "Cliente Uno" || customers[0].Identification != "900111222" {
		t.Fatalf("unexpected customers: %+v", customers)
	}
}

func TestAdapter_CreateInvoice_ManualNumberAndIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
		case r.URL.Path == "/v1/invoices":
			if r.Header.Get("Idempotency-Key") != "o7d99" {
				t.Errorf("missing Idempotency-Key header, got %q", r.Header.Get("Idempotency-Key"))
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("bad body: %v", err)
			}
			if body["number"] != "FAC1-00124" {
				t.Fatalf("expected manual number in payload, got %v", body["number"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"inv-9","number":"FAC1-00124","status":"pending"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	inv, err := adapter.CreateInvoice(context.Background(), 7, &domain.InvoiceRequest{
		DealID: 99, Customer: domain.CustomerInfo{Name: "X", Identification: "1"}, Description: "d", Number: "FAC1-00124",
	})
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}
	if inv.ExternalID != "inv-9" {
		t.Fatalf("unexpected invoice: %+v", inv)
	}
}

func TestAdapter_ValidateCredentials_CompanyEndpointAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	company, err := adapter.ValidateCredentials(context.Background(), "c", "s")
	if err != nil {
		t.Fatalf("credentials must validate on token grant alone: %v", err)
	}
	if company.Nit != "" || company.Name != "" {
		t.Fatalf("expected empty company when endpoint absent, got %+v", company)
	}
}

func TestIncrementNumber(t *testing.T) {
	cases := map[string]string{
		"FAC1-00123": "FAC1-00124",
		"FV 99":      "FV 100",
		"999":        "1000",
		"AB":         "AB",
	}
	for in, want := range cases {
		if got := incrementNumber(in); got != want {
			t.Errorf("incrementNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAdapter_ValidateCredentials_CompanyEndpointPresent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
			return
		}
		if r.URL.Path == "/v1/company" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"nit":"900111222","name":"Mi Empresa"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	company, err := adapter.ValidateCredentials(context.Background(), "c", "s")
	if err != nil {
		t.Fatalf("ValidateCredentials failed: %v", err)
	}
	if company.Nit != "900111222" || company.Name != "Mi Empresa" {
		t.Fatalf("unexpected company: %+v", company)
	}
}

func TestAdapter_ValidateCredentials_CompanyPresentWithMismatchData(t *testing.T) {
	// The adapter returns provider data as-is; the mismatch decision belongs
	// to the connection service (covered there). This asserts data passthrough.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t"}`)
			return
		}
		if r.URL.Path == "/v1/company" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"nit":"999888777","name":"Otra Empresa"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "c", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	company, err := adapter.ValidateCredentials(context.Background(), "c", "s")
	if err != nil {
		t.Fatalf("ValidateCredentials failed: %v", err)
	}
	if company.Nit != "999888777" {
		t.Fatalf("expected provider NIT passthrough, got %q", company.Nit)
	}
}

func TestAdapter_ValidateCredentials_TokenGrantRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_client"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	adapter := NewAdapter(&Config{ClientID: "bad", ClientSecret: "bad", BaseURL: srv.URL, TokenURL: srv.URL + "/token"}, nil, nil)
	if _, err := adapter.ValidateCredentials(context.Background(), "bad", "bad"); err == nil {
		t.Fatal("expected error for rejected token grant")
	}
}
