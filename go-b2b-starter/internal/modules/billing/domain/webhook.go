package domain

// Webhook signature header names for the Svix scheme used by Polar.sh.
const (
	PolarWebhookIDHeader        = "webhook-id"
	PolarWebhookTimestampHeader = "webhook-timestamp"
	PolarWebhookSignatureHeader = "webhook-signature"
)

// MercadoPagoWebhookSignatureHeader is the signature header name used by
// MercadoPago IPN notifications.
const MercadoPagoWebhookSignatureHeader = "x-signature"

// WebhookVerifier verifies inbound billing-provider webhook signatures.
//
// Implementations live in the infra layer (billing/infra/polar,
// billing/infra/mercadopago) so the HTTP handler depends only on this domain
// interface, never on provider SDKs or transport packages.
type WebhookVerifier interface {
	// VerifyPolar verifies a Svix-scheme signature (used by Polar.sh).
	VerifyPolar(payload []byte, msgID, msgTimestamp, signature, secret string) error

	// VerifyMercadoPago verifies a MercadoPago IPN signature.
	VerifyMercadoPago(payload []byte, signature, secret string) error
}
