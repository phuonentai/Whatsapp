## Context

The paywall spec and the feature layer already treat `trialing` as active: `paywall.IsActiveStatus` and `billingFeatureProvider.GetEntitlement` (`billing_provider.go:41`) both include `trialing`. The billing status/quota layer contradicts them — `GetBillingStatus` (`get_billing_status_service.go:16,39`), the three quota services (`check_quota_availability_service.go:54`, `verify_and_consume_quota_service.go:52`, `consume_invoice_quota_service.go:54`), `needsFallbackVerification` (`verify_and_consume_quota_service.go:82`), and the SQL `can_process_invoice` gate (`subscription_billing.sql:112`) all recognize only `"active"`. Result: a trialing org receives 402 on paywalled routes and cannot process invoices. No schema or provider change is involved.

## Goals / Non-Goals

**Goals:**
- Make `trialing` evaluate as active everywhere status is derived for gating and quota
- Single source for the active-status predicate in the billing services layer
- Keep behavior identical for every other status

**Non-Goals:**
- Creating trials (see `new-client-billing-lifecycle`)
- Dunning, status-set changes, paywall middleware changes
- Provider adapter changes

## Decisions

### 1. One helper in the billing services package

Introduce `isActiveSubscriptionStatus(status string) bool` (`"active" || "trialing"`) in the services package and use it in `GetBillingStatus`, `buildStatusReason`, the three quota services, and `needsFallbackVerification`.

**Alternatives considered:**
- Reuse `paywall.IsActiveStatus` directly: avoids duplication but imports the paywall package into services (the adapter currently maps billing → paywall; keeping the direction one-way preserves layering). Accepted trade-off: the helper body is three lines and mirrors the spec.

### 2. SQL gate updated, SQLC regenerated

`can_process_invoice` becomes `WHEN s.subscription_status IN ('active','trialing') AND q.invoice_count > 0`, then `make sqlc` regenerates `gen/subscription_billing.sql.go`.

**Alternatives considered:**
- Compute the gate in Go after reading raw rows: would require changing the `GetQuotaStatus` query shape and every consumer; rejected — the SQL gate is the single read path used by all quota services.

## Risks / Trade-offs

- **[Risk] `make sqlc` regenerates generated files that other in-flight work may also touch** → Mitigation: run the repo's `make sqlc` target (docker cli container, pinned version); review the diff before committing.
- **[Risk] Trialing orgs with a provider-side trial now pass quota checks until period end** → Intended: trial grants access per spec; the period boundary (`current_period_end`) is enforced by the provider webhook that moves status off `trialing` at trial end.

## Migration Plan

1. Add the helper and update the five service call sites.
2. Update the SQL query; run `make sqlc`.
3. `go build ./...`, `go test ./...`, `go vet ./internal/modules/...`.
4. Rollback: revert the code + query changes (no migration; no provider/St~ytch state touched).

## Open Questions

- Whether to also include `trialing` in the monitoring queries (`ListActiveSubscriptions`, `ListQuotasNearLimit` currently filter `= 'active'`) — deferred, monitoring-only.
