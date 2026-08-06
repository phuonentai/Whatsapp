# Billing Guide

The billing system integrates with Polar.sh for subscription management, usage-based billing, and payment processing with a hybrid sync strategy.

## Architecture

**Polar.sh**: Payment provider for subscriptions and metering
**Hybrid Sync**: Webhooks + on-demand fetching
**Paywall Middleware**: Protects routes based on subscription status
**Quota Tracking**: Usage-based billing with meters

## Core Concepts

### Subscriptions

Managed in Polar.sh, synced to local database.

**Subscription states:**
- `active` - Valid subscription
- `incomplete` - Payment pending
- `cancelled` - Subscription cancelled
- `unpaid` - Payment failed

### Quota Tracking

Track usage for metered billing.

**How it works:**
1. User performs action (API call, file upload, etc.)
2. System increments local quota counter
3. Periodically sync usage to Polar meters
4. Polar charges based on usage

### Billing Status

Represents organization's billing state:

- **Subscription**: Active subscription details
- **Quota Usage**: Current usage vs limits
- **Payment Status**: Last payment result
- **Metering**: Usage meters for billing

## Hybrid Sync Strategy

Combines webhooks with on-demand fetching for reliability.

### Webhook Path (Real-time)

```
Polar Event → Webhook → Update Database
```

**Handles:**
- Subscription created/updated/cancelled
- Payment succeeded/failed
- Customer created/updated

### Lazy Guarding (On-demand)

```
API Request → Check Subscription → Fetch if stale → Update Database
```

**When used:**
- Webhook delivery failed
- Data drift detected
- Initial subscription fetch

**Benefits:**
- Self-healing system
- No critical webhook dependency
- Always up-to-date data

## Paywall Middleware

Protects routes based on subscription requirements.

### Basic Usage

```go
router.POST("/premium-feature",
    paywallMiddleware.RequireActiveSubscription(),
    handler.PremiumFeature)
```

### Quota-Based Protection

```go
router.POST("/api-call",
    paywallMiddleware.RequireQuota("api_calls", 1),
    handler.APICall)
```

**What it does:**
1. Checks organization has active subscription
2. Verifies quota available
3. Increments usage counter
4. Returns 402 (Payment Required) if quota exceeded

### Feature-Based Protection

```go
router.POST("/advanced-feature",
    paywallMiddleware.RequireFeature("advanced_analytics"),
    handler.AdvancedFeature)
```

Checks if subscription plan includes specific feature.

## Webhook Processing

Polar sends webhooks for billing events.

### Webhook Handler

Located in `internal/billing/polar_handler.go`.

**Events handled:**
- `subscription.created`
- `subscription.updated`
- `subscription.canceled`
- `checkout.created`
- `checkout.updated`

### Verification

Webhooks are verified using Polar webhook secret:

```env
POLAR_WEBHOOK_SECRET=whsec_xxx
```

Invalid signatures are rejected.

## Usage Tracking

Track resource usage for billing.

### Recording Usage

```go
func (s *service) ProcessAction(ctx context.Context, orgID int32) error {
    // Perform action
    result, err := s.doAction(ctx)
    if err != nil {
        return err
    }

    // Record usage
    err = s.billingService.IncrementQuota(ctx, orgID, "actions", 1)
    if err != nil {
        // Log error but don't fail the operation
        log.Error("failed to record usage", zap.Error(err))
    }

    return nil
}
```

### Meter Ingestion

Usage synced to Polar periodically:

1. Accumulate usage locally
2. Batch send to Polar meters API
3. Polar charges based on metered usage

Configured in `internal/billing/app/services/metering_service.go`.

## Configuration

```env
# Polar.sh
POLAR_ACCESS_TOKEN=polar_xxx
POLAR_WEBHOOK_SECRET=whsec_xxx
POLAR_ORGANIZATION_ID=org_xxx
```

## Common Patterns

### Check Subscription Status

```go
func (h *Handler) GetFeature(c *gin.Context) {
    orgID := auth.GetOrganizationID(c)

    status, err := h.billingService.GetBillingStatus(ctx, orgID)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to get billing status"})
        return
    }

    if status.Subscription == nil || !status.Subscription.IsActive() {
        c.JSON(402, gin.H{"error": "active subscription required"})
        return
    }

    // Proceed with feature
}
```

### Track Usage

```go
func (s *service) ProcessFile(ctx context.Context, orgID int32, file *File) error {
    // Process file
    err := s.processor.Process(file)
    if err != nil {
        return err
    }

    // Record usage
    s.billingService.IncrementQuota(ctx, orgID, "files_processed", 1)

    return nil
}
```

### Handle Payment Failures

```go
func (h *WebhookHandler) HandlePaymentFailed(ctx context.Context, event *Event) error {
    // Update subscription status
    err := h.billingService.UpdateSubscriptionStatus(ctx, event.SubscriptionID, "unpaid")
    if err != nil {
        return err
    }

    // Notify organization
    h.notificationService.SendPaymentFailure(ctx, event.OrganizationID)

    return nil
}
```

## File Locations

| Component | Path |
|-----------|------|
| Billing domain | `internal/billing/domain/` |
| Billing service | `internal/billing/app/services/` |
| Polar adapter | `internal/billing/infra/adapters/polar/` |
| Paywall middleware | `internal/paywall/` |
| Polar client | `internal/polar/` |
| Webhook handlers | `internal/billing/` |

## Next Steps

- **API protection**: Use paywall middleware in routes
- **Usage tracking**: Implement quota consumption
- **Polar documentation**: https://docs.polar.sh/
