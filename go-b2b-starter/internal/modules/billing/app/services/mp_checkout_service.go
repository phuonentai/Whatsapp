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

	creator, ok := s.mpProvider.(interface {
		CreateCheckoutSession(ctx context.Context, planID, externalReference string) (*domain.CheckoutSessionResponse, error)
	})
	if !ok {
		return nil, fmt.Errorf("MercadoPago provider does not support checkout creation")
	}

	checkoutSession, err := creator.CreateCheckoutSession(ctx, planID, fmt.Sprintf("%v", externalCustomerID))
	if err != nil {
		return nil, fmt.Errorf("failed to create MercadoPago checkout: %w", err)
	}

	s.logger.Info("MercadoPago checkout created", map[string]any{
		"checkout_id": checkoutSession.ID,
		"org_id":      orgID,
		"has_init":    checkoutSession.InitPoint != "",
	})

	return &domain.BillingStatus{
		OrganizationID:        orgID,
		ExternalID:            fmt.Sprintf("%v", externalCustomerID),
		HasActiveSubscription: false,
		CanProcessInvoices:    false,
		InvoiceCount:          0,
		Reason:                "Checkout initiated",
		CheckedAt:             time.Now(),
		CheckoutURL:           checkoutSession.InitPoint,
	}, nil
}

func (s *billingService) CancelMPSubscription(ctx context.Context, subscriptionID string) (*domain.BillingStatus, error) {
	externalCustomerID := ctx.Value("stytch_org_id")
	if externalCustomerID == nil {
		return nil, fmt.Errorf("organization context required for subscription cancellation")
	}

	organizationID, err := s.orgAdapter.GetOrganizationIDByStytchOrgID(ctx, fmt.Sprintf("%v", externalCustomerID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization: %w", err)
	}

	canceller, ok := s.mpProvider.(interface {
		CancelSubscription(ctx context.Context, subscriptionID string) error
	})
	if !ok {
		return nil, fmt.Errorf("MercadoPago provider does not support subscription cancellation")
	}

	if err := canceller.CancelSubscription(ctx, subscriptionID); err != nil {
		return nil, fmt.Errorf("failed to cancel MercadoPago subscription: %w", err)
	}

	now := time.Now()
	subscription := &domain.Subscription{
		OrganizationID:     organizationID,
		ExternalCustomerID: fmt.Sprintf("%v", externalCustomerID),
		SubscriptionID:     subscriptionID,
		SubscriptionStatus: "canceled",
		CancelAtPeriodEnd:  false,
		CanceledAt:         &now,
	}

	if _, err := s.repo.UpsertSubscription(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to mark subscription canceled: %w", err)
	}

	s.logger.Info("MercadoPago subscription cancelled", map[string]any{
		"subscription_id": subscriptionID,
		"organization_id": organizationID,
	})

	return &domain.BillingStatus{
		OrganizationID:        organizationID,
		ExternalID:            fmt.Sprintf("%v", externalCustomerID),
		HasActiveSubscription: false,
		CanProcessInvoices:    false,
		InvoiceCount:          0,
		Reason:                "Subscription cancelled",
		CheckedAt:             time.Now(),
	}, nil
}

func (s *billingService) VerifyMPPayment(ctx context.Context, paymentID string) (*domain.BillingStatus, error) {
	s.logger.Info("Verifying MercadoPago payment", map[string]any{
		"payment_id": paymentID,
	})

	checkoutSession, err := s.mpProvider.GetCheckoutSessionWithPolling(ctx, paymentID)
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

	if err := s.resolver.SetBillingProvider(ctx, organizationID, "mercadopago"); err != nil {
		return nil, fmt.Errorf("failed to set billing provider: %w", err)
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
