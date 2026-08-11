# Tasks

## 1. Capability spec

- [x] 1.1 [OPS-GOV] Create delta spec `paywall` at `openspec/changes/add-paywall-access-gating/specs/paywall/spec.md` covering the subscription status state machine, 402 error contract, lazy-guard refresh, blocking/optional middleware modes, adapter contract, and named middleware registration. Verify: `openspec validate --change add-paywall-access-gating` exits 0.

## 2. Verification gate

- [x] 2.1 [OPS-GOV] Confirm the spec matches implemented behavior: `grep -n "RefreshSubscriptionStatus" go-b2b-starter/internal/modules/paywall/middleware.go go-b2b-starter/internal/modules/billing/infra/adapters/status_provider.go` returns matches for the lazy-guard path. Verify: both files listed.
- [x] 2.2 [OPS-GOV] Confirm the 402 error contract matches code: `grep -n "payment_failed\|subscription_canceled\|payment_required\|subscription_inactive" go-b2b-starter/internal/modules/paywall/middleware.go` returns matches. Verify: all four codes present.
- [x] 2.3 [OPS-GOV] Confirm named middleware registration matches code: `grep -n "RegisterNamedMiddleware" go-b2b-starter/internal/modules/paywall/provider.go` returns matches for `paywall`, `paywall_optional`, and legacy aliases. Verify: at least four registrations.
- [x] 2.4 [OPS-GOV] Run `openspec validate` on the change; fix any schema errors. Verify: command exits 0.
- [x] 2.5 [OPS-GOV] Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>` to this file. Verify: entry present.

**Archive decision:** archived via `/opsx-archive` on 2026-08-10. Delta spec synced to living spec `openspec/specs/paywall/spec.md`.
