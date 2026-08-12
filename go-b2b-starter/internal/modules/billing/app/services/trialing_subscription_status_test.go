package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// fakeQuotaRepo stubs the subscription repository for status/quota tests.
type fakeQuotaRepo struct {
	domain.SubscriptionRepository
	quotaStatus  *domain.QuotaStatus
	decremented  *domain.QuotaTracking
	getErr       error
	decrementErr error
}

func (f *fakeQuotaRepo) GetQuotaStatus(ctx context.Context, organizationID int32) (*domain.QuotaStatus, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.quotaStatus, nil
}

func (f *fakeQuotaRepo) DecrementInvoiceCount(ctx context.Context, organizationID int32) (*domain.QuotaTracking, error) {
	if f.decrementErr != nil {
		return nil, f.decrementErr
	}
	if f.decremented != nil {
		return f.decremented, nil
	}
	return &domain.QuotaTracking{InvoiceCount: f.quotaStatus.InvoiceCount - 1}, nil
}

// fakeTrialingOrgAdapter fails external-org resolution so the best-effort
// Polar meter goroutine in ConsumeInvoiceQuota logs and returns safely.
type fakeTrialingOrgAdapter struct {
	domain.OrganizationAdapter
}

func (fakeTrialingOrgAdapter) GetStytchOrgID(ctx context.Context, organizationID int32) (string, error) {
	return "", errors.New("no external org")
}

func newTrialingService(repo domain.SubscriptionRepository) *billingService {
	return &billingService{
		repo:       repo,
		orgAdapter: fakeTrialingOrgAdapter{},
		logger:     fakeGrantLogger{},
	}
}

func TestGetBillingStatus_TrialingReportsActive(t *testing.T) {
	svc := newTrialingService(&fakeQuotaRepo{
		quotaStatus: &domain.QuotaStatus{
			SubscriptionStatus: "trialing",
			CurrentPeriodEnd:   time.Now().Add(7 * 24 * time.Hour), // trial not expired
			InvoiceCount:       5,
			CanProcessInvoice:  true,
		},
	})

	status, err := svc.GetBillingStatus(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, status.HasActiveSubscription)
	assert.True(t, status.CanProcessInvoices)
	assert.Equal(t, "ok", status.Reason)
}

func TestGetBillingStatus_TrialingWithoutQuotaIsQuotaBlockedNotPaywallBlocked(t *testing.T) {
	svc := newTrialingService(&fakeQuotaRepo{
		quotaStatus: &domain.QuotaStatus{
			SubscriptionStatus: "trialing",
			CurrentPeriodEnd:   time.Now().Add(7 * 24 * time.Hour), // trial not expired
			InvoiceCount:       0,
			CanProcessInvoice:  false,
		},
	})

	status, err := svc.GetBillingStatus(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, status.HasActiveSubscription)
	assert.False(t, status.CanProcessInvoices)
	assert.Equal(t, "invoice quota exceeded", status.Reason)
}

func TestGetBillingStatus_InactiveStatusReportsInactive(t *testing.T) {
	svc := newTrialingService(&fakeQuotaRepo{
		quotaStatus: &domain.QuotaStatus{
			SubscriptionStatus: "canceled",
			InvoiceCount:       5,
			CanProcessInvoice:  false,
		},
	})

	status, err := svc.GetBillingStatus(context.Background(), 42)
	require.NoError(t, err)
	assert.False(t, status.HasActiveSubscription)
	assert.Equal(t, "subscription status: canceled", status.Reason)
}

func TestCheckQuotaAvailability_TrialingReportsActive(t *testing.T) {
	t.Run("trialing with exhausted quota is quota-blocked but active", func(t *testing.T) {
		svc := newTrialingService(&fakeQuotaRepo{
			quotaStatus: &domain.QuotaStatus{
				SubscriptionStatus: "trialing",
				InvoiceCount:       0,
				CanProcessInvoice:  false,
			},
		})

		status, err := svc.CheckQuotaAvailability(context.Background(), 42)
		require.ErrorIs(t, err, domain.ErrQuotaExceeded)
		require.NotNil(t, status)
		assert.True(t, status.HasActiveSubscription)
		assert.False(t, status.CanProcessInvoices)
	})

	t.Run("trialing with remaining quota passes", func(t *testing.T) {
		svc := newTrialingService(&fakeQuotaRepo{
			quotaStatus: &domain.QuotaStatus{
				SubscriptionStatus: "trialing",
				InvoiceCount:       20,
				CanProcessInvoice:  true,
			},
		})

		status, err := svc.CheckQuotaAvailability(context.Background(), 42)
		require.NoError(t, err)
		assert.True(t, status.HasActiveSubscription)
		assert.True(t, status.CanProcessInvoices)
		assert.Equal(t, int32(20), status.InvoiceCount)
	})
}

func TestVerifyAndConsumeQuota_TrialingReportsActive(t *testing.T) {
	t.Run("trialing with exhausted quota is quota-blocked but active", func(t *testing.T) {
		svc := newTrialingService(&fakeQuotaRepo{
			quotaStatus: &domain.QuotaStatus{
				SubscriptionStatus: "trialing",
				InvoiceCount:       0,
				CanProcessInvoice:  false,
			},
		})

		status, err := svc.VerifyAndConsumeQuota(context.Background(), 42)
		require.ErrorIs(t, err, domain.ErrQuotaExceeded)
		require.NotNil(t, status)
		assert.True(t, status.HasActiveSubscription)
		assert.False(t, status.CanProcessInvoices)
	})

	t.Run("trialing with remaining quota consumes an invoice", func(t *testing.T) {
		repo := &fakeQuotaRepo{
			quotaStatus: &domain.QuotaStatus{
				SubscriptionStatus: "trialing",
				InvoiceCount:       20,
				CanProcessInvoice:  true,
			},
		}
		svc := newTrialingService(repo)

		status, err := svc.VerifyAndConsumeQuota(context.Background(), 42)
		require.NoError(t, err)
		assert.True(t, status.HasActiveSubscription)
		assert.True(t, status.CanProcessInvoices)
		assert.Equal(t, int32(19), status.InvoiceCount)
	})
}

func TestConsumeInvoiceQuota_TrialingReportsActive(t *testing.T) {
	repo := &fakeQuotaRepo{
		quotaStatus: &domain.QuotaStatus{
			SubscriptionStatus: "trialing",
			InvoiceCount:       3,
			CanProcessInvoice:  true,
		},
		decremented: &domain.QuotaTracking{InvoiceCount: 2},
	}
	svc := newTrialingService(repo)

	status, err := svc.ConsumeInvoiceQuota(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, status.HasActiveSubscription)
	assert.True(t, status.CanProcessInvoices)
	assert.Equal(t, int32(2), status.InvoiceCount)
}

func TestGetBillingStatus_ErrSubscriptionNotFoundPropagatesUnwrapped(t *testing.T) {
	// Design D1 regression pin: the /status handler compares with direct equality.
	svc := newTrialingService(&fakeQuotaRepo{
		getErr: domain.ErrSubscriptionNotFound,
	})

	_, err := svc.GetBillingStatus(context.Background(), 42)
	require.ErrorIs(t, err, domain.ErrSubscriptionNotFound)
	// Direct equality (handler's comparison):
	require.True(t, err == domain.ErrSubscriptionNotFound)
}

func TestGetBillingStatus_DBErrorPropagatesWrapped(t *testing.T) {
	svc := newTrialingService(&fakeQuotaRepo{
		getErr: errors.New("connection refused"),
	})

	_, err := svc.GetBillingStatus(context.Background(), 42)
	require.Error(t, err)
	require.False(t, errors.Is(err, domain.ErrSubscriptionNotFound))
}

func TestGetBillingStatus_TrialExpiredReportsInactive(t *testing.T) {
	// Design D6: trialing with now() > current_period_end is expired.
	svc := newTrialingService(&fakeQuotaRepo{
		quotaStatus: &domain.QuotaStatus{
			SubscriptionStatus: "trialing",
			CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour), // expired 1 hour ago
			InvoiceCount:       5,
			CanProcessInvoice:  true,
		},
	})

	status, err := svc.GetBillingStatus(context.Background(), 42)
	require.NoError(t, err)
	assert.False(t, status.HasActiveSubscription, "expired trial must be inactive")
	assert.Equal(t, "trial expired", status.Reason)
}

func TestNeedsFallbackVerification_TrialingWithQuota(t *testing.T) {
	svc := newTrialingService(&fakeQuotaRepo{})

	tests := []struct {
		name   string
		status *domain.QuotaStatus
		want   bool
	}{
		{
			name:   "trialing with quota does not need fallback",
			status: &domain.QuotaStatus{SubscriptionStatus: "trialing", InvoiceCount: 20},
			want:   false,
		},
		{
			name:   "trialing near quota limit still needs fallback",
			status: &domain.QuotaStatus{SubscriptionStatus: "trialing", InvoiceCount: 5},
			want:   true,
		},
		{
			name:   "active with quota does not need fallback",
			status: &domain.QuotaStatus{SubscriptionStatus: "active", InvoiceCount: 20},
			want:   false,
		},
		{
			name:   "inactive status with quota needs fallback",
			status: &domain.QuotaStatus{SubscriptionStatus: "canceled", InvoiceCount: 20},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, svc.needsFallbackVerification(tt.status))
		})
	}
}
