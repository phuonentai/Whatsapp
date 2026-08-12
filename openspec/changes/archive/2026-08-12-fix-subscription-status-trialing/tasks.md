## 1. Status derivation [BE-DOMAIN]

- [x] 1.1 Add `isActiveSubscriptionStatus` helper (`"active" || "trialing"`) in `internal/modules/billing/app/services` and use it in `GetBillingStatus` (`HasActiveSubscription`) and `buildStatusReason`
- [x] 1.2 Use the helper in `CheckQuotaAvailability`, `VerifyAndConsumeQuota`, and `ConsumeInvoiceQuota` `HasActiveSubscription` fields
- [x] 1.3 Update `needsFallbackVerification` to `!isActiveSubscriptionStatus(status.SubscriptionStatus)`

## 2. SQL gate [DB-SQLC]

- [x] 2.1 Update `can_process_invoice` in `internal/db/postgres/sqlc/query/subscription_billing.sql` to `s.subscription_status IN ('active','trialing') AND q.invoice_count > 0`
- [x] 2.2 Run `make sqlc` and confirm `gen/subscription_billing.sql.go` reflects the new CASE expression

## 3. Verification [OPS-GOV]

- [x] 3.1 Add unit tests: trialing org → `GetBillingStatus` active, quota services report `HasActiveSubscription=true`, `needsFallbackVerification` false for trialing with quota
- [x] 3.2 Run `go build ./...`, `go vet ./internal/modules/...`, `go test ./...` — all pass
- [x] 3.3 Record verification results and archive decision in `tasks.md`

## Verification results (2026-08-11)

- `go test ./internal/modules/billing/...` → exit 0 (`ok github.com/moasq/go-b2b-starter/internal/modules/billing/app/services`)
- New unit tests (all PASS): `TestGetBillingStatus_TrialingReportsActive`, `TestGetBillingStatus_TrialingWithoutQuotaIsQuotaBlockedNotPaywallBlocked`, `TestGetBillingStatus_InactiveStatusReportsInactive`, `TestCheckQuotaAvailability_TrialingReportsActive`, `TestVerifyAndConsumeQuota_TrialingReportsActive`, `TestConsumeInvoiceQuota_TrialingReportsActive`, `TestNeedsFallbackVerification_TrialingWithQuota`
- `go build ./...` → exit 0
- `go vet ./internal/modules/...` → exit 0
- `go test ./...` → exit 0 (full suite green)
- SQLC regeneration: `docker compose -f deps/docker-compose.yml run --no-deps --rm -w /workspace/internal/db/postgres/sqlc cli sqlc generate` → exit 0 (used in place of `make sqlc`, which is blocked by a local postgres port conflict); `gen/subscription_billing.sql.go:162` now emits `WHEN s.subscription_status IN ('active','trialing') AND q.invoice_count > 0`

**Archive decision (2026-08-11):** ready — all 8 tasks `[x]`, every go gate exits 0, and the delta spec scenarios (trialing + quota processes invoices; trialing passes the paywall; trialing without quota is quota-blocked not paywall-blocked) are covered by the new unit tests. Archiving itself is performed by the orchestrator via `/opsx-archive`; no blocking dependencies remain (no schema change, no provider/credential involvement).
