package mercadopago

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	mp "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
)

var _ domain.BillingProvider = (*mpAdapter)(nil)

type mpAdapter struct {
	client *mp.Client
	logger logger.Logger
	cfg    *mp.Config
}

func NewMPAdapter(client *mp.Client, log logger.Logger, cfg *mp.Config) domain.BillingProvider {
	return &mpAdapter{
		client: client,
		logger: log,
		cfg:    cfg,
	}
}

// CreateCheckoutSession creates a MercadoPago preapproval for the given plan
// and returns the hosted Checkout Pro redirect URL (init_point).
func (a *mpAdapter) CreateCheckoutSession(ctx context.Context, planID, externalReference string) (*domain.CheckoutSessionResponse, error) {
	body := map[string]any{
		"preapproval_plan_id": planID,
		"external_reference":  externalReference,
		"reason":              "Subscription checkout",
		"back_url":            a.cfg.BackURL,
		"auto_recurring":      true,
	}

	// Attach the plan's configured invoice quota as preapproval metadata so
	// verify/sync/webhook paths can seed local quota tracking (MP plans carry
	// no product metadata of their own).
	if invoiceCount, ok := a.invoiceCountForPlan(planID); ok {
		body["metadata"] = map[string]any{"invoice_count_max": invoiceCount}
	}

	resp, err := a.client.Post(ctx, "/preapproval", body)
	if err != nil {
		return nil, fmt.Errorf("failed to call MercadoPago preapproval API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MercadoPago preapproval API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		InitPoint string `json:"init_point"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode MercadoPago preapproval response: %w", err)
	}

	a.logger.Info("MercadoPago preapproval created", loggerdomain.Fields{
		"preapproval_id": result.ID,
		"external_ref":   externalReference,
		"status":         result.Status,
		"init_point":     result.InitPoint,
	})

	return &domain.CheckoutSessionResponse{
		ID:         result.ID,
		Status:     result.Status,
		CustomerID: externalReference,
		InitPoint:  result.InitPoint,
	}, nil
}

func (a *mpAdapter) GetSubscription(ctx context.Context, externalCustomerID string) (*domain.Subscription, error) {
	endpoint := fmt.Sprintf("/preapproval/search?external_reference=%s", externalCustomerID)

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to call MercadoPago API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MercadoPago API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			ID             string `json:"id"`
			Reason         string `json:"reason"`
			ExternalRef    string `json:"external_reference"`
			Status         string `json:"status"`
			NextPayment    string `json:"next_payment_date"`
			DateCreated    string `json:"date_created"`
			LastModified   string `json:"last_modified"`
			AutoRecurring  *struct {
				Frequency       int    `json:"frequency"`
				FrequencyType   string `json:"frequency_type"`
				StartDate       string `json:"start_date"`
				EndDate         string `json:"end_date"`
				TransactionAmt  string `json:"transaction_amount"`
				CurrencyID      string `json:"currency_id"`
			} `json:"auto_recurring"`
			PayerID    int            `json:"payer_id"`
			PaymentID  string         `json:"payment_method_id"`
			Metadata   map[string]any `json:"metadata"`
		} `json:"results"`
		Paging struct {
			Total int `json:"total"`
		} `json:"paging"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode MercadoPago preapproval search response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, domain.ErrSubscriptionNotFound
	}

	mpSub := result.Results[0]

	var periodStart, periodEnd time.Time
	if mpSub.AutoRecurring != nil {
		periodStart, _ = time.Parse(time.RFC3339, mpSub.AutoRecurring.StartDate)
		periodEnd, _ = time.Parse(time.RFC3339, mpSub.AutoRecurring.EndDate)
	}
	if periodStart.IsZero() {
		periodStart, _ = time.Parse(time.RFC3339, mpSub.DateCreated)
	}
	if periodEnd.IsZero() && mpSub.NextPayment != "" {
		periodEnd, _ = time.Parse(time.RFC3339, mpSub.NextPayment)
	}

	a.logger.Info("MercadoPago preapproval sync completed", loggerdomain.Fields{
		"external_ref":    externalCustomerID,
		"preapproval_id":  mpSub.ID,
		"status":          mpSub.Status,
		"reason":          mpSub.Reason,
	})

	subscription := &domain.Subscription{
		ExternalCustomerID: externalCustomerID,
		SubscriptionID:     mpSub.ID,
		// Map the raw MP status (authorized/paused/cancelled) to the canonical
		// domain status so the SQL gate and GetBillingStatus recognize it.
		SubscriptionStatus: MapMPStatus(mpSub.Status),
		ProductID:          "",
		ProductName:        mpSub.Reason,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Metadata: map[string]any{
			"provider":          "mercadopago",
			"payer_id":          mpSub.PayerID,
			"payment_method_id": mpSub.PaymentID,
		},
	}

	// Read back the quota attached at checkout time so verify/sync paths seed
	// a nonzero local quota. Tolerates numeric and string metadata values.
	if raw, ok := mpSub.Metadata["invoice_count_max"]; ok {
		if count, ok := parseInvoiceCountMax(raw); ok {
			subscription.Metadata["invoice_count_max"] = count
		}
	}

	return subscription, nil
}

// invoiceCountForPlan resolves the configured invoice quota for a preapproval
// plan id. It reports ok=false for unmapped plans so no quota metadata is
// attached to preapprovals for unknown plans.
func (a *mpAdapter) invoiceCountForPlan(planID string) (int32, bool) {
	if planID == "" {
		return 0, false
	}
	if a.cfg.CheckoutPlanID != "" && planID == a.cfg.CheckoutPlanID {
		return a.cfg.CheckoutInvoiceCount, true
	}
	if a.cfg.BusinessPlanID != "" && planID == a.cfg.BusinessPlanID {
		return a.cfg.BusinessInvoiceCount, true
	}
	return 0, false
}

// parseInvoiceCountMax normalizes a preapproval metadata value into an int32
// invoice quota, tolerating JSON numbers and string representations.
func parseInvoiceCountMax(raw any) (int32, bool) {
	switch v := raw.(type) {
	case float64:
		return int32(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(n), true
	case string:
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return 0, false
		}
		return int32(n), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	default:
		return 0, false
	}
}

func (a *mpAdapter) GetCheckoutSession(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	endpoint := fmt.Sprintf("/v1/payments/%s", sessionID)

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to call MercadoPago payments API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("%w: %s", domain.ErrCheckoutSessionNotFound, sessionID)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MercadoPago payments API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID              int64  `json:"id"`
		Status          string `json:"status"`
		StatusDetail    string `json:"status_detail"`
		ExternalRef     string `json:"external_reference"`
		TransactionAmt  float64 `json:"transaction_amount"`
		DateCreated     string `json:"date_created"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode MercadoPago payment response: %w", err)
	}

	a.logger.Info("MercadoPago payment retrieved", loggerdomain.Fields{
		"payment_id":   result.ID,
		"status":       result.Status,
		"external_ref": result.ExternalRef,
	})

	checkoutStatus := result.Status
	switch result.Status {
	case "approved":
		checkoutStatus = "succeeded"
	case "rejected", "cancelled", "refunded", "charged_back":
		checkoutStatus = "failed"
	default:
		checkoutStatus = "pending"
	}

	createdAt, _ := time.Parse(time.RFC3339, result.DateCreated)

	checkoutSession := &domain.CheckoutSessionResponse{
		ID:         strconv.FormatInt(result.ID, 10),
		Status:     checkoutStatus,
		CustomerID: result.ExternalRef,
		Amount:     int64(result.TransactionAmt),
		CreatedAt:  createdAt,
	}

	return checkoutSession, nil
}

func (a *mpAdapter) GetCheckoutSessionWithPolling(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	const (
		pollInterval = 2 * time.Second
		maxDuration  = 10 * time.Second
	)

	deadline := time.Now().Add(maxDuration)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	session, err := a.GetCheckoutSession(ctx, sessionID)
	if err == nil && session.Status == "succeeded" {
		return session, nil
	}

	if err == nil {
		a.logger.Debug("MercadoPago payment polling started", loggerdomain.Fields{
			"payment_id":   sessionID,
			"status":       session.Status,
			"max_duration": maxDuration.String(),
		})
	} else if !isRetryableError(err) {
		return nil, err
	} else {
		a.logger.Debug("MercadoPago payment initial attempt failed, will retry", loggerdomain.Fields{
			"payment_id": sessionID,
			"error":      err.Error(),
		})
	}

	attemptCount := 1
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			attemptCount++
			session, err := a.GetCheckoutSession(ctx, sessionID)

			if err == nil {
				a.logger.Debug("MercadoPago payment polling attempt", loggerdomain.Fields{
					"payment_id": sessionID,
					"attempt":    attemptCount,
					"status":     session.Status,
				})
				if session.Status == "succeeded" {
					a.logger.Info("MercadoPago payment polling succeeded", loggerdomain.Fields{
						"payment_id": sessionID,
						"attempts":   attemptCount,
					})
					return session, nil
				}
				continue
			}

			if !isRetryableError(err) {
				a.logger.Warn("MercadoPago payment polling non-retryable error", loggerdomain.Fields{
					"payment_id": sessionID,
					"attempt":    attemptCount,
					"error":      err.Error(),
				})
				return nil, err
			}

			a.logger.Debug("MercadoPago payment polling attempt failed, retrying", loggerdomain.Fields{
				"payment_id": sessionID,
				"attempt":    attemptCount,
				"error":      err.Error(),
			})
		}
	}

	lastStatus := "unknown"
	if session != nil {
		lastStatus = session.Status
	}
	a.logger.Warn("MercadoPago payment polling timeout", loggerdomain.Fields{
		"payment_id":  sessionID,
		"attempts":    attemptCount,
		"last_status": lastStatus,
	})
	return nil, fmt.Errorf("payment verification timed out after 10 seconds (last status: %s)", lastStatus)
}

// CancelSubscription cancels a MercadoPago preapproval (subscription).
func (a *mpAdapter) CancelSubscription(ctx context.Context, subscriptionID string) error {
	body := map[string]any{
		"status": "cancelled",
	}

	resp, err := a.client.Put(ctx, fmt.Sprintf("/preapproval/%s", subscriptionID), body)
	if err != nil {
		return fmt.Errorf("failed to call MercadoPago preapproval API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MercadoPago preapproval API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	a.logger.Info("MercadoPago preapproval cancelled", loggerdomain.Fields{
		"preapproval_id": subscriptionID,
	})
	return nil
}

func (a *mpAdapter) IngestMeterEvent(ctx context.Context, externalCustomerID string, meterSlug string, amount int32) error {
	a.logger.Debug("MercadoPago IngestMeterEvent no-op (meter events not supported)", loggerdomain.Fields{
		"external_customer_id": externalCustomerID,
		"meter_slug":           meterSlug,
		"amount":               amount,
	})
	return nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	if strings.Contains(errStr, "checkout session not found") || strings.Contains(errStr, "404") {
		return false
	}

	if strings.Contains(errStr, "returned status 400") ||
		strings.Contains(errStr, "returned status 401") ||
		strings.Contains(errStr, "returned status 403") {
		return false
	}

	return true
}
