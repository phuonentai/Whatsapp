package features

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

type billingFeatureProvider struct {
	subRepo domain.SubscriptionRepository
	store   sqlc.Store
	logger  logger.Logger
}

func NewBillingFeatureProvider(
	subRepo domain.SubscriptionRepository,
	store sqlc.Store,
	logger logger.Logger,
) features.FeatureProvider {
	return &billingFeatureProvider{
		subRepo: subRepo,
		store:   store,
		logger:  logger,
	}
}

func (p *billingFeatureProvider) GetEntitlement(ctx context.Context, orgID int32) (*features.Entitlement, error) {
	sub, err := p.subRepo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return &features.Entitlement{
			Features: make(map[string]bool),
			Quotas:   make(map[string]int32),
			Usage:    make(map[string]int32),
		}, nil
	}

	isActive := sub.SubscriptionStatus == "active" || sub.SubscriptionStatus == "trialing"
	isGracePeriod := sub.SubscriptionStatus == "past_due"
	isReadOnly := !isActive && !isGracePeriod

	featuresMap := parseCRMFeatures(sub.Metadata, isActive || isGracePeriod)
	quotas := parseQuotas(sub.Metadata)
	usage := p.getUsage(ctx, orgID)

	return &features.Entitlement{
		Features:      featuresMap,
		Quotas:        quotas,
		Usage:         usage,
		IsReadOnly:    isReadOnly,
		IsGracePeriod: isGracePeriod,
		PlanName:      sub.PlanName,
	}, nil
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

	raw, ok := metadata["crm_features"]
	if !ok {
		return result
	}

	str, ok := raw.(string)
	if !ok {
		return result
	}

	for _, feature := range strings.Split(str, ",") {
		feature = strings.TrimSpace(feature)
		if feature != "" {
			result[feature] = true
		}
	}

	return result
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

	return usage
}
