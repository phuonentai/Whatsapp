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
	creds      domain.CredentialProvider
	debug      bool
}

// NewAdapter builds the Siigo adapter. client may be nil (default HTTP client
// used). creds may be nil, in which case the adapter uses the env credentials
// from cfg (platform-level single account, legacy mode).
func NewAdapter(cfg *Config, client httpClient, creds domain.CredentialProvider) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	a := &Adapter{cfg: cfg, http: client, debug: cfg.Debug, creds: creds}
	a.tokenCache = newTokenCache(a.fetchToken)
	return a
}

// fetchToken performs the OAuth2 client_credentials grant for an
// organization, resolving credentials from the per-org credential provider
// (or env config when no provider is wired).
func (a *Adapter) fetchToken(orgID int32) (string, error) {
	clientID, clientSecret := a.cfg.ClientID, a.cfg.ClientSecret
	if a.creds != nil {
		var err error
		clientID, clientSecret, err = a.creds.ResolveCredentials(context.Background(), orgID)
		if err != nil {
			return "", fmt.Errorf("%w: %v", domain.ErrCredentialResolution, err)
		}
	}
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("%w: empty credentials for organization %d", domain.ErrCredentialResolution, orgID)
	}
	form := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
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

func (a *Adapter) do(ctx context.Context, orgID int32, method, path string, body any, extraHeaders ...http.Header) (*http.Response, error) {
	token, err := a.tokenCache.Get(orgID)
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
	for _, h := range extraHeaders {
		for k, vals := range h {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("siigo request failed: %w", err)
	}
	// Transparent retry once on expired-token 401.
	if resp.StatusCode == http.StatusUnauthorized {
		a.tokenCache.invalidate(orgID)
		_ = resp.Body.Close()

		token, err := a.tokenCache.Get(orgID)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return a.http.Do(req)
	}
	return resp, nil
}

// idempotencyKey builds the provider Idempotency-Key header value (spike:
// alphanumeric, no specials, max 30 chars). One key per (org, deal) makes
// retries idempotent — a retried POST returns the previously created
// comprobante instead of duplicating it.
func idempotencyKey(orgID, dealID int32) string {
	return fmt.Sprintf("o%dd%d", orgID, dealID)
}

// ValidateCredentials verifies a credential pair against the provider (token
// grant + company lookup) without touching the cache or any stored state.
// Used by the connect flow before persisting anything. Credentials never
// appear in logs or returned errors.
func (a *Adapter) ValidateCredentials(ctx context.Context, clientID, clientSecret string) (domain.ProviderCompany, error) {
	form := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.TokenURL, bytes.NewBufferString(form))
	if err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.http.Do(req)
	if err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("siigo token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 1024))
		return domain.ProviderCompany{}, fmt.Errorf("%w: token request returned %d", domain.ErrInvalidCredentials, resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("siigo token response decode failed: %w", err)
	}
	if payload.AccessToken == "" {
		return domain.ProviderCompany{}, fmt.Errorf("%w: token response missing access_token", domain.ErrInvalidCredentials)
	}

	companyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.BaseURL+"/v1/company", nil)
	if err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("failed to build company request: %w", err)
	}
	companyReq.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	companyReq.Header.Set("Accept", "application/json")

	companyResp, err := a.http.Do(companyReq)
	if err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("siigo company request failed: %w", err)
	}
	defer companyResp.Body.Close()

	// Spike (2026-08): Siigo's REST API exposes NO company resource — /v1/company
	// returns 404. Credential validation therefore succeeds on the token grant
	// alone; the company lookup is best-effort and returns empty data when the
	// endpoint is absent, so connect does not fail on a missing NIT source.
	if companyResp.StatusCode == http.StatusNotFound || companyResp.StatusCode == http.StatusMethodNotAllowed {
		return domain.ProviderCompany{}, nil
	}
	if companyResp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(io.LimitReader(companyResp.Body, 1024))
		return domain.ProviderCompany{}, fmt.Errorf("%w: company request returned %d", domain.ErrInvalidCredentials, companyResp.StatusCode)
	}

	var company struct {
		Nit  string `json:"nit"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(companyResp.Body).Decode(&company); err != nil {
		return domain.ProviderCompany{}, fmt.Errorf("failed to decode company response: %w", err)
	}
	return domain.ProviderCompany{Nit: company.Nit, Name: company.Name}, nil
}

// GetNumeration reads the organization's active DIAN numeration. Spike
// verified (2026-08): Siigo's REST API exposes no numeration resource and
// invoices number automatically under the company's resolution — mode auto,
// resolution fields empty. Manual mode (platform-supplied number) is
// available behind config for providers that require it.
func (a *Adapter) GetNumeration(ctx context.Context, orgID int32) (domain.NumerationInfo, error) {
	if domain.NumerationMode(a.cfg.NumberingMode) != domain.NumerationManual {
		return domain.NumerationInfo{Mode: domain.NumerationAuto}, nil
	}
	// Manual mode: best-effort next number = last issued + 1 (page 0 returns
	// the most recent invoices). Idempotency-Key protects against duplicates.
	resp, err := a.do(ctx, orgID, http.MethodGet, "/v1/invoices?page=0", nil)
	if err != nil {
		return domain.NumerationInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.NumerationInfo{}, fmt.Errorf("%w: numeration read returned %d", domain.ErrProvider, resp.StatusCode)
	}
	var invoices []struct {
		Number string `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invoices); err != nil {
		return domain.NumerationInfo{}, fmt.Errorf("failed to decode invoice list: %w", err)
	}
	next := ""
	if len(invoices) > 0 && invoices[0].Number != "" {
		next = incrementNumber(invoices[0].Number)
	}
	return domain.NumerationInfo{Mode: domain.NumerationManual, NextNumber: next}, nil
}

// ListCustomers fetches one page of provider customers (page is 0-based).
func (a *Adapter) ListCustomers(ctx context.Context, orgID int32, page int32) ([]domain.CustomerRecord, error) {
	resp, err := a.do(ctx, orgID, http.MethodGet, fmt.Sprintf("/v1/customers?page=%d&page_size=100", page), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: customer list returned %d", domain.ErrProvider, resp.StatusCode)
	}
	var items []struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		Identification     string `json:"identification"`
		IdentificationType string `json:"identification_type"`
		Email              string `json:"email"`
		Phone              string `json:"phone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode customer list: %w", err)
	}
	records := make([]domain.CustomerRecord, len(items))
	for i, it := range items {
		records[i] = domain.CustomerRecord{
			ExternalID: it.ID, Name: it.Name,
			Identification: it.Identification, IdentificationType: it.IdentificationType,
			Email: it.Email, Phone: it.Phone,
		}
	}
	return records, nil
}

// incrementNumber raises an alphanumeric consecutive by 1 (e.g. "FAC1-00123"
// → "FAC1-00124"). Falls back to the raw value when not parseable.
func incrementNumber(number string) string {
	idx := len(number)
	for i := len(number) - 1; i >= 0; i-- {
		if number[i] < '0' || number[i] > '9' {
			break
		}
		idx = i
	}
	if idx == len(number) {
		return number
	}
	n := 0
	for _, r := range number[idx:] {
		n = n*10 + int(r-'0')
	}
	digits := len(number) - idx
	next := fmt.Sprintf("%0*d", digits, n+1)
	return number[:idx] + next
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

	resp, err := a.do(ctx, orgID, http.MethodPost, "/v1/customers", payload)
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
	// Manual numbering mode: supply the confirmed consecutive. Auto mode
	// (Siigo default) assigns the number automatically.
	if req.Number != "" {
		payload["number"] = req.Number
	}

	reqWithBody, err := a.do(ctx, orgID, http.MethodPost, "/v1/invoices", payload, http.Header{
		// Idempotency-Key (spike-verified, max 30 chars alphanumeric): safe
		// retries without duplicate comprobantes. One key per (org, deal).
		"Idempotency-Key": {idempotencyKey(orgID, req.DealID)},
	})
	if err != nil {
		return nil, err
	}
	defer reqWithBody.Body.Close()

	if reqWithBody.StatusCode != http.StatusCreated && reqWithBody.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(reqWithBody.Body, 1024))
		return nil, fmt.Errorf("%w: create invoice returned %d: %s", domain.ErrProvider, reqWithBody.StatusCode, string(body))
	}

	var out struct {
		ID     string `json:"id"`
		Number string `json:"number"`
		Cufe   string `json:"cufe"`
		Status string `json:"status"`
		PdfURL string `json:"pdf_url"`
	}
	if err := json.NewDecoder(reqWithBody.Body).Decode(&out); err != nil {
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
func (a *Adapter) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {	resp, err := a.do(ctx, orgID, http.MethodGet, "/v1/invoices/"+externalID, nil)
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
