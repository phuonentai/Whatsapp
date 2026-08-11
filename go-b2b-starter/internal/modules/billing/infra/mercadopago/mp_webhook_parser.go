package mercadopago

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// Verifier adapts MercadoPago signature verification to the billing domain
// WebhookVerifier interface.
type Verifier struct{}

// NewVerifier creates a MercadoPago webhook verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// VerifyPolar is not implemented by the MercadoPago adapter.
func (v *Verifier) VerifyPolar(payload []byte, msgID, msgTimestamp, signature, secret string) error {
	return fmt.Errorf("mercadopago verifier does not support polar webhooks")
}

// VerifyMercadoPago verifies a MercadoPago IPN webhook signature.
func (v *Verifier) VerifyMercadoPago(payload []byte, signature, secret string) error {
	return VerifyWebhookSignature(payload, signature, secret)
}

// ensure interface compliance
var _ domain.WebhookVerifier = (*Verifier)(nil)

// VerifyWebhookSignature verifies a MercadoPago IPN webhook signature.
// It supports both the current MercadoPago format
// ("ts=<timestamp>,v1=<hex hmac>" over "id:<id>;request-id:<request_id>;ts:<ts>;<body>")
// and a plain raw-hex HMAC-SHA256 of the body (legacy/test clients).
func VerifyWebhookSignature(payload []byte, signatureHeader string, secret string) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}

	params := parseSignatureParams(signatureHeader)
	if ts, ok := params["ts"]; ok {
		if v1, ok := params["v1"]; ok {
			expected, err := hex.DecodeString(v1)
			if err != nil {
				return fmt.Errorf("invalid signature v1 format: %w", err)
			}
			signingInput := signatureSigningInput(payload, ts)
			if hmacMatches(expected, []byte(secret), []byte(signingInput)) {
				return nil
			}
			return fmt.Errorf("webhook signature mismatch")
		}
	}

	expectedMAC, err := hex.DecodeString(signatureHeader)
	if err != nil {
		return fmt.Errorf("invalid signature header format: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	if !hmac.Equal(expectedMAC, expected) {
		return fmt.Errorf("webhook signature mismatch")
	}

	return nil
}

func parseSignatureParams(header string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return params
}

// signatureSigningInput builds the MercadoPago signing input:
// "id:<notification_id>;request-id:<request_id>;ts:<timestamp>;<raw body>".
// id and request_id are read from the notification payload.
func signatureSigningInput(payload []byte, ts string) string {
	var notification struct {
		ID        string `json:"id"`
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(payload, &notification)
	return fmt.Sprintf("id:%s;request-id:%s;ts:%s;%s", notification.ID, notification.RequestID, ts, payload)
}

func hmacMatches(expectedHex, secret, input []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(input)
	return hmac.Equal(expectedHex, mac.Sum(nil))
}

type MPWebhookPayload struct {
	Action  string          `json:"action"`
	APIVer  string          `json:"api_version"`
	Data    json.RawMessage `json:"data"`
	DateCreated string      `json:"date_created"`
	ID      int64           `json:"id"`
	LiveMode bool           `json:"live_mode"`
	Type    string          `json:"type"`
	UserID  int64           `json:"user_id"`
}

type MPSubscriptionEventData struct {
	ID              string `json:"id"`
	ExternalRef     string `json:"external_reference"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
	PayerEmail      string `json:"payer_email"`
	PaymentMethodID string `json:"payment_method_id"`
	NextPaymentDate string `json:"next_payment_date"`
	DateCreated     string `json:"date_created"`
	LastModified    string `json:"last_modified"`
	AutoRecurring   *struct {
		Frequency      int    `json:"frequency"`
		FrequencyType  string `json:"frequency_type"`
		StartDate      string `json:"start_date"`
		EndDate        string `json:"end_date"`
		TransactionAmt string `json:"transaction_amount"`
		CurrencyID     string `json:"currency_id"`
	} `json:"auto_recurring"`
}

type MPPaymentEventData struct {
	ID          int64  `json:"id"`
	ExternalRef string `json:"external_reference"`
	Status      string `json:"status"`
	StatusDetail string `json:"status_detail"`
	PayerEmail  string `json:"payer"`
	Amount      float64 `json:"transaction_amount"`
	Description string `json:"description"`
	DateCreated string `json:"date_created"`
}

func ParseWebhookPayload(payload json.RawMessage) (*MPWebhookPayload, error) {
	var wh MPWebhookPayload
	if err := json.Unmarshal(payload, &wh); err != nil {
		return nil, fmt.Errorf("failed to parse MercadoPago webhook payload: %w", err)
	}
	return &wh, nil
}

func ParseSubscriptionEventData(raw json.RawMessage) (*domain.SubscriptionEventData, error) {
	var event MPSubscriptionEventData
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, fmt.Errorf("failed to parse MercadoPago subscription event: %w", err)
	}

	if event.ExternalRef == "" {
		return nil, fmt.Errorf("MercadoPago webhook missing external_reference")
	}

	data := &domain.SubscriptionEventData{
		SubscriptionID:     event.ID,
		ExternalCustomerID: event.ExternalRef,
		Status:             event.Status,
		ProductName:        event.Reason,
	}

	if event.AutoRecurring != nil {
		if start, err := time.Parse(time.RFC3339, event.AutoRecurring.StartDate); err == nil {
			data.CurrentPeriodStart = start
		}
		if end, err := time.Parse(time.RFC3339, event.AutoRecurring.EndDate); err == nil {
			data.CurrentPeriodEnd = end
		}
	}
	if data.CurrentPeriodStart.IsZero() {
		if t, err := time.Parse(time.RFC3339, event.DateCreated); err == nil {
			data.CurrentPeriodStart = t
		}
	}
	if data.CurrentPeriodEnd.IsZero() && event.NextPaymentDate != "" {
		if t, err := time.Parse(time.RFC3339, event.NextPaymentDate); err == nil {
			data.CurrentPeriodEnd = t
		}
	}

	return data, nil
}

func ParsePaymentEventData(raw json.RawMessage) (*domain.CheckoutSessionResponse, error) {
	var event MPPaymentEventData
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, fmt.Errorf("failed to parse MercadoPago payment event: %w", err)
	}

	checkoutStatus := event.Status
	switch event.Status {
	case "approved":
		checkoutStatus = "succeeded"
	case "rejected", "cancelled", "refunded", "charged_back":
		checkoutStatus = "failed"
	default:
		checkoutStatus = "pending"
	}

	createdAt, _ := time.Parse(time.RFC3339, event.DateCreated)

	return &domain.CheckoutSessionResponse{
		ID:         fmt.Sprintf("%d", event.ID),
		Status:     checkoutStatus,
		CustomerID: event.ExternalRef,
		Amount:     int64(event.Amount),
		CreatedAt:  createdAt,
	}, nil
}

func MapMPStatus(status string) string {
	switch strings.ToLower(status) {
	case "authorized", "active", "approved":
		return "active"
	case "pending":
		return "pending"
	case "paused":
		return "past_due"
	case "cancelled", "canceled":
		return "canceled"
	default:
		return status
	}
}
