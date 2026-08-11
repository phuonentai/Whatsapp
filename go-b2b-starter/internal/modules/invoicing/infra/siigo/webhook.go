package siigo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// WebhookSignatureHeader is the header carrying the HMAC-SHA256 hex digest of
// the raw webhook body. ASSUMPTION (unverified, no network): Siigo signs
// webhooks as hex(HMAC-SHA256(body, secret)). Verify against live sandbox at
// deployment (tasks 11.x) and adjust here if the scheme differs.
const WebhookSignatureHeader = domain.WebhookSignatureHeader

// Verifier adapts Siigo webhook signature verification to the invoicing domain
// WebhookVerifier interface.
type Verifier struct{}

// NewVerifier creates a Siigo webhook verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// Verify verifies a Siigo webhook signature.
func (v *Verifier) Verify(payload []byte, signature, secret string) error {
	return VerifyWebhookSignature(payload, signature, secret)
}

// ensure interface compliance
var _ domain.WebhookVerifier = (*Verifier)(nil)

// VerifyWebhookSignature verifies the HMAC-SHA256 signature of a webhook body
// using constant-time comparison. Empty secret fails closed.
func VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}
	if signature == "" {
		return fmt.Errorf("missing signature header %s", WebhookSignatureHeader)
	}

	provided, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature header format: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	if !hmac.Equal(provided, expected) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}
