// Command mock-siigo is the offline e2e mock of the Siigo REST API.
//
// It implements the adapter surface (internal/modules/invoicing/infra/siigo)
// against the spike-verified contract (add-siigo-onboarding-data task 1.1):
//
//   - POST {tokenURL}      OAuth2 client_credentials grant → {"access_token": "mock-token"}
//   - GET  /v1/company     404 — Siigo exposes NO company resource (spike finding;
//     exercises the adapter's tolerant ValidateCredentials path)
//   - GET  /v1/customers?page={n}&page_size=100   0-based pagination, short page ends pull
//   - POST /v1/customers    upsert: deterministic id from identification, 201 {id, name}
//   - POST /v1/invoices    consecutive numbering (FAC1-00001…); Idempotency-Key
//     dedupe: same key → returns the first-created invoice
//   - GET  /v1/invoices/{id}  status lookup (valid, with CUFE + pdf_url)
//
// All state is in-memory and resets on restart; the e2e runner drops the DB per
// run, so the mock resets in lockstep. No real Siigo traffic ever occurs in e2e.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const (
	pageSize      = 100
	invoicePrefix = "FAC1-"
)

// customers is the deterministic roster returned by /v1/customers.
var customers = []map[string]any{
	{"id": "cust-1", "name": "Cliente Uno", "identification": "900111222", "identification_type": "NIT", "email": "cliente1@mock.co", "phone": "+57300111111"},
	{"id": "cust-2", "name": "Cliente Dos", "identification": "900333444", "identification_type": "NIT", "email": "cliente2@mock.co", "phone": "+57300222222"},
	{"id": "cust-3", "name": "Cliente Tres", "identification": "800555666", "identification_type": "NIT", "email": "", "phone": ""},
	{"id": "cust-4", "name": "Cliente Cuatro", "identification": "900777888", "identification_type": "NIT", "email": "cliente4@mock.co", "phone": "+57300444444"},
}

type mockState struct {
	mu       sync.Mutex
	nextNum  int
	invoices map[string]map[string]any // id -> invoice
	byKey    map[string]string         // idempotency key -> invoice id
}

func newMockState() *mockState {
	return &mockState{nextNum: 1, invoices: map[string]map[string]any{}, byKey: map[string]string{}}
}

func (s *mockState) createInvoice(key string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" {
		if id, ok := s.byKey[key]; ok {
			return s.invoices[id]
		}
	}
	id := fmt.Sprintf("inv-%d", s.nextNum)
	number := fmt.Sprintf("%s%05d", invoicePrefix, s.nextNum)
	s.nextNum++
	inv := map[string]any{
		"id": id, "number": number, "status": "valid",
		"cufe":    fmt.Sprintf("CUFE-MOCK-%d", s.nextNum-1),
		"pdf_url": fmt.Sprintf("https://mock-siigo.local/pdf/%s", id),
	}
	s.invoices[id] = inv
	if key != "" {
		s.byKey[key] = id
	}
	return inv
}

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()

	state := newMockState()

	mux := http.NewServeMux()

	// POST {tokenURL}: OAuth2 client_credentials grant (any credentials accepted).
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "mock-token"})
	})

	// GET /v1/company → 404: Siigo exposes no company resource (spike finding).
	mux.HandleFunc("/v1/company", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": 404, "message": "Resource not found"})
	})

	// GET /v1/customers: 0-based pagination; short page ends the pull.
	// POST /v1/customers: upsert by identification; deterministic id.
	mux.HandleFunc("/v1/customers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var payload struct {
				Name               string `json:"name"`
				Identification     string `json:"identification"`
				IdentificationType string `json:"identification_type"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			id := fmt.Sprintf("cust-%s", payload.Identification)
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":   id,
				"name": payload.Name,
			})
		case http.MethodGet:
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			start := page * pageSize
			if start < 0 {
				start = 0
			}
			end := start + pageSize
			if end > len(customers) {
				end = len(customers)
			}
			pageItems := customers[start:end]
			if pageItems == nil {
				pageItems = []map[string]any{}
			}
			writeJSON(w, http.StatusOK, pageItems)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// POST /v1/invoices: consecutive numbering + Idempotency-Key dedupe.
	mux.HandleFunc("/v1/invoices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			key := r.Header.Get("Idempotency-Key")
			inv := state.createInvoice(key)
			writeJSON(w, http.StatusCreated, inv)
		case http.MethodGet:
			http.NotFound(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /v1/invoices/{id}
	mux.HandleFunc("/v1/invoices/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/invoices/")
		state.mu.Lock()
		inv, ok := state.invoices[id]
		state.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": 404, "message": "Resource not found"})
			return
		}
		writeJSON(w, http.StatusOK, inv)
	})

	// GET /healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	log.Printf("mock-siigo listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
