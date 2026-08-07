package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type TextMessageRequest struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

type TextMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type APIErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		ErrorData *struct {
			Details string `json:"details"`
		} `json:"error_data,omitempty"`
	} `json:"error"`
}

func (c *Client) SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/messages", graphAPIURL, apiVersion, phoneNumberID)

	payload := TextMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
	}
	payload.Text.Body = body

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("whatsapp api error (code %d): %s", apiErr.Error.Code, apiErr.Error.Message)
		}
		return "", fmt.Errorf("whatsapp api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result TextMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("whatsapp api returned no message ID")
	}

	return result.Messages[0].ID, nil
}

type CircuitBreaker struct {
	mu             sync.Mutex
	failures       int
	lastFailureAt  time.Time
	state          CircuitState
	threshold      int
	cooldown       time.Duration
	halfOpenProbes int
	halfOpenMax    int
}

type CircuitState int

const (
	CircuitClosed   CircuitState = 0
	CircuitOpen     CircuitState = 1
	CircuitHalfOpen CircuitState = 2
)

func NewCircuitBreaker(threshold int, cooldown time.Duration, halfOpenProbes int) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:      threshold,
		cooldown:       cooldown,
		halfOpenProbes: halfOpenProbes,
		halfOpenMax:    halfOpenProbes,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureAt) > cb.cooldown {
			cb.state = CircuitHalfOpen
			cb.halfOpenProbes = cb.halfOpenMax
		} else {
			return false
		}
		fallthrough
	case CircuitHalfOpen:
		if cb.halfOpenProbes <= 0 {
			cb.state = CircuitOpen
			cb.lastFailureAt = time.Now()
			return false
		}
		cb.halfOpenProbes--
		if cb.halfOpenProbes <= 0 {
			cb.state = CircuitOpen
			cb.lastFailureAt = time.Now()
		}
		return true
	default:
		return true
	}
}

func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}

func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureAt = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

type ClientWithBreaker struct {
	client  *Client
	breaker *CircuitBreaker
}

func NewClientWithBreaker(threshold int, cooldown time.Duration, halfOpenProbes int) *ClientWithBreaker {
	return &ClientWithBreaker{
		client:  NewClient(),
		breaker: NewCircuitBreaker(threshold, cooldown, halfOpenProbes),
	}
}

func (c *ClientWithBreaker) SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error) {
	if !c.breaker.Allow() {
		return "", fmt.Errorf("circuit breaker open: request blocked")
	}

	msgID, err := c.client.SendTextMessage(ctx, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body)
	if err != nil {
		c.breaker.Failure()
		return "", err
	}

	c.breaker.Success()
	return msgID, nil
}
