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

// ClientConfig holds the transport-level Meta app settings for Instagram
// Graph API calls. Values are lenient: the Instagram feature is optional, so
// unset env vars fall back to defaults instead of failing startup.
type ClientConfig struct {
	AppID      string
	AppSecret  string
	APIBase    string // e.g. https://graph.facebook.com
	APIVersion string // e.g. v21.0
}

// IGUser describes an Instagram user resolved from the Graph API.
type IGUser struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	ProfilePictureURL string `json:"profile_picture_url"`
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

// IGClient is the seam for the Instagram Graph API (product `messaging`);
// a mock implements it in tests.
type IGClient interface {
	// SendTextMessage sends a DM text message to the recipient IG user.
	// Returns the provider message id on success.
	SendTextMessage(ctx context.Context, accessToken, baseURL, apiVersion, igUserID, recipientID, text string) (string, error)
	// GetIGUser resolves an Instagram user's username and profile picture.
	GetIGUser(ctx context.Context, accessToken, baseURL, apiVersion, igUserID string) (*IGUser, error)
	// RefreshToken exchanges the current token for a new long-lived token
	// via the fb_exchange_token grant. Returns the token and its expiry.
	RefreshToken(ctx context.Context, appID, appSecret, baseURL, apiVersion, token string) (string, *time.Time, error)
}

// igClient is the real Instagram Graph API implementation, guarded by a
// circuit breaker (threshold 5, cooldown 10s, half-open probe 2).
type igClient struct {
	cfg     ClientConfig
	http    *http.Client
	breaker *whatsapp.CircuitBreaker
}

// NewIGClient builds the real Instagram Graph API client. A nil httpClient
// yields a default one.
func NewIGClient(cfg ClientConfig, httpClient *http.Client) IGClient {
	return newIGClient(cfg, httpClient, whatsapp.NewCircuitBreaker(5, 10*time.Second, 2))
}

// newIGClient is the unexported constructor used by tests to inject a breaker.
func newIGClient(cfg ClientConfig, httpClient *http.Client, breaker *whatsapp.CircuitBreaker) IGClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.APIBase == "" {
		cfg.APIBase = "https://graph.facebook.com"
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v21.0"
	}
	return &igClient{cfg: cfg, http: httpClient, breaker: breaker}
}

// run wraps a Graph call with circuit-breaker semantics.
func (c *igClient) run(fn func() error) error {
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
	} `json:"error"`
}

func (c *igClient) doJSON(ctx context.Context, method, apiBase, apiVersion, path string, token string, form url.Values, bodyJSON any, out any) error {
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
	if method == http.MethodPost && bodyJSON != nil {
		raw, err := json.Marshal(bodyJSON)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	} else if method == http.MethodPost && len(form) > 0 {
		bodyReader = bytes.NewReader([]byte(form.Encode()))
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create graph request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if method == http.MethodPost {
		if bodyJSON != nil {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
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

// SendTextMessage posts a DM to the recipient via the Instagram Messaging API:
// POST {base}/{version}/{ig_user_id}/messages with
// {"recipient": {"id": <recipient>}, "message": {"text": <text>}}.
func (c *igClient) SendTextMessage(ctx context.Context, accessToken, baseURL, apiVersion, igUserID, recipientID, text string) (string, error) {
	var messageID string
	err := c.run(func() error {
		payload := map[string]any{
			"recipient": map[string]string{"id": recipientID},
			"message":   map[string]string{"text": text},
		}
		var out struct {
			MessageID string `json:"message_id"`
		}
		if err := c.doJSON(ctx, http.MethodPost, baseURL, apiVersion, igUserID+"/messages", accessToken, nil, payload, &out); err != nil {
			return err
		}
		if out.MessageID == "" {
			return fmt.Errorf("graph api returned no message id")
		}
		messageID = out.MessageID
		return nil
	})
	return messageID, err
}

// GetIGUser resolves username and profile picture for an IG user.
func (c *igClient) GetIGUser(ctx context.Context, accessToken, baseURL, apiVersion, igUserID string) (*IGUser, error) {
	var user *IGUser
	err := c.run(func() error {
		form := url.Values{}
		form.Set("fields", "username,profile_picture_url")
		var out IGUser
		if err := c.doJSON(ctx, http.MethodGet, baseURL, apiVersion, igUserID, accessToken, form, nil, &out); err != nil {
			return err
		}
		user = &out
		return nil
	})
	return user, err
}

// RefreshToken exchanges the current long-lived token for a new one via
// grant_type=fb_exchange_token. Returns the token and expiry derived from
// expires_in (nil when the response omits it).
func (c *igClient) RefreshToken(ctx context.Context, appID, appSecret, baseURL, apiVersion, token string) (string, *time.Time, error) {
	var newToken string
	var expiry *time.Time
	err := c.run(func() error {
		form := url.Values{}
		form.Set("grant_type", "fb_exchange_token")
		form.Set("client_id", appID)
		form.Set("client_secret", appSecret)
		form.Set("fb_exchange_token", token)
		var out struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int64  `json:"expires_in"`
		}
		if err := c.doJSON(ctx, http.MethodGet, baseURL, apiVersion, "oauth/access_token", "", form, nil, &out); err != nil {
			return err
		}
		if out.AccessToken == "" {
			return fmt.Errorf("graph api returned no access token")
		}
		newToken = out.AccessToken
		if out.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
			expiry = &t
		}
		return nil
	})
	return newToken, expiry, err
}
