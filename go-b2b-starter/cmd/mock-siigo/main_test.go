package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	state := newMockState()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "mock-token"})
	})
	mux.HandleFunc("/v1/company", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": 404})
	})
	mux.HandleFunc("/v1/customers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var payload struct {
				Name           string `json:"name"`
				Identification string `json:"identification"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":   fmt.Sprintf("cust-%s", payload.Identification),
				"name": payload.Name,
			})
		case http.MethodGet:
			page := 0
			if v := r.URL.Query().Get("page"); v != "" {
				_, _ = fmtSscan(v, &page)
			}
			start := page * pageSize
			end := start + pageSize
			if end > len(customers) {
				end = len(customers)
			}
			writeJSON(w, http.StatusOK, customers[start:end])
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/invoices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusCreated, state.createInvoice(r.Header.Get("Idempotency-Key")))
	})
	mux.HandleFunc("/v1/invoices/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/v1/invoices/"):]
		state.mu.Lock()
		inv, ok := state.invoices[id]
		state.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": 404})
			return
		}
		writeJSON(w, http.StatusOK, inv)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func fmtSscan(s string, out *int) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	*out = n
	return n, nil
}

func doJSON(t *testing.T, method, url, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	return resp.StatusCode, payload
}

func TestMock_TokenGrant(t *testing.T) {
	srv := newTestServer(t)
	status, payload := doJSON(t, http.MethodPost, srv.URL+"/token", "", nil)
	if status != http.StatusOK || payload["access_token"] != "mock-token" {
		t.Fatalf("unexpected token response: %d %v", status, payload)
	}
}

func TestMock_CompanyEndpointAbsent(t *testing.T) {
	srv := newTestServer(t)
	status, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/company", "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for /v1/company, got %d", status)
	}
}

func TestMock_CustomerPagination(t *testing.T) {
	srv := newTestServer(t)
	// All 4 customers on page 0 (short page ends the pull loop).
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/customers?page=0", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var arr []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 4 {
		t.Fatalf("expected 4 customers on page 0, got %d", len(arr))
	}
}

func TestMock_CustomerUpsert(t *testing.T) {
	srv := newTestServer(t)

	// POST returns the {id, name} object the adapter decodes (201).
	status, created := doJSON(t, http.MethodPost, srv.URL+"/v1/customers",
		`{"name":"Negocio Manual","identification":"500000001","identification_type":"NIT"}`, nil)
	if status != http.StatusCreated {
		t.Fatalf("upsert failed: %d", status)
	}
	if created["id"] != "cust-500000001" || created["name"] != "Negocio Manual" {
		t.Fatalf("unexpected upsert response: %v", created)
	}

	// Same identification → same deterministic id (idempotent upsert).
	status, again := doJSON(t, http.MethodPost, srv.URL+"/v1/customers",
		`{"name":"Negocio Manual","identification":"500000001","identification_type":"NIT"}`, nil)
	if status != http.StatusCreated || again["id"] != created["id"] {
		t.Fatalf("upsert must be idempotent by identification: %d %v", status, again)
	}
}

func TestMock_InvoiceNumberingAndIdempotency(t *testing.T) {
	srv := newTestServer(t)

	status, first := doJSON(t, http.MethodPost, srv.URL+"/v1/invoices", `{}`, map[string]string{"Idempotency-Key": "o7d99"})
	if status != http.StatusCreated {
		t.Fatalf("create failed: %d", status)
	}
	if first["number"] != "FAC1-00001" {
		t.Fatalf("expected first number FAC1-00001, got %v", first["number"])
	}

	// Same key → same invoice, no duplicate, no number consumed.
	status, second := doJSON(t, http.MethodPost, srv.URL+"/v1/invoices", `{}`, map[string]string{"Idempotency-Key": "o7d99"})
	if status != http.StatusCreated {
		t.Fatalf("dedupe create failed: %d", status)
	}
	if second["id"] != first["id"] {
		t.Fatalf("same key must return the same invoice: %v vs %v", first["id"], second["id"])
	}

	// Different key → next consecutive number.
	status, third := doJSON(t, http.MethodPost, srv.URL+"/v1/invoices", `{}`, map[string]string{"Idempotency-Key": "o8d1"})
	if status != http.StatusCreated {
		t.Fatal(status)
	}
	if third["number"] != "FAC1-00002" {
		t.Fatalf("expected FAC1-00002, got %v", third["number"])
	}

	// Status lookup resolves the invoice.
	status, got := doJSON(t, http.MethodGet, srv.URL+"/v1/invoices/"+first["id"].(string), "", nil)
	if status != http.StatusOK || got["status"] != "valid" {
		t.Fatalf("unexpected status lookup: %d %v", status, got)
	}
}
