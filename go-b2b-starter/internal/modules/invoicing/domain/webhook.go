package domain

// WebhookSignatureHeader is the header carrying the HMAC-SHA256 hex digest of
// the raw Siigo webhook body.
const WebhookSignatureHeader = "x-siigo-signature"

// WebhookVerifier verifies inbound invoicing-provider webhook signatures.
//
// Implementations live in the infra layer (invoicing/infra/siigo) so the HTTP
// handler depends only on this domain interface, never on provider SDKs or
// transport packages.
type WebhookVerifier interface {
	// Verify verifies a provider webhook signature over the raw payload.
	Verify(payload []byte, signature, secret string) error
}
