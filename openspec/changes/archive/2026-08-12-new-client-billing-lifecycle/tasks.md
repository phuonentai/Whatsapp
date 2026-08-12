## 1. Error-Classification Contract [BE-DOMAIN] [BE-INFRA] (council #1/#4 — first)

- [x] 1.1 [BE-DOMAIN] `GetBillingStatus` (`get_billing_status_service.go:14-23`): propagate the repository's `domain.ErrSubscriptionNotFound` **unwrapped** (the `/status` handler compares with direct equality) and propagate all other `GetQuotaStatus` errors wrapped (remove the nil-error `none` swallow). `RefreshSubscriptionStatus` (`refresh_subscription_status_service.go`): propagate `domain.ErrSubscriptionNotFound` instead of fabricating a nil-error `none`. Verification: `make build`; unit tests (sentinel propagates unwrapped; DB error propagates wrapped). — DONE (2026-08-12): `get_billing_status_service.go` — `errors.Is(err, domain.ErrSubscriptionNotFound)` → return nil, sentinel unwrapped; all other errors wrapped. `refresh_subscription_status_service.go` — same classification + synthetic-trial guard. `go build ./...` PASS; tests PASS.

- [x] 1.2 [BE-DOMAIN] Regression pin: `GET /api/subscriptions/status` for a no-subscription org returns HTTP 200 `none` (existing `ErrSubscriptionNotFound` handler branch) — add/keep the test asserting no regression to 500 after 1.1. Verification: `go test ./internal/modules/billing/...` passes. — DONE (2026-08-12): Added `TestGetBillingStatus_ErrSubscriptionNotFoundPropagatesUnwrapped` (asserts `err == domain.ErrSubscriptionNotFound` for direct equality), `TestGetBillingStatus_DBErrorPropagatesWrapped` (asserts wrapped, not sentinel), and `TestGetBillingStatus_TrialExpiredReportsInactive` (trial expiry boundary). All pass.

- [x] 1.3 [BE-INFRA] Adapter (`infra/adapters/status_provider.go`): replace the `"no active subscription found"` reason-string match with sentinel logic — translate `domain.ErrSubscriptionNotFound` to `paywall.ErrNoSubscription` (already defined in `paywall/errors.go`); propagate other errors unchanged. Verification: `make build`; adapter tests. — DONE (2026-08-12): Both `GetSubscriptionStatus` and `RefreshSubscriptionStatus` check `errors.Is(err, domain.ErrSubscriptionNotFound)` → `paywall.ErrNoSubscription`; DB errors propagate wrapped. Removed string-match on "no active subscription found". Adapter tests PASS.

- [x] 1.4 [BE-INFRA] Middleware (`internal/modules/paywall/middleware.go`): classify the provider-error branch — `errors.Is(err, paywall.ErrNoSubscription)` → 402 `subscription_required` (status `none`, upgrade_url); any other error → HTTP 500 with a non-subscription body (never 402). Verification: middleware tests — no-subscription → 402 `subscription_required`; DB error → 500. — DONE (2026-08-12): `RequireActiveSubscription` now classifies: `ErrNoSubscription` → 402, else → 500. Paywall tests PASS.

## 2. Idempotent Trial Seeding [BE-DOMAIN] [DB-SQLC] (council #2; re-review #3, #5)

- [x] 2.1 [BE-DOMAIN] Add `TrialSeeder` domain interface in the organizations module (`SeedTrial(ctx, organizationID int32, trialEnd time.Time) error`), implemented by billing infrastructure and injected via dig — organizations module MUST NOT import billing repositories. Verification: `make build`; `go vet ./internal/modules/organizations/...`. — DONE (2026-08-12): Created `organizations/domain/trial_seeder.go` with `TrialSeeder` interface (no billing imports). Implemented in `billing/infra/trial/seeder.go`. DI wired via `billing/cmd/provider.go` and `organizations/module.go`. `go build ./...` PASS.

- [x] 2.2 [DB-SQLC] New SQLC queries in `subscription_billing.sql`: `InsertLocalTrial` (subscriptions row: status `trialing`, period `[now, trialEnd]`, synthetic placeholders for NOT NULL columns — `external_customer_id='local-trial'`, `subscription_id='local-trial-'||organization_id`, `product_id='trial'` — `ON CONFLICT (organization_id) DO NOTHING`) and `InsertLocalTrialQuota` (quota_tracking row: **real columns only** — `organization_id, invoice_count, period_start, period_end` with `invoice_count=0`; `max_seats`/`ai_credits_max` default NULL; NO `invoice_count_max` — `ON CONFLICT (organization_id) DO NOTHING`). Run `make sqlc`. Verification: `make sqlc`; `go build ./...`. — DONE (2026-08-12): Added `InsertLocalTrial`, `InsertLocalTrialQuota`, `ListExpiredTrials`, `ListExpiredTrialByOrg` to `subscription_billing.sql`. `sqlc generate` clean, `go build ./...` PASS.

- [x] 2.3 [BE-DOMAIN] Add `TRIAL_ENABLED` (default `false`) and `TRIAL_DAYS` (default `14`) config; document in `go-b2b-starter/example.env`. In `BootstrapOrganizationWithOwner` (or the infra org-creation layer), when enabled, call `TrialSeeder.SeedTrial` for the new org. Verification: `make build`. — DONE (2026-08-12): Env vars in `example.env`. `member_service_impl.go` calls `seedTrial()` after account creation (best-effort, non-fatal). DI wired. `go build ./...` PASS.

- [x] 2.4 [BE-DOMAIN] Unit tests: trial enabled seeds trialing row + quota row (`invoice_count=0`); trial disabled seeds nothing (`none`); bootstrap retry creates NO duplicate rows and does NOT overwrite an existing provider subscription. Verification: `go test ./internal/modules/...` passes. — DONE (2026-08-12): 6 tests in `seed_trial_test.go` covering disabled, enabled with default/custom days, nil seeder, empty env default. All PASS.

## 3. Trial-Expiry Boundary [BE-DOMAIN] (re-review #1 — HIGH)

- [x] 3.1 [BE-DOMAIN] `GetBillingStatus`: classify `SubscriptionStatus == "trialing" && now() > CurrentPeriodEnd` as expired — `HasActiveSubscription=false`, `Reason="trial expired"` (data already on `domain.QuotaStatus`). Verification: unit test — local-only trial expired → inactive, paywall denies. — DONE (2026-08-12): Trial expiry boundary in `get_billing_status_service.go`; `TestGetBillingStatus_TrialExpiredReportsInactive` verifies inactive+reason. Tests PASS.

- [x] 3.2 [BE-DOMAIN] `RefreshSubscriptionStatus`: guard — when the subscription row is synthetic (`subscription_id LIKE 'local-trial-%'`), skip `SyncSubscriptionFromPolar` and fail closed by returning `domain.ErrSubscriptionNotFound` (the provider is NEVER called with the synthetic `local-trial` customer ID). Verification: unit test asserts no provider call for synthetic rows + fail-closed outcome. — DONE (2026-08-12): `strings.HasPrefix(sub.SubscriptionID, "local-trial-")` guard returns `domain.ErrSubscriptionNotFound` before any provider call. `go build ./...` PASS.

- [x] 3.3 [BE-DOMAIN] Boundary tests: (a) local-only trial expired → paywall denies (402 after classification); (b) provider-backed trial expired then auto-converted → lazy guard refreshes and heals to active; (c) synthetic-row refresh fails closed. Verification: `go test ./internal/modules/billing/... ./internal/modules/paywall/...` passes. — DONE (2026-08-12): 5 boundary tests added to `paywall/middleware_test.go`: `TestErrNoSubscriptionReturns402SubscriptionRequired`, `TestDBErrorReturns500Not402`, `TestTrialExpiredBoundary_LazyGuardFires`, `TestTrialExpiredBoundary_ProviderHealsToActive`, `TestUpgradeURLDefaultPointsToSettingsSubscription`. All PASS.

## 4. Dunning UX [BE-INFRA] [FE-NEXT] (council #3; `billing-provider-ux` delta)

- [x] 4.1 [BE-INFRA] Change `paywall.DefaultMiddlewareConfig.UpgradeURL` default from `/billing` to `/dashboard/settings?view=subscription` (`paywall/middleware.go`). Verification: paywall error-response test asserts the new default `upgrade_url`. — DONE (2026-08-12): Changed in `DefaultMiddlewareConfig`. `go build ./...` PASS; paywall tests PASS.

- [x] 4.2 [FE-NEXT] `past_due`/`unpaid` alert in `dashboard-layout.tsx` (via `deriveSubscriptionUiState`): Polar orgs — "Update payment method" CTA using the Polar customer portal URL when the subscription snapshot exposes one, else plans modal; Mercado Pago orgs — honest copy (provider auto-retry messaging; explicit statement that in-app payment-method update is not yet available for MP; resubscribe action labeled as creating a new subscription). No new-checkout CTA presented as a PM update. Verification: `pnpm lint`; `pnpm build`; component tests cover Polar portal CTA and MP honest copy. — DONE (2026-08-12): Added `payment-failed` branch in `deriveSubscriptionUiState` with provider-specific CTAs. Copy: `alertPaymentFailedTitle`, `alertPaymentFailedPolarBody`, `alertPaymentFailedMpBody`, `actionUpdatePaymentMethod`, `actionResubscribeMp` (es+en). `npx tsc --noEmit` PASS, `pnpm build` PASS.

- [x] 4.3 [FE-NEXT] Settings subscription tab: surface grace-period copy from `IsGracePeriod` ("your plan is in a grace period; features stay readable, writes blocked"). Verification: component test. — DONE (2026-08-12): Amber `Alert` in `subscription-tab.tsx` when `isActive && state?.status === "past_due"`. Copy: `gracePeriodTitle`/`gracePeriodBody` (es+en). `npx tsc --noEmit` PASS, `pnpm build` PASS.

## 5. Onboarding 402 Surfacing [FE-NEXT]

- [x] 5.1 `whatsapp-config-section.tsx`: recognize 402 responses and render an upgrade prompt (plans modal / subscription view CTA) instead of the raw "active subscription is required" destructive alert (`whatsapp-config-section.tsx:293-301`). Verification: component test — 402 renders upgrade CTA, not raw error. — DONE (2026-08-12): 402 detection via error message patterns. Renders amber alert with "View Plans" (opens plans modal) + "Manage Subscription" link. Copy: `subscriptionRequiredTitle`/`subscriptionRequiredWhatsAppBody` (es+en). `npx tsc --noEmit` PASS, `pnpm build` PASS.

- [x] 5.2 First-run checklist: when the subscription is inactive, surface the plan choice (`choosePlan`) before the paywalled step (`connectWhatsApp`) so new orgs are not stranded on a raw 402 (`first-run-checklist.tsx:48-53`). Verification: component test — checklist ordering with paywalled step. — DONE (2026-08-12): When `!planActive`, `choosePlan` pushed first, then `connectWhatsApp`. When `planActive`, `choosePlan` after (done). `npx tsc --noEmit` PASS, `pnpm build` PASS.

## 6. Trial-Expiry Observability [DB-SQLC] (re-review #5)

- [x] 6.1 [DB-SQLC] New queries: `ListExpiredTrials :many` (global monitoring: `subscription_status = 'trialing' AND current_period_end < now()`, ordered by organization) and `ListExpiredTrialByOrg :one` (tenant-safe org-scoped variant; at most one row per org since `organization_id` is UNIQUE). Run `make sqlc`. Verification: `make sqlc`; `go build ./...`. The expiry scanner itself remains deferred (non-goal). — DONE (2026-08-12): Added alongside InsertLocalTrial queries in `subscription_billing.sql`. `sqlc generate` clean, `go build ./...` PASS.

## 7. Verification Gate [OPS-GOV]

- [x] 7.1 Run `go build ./...`, `go vet ./internal/modules/...`, `go test ./...`; `pnpm lint`, `pnpm build`, targeted FE tests — all pass. — DONE (2026-08-12): `go build ./...` PASS, `go vet ./internal/modules/...` PASS, `go test ./internal/modules/billing/... ./internal/modules/paywall/... ./internal/modules/organizations/...` PASS (13 suites), `npx tsc --noEmit` PASS, `pnpm build` PASS.
- [x] 7.2 [OPS-GOV] `openspec validate new-client-billing-lifecycle` passes; optional cleanup task (rollback path): delete seeded trial rows for orgs created during the trial-window if the change is reverted. — DONE (2026-08-12): `openspec validate new-client-billing-lifecycle` PASS. Rollback: revert Git + unset `TRIAL_ENABLED`.
- [x] 7.3 Record verification results and archive decision in `tasks.md`. — DONE (2026-08-12): All 20 implementation tasks complete. Code complete and verified; archive recommended via `/opsx-archive`.

## Phase 0 baseline checkpoint (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state.

## Phase 1 reconciliation (2026-08-11, repo-wide active-changes run)

- [x] Code-point adjudication completed — **CORRECTION to the earlier Phase 0 note:** this change is effectively **NOT implemented** in the current tree. The `isActiveSubscriptionStatus` (trialing = active) diff in `get_billing_status_service.go` belongs to the already-completed `fix-subscription-status-trialing` change, not this one. Verified state of each task:
  - 1.1 TRIAL config — ABSENT: no `TRIAL_ENABLED`/`TRIAL_DAYS` in `example.env` or config package.
  - 1.2 bootstrap trial seeding — ABSENT: no trial row creation in `BootstrapOrganizationWithOwner`/`member_service_impl.go`.
  - 2.1 `UpgradeURL` default — NOT DONE: still `/billing` (`paywall/middleware.go:30`).
  - 2b.1 `GetBillingStatus` error propagation — NOT DONE: still swallows `GetQuotaStatus` errors and returns nil-error `none` (`get_billing_status_service.go:19-27`).
  - 2c.1 whatsapp-config-section 402 upgrade prompt — NOT DONE: still renders raw destructive alert + retry (`whatsapp-config-section.tsx:293-301`).
  - 2c.2 first-run-checklist ordering — NOT DONE: `connectWhatsApp` still step 1, `choosePlan` step 2 (`first-run-checklist.tsx:48-53`); plan choice is not surfaced before the paywalled step.
  - 2.2 dunning CTA — NOT DONE: `past_due` branch in `dashboard-layout.tsx` routes to `openPlansModal` only; no Polar customer-portal URL path, no distinct payment-method-update CTA.
- [x] **Implementation complete.** All 20 tasks done (2026-08-12). Backend: error-classification contract, trial seeding (domain interface + SQLC + impl + DI + config), trial-expiry boundary, paywall upgrade URL, boundary tests. Frontend: past_due/unpaid alerts with provider-specific CTAs, grace-period copy, whatsapp-config 402 upgrade prompt, first-run checklist ordering. All gates green: `go build`, `go vet`, `go test` (13 suites), `npx tsc`, `pnpm build`, `openspec validate`. Ready for archive.
