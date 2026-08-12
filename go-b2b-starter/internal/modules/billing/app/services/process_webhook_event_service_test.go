package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/features"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	registryDomain "github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
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

// TestHandleCustomerUpdated_MetadataOnlyPreservesCount covers the
// non-destructive upsert contract (3.2): a customer.updated webhook whose
// metadata carries no invoice_count must pass the -1 "no new value" sentinel
// and must NOT clobber the stored count.
func TestHandleCustomerUpdated_MetadataOnlyPreservesCount(t *testing.T) {
	repo := &fakeUpsertRepo{storedQuota: &domain.QuotaTracking{
		OrganizationID: 42,
		InvoiceCount:   25,
		MaxSeats:       5,
	}}
	svc := newUpsertService(repo, &fakeSyncModuleSvc{})

	err := svc.ProcessWebhookEvent(context.Background(), "customer.updated", map[string]any{
		"customer": map[string]any{
			"id":          "cus_1",
			"external_id": "org_stytch_1",
			"metadata": map[string]any{
				"tier": "growth", // no invoice_count -> metadata-only update
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.quotaUpsertCalls)
	// The service forwards the -1 sentinel (SQL preserves the stored value).
	assert.Equal(t, int32(-1), repo.quota.InvoiceCount, "service must pass -1 for metadata-only updates")
	// Emulated SQL semantics: the stored count survives the metadata-only write.
	require.NotNil(t, repo.storedQuota)
	assert.Equal(t, int32(25), repo.storedQuota.InvoiceCount, "stored invoice_count must remain unchanged")
	assert.Equal(t, int32(5), repo.storedQuota.MaxSeats, "stored max_seats must remain unchanged")
}

// TestHandleCustomerUpdated_CountCarryingUpdateReplacesCount covers the
// other side of the contract (3.2): an update that carries invoice_count
// replaces the stored value.
func TestHandleCustomerUpdated_CountCarryingUpdateReplacesCount(t *testing.T) {
	repo := &fakeUpsertRepo{storedQuota: &domain.QuotaTracking{
		OrganizationID: 42,
		InvoiceCount:   25,
		MaxSeats:       5,
	}}
	svc := newUpsertService(repo, &fakeSyncModuleSvc{})

	err := svc.ProcessWebhookEvent(context.Background(), "customer.updated", map[string]any{
		"customer": map[string]any{
			"id":          "cus_1",
			"external_id": "org_stytch_1",
			"metadata": map[string]any{
				"invoice_count": "40",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.quotaUpsertCalls)
	assert.Equal(t, int32(40), repo.quota.InvoiceCount, "count-carrying update passes the new count")
	require.NotNil(t, repo.storedQuota)
	assert.Equal(t, int32(40), repo.storedQuota.InvoiceCount, "stored count is replaced by the carried value")
}

// TestHandleMeterGrantEvent_AiTokensDoesNotInflateConsumedCounts covers
// (3.2): the ai.tokens meter grant refreshes the AI credit allowance via the
// ai-usage repository only; it must never touch (or inflate) the consumed
// invoice_count on the quota row.
func TestHandleMeterGrantEvent_AiTokensDoesNotInflateConsumedCounts(t *testing.T) {
	repo := &fakeUpsertRepo{storedQuota: &domain.QuotaTracking{
		OrganizationID: 42,
		InvoiceCount:   25,
	}}
	aiRepo := &fakeGrantAiRepo{}
	svc := &billingService{
		repo:       repo,
		aiRepo:     aiRepo,
		orgAdapter: &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}},
		logger:     fakeGrantLogger{},
	}

	err := svc.ProcessWebhookEvent(context.Background(), "meter.grant.updated", map[string]any{
		"meter_slug":           "ai.tokens",
		"external_customer_id": "org_stytch_1",
		"balance":              map[string]any{"available": 5000},
	})
	require.NoError(t, err)
	require.Len(t, aiRepo.updates, 1)
	assert.Equal(t, int32(5000), aiRepo.updates[0], "ai allowance refreshed from the grant")
	// The quota row is untouched: no upsert, count not inflated.
	assert.Equal(t, 0, repo.quotaUpsertCalls, "ai.tokens grant must not write the quota row")
	require.NotNil(t, repo.storedQuota)
	assert.Equal(t, int32(25), repo.storedQuota.InvoiceCount, "consumed invoice_count must not be inflated")
}

// fakeUpsertRepo captures subscription/quota upserts and serves stored rows.
// It emulates the SQL preservation contract for quota upserts: an incoming
// InvoiceCount/MaxSeats of -1 means "no new value" and preserves the stored
// value (mirroring COALESCE(NULLIF($n::int,-1), stored) in the real query),
// and reads return a copy so caller-side mutation cannot clobber the row.
type fakeUpsertRepo struct {
	domain.SubscriptionRepository
	upserts          []*domain.Subscription
	stored           *domain.Subscription
	quota            *domain.QuotaTracking
	quotaUpsertCalls int
	// storedQuota is the emulated DB row, distinct from the as-passed quota.
	storedQuota *domain.QuotaTracking
}

func (f *fakeUpsertRepo) UpsertSubscription(ctx context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
	f.upserts = append(f.upserts, sub)
	f.stored = sub
	return sub, nil
}

func (f *fakeUpsertRepo) GetSubscriptionByOrgID(ctx context.Context, orgID int32) (*domain.Subscription, error) {
	if f.stored == nil {
		return nil, domain.ErrSubscriptionNotFound
	}
	return f.stored, nil
}

func (f *fakeUpsertRepo) UpsertQuota(ctx context.Context, quota *domain.QuotaTracking) (*domain.QuotaTracking, error) {
	f.quotaUpsertCalls++
	f.quota = quota

	// Emulate the SQL contract: -1 ("no new value") preserves the stored
	// count; anything else replaces it. A fresh row with -1 seeds 0.
	persisted := &domain.QuotaTracking{}
	if f.storedQuota != nil {
		*persisted = *f.storedQuota
	}
	persisted.OrganizationID = quota.OrganizationID
	if quota.InvoiceCount != -1 {
		persisted.InvoiceCount = quota.InvoiceCount
	}
	if quota.MaxSeats != -1 {
		persisted.MaxSeats = quota.MaxSeats
	}
	persisted.PeriodStart = quota.PeriodStart
	persisted.PeriodEnd = quota.PeriodEnd
	persisted.LastSyncedAt = quota.LastSyncedAt
	f.storedQuota = persisted
	return quota, nil
}

func (f *fakeUpsertRepo) GetQuotaByOrgID(ctx context.Context, orgID int32) (*domain.QuotaTracking, error) {
	if f.storedQuota == nil {
		return nil, domain.ErrQuotaNotFound
	}
	cp := *f.storedQuota
	return &cp, nil
}

// fakeSyncModuleSvc records SyncModulesFromMetadata invocations.
type fakeSyncModuleSvc struct {
	registryServices.ModuleService
	syncCalls [][]string
}

func (f *fakeSyncModuleSvc) SyncModulesFromMetadata(ctx context.Context, orgID int32, moduleKeys []string) error {
	f.syncCalls = append(f.syncCalls, append([]string(nil), moduleKeys...))
	return nil
}

// newUpsertService wires a billingService with capturing fakes.
func newUpsertService(repo *fakeUpsertRepo, modSvc *fakeSyncModuleSvc) *billingService {
	return &billingService{
		repo:          repo,
		orgAdapter:    &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}},
		moduleService: modSvc,
		logger:        fakeGrantLogger{},
	}
}

// polarUpsertPayload is a subscription.updated payload whose product metadata
// carries module, crm/ai feature, and quota keys.
func polarUpsertPayload() map[string]any {
	return map[string]any{
		"id":     "sub_123",
		"status": "active",
		"customer": map[string]any{
			"id":          "cus_1",
			"external_id": "org_stytch_1",
		},
		"product": map[string]any{
			"id":   "prod_1",
			"name": "Growth",
			"metadata": map[string]any{
				"module_tickets": "true",
				"crm_features":   "crm_contacts_manage,crm_deals",
				"ai_features":    "ai_assistant",
				"invoice_count":  "25",
			},
		},
		"current_period_start": "2026-01-01T00:00:00Z",
		"current_period_end":   "2026-02-01T00:00:00Z",
	}
}

// polarUpsertPayloadWithoutMetadata is a subscription.updated payload that
// carries no product metadata.
func polarUpsertPayloadWithoutMetadata() map[string]any {
	return map[string]any{
		"id":     "sub_456",
		"status": "active",
		"customer": map[string]any{
			"id":          "cus_2",
			"external_id": "org_stytch_1",
		},
		"product": map[string]any{
			"id":   "prod_2",
			"name": "Base",
		},
		"current_period_start": "2026-01-01T00:00:00Z",
		"current_period_end":   "2026-02-01T00:00:00Z",
	}
}

func TestHandleSubscriptionUpsert_PersistsProductMetadata(t *testing.T) {
	repo := &fakeUpsertRepo{}
	modSvc := &fakeSyncModuleSvc{}
	svc := newUpsertService(repo, modSvc)

	err := svc.ProcessWebhookEvent(context.Background(), "subscription.updated", polarUpsertPayload())
	require.NoError(t, err)
	require.Len(t, repo.upserts, 1)

	sub := repo.upserts[0]
	require.NotNil(t, sub.Metadata, "upserted row must keep product metadata")
	productMetadata, ok := sub.Metadata["product_metadata"].(map[string]any)
	require.True(t, ok, "product metadata must be persisted under the canonical nested key")
	assert.Equal(t, "true", productMetadata["module_tickets"])
	assert.Equal(t, "crm_contacts_manage,crm_deals", productMetadata["crm_features"])
	assert.Equal(t, "ai_assistant", productMetadata["ai_features"])
	assert.Equal(t, "25", productMetadata["invoice_count"])

	// Module sync receives the actual keys derived from the persisted metadata.
	require.Len(t, modSvc.syncCalls, 1)
	assert.ElementsMatch(t, []string{"tickets"}, modSvc.syncCalls[0])

	// The quota path keeps seeding from the same parsed metadata.
	require.NotNil(t, repo.quota)
	assert.Equal(t, int32(25), repo.quota.InvoiceCount)
}

func TestHandleSubscriptionUpsert_NoMetadataDoesNotDisableModules(t *testing.T) {
	repo := &fakeUpsertRepo{}
	modSvc := &fakeSyncModuleSvc{}
	svc := newUpsertService(repo, modSvc)

	err := svc.ProcessWebhookEvent(context.Background(), "subscription.updated", polarUpsertPayloadWithoutMetadata())
	require.NoError(t, err)
	require.Len(t, repo.upserts, 1)

	// Nothing to persist: the row carries no product metadata.
	assert.Nil(t, repo.upserts[0].Metadata)

	// The module sync is invoked with an empty key set; SyncModulesFromMetadata
	// treats that as "no change", so org modules are not disabled.
	require.Len(t, modSvc.syncCalls, 1)
	assert.Empty(t, modSvc.syncCalls[0])
}

func TestHandleSubscriptionUpsert_MPWebhookWithoutMetadataDoesNotDisableModules(t *testing.T) {
	repo := &fakeUpsertRepo{}
	modSvc := &fakeSyncModuleSvc{}
	svc := newUpsertService(repo, modSvc)

	err := svc.ProcessMPWebhookEvent(context.Background(), json.RawMessage(`{
		"type": "subscription_authorized",
		"id": 9,
		"data": {"id": "preapproval-1", "external_reference": "org_stytch_1", "status": "authorized"}
	}`))
	require.NoError(t, err)
	require.Len(t, repo.upserts, 1)

	// MP events carry no product metadata; the row must not fabricate any.
	assert.Nil(t, repo.upserts[0].Metadata)

	require.Len(t, modSvc.syncCalls, 1)
	assert.Empty(t, modSvc.syncCalls[0])
}

// fakeCatalogRepo serves the module catalog for entitlement resolution.
type fakeCatalogRepo struct {
	registryDomain.ModuleRepository
	modules []*registryDomain.Module
}

func (f *fakeCatalogRepo) ListActive(ctx context.Context) ([]*registryDomain.Module, error) {
	return f.modules, nil
}

func (f *fakeCatalogRepo) GetByKey(ctx context.Context, key string) (*registryDomain.Module, error) {
	for _, m := range f.modules {
		if m.Key == key {
			return m, nil
		}
	}
	return nil, registryDomain.ErrModuleNotFound
}

// fakeOrgModStateRepo serves per-org module state for entitlement resolution.
type fakeOrgModStateRepo struct {
	registryDomain.OrganizationModuleRepository
	orgMods []*registryDomain.OrganizationModule
}

func (f *fakeOrgModStateRepo) ListByOrg(ctx context.Context, orgID int32) ([]*registryDomain.OrganizationModule, error) {
	return f.orgMods, nil
}

func (f *fakeOrgModStateRepo) GetByKey(ctx context.Context, orgID int32, moduleKey string) (*registryDomain.OrganizationModule, error) {
	for _, om := range f.orgMods {
		if om.ModuleKey == moduleKey {
			return om, nil
		}
	}
	return nil, nil
}

func (f *fakeOrgModStateRepo) UpsertConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*registryDomain.OrganizationModule, error) {
	return &registryDomain.OrganizationModule{OrganizationID: orgID, ModuleKey: moduleKey, Config: config}, nil
}

func (f *fakeOrgModStateRepo) Delete(ctx context.Context, orgID int32, moduleKey string) error {
	return nil
}

// fakeEntitlementStore stands in for the sqlc.Store usage-count queries.
type fakeEntitlementStore struct {
	sqlc.Store
}

func (f *fakeEntitlementStore) CountContactsByOrganization(ctx context.Context, orgID int32) (int64, error) {
	return 0, nil
}

func (f *fakeEntitlementStore) CountDealsByOrganization(ctx context.Context, orgID int32) (int64, error) {
	return 0, nil
}

// fakeEntitlementAiRepo reports no AI usage for the period.
type fakeEntitlementAiRepo struct {
	domain.AiUsageRepository
}

func (f *fakeEntitlementAiRepo) GetAiUsageByOrgAndPeriod(ctx context.Context, organizationID int32, periodStart time.Time) (*domain.AiUsage, error) {
	return nil, domain.ErrAiUsageNotFound
}

func TestEntitlementReflectsPersistedMetadataAfterWebhook(t *testing.T) {
	repo := &fakeUpsertRepo{}
	modSvc := &fakeSyncModuleSvc{}
	svc := newUpsertService(repo, modSvc)

	// A subscription webhook persists product metadata on the row...
	err := svc.ProcessWebhookEvent(context.Background(), "subscription.updated", polarUpsertPayload())
	require.NoError(t, err)
	require.Len(t, repo.upserts, 1)

	// ...and the entitlement provider reads it back without any verify/refresh.
	moduleSvc := registryServices.NewModuleService(
		&fakeCatalogRepo{modules: []*registryDomain.Module{
			{Key: "tickets", Name: "Tickets", GrantedFeatures: []string{"tickets_module"}, IsActive: true},
			{Key: "analytics", Name: "Analytics", GrantedFeatures: []string{"analytics_module"}, IsActive: true},
		}},
		&fakeOrgModStateRepo{},
		fakeGrantLogger{},
	)
	provider := features.NewBillingFeatureProvider(
		repo,
		&fakeEntitlementAiRepo{},
		&fakeEntitlementStore{},
		fakeGrantLogger{},
		moduleSvc,
	)

	ent, err := provider.GetEntitlement(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, ent.Features["crm_contacts_manage"], "crm feature from persisted product metadata")
	assert.True(t, ent.Features["crm_deals"], "crm feature from persisted product metadata")
	assert.True(t, ent.Features["ai_assistant"], "ai feature from persisted product metadata")
	assert.True(t, ent.Modules["tickets"].Enabled, "module key from persisted product metadata")
	assert.Equal(t, int32(25), repo.quota.InvoiceCount, "quota still seeded from the same event")
}
