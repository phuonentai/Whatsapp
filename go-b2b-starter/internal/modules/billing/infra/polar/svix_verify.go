package polar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// Webhook headers used by the Svix webhook signature scheme
const (
	WebhookIDHeader        = domain.PolarWebhookIDHeader
	WebhookTimestampHeader = domain.PolarWebhookTimestampHeader
	WebhookSignatureHeader = domain.PolarWebhookSignatureHeader
)

// Verifier adapts the Svix signature verification to the billing domain
// WebhookVerifier interface.
type Verifier struct{}

// NewVerifier creates a Polar webhook verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// VerifyPolar verifies a Svix-scheme webhook signature.
func (v *Verifier) VerifyPolar(payload []byte, msgID, msgTimestamp, signature, secret string) error {
	return VerifyWebhookSignature(payload, msgID, msgTimestamp, signature, secret)
}

// VerifyMercadoPago is not implemented by the Polar adapter.
func (v *Verifier) VerifyMercadoPago(payload []byte, signature, secret string) error {
	return fmt.Errorf("polar verifier does not support mercadopago webhooks")
}

// ensure interface compliance
var _ domain.WebhookVerifier = (*Verifier)(nil)

// webhookTolerance is the maximum age (in seconds) accepted for a webhook
// timestamp before it is considered expired. Matches Svix's default.
const webhookTolerance = 5 * 60

// VerifyWebhookSignature verifies a Svix-style webhook signature
// (the scheme used by Polar.sh): the signature header contains
// space-separated signatures prefixed with the version (e.g. "v1,<base64>"),
// each computed as base64(HMAC-SHA256(secret, msg_id + "." + msg_timestamp +
// "." + payload)) with constant-time comparison and a timestamp tolerance
// window.
func VerifyWebhookSignature(payload []byte, msgID, msgTimestamp, signatureHeader, secret string) error {
	if signatureHeader == "" {
		return fmt.Errorf("missing %s header", WebhookSignatureHeader)
	}
	if msgID == "" {
		return fmt.Errorf("missing %s header", WebhookIDHeader)
	}
	if msgTimestamp == "" {
		return fmt.Errorf("missing %s header", WebhookTimestampHeader)
	}
	if secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}

	timestamp, err := strconv.ParseInt(msgTimestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook timestamp: %w", err)
	}

	now := time.Now().Unix()
	if diff := now - timestamp; diff < -webhookTolerance || diff > webhookTolerance {
		return fmt.Errorf("webhook timestamp %d is outside the tolerance window", timestamp)
	}

	secretKey := []byte(secret)

	signingInput := fmt.Sprintf("%s.%s.%s", msgID, msgTimestamp, payload)

	expectedMAC := hmac.New(sha256.New, secretKey)
	expectedMAC.Write([]byte(signingInput))
	expected := base64.StdEncoding.EncodeToString(expectedMAC.Sum(nil))

	for _, candidate := range strings.Fields(signatureHeader) {
		parts := strings.SplitN(candidate, ",", 2)
		if len(parts) != 2 {
			continue
		}
		version, sig := parts[0], parts[1]
		if version != "v1" {
			continue
		}
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}

	return fmt.Errorf("webhook signature mismatch")
}
