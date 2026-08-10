package domain

import "context"

// PaymentRail creates and verifies one-shot payment preferences at the
// payment provider (MercadoPago). The provider is a rail; the local DB is
// the system of record.
type PaymentRail interface {
	// CreatePreference creates a one-shot checkout preference priced at
	// unitPriceCOP (base amount + platform commission) and returns the
	// hosted checkout URL (init_point) plus the preference id.
	CreatePreference(ctx context.Context, orgID, dealID int32, unitPriceCOP int64, currency string) (initPoint, preferenceID string, err error)
	// VerifyPayment returns the provider-side payment detail (status
	// normalized to the local vocabulary, plus correlation keys).
	VerifyPayment(ctx context.Context, paymentID string) (*PaymentDetail, error)
}

// PaymentDetail is the provider-side projection used to correlate and
// verify a payment event.
type PaymentDetail struct {
	Status        PaymentStatus
	PreferenceID  string
	ExternalRef   string
	TransactionAmount int64
}
