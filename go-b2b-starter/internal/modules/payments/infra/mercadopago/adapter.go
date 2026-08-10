// Package mercadopago implements the payments PaymentRail over the platform
// MercadoPago client (Checkout Preferences for one-shot payment links).
package mercadopago

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	mp "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
)

var _ domain.PaymentRail = (*paymentRail)(nil)

type paymentRail struct {
	client  *mp.Client
	logger  loggerDomain.Logger
	backURL string
}

func NewPaymentRail(client *mp.Client, log loggerDomain.Logger, backURL string) domain.PaymentRail {
	return &paymentRail{client: client, logger: log, backURL: backURL}
}

// CreatePreference creates a one-shot checkout preference. The external
// reference is namespaced ("deal:<id>") so it never collides with the
// subscription preapproval reference space.
func (r *paymentRail) CreatePreference(ctx context.Context, orgID, dealID int32, unitPriceCOP int64, currency string) (string, string, error) {
	body := map[string]any{
		"items": []map[string]any{
			{
				"title":        "Pago de negocio",
				"quantity":     1,
				"unit_price":   unitPriceCOP,
				"currency_id":  currency,
			},
		},
		"external_reference": fmt.Sprintf("deal:%d", dealID),
		"back_urls": map[string]any{
			"success": r.backURL,
			"failure": r.backURL,
			"pending": r.backURL,
		},
		"auto_return": "approved",
		"purpose":     "wallet_purchase",
	}

	resp, err := r.client.Post(ctx, "/checkout/preferences", body)
	if err != nil {
		return "", "", fmt.Errorf("failed to call MercadoPago preferences API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("MercadoPago preferences API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID        string `json:"id"`
		InitPoint string `json:"init_point"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode MercadoPago preferences response: %w", err)
	}

	r.logger.Info("MercadoPago payment preference created", loggerDomain.Fields{
		"preference_id": result.ID,
		"external_ref":  fmt.Sprintf("deal:%d", dealID),
		"unit_price":    unitPriceCOP,
	})

	return result.InitPoint, result.ID, nil
}

// VerifyPayment retrieves a payment and normalizes its status to the local
// vocabulary, plus the correlation keys (preference id, external reference).
func (r *paymentRail) VerifyPayment(ctx context.Context, paymentID string) (*domain.PaymentDetail, error) {
	resp, err := r.client.Get(ctx, fmt.Sprintf("/v1/payments/%s", paymentID))
	if err != nil {
		return nil, fmt.Errorf("failed to call MercadoPago payments API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MercadoPago payments API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID              int64   `json:"id"`
		Status          string  `json:"status"`
		ExternalRef     string  `json:"external_reference"`
		PreferenceID    string  `json:"preference_id"`
		TransactionAmt  float64 `json:"transaction_amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode MercadoPago payment response: %w", err)
	}

	detail := &domain.PaymentDetail{
		Status:            mapStatus(result.Status),
		PreferenceID:      result.PreferenceID,
		ExternalRef:       result.ExternalRef,
		TransactionAmount: int64(result.TransactionAmt),
	}

	r.logger.Info("MercadoPago payment verified", loggerDomain.Fields{
		"payment_id":   paymentID,
		"status":       result.Status,
		"preference_id": result.PreferenceID,
	})

	return detail, nil
}

func mapStatus(status string) domain.PaymentStatus {
	switch status {
	case "approved":
		return domain.PaymentStatusPaid
	case "rejected", "cancelled", "refunded", "charged_back":
		return domain.PaymentStatusFailed
	default:
		return domain.PaymentStatusPending
	}
}
