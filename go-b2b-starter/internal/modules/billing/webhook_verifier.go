package billing

import (
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/mercadopago"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/polar"
)

// webhookVerifier routes each provider's signature verification to the
// matching infra adapter. Handlers depend on the domain.WebhookVerifier
// interface and never import provider infra packages directly.
type webhookVerifier struct {
	polarVerifier       domain.WebhookVerifier
	mercadopagoVerifier domain.WebhookVerifier
}

func (v *webhookVerifier) VerifyPolar(payload []byte, msgID, msgTimestamp, signature, secret string) error {
	return v.polarVerifier.VerifyPolar(payload, msgID, msgTimestamp, signature, secret)
}

func (v *webhookVerifier) VerifyMercadoPago(payload []byte, signature, secret string) error {
	return v.mercadopagoVerifier.VerifyMercadoPago(payload, signature, secret)
}

var _ domain.WebhookVerifier = (*webhookVerifier)(nil)

// newWebhookVerifier builds a composite verifier over the provider adapters.
func newWebhookVerifier() domain.WebhookVerifier {
	return &webhookVerifier{
		polarVerifier:       polar.NewVerifier(),
		mercadopagoVerifier: mercadopago.NewVerifier(),
	}
}
