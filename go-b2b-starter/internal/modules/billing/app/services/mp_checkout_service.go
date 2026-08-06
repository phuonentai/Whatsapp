package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

func (s *billingService) CreateMPCheckout(ctx context.Context, planID string) (*domain.BillingStatus, error) {
	externalCustomerID := ctx.Value("stytch_org_id")
	if externalCustomerID == nil {
		return nil, fmt.Errorf("organization context required for checkout creation")
	}

	orgID, err := s.orgAdapter.GetOrganizationIDByStytchOrgID(ctx, fmt.Sprintf("%v", externalCustomerID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization: %w", err)
	}

	s.logger.Info("Creating MercadoPago checkout", map[string]any{
		"plan_id": planID,
		"org_id":  orgID,
	})

	return &domain.BillingStatus{
		OrganizationID:        orgID,
		ExternalID:            fmt.Sprintf("%v", externalCustomerID),
		HasActiveSubscription: false,
		CanProcessInvoices:    false,
		InvoiceCount:          0,
		Reason:                "Checkout initiated",
		CheckedAt:             time.Now(),
	}, nil
}

func (s *billingService) VerifyMPPayment(ctx context.Context, paymentID string) (*domain.BillingStatus, error) {
	s.logger.Info("Verifying MercadoPago payment", map[string]any{
		"payment_id": paymentID,
	})

	checkoutSession, err := s.billingProvider.GetCheckoutSessionWithPolling(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment: %w", err)
	}

	if checkoutSession.Status != "succeeded" {
		return &domain.BillingStatus{
			HasActiveSubscription: false,
			CanProcessInvoices:    false,
			Reason:                fmt.Sprintf("payment status is %s", checkoutSession.Status),
			CheckedAt:             time.Now(),
		}, nil
	}

	externalCustomerID := checkoutSession.CustomerID
	if externalCustomerID == "" {
		return nil, fmt.Errorf("payment has no external reference")
	}

	organizationID, err := s.orgAdapter.GetOrganizationIDByStytchOrgID(ctx, externalCustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to map customer ID to organization: %w", err)
	}

	subscription, err := s.billingProvider.GetSubscription(ctx, externalCustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription from MercadoPago: %w", err)
	}

	subscription.OrganizationID = organizationID
	_, err = s.repo.UpsertSubscription(ctx, subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	invoiceCountMax := int32(0)
	if metadata, ok := subscription.Metadata["invoice_count_max"].(int32); ok {
		invoiceCountMax = metadata
	} else if val, ok := subscription.Metadata["invoice_count_max"].(string); ok {
		if count, err := strconv.ParseInt(val, 10, 32); err == nil {
			invoiceCountMax = int32(count)
		}
	}

	quota := &domain.QuotaTracking{
		OrganizationID: organizationID,
		InvoiceCount:   invoiceCountMax,
		PeriodStart:    subscription.CurrentPeriodStart,
		PeriodEnd:      subscription.CurrentPeriodEnd,
		LastSyncedAt:   &time.Time{},
	}
	*quota.LastSyncedAt = time.Now()

	_, err = s.repo.UpsertQuota(ctx, quota)
	if err != nil {
		return nil, fmt.Errorf("failed to save quota: %w", err)
	}

	s.logger.Info("MercadoPago payment verified", map[string]any{
		"payment_id":      paymentID,
		"organization_id": organizationID,
		"subscription_id": subscription.SubscriptionID,
		"invoice_count":   invoiceCountMax,
	})

	return &domain.BillingStatus{
		OrganizationID:        organizationID,
		ExternalID:            externalCustomerID,
		HasActiveSubscription: subscription.SubscriptionStatus == "active" || subscription.SubscriptionStatus == "trialing" || subscription.SubscriptionStatus == "authorized",
		CanProcessInvoices:    (subscription.SubscriptionStatus == "active" || subscription.SubscriptionStatus == "authorized") && invoiceCountMax > 0,
		InvoiceCount:          invoiceCountMax,
		Reason:                "Payment verified successfully",
		CheckedAt:             time.Now(),
	}, nil
}
