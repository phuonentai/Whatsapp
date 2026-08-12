package graphapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// ClientConfig holds the transport-level Meta app settings for Graph API calls.
type ClientConfig struct {
	AppID      string
	AppSecret  string
	APIBase    string // e.g. https://graph.facebook.com
	APIVersion string // e.g. v21.0
}

// MetaConfig carries the values the browser SDK needs to start Embedded Signup.
type MetaConfig struct {
	AppID       string `json:"app_id"`
	ConfigID    string `json:"config_id"`
	RedirectURI string `json:"redirect_uri"`
}

// GraphError is a structured Meta Graph API error.
type GraphError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Subcode int    `json:"error_subcode"`
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("graph api error (code %d): %s", e.Code, e.Message)
}

// WABAPhoneNumber describes a phone number inside a WhatsApp Business Account.
type WABAPhoneNumber struct {
	ID                     string `json:"id"`
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	CodeVerificationStatus string `json:"code_verification_status"`
	Certificate            string `json:"certificate"`
}

// WABAInfo describes a WhatsApp Business Account and its phone numbers.
type WABAInfo struct {
	ID           string            `json:"id"`
	DisplayName  string            `json:"display_name"`
	PhoneNumbers []WABAPhoneNumber `json:"phone_numbers"`
}

// Client is the seam for Meta Graph API integration; a mock implements it in tests.
type Client interface {
	// ExchangeCode trades the embedded-signup authorization code for a user access token.
	ExchangeCode(ctx context.Context, code string) (string, error)
	// ResolveBusiness returns the id of the first business the user token can manage.
	ResolveBusiness(ctx context.Context, userToken string) (string, error)
	// ResolveWABAAndNumbers lists the WABAs owned by the business with their phone numbers.
	ResolveWABAAndNumbers(ctx context.Context, userToken, businessID string) ([]WABAInfo, error)
	// CreateSystemUser provisions a Business Integration system user and returns its long-lived access token.
	CreateSystemUser(ctx context.Context, userToken, businessID, appID string) (string, error)
	// SubscribeWABA subscribes the app to the WABA's webhooks.
	SubscribeWABA(ctx context.Context, userToken, wabaID string) error
	// RegisterAppSubscriptions configures the app's WhatsApp subscription to deliver to callbackURL.
	RegisterAppSubscriptions(ctx context.Context, userToken, appID, callbackURL, verifyToken string) error
	// SendTestMessage sends a text message through the Cloud API (used for TTV validation).
	SendTestMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to string) error
	// SubmitTemplate creates a message template at Meta and returns its template id.
	SubmitTemplate(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, name, language, category, body string) (string, error)
	// GetTemplateStatus fetches a template's approval status from Meta.
	GetTemplateStatus(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, metaTemplateID string) (string, error)
}

// client is the real Graph API implementation, guarded by a two-tier circuit breaker
// (threshold 5, cooldown 10s, half-open probe 2) matching the repo's resilience idiom.
type client struct {
	cfg       ClientConfig
	http      *http.Client
	breaker   *whatsapp.CircuitBreaker
	userAgent string
}

// NewClient builds the real Graph API client. A nil httpClient yields a default one.
func NewClient(cfg ClientConfig, httpClient *http.Client) Client {
	return newClient(cfg, httpClient, whatsapp.NewCircuitBreaker(5, 10*time.Second, 2))
}

// newClient is the unexported constructor used by tests to inject a breaker.
func newClient(cfg ClientConfig, httpClient *http.Client, breaker *whatsapp.CircuitBreaker) Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &client{
		cfg:       cfg,
		http:      httpClient,
		breaker:   breaker,
		userAgent: "go-b2b-starter/1.0",
	}
}

// run wraps a Graph call with circuit-breaker semantics.
func (c *client) run(fn func() error) error {
	if !c.breaker.Allow() {
		return fmt.Errorf("circuit breaker open: graph api request blocked")
	}
	if err := fn(); err != nil {
		c.breaker.Failure()
		return err
	}
	c.breaker.Success()
	return nil
}

type graphErrorResponse struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		ErrorUserMsg string `json:"error_user_msg"`
	} `json:"error"`
}

func (c *client) doJSON(ctx context.Context, method, apiBase, apiVersion, path string, token string, form url.Values, out any) error {
	base := apiBase
	if base == "" {
		base = c.cfg.APIBase
	}
	version := apiVersion
	if version == "" {
		version = c.cfg.APIVersion
	}

	u := base + "/" + version + "/" + path
	if method == http.MethodGet && len(form) > 0 {
		u = u + "?" + form.Encode()
	}

	var bodyReader io.Reader
	if method == http.MethodPost && len(form) > 0 {
		bodyReader = bytes.NewReader([]byte(form.Encode()))
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create graph request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("graph request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read graph response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr graphErrorResponse
		if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error.Message != "" {
			return &GraphError{
				Code:    apiErr.Error.Code,
				Message: apiErr.Error.Message,
				Type:    apiErr.Error.Type,
				Subcode: apiErr.Error.ErrorSubcode,
			}
		}
		return fmt.Errorf("graph api error (status %d): %s", resp.StatusCode, string(raw))
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("failed to parse graph response: %w", err)
		}
	}
	return nil
}

func (c *client) ExchangeCode(ctx context.Context, code string) (string, error) {
	var token string
	err := c.run(func() error {
		form := url.Values{}
		form.Set("client_id", c.cfg.AppID)
		form.Set("client_secret", c.cfg.AppSecret)
		form.Set("redirect_uri", "")
		form.Set("code", code)
		var out struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int64  `json:"expires_in"`
		}
		if err := c.doJSON(ctx, http.MethodGet, c.cfg.APIBase, c.cfg.APIVersion, "oauth/access_token", "", form, &out); err != nil {
			return err
		}
		if out.AccessToken == "" {
			return fmt.Errorf("graph api returned no access token")
		}
		token = out.AccessToken
		return nil
	})
	return token, err
}

func (c *client) ResolveBusiness(ctx context.Context, userToken string) (string, error) {
	var businessID string
	err := c.run(func() error {
		form := url.Values{}
		form.Set("fields", "id,name")
		var out struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := c.doJSON(ctx, http.MethodGet, c.cfg.APIBase, c.cfg.APIVersion, "me/businesses", userToken, form, &out); err != nil {
			return err
		}
		if len(out.Data) == 0 {
			return fmt.Errorf("no business found for user token")
		}
		businessID = out.Data[0].ID
		return nil
	})
	return businessID, err
}

func (c *client) ResolveWABAAndNumbers(ctx context.Context, userToken, businessID string) ([]WABAInfo, error) {
	var wabas []WABAInfo
	err := c.run(func() error {
		form := url.Values{}
		form.Set("fields", "id,name")
		var wabaOut struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := c.doJSON(ctx, http.MethodGet, c.cfg.APIBase, c.cfg.APIVersion, businessID+"/owned_whatsapp_business_accounts", userToken, form, &wabaOut); err != nil {
			return err
		}
		for _, w := range wabaOut.Data {
			info := WABAInfo{ID: w.ID, DisplayName: w.Name}
			phoneForm := url.Values{}
			phoneForm.Set("fields", "id,display_phone_number,verified_name,code_verification_status,certificate")
			var phoneOut struct {
				Data []struct {
					ID                     string `json:"id"`
					DisplayPhoneNumber     string `json:"display_phone_number"`
					VerifiedName           string `json:"verified_name"`
					CodeVerificationStatus string `json:"code_verification_status"`
					Certificate            string `json:"certificate"`
				} `json:"data"`
			}
			if err := c.doJSON(ctx, http.MethodGet, c.cfg.APIBase, c.cfg.APIVersion, w.ID+"/phone_numbers", userToken, phoneForm, &phoneOut); err != nil {
				return err
			}
			for _, p := range phoneOut.Data {
				info.PhoneNumbers = append(info.PhoneNumbers, WABAPhoneNumber{
					ID:                     p.ID,
					DisplayPhoneNumber:     p.DisplayPhoneNumber,
					VerifiedName:           p.VerifiedName,
					CodeVerificationStatus: p.CodeVerificationStatus,
					Certificate:            p.Certificate,
				})
			}
			wabas = append(wabas, info)
		}
		return nil
	})
	return wabas, err
}

func (c *client) CreateSystemUser(ctx context.Context, userToken, businessID, appID string) (string, error) {
	var sysToken string
	err := c.run(func() error {
		form := url.Values{}
		form.Set("name", "Go B2B Starter Integration")
		form.Set("role", "EMPLOYEE")
		form.Set("system_user_type", "business_integration")
		form.Set("app_id", appID)
		var created struct {
			ID string `json:"id"`
		}
		if err := c.doJSON(ctx, http.MethodPost, c.cfg.APIBase, c.cfg.APIVersion, businessID+"/business_system_users", userToken, form, &created); err != nil {
			return err
		}
		if created.ID == "" {
			return fmt.Errorf("graph api returned no system user id")
		}

		tokenForm := url.Values{}
		tokenForm.Set("generate_access_token", "true")
		var tokenOut struct {
			AccessToken string `json:"access_token"`
		}
		if err := c.doJSON(ctx, http.MethodGet, c.cfg.APIBase, c.cfg.APIVersion, created.ID+"/access_tokens", userToken, tokenForm, &tokenOut); err != nil {
			return err
		}
		if tokenOut.AccessToken == "" {
			return fmt.Errorf("graph api returned no system user access token")
		}
		sysToken = tokenOut.AccessToken
		return nil
	})
	return sysToken, err
}

func (c *client) SubscribeWABA(ctx context.Context, userToken, wabaID string) error {
	return c.run(func() error {
		return c.doJSON(ctx, http.MethodPost, c.cfg.APIBase, c.cfg.APIVersion, wabaID+"/subscribed_apps", userToken, url.Values{}, nil)
	})
}

func (c *client) RegisterAppSubscriptions(ctx context.Context, userToken, appID, callbackURL, verifyToken string) error {
	return c.run(func() error {
		form := url.Values{}
		form.Set("object", "whatsapp_business_account")
		form.Set("callback_url", callbackURL)
		form.Set("verify_token", verifyToken)
		form.Set("fields", "messages,statuses")
		return c.doJSON(ctx, http.MethodPost, c.cfg.APIBase, c.cfg.APIVersion, appID+"/subscriptions", userToken, form, nil)
	})
}

func (c *client) SendTestMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to string) error {
	return c.run(func() error {
		base := c.cfg.APIBase
		if graphAPIURL != "" {
			base = graphAPIURL
		}
		version := c.cfg.APIVersion
		if apiVersion != "" {
			version = apiVersion
		}

		payload := map[string]any{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                to,
			"type":              "text",
			"text":              map[string]string{"body": "Your WhatsApp is connected!"},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal test message: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/"+version+"/"+phoneNumberID+"/messages", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create test message request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("test message request failed: %w", err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read test message response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			var apiErr graphErrorResponse
			if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error.Message != "" {
				return &GraphError{Code: apiErr.Error.Code, Message: apiErr.Error.Message, Type: apiErr.Error.Type}
			}
			return fmt.Errorf("graph api error (status %d): %s", resp.StatusCode, string(raw))
		}
		return nil
	})
}

// SubmitTemplate creates a message template at Meta for the org's phone number
// and returns the Meta-assigned template id. The body components use the same
// {{N}} placeholders stored locally; Meta validates the component count.
func (c *client) SubmitTemplate(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, name, language, category, body string) (string, error) {
	var templateID string
	err := c.run(func() error {
		base := c.cfg.APIBase
		if graphAPIURL != "" {
			base = graphAPIURL
		}
		version := c.cfg.APIVersion
		if apiVersion != "" {
			version = apiVersion
		}

		components, err := json.Marshal([]map[string]any{{
			"type": "BODY",
			"text": body,
		}})
		if err != nil {
			return fmt.Errorf("failed to marshal template components: %w", err)
		}

		form := url.Values{}
		form.Set("name", name)
		form.Set("language", language)
		form.Set("category", category)
		form.Set("components", string(components))

		var created struct {
			ID string `json:"id"`
		}
		if err := c.doJSON(ctx, http.MethodPost, base, version, phoneNumberID+"/message_templates", accessToken, form, &created); err != nil {
			return err
		}
		if created.ID == "" {
			return fmt.Errorf("graph api returned no template id")
		}
		templateID = created.ID
		return nil
	})
	return templateID, err
}

// GetTemplateStatus fetches a template's approval status from Meta. Returns
// the raw status string (e.g. APPROVED, REJECTED, PAUSED, IN_APPEAL).
func (c *client) GetTemplateStatus(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, metaTemplateID string) (string, error) {
	var status string
	err := c.run(func() error {
		base := c.cfg.APIBase
		if graphAPIURL != "" {
			base = graphAPIURL
		}
		version := c.cfg.APIVersion
		if apiVersion != "" {
			version = apiVersion
		}

		form := url.Values{}
		form.Set("fields", "status")
		var out struct {
			Status string `json:"status"`
		}
		if err := c.doJSON(ctx, http.MethodGet, base, version, metaTemplateID, accessToken, form, &out); err != nil {
			return err
		}
		status = out.Status
		return nil
	})
	return status, err
}
