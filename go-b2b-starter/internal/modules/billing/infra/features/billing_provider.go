package features

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

type billingFeatureProvider struct {
	subRepo       domain.SubscriptionRepository
	aiRepo        domain.AiUsageRepository
	store         sqlc.Store
	logger        logger.Logger
	moduleService registryServices.ModuleService
}

func NewBillingFeatureProvider(
	subRepo domain.SubscriptionRepository,
	aiRepo domain.AiUsageRepository,
	store sqlc.Store,
	logger logger.Logger,
	moduleService registryServices.ModuleService,
) features.FeatureProvider {
	return &billingFeatureProvider{
		subRepo:       subRepo,
		aiRepo:        aiRepo,
		store:         store,
		logger:        logger,
		moduleService: moduleService,
	}
}

func (p *billingFeatureProvider) GetEntitlement(ctx context.Context, orgID int32) (*features.Entitlement, error) {
	sub, err := p.subRepo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return &features.Entitlement{
			Features: make(map[string]bool),
			Quotas:   make(map[string]int32),
			Usage:    make(map[string]int32),
			Modules:  make(map[string]features.ModuleState),
		}, nil
	}

	isActive := sub.SubscriptionStatus == "active" || sub.SubscriptionStatus == "trialing"
	isGracePeriod := sub.SubscriptionStatus == "past_due"
	isReadOnly := !isActive && !isGracePeriod

	featuresMap := parseCRMFeatures(sub.Metadata, isActive || isGracePeriod)
	featuresMap = parseAIFeatures(featuresMap, sub.Metadata, isActive || isGracePeriod)

	// Grant-base paid-plan flags (conversation-row-scoping, Decisión 8 del
	// design): siempre activos para suscripciones activas/trialing/past_due;
	// free/inactiva → false (bandeja org-scope).
	for k, v := range basePaidPlanFeatures(isActive || isGracePeriod) {
		featuresMap[k] = v
	}
	quotas := parseQuotas(sub.Metadata)
	usage := p.getUsage(ctx, orgID)

	// Module entitlements: union plan features with purchased module features.
	modules := p.resolveModules(ctx, orgID, sub.Metadata, isActive || isGracePeriod)
	for _, state := range modules {
		if !state.Enabled {
			continue
		}
		for _, feature := range state.Features {
			featuresMap[feature] = true
		}
	}

	return &features.Entitlement{
		Features:      featuresMap,
		Quotas:        quotas,
		Usage:         usage,
		IsReadOnly:    isReadOnly,
		IsGracePeriod: isGracePeriod,
		PlanName:      sub.PlanName,
		Modules:       modules,
	}, nil
}

// defaultGrantedModules are always enabled for active subscriptions,
// independent of purchased product metadata (read-only base features).
var defaultGrantedModules = []string{"analytics"}

// basePaidPlanFeatures devuelve los flags de grant base de planes pagos.
// conversation_row_scoping (conversation-row-scoping) solo es true con
// suscripción activa/trialing/past_due; free/inactiva → vacío (flag false).
func basePaidPlanFeatures(enabled bool) map[string]bool {
	if !enabled {
		return map[string]bool{}
	}
	return map[string]bool{
		"conversation_row_scoping": true,
	}
}

// resolveModules determines per-org module state from subscription metadata
// module keys (e.g., "module_tickets") merged with stored per-org configs.
// Inactive subscriptions never receive module features.
func (p *billingFeatureProvider) resolveModules(
	ctx context.Context,
	orgID int32,
	metadata map[string]any,
	enabled bool,
) map[string]features.ModuleState {
	result := make(map[string]features.ModuleState)
	if !enabled {
		return result
	}

	metadataKeys := parseModuleKeys(metadata)
	if len(metadataKeys) == 0 {
		metadataKeys = []string{}
	}
	// Base-plan modules are always granted to active subscriptions.
	metadataKeys = append(metadataKeys, defaultGrantedModules...)

	active, err := p.moduleService.ListCatalogInternal(ctx)
	if err != nil {
		p.logger.Error("failed to list module catalog for entitlement", map[string]any{
			"organization_id": orgID,
			"error":           err.Error(),
		})
		return result
	}
	byKey := make(map[string]features.ModuleState, len(active))
	for _, m := range active {
		byKey[m.Key] = features.ModuleState{
			Enabled:  false,
			Features: m.GrantedFeatures,
		}
	}

	// Per-org configs (may be empty if the billing webhook sync has not run).
	configByKey := p.orgModuleConfigs(ctx, orgID)

	for _, key := range metadataKeys {
		state, ok := byKey[key]
		if !ok {
			continue
		}
		state.Enabled = true
		state.Config = configByKey[key]
		if state.Config == nil {
			state.Config = map[string]any{}
		}
		result[key] = state
	}
	return result
}

func (p *billingFeatureProvider) orgModuleConfigs(ctx context.Context, orgID int32) map[string]map[string]any {
	configs := make(map[string]map[string]any)
	_, orgMods, err := p.moduleService.ListOrgModules(ctx, orgID)
	if err != nil {
		p.logger.Error("failed to resolve org modules for entitlement", map[string]any{
			"organization_id": orgID,
			"error":           err.Error(),
		})
		return configs
	}
	for _, om := range orgMods {
		configs[om.ModuleKey] = om.Config
	}
	return configs
}

func parseCRMFeatures(metadata map[string]any, enabled bool) map[string]bool {
	result := make(map[string]bool)

	if !enabled {
		result["crm_contacts_manage"] = false
		result["crm_companies"] = false
		result["crm_deals"] = false
		result["crm_activities"] = false
		result["crm_tags"] = false
		return result
	}

	for _, raw := range []any{metadata["crm_features"], nestedValue(metadata, "crm_features")} {
		switch v := raw.(type) {
		case string:
			for _, feature := range strings.Split(v, ",") {
				feature = strings.TrimSpace(feature)
				if feature != "" {
					result[feature] = true
				}
			}
		}
	}

	return result
}

// parseAIFeatures derives AI feature flags from subscription metadata.
// Supports the direct `ai_assistant` key (boolean or "true"/"false") and a
// comma-separated `ai_features` list, mirroring the `crm_features` pattern.
func parseAIFeatures(base map[string]bool, metadata map[string]any, enabled bool) map[string]bool {
	if !enabled {
		base["ai_assistant"] = false
		return base
	}

	if raw, ok := metadata["ai_assistant"]; ok {
		base["ai_assistant"] = parseBoolValue(raw)
	}
	for _, raw := range []any{metadata["ai_features"], nestedValue(metadata, "ai_features")} {
		switch v := raw.(type) {
		case string:
			for _, feature := range strings.Split(v, ",") {
				feature = strings.TrimSpace(feature)
				if feature != "" {
					base[feature] = true
				}
			}
		}
	}

	return base
}

func parseBoolValue(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true
		default:
			return false
		}
	}
	return false
}

// parseModuleKeys extracts namespaced module keys (e.g., "module_tickets")
// from subscription metadata, checking both the top level and the nested
// "product_metadata" map used by the Polar adapter.
func parseModuleKeys(metadata map[string]any) []string {
	seen := make(map[string]bool)
	var keys []string
	add := func(k string) {
		if seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range metadata {
		if strings.HasPrefix(k, "module_") {
			add(strings.TrimPrefix(k, "module_"))
		}
	}
	if nested := nestedMap(metadata, "product_metadata"); nested != nil {
		for k := range nested {
			if strings.HasPrefix(k, "module_") {
				add(strings.TrimPrefix(k, "module_"))
			}
		}
	}
	return keys
}

// nestedValue looks up a key inside the nested "product_metadata" map.
func nestedValue(metadata map[string]any, key string) any {
	if nested := nestedMap(metadata, "product_metadata"); nested != nil {
		return nested[key]
	}
	return nil
}

// nestedMap returns the "product_metadata" value as map[string]any, supporting
// both map[string]any and map[string]string shapes.
func nestedMap(metadata map[string]any, key string) map[string]any {
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	switch m := raw.(type) {
	case map[string]any:
		return m
	case map[string]string:
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	return nil
}

func parseQuotas(metadata map[string]any) map[string]int32 {
	quotas := make(map[string]int32)

	if raw, ok := metadata["max_contactos"]; ok {
		if val, err := strconv.ParseInt(fmt.Sprintf("%v", raw), 10, 32); err == nil {
			quotas["contacts"] = int32(val)
		}
	}

	if raw, ok := metadata["max_negocios"]; ok {
		if val, err := strconv.ParseInt(fmt.Sprintf("%v", raw), 10, 32); err == nil {
			quotas["deals"] = int32(val)
		}
	}

	if raw, ok := metadata["ai_credits_max"]; ok {
		if val, err := strconv.ParseInt(fmt.Sprintf("%v", raw), 10, 32); err == nil {
			quotas["ai_credits"] = int32(val)
		}
	}

	return quotas
}

func (p *billingFeatureProvider) getUsage(ctx context.Context, orgID int32) map[string]int32 {
	usage := make(map[string]int32)

	contactCount, err := p.store.CountContactsByOrganization(ctx, orgID)
	if err == nil {
		usage["contacts"] = int32(contactCount)
	}

	dealCount, err := p.store.CountDealsByOrganization(ctx, orgID)
	if err == nil {
		usage["deals"] = int32(dealCount)
	}

	p.mergeAiUsage(ctx, orgID, usage)

	return usage
}

// mergeAiUsage adds the current-period AI ledger state to the entitlement
// usage map: token totals, credits used, and credits remaining.
func (p *billingFeatureProvider) mergeAiUsage(ctx context.Context, orgID int32, usage map[string]int32) {
	usage["ai_tokens_input"] = 0
	usage["ai_tokens_output"] = 0
	usage["ai_tokens_embedding"] = 0
	usage["ai_credits_used"] = 0
	usage["ai_credits_remaining"] = 0

	periodStart, ok := p.currentPeriodStart(ctx, orgID)
	if !ok {
		return
	}

	aiUsage, err := p.aiRepo.GetAiUsageByOrgAndPeriod(ctx, orgID, periodStart)
	if err != nil {
		if !errors.Is(err, domain.ErrAiUsageNotFound) {
			p.logger.Error("failed to load ai usage for entitlement", map[string]any{
				"organization_id": orgID,
				"error":           err.Error(),
			})
		}
		return
	}

	maxCredits := p.aiCreditsMax(ctx, orgID)

	usage["ai_tokens_input"] = int32(aiUsage.TokensInput)
	usage["ai_tokens_output"] = int32(aiUsage.TokensOutput)
	usage["ai_tokens_embedding"] = int32(aiUsage.TokensEmbedding)
	usage["ai_credits_used"] = aiUsage.CreditsUsed
	if remaining := maxCredits - aiUsage.CreditsUsed; remaining > 0 {
		usage["ai_credits_remaining"] = remaining
	}
}

// currentPeriodStart resolves the org's active billing period start for the
// AI ledger lookup.
func (p *billingFeatureProvider) currentPeriodStart(ctx context.Context, orgID int32) (time.Time, bool) {
	if quota, err := p.subRepo.GetQuotaByOrgID(ctx, orgID); err == nil && !quota.PeriodStart.IsZero() {
		return quota.PeriodStart, true
	}
	if sub, err := p.subRepo.GetSubscriptionByOrgID(ctx, orgID); err == nil {
		return sub.CurrentPeriodStart, true
	}
	return time.Time{}, false
}

// aiCreditsMax reads the org's period AI credit allowance from quota tracking.
func (p *billingFeatureProvider) aiCreditsMax(ctx context.Context, orgID int32) int32 {
	quota, err := p.subRepo.GetQuotaByOrgID(ctx, orgID)
	if err != nil {
		return 0
	}
	return quota.AiCreditsMax
}
