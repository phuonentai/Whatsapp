package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// RefreshSubscriptionStatus forces a sync with Polar API and returns updated status.
// This is the lazy guarding mechanism - used when DB says expired but we want
// to double-check with the provider in case we missed a webhook.
func (s *billingService) RefreshSubscriptionStatus(ctx context.Context, organizationID int32) (*domain.BillingStatus, error) {
	// Step 1: Check if subscription exists in database
	sub, err := s.repo.GetSubscriptionByOrgID(ctx, organizationID)
	if err != nil {
		// No subscription exists — propagate the sentinel unwrapped (the
		// /status handler and the status-provider adapter compare directly).
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription for refresh: %w", err)
	}

	// Synthetic-customer guard (design D6): local-trial rows MUST NOT trigger
	// a provider call. The external_customer_id == 'local-trial' is never
	// valid at Polar/MP; calling it would leak a synthetic ID. Fail closed.
	if strings.HasPrefix(sub.SubscriptionID, "local-trial-") {
		s.logger.Info("Synthetic trial row — refusing provider refresh", map[string]any{
			"organization_id": organizationID,
			"subscription_id": sub.SubscriptionID,
		})
		return nil, domain.ErrSubscriptionNotFound
	}

	// Step 2: Sync subscription from Polar API
	if err := s.SyncSubscriptionFromPolar(ctx, organizationID); err != nil {
		// Sync failed - return error
		return nil, fmt.Errorf("failed to refresh subscription from Polar: %w", err)
	}

	// Step 3: Get fresh billing status from database (after sync)
	billingStatus, err := s.GetBillingStatus(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing status after refresh: %w", err)
	}

	s.logger.Info("Subscription status refreshed", map[string]any{
		"organization_id":         organizationID,
		"has_active_subscription": billingStatus.HasActiveSubscription,
		"invoice_count":           billingStatus.InvoiceCount,
	})

	// Console log for refresh completion
	fmt.Printf("🔄 SUBSCRIPTION REFRESHED - Org: %d | Active: %v | Invoice Count: %d | Reason: %s\n",
		organizationID, billingStatus.HasActiveSubscription, billingStatus.InvoiceCount, billingStatus.Reason)

	return billingStatus, nil
}
