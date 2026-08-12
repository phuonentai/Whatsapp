package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// isActiveSubscriptionStatus reports whether a subscription status grants
// active access. Mirrors paywall.IsActiveStatus; trialing is active.
func isActiveSubscriptionStatus(status string) bool {
	return status == "active" || status == "trialing"
}

func (s *billingService) GetBillingStatus(ctx context.Context, organizationID int32) (*domain.BillingStatus, error) {
	// Get quota status from database
	quotaStatus, err := s.repo.GetQuotaStatus(ctx, organizationID)
	if err != nil {
		// The /status handler compares with direct equality (err == domain.ErrSubscriptionNotFound),
		// so return the sentinel unwrapped.
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, domain.ErrSubscriptionNotFound
		}
		// DB failures propagate wrapped — never a silent nil-error "none".
		return nil, fmt.Errorf("get quota status: %w", err)
	}

	// Enforce trial expiry at the status boundary (design D6).
	// When a trial has passed current_period_end, classify as expired —
	// the paywall lazy guard fires and a provider-backed trial heals;
	// a local-only trial fails closed to 402.
	if quotaStatus.SubscriptionStatus == "trialing" && time.Now().After(quotaStatus.CurrentPeriodEnd) {
		return &domain.BillingStatus{
			OrganizationID:        organizationID,
			HasActiveSubscription: false,
			CanProcessInvoices:    false,
			InvoiceCount:          quotaStatus.InvoiceCount,
			Reason:                "trial expired",
			CheckedAt:             time.Now(),
		}, nil
	}

	// Build billing status from quota status
	return &domain.BillingStatus{
		OrganizationID:        organizationID,
		HasActiveSubscription: isActiveSubscriptionStatus(quotaStatus.SubscriptionStatus),
		CanProcessInvoices:    quotaStatus.CanProcessInvoice,
		InvoiceCount:          quotaStatus.InvoiceCount,
		Reason:                s.buildStatusReason(quotaStatus),
		CheckedAt:             time.Now(),
	}, nil
}

func (s *billingService) buildStatusReason(status *domain.QuotaStatus) string {
	if !status.CanProcessInvoice {
		if !isActiveSubscriptionStatus(status.SubscriptionStatus) {
			return fmt.Sprintf("subscription status: %s", status.SubscriptionStatus)
		}
		return "invoice quota exceeded"
	}
	return "ok"
}
