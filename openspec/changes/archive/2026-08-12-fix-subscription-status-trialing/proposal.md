## Why

The paywall spec (`openspec/specs/paywall/spec.md`) requires `trialing` to evaluate as active (`IsActive=true`), and the FeatureProvider already honors that (`billing_provider.go:41`). But the billing status/quota layer treats only `"active"` as active, so trial orgs are denied on paywalled routes and cannot process invoices: `GetBillingStatus` reports `HasActiveSubscription=false`, the SQL `can_process_invoice` gate is false, and the paywall adapter passes that straight to the 402 middleware. A trial is therefore a hard lockout, not a trial.

## What Changes

- `get_billing_status_service.go:16,39` — `HasActiveSubscription` and `buildStatusReason` SHALL treat `trialing` as active (introduce a single `isActiveSubscriptionStatus` helper mirroring `paywall.IsActiveStatus`).
- `check_quota_availability_service.go:54`, `verify_and_consume_quota_service.go:52`, `consume_invoice_quota_service.go:54` — quota-service `HasActiveSubscription` SHALL use the same helper.
- `verify_and_consume_quota_service.go:82` — `needsFallbackVerification` SHALL NOT force a provider sync for trialing orgs (`!isActiveSubscriptionStatus` instead of `!= "active"`).
- `internal/db/postgres/sqlc/query/subscription_billing.sql:112` — `can_process_invoice` SHALL be `WHEN s.subscription_status IN ('active','trialing') AND q.invoice_count > 0`; regenerate SQLC (`make sqlc`).
- No change to the set of statuses stored or to paywall middleware behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `paywall`: extend the "Trialing subscription grants access" requirement with quota-enforcement scenarios — a trialing org with remaining quota SHALL process invoices, and a trialing org SHALL NOT receive 402 from `RequireActiveSubscription`. The existing statuses/`IsActive` mapping is unchanged; this only pins the quota tie-in that the current code contradicts.

## Non-Goals

- Creating or seeding trials (covered by `new-client-billing-lifecycle`).
- Dunning / late-payment flows (covered by `new-client-billing-lifecycle`).
- Changing stored status values, paywall middleware, or the feature-gating layer (already trialing-correct).
- No local credential storage is introduced; no Stytch API contract changes.

## Rollback

- **Git**: revert the code + SQLC changes in this change; no migration is required (no schema change — the query edit is regenerated from SQL source).
- **Stytch**: no tenant policy state is touched; subscription status lives only in local DB and provider accounts, so no provider-side rollback is needed.
