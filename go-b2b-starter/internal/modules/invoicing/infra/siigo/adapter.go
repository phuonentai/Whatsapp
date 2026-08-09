package siigo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// httpClient abstracts the transport so tests can inject a mock server.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter implements domain.InvoicingProvider against the Siigo REST API.
// Kept transport-coupled here (infra layer) per the governance rule; domain
// imports no SDKs.
type Adapter struct {
	cfg        *Config
	http       httpClient
	tokenCache *tokenCache
	debug      bool
}

func NewAdapter(cfg *Config, client httpClient) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	a := &Adapter{cfg: cfg, http: client, debug: cfg.Debug}
	a.tokenCache = newTokenCache(a.fetchToken)
	return a
}

// fetchToken performs the OAuth2 client_credentials grant.
func (a *Adapter) fetchToken() (string, error) {
	form := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s",
		a.cfg.ClientID, a.cfg.ClientSecret)
	req, err := http.NewRequest(http.MethodPost, a.cfg.TokenURL, bytes.NewBufferString(form))
	if err != nil {
		return "", fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("siigo token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("siigo token request returned %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("siigo token response decode failed: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("siigo token response missing access_token")
	}
	return payload.AccessToken, nil
}

func (a *Adapter) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	token, err := a.tokenCache.Get()
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.cfg.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("siigo request failed: %w", err)
	}
	// Transparent retry once on expired-token 401.
	if resp.StatusCode == http.StatusUnauthorized {
		a.tokenCache.invalidate()
		_ = resp.Body.Close()

		token, err := a.tokenCache.Get()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return a.http.Do(req)
	}
	return resp, nil
}

// UpsertCustomer creates or finds a customer by identification.
func (a *Adapter) UpsertCustomer(ctx context.Context, orgID int32, customer domain.CustomerInfo) (*domain.CustomerRef, error) {
	payload := map[string]any{
		"name":              customer.Name,
		"type":              "Customer",
		"identification":    customer.Identification,
		"identification_type": customer.IdentificationType,
	}
	if customer.Email != "" {
		payload["email"] = customer.Email
	}
	if customer.Phone != "" {
		payload["phone"] = customer.Phone
	}

	resp, err := a.do(ctx, http.MethodPost, "/v1/customers", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: upsert customer returned %d: %s", domain.ErrProvider, resp.StatusCode, string(body))
	}

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode customer response: %w", err)
	}
	return &domain.CustomerRef{ExternalID: out.ID, Name: out.Name}, nil
}

// CreateInvoice creates a sales invoice at the provider.
func (a *Adapter) CreateInvoice(ctx context.Context, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error) {
	payload := map[string]any{
		"document": map[string]any{
			"id": "FV", // factura de venta
		},
		"customer": map[string]any{
			"identification": req.Customer.Identification,
			"name":           req.Customer.Name,
		},
		"items": []map[string]any{{
			"description": req.Description,
			"quantity":    1,
			"price":       amountOrZero(req.Amount),
		}},
	}

	resp, err := a.do(ctx, http.MethodPost, "/v1/invoices", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: create invoice returned %d: %s", domain.ErrProvider, resp.StatusCode, string(body))
	}

	var out struct {
		ID     string `json:"id"`
		Number string `json:"number"`
		Cufe   string `json:"cufe"`
		Status string `json:"status"`
		PdfURL string `json:"pdf_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode invoice response: %w", err)
	}

	status := domain.InvoiceStatusPending
	switch out.Status {
	case "valid":
		status = domain.InvoiceStatusValid
	case "invalid":
		status = domain.InvoiceStatusInvalid
	case "errored":
		status = domain.InvoiceStatusErrored
	}

	return &domain.Invoice{
		OrganizationID: orgID,
		DealID:         req.DealID,
		ExternalID:     out.ID,
		Cufe:           out.Cufe,
		Status:         status,
		PdfURL:         out.PdfURL,
		Amount:         req.Amount,
		Currency:       req.Currency,
	}, nil
}

// GetInvoiceStatus fetches the current status of an invoice from the provider.
func (a *Adapter) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {	resp, err := a.do(ctx, http.MethodGet, "/v1/invoices/"+externalID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: get invoice returned %d: %s", domain.ErrProvider, resp.StatusCode, string(body))
	}

	var out struct {
		ID     string `json:"id"`
		Cufe   string `json:"cufe"`
		Status string `json:"status"`
		PdfURL string `json:"pdf_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode invoice status response: %w", err)
	}

	status := domain.InvoiceStatusPending
	switch out.Status {
	case "valid":
		status = domain.InvoiceStatusValid
	case "invalid":
		status = domain.InvoiceStatusInvalid
	case "errored":
		status = domain.InvoiceStatusErrored
	}

	return &domain.Invoice{
		ExternalID: out.ID,
		Cufe:       out.Cufe,
		Status:     status,
		PdfURL:     out.PdfURL,
	}, nil
}

func amountOrZero(amount *float64) float64 {
	if amount == nil {
		return 0
	}
	return *amount
}
