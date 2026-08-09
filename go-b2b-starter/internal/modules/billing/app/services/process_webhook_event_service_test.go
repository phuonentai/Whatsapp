package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// fakeGrantOrgAdapter resolves Stytch org IDs to local IDs.
type fakeGrantOrgAdapter struct {
	domain.OrganizationAdapter
	orgByStytch map[string]int32
}

func (f *fakeGrantOrgAdapter) GetOrganizationIDByStytchOrgID(ctx context.Context, stytchOrgID string) (int32, error) {
	return f.orgByStytch[stytchOrgID], nil
}

// fakeGrantAiRepo captures AI allowance updates.
type fakeGrantAiRepo struct {
	domain.AiUsageRepository
	updates []int32
	err     error
}

func (f *fakeGrantAiRepo) UpdateAiCreditsMax(ctx context.Context, organizationID int32, creditsMax int32, periodStart, periodEnd time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.updates = append(f.updates, creditsMax)
	return nil
}

type fakeGrantLogger struct{}

func (fakeGrantLogger) Debug(msg string, fields ...logger.Fields) {}
func (fakeGrantLogger) Info(msg string, fields ...logger.Fields)  {}
func (fakeGrantLogger) Warn(msg string, fields ...logger.Fields)  {}
func (fakeGrantLogger) Error(msg string, fields ...logger.Fields) {}
func (fakeGrantLogger) Fatal(msg string, fields ...logger.Fields) {}
func (fakeGrantLogger) WithFields(fields logger.Fields) logger.Logger { return fakeGrantLogger{} }

// newGrantService builds a billingService with fakes for grant handling.
func newGrantService(aiRepo domain.AiUsageRepository, orgAdapter domain.OrganizationAdapter) *billingService {
	return &billingService{
		aiRepo:     aiRepo,
		orgAdapter: orgAdapter,
		logger:     fakeGrantLogger{},
	}
}

func TestHandleMeterGrantEvent_AiTokensSlugUpdatesAllowance(t *testing.T) {
	aiRepo := &fakeGrantAiRepo{}
	orgAdapter := &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}}
	svc := newGrantService(aiRepo, orgAdapter)

	err := svc.handleMeterGrantEvent(context.Background(), map[string]any{
		"meter_slug":           "ai.tokens",
		"external_customer_id": "org_stytch_1",
		"balance":              map[string]any{"available": 5000},
	})
	require.NoError(t, err)
	require.Len(t, aiRepo.updates, 1)
	assert.Equal(t, int32(5000), aiRepo.updates[0])
}

func TestHandleMeterGrantEvent_UnrelatedSlugIsIgnored(t *testing.T) {
	aiRepo := &fakeGrantAiRepo{}
	svc := newGrantService(aiRepo, &fakeGrantOrgAdapter{})

	err := svc.handleMeterGrantEvent(context.Background(), map[string]any{
		"meter_slug":           "some.other.meter",
		"external_customer_id": "org_stytch_1",
		"balance":              map[string]any{"available": 100},
	})
	require.NoError(t, err)
}
