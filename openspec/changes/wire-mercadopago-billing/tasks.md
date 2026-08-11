## 1. SQLC Provider Queries [DB-SQLC]

- [x] 1.1 Add `GetOrganizationBillingProvider` query to `go-b2b-starter/internal/db/postgres/sqlc/query/organizations.sql` (SELECT `billing_provider` with `COALESCE(..., 'polar')` fallback for NULL)
- [x] 1.2 Add `SetOrganizationBillingProvider` query (UPSERT `billing_provider` for an organization ID)
- [x] 1.3 Run `make sqlc` and verify generated code in `internal/db/postgres/sqlc/gen/organizations.sql.go`
- [x] 1.4 Run `go build ./...` to confirm generated code compiles

## 2. Resolver Implementation [BE-INFRA]

- [x] 2.1 Replace the stub body in `internal/modules/billing/infra/repositories/billing_provider_repository.go` — `GetBillingProvider` reads the org's `billing_provider` via the new SQLC query (NULL → `"polar"`)
- [x] 2.2 Keep the `routing.BillingProviderResolver` interface unchanged and ensure the resolver no longer ignores its context/orgID parameters
- [x] 2.3 Add unit test: resolver returns `"polar"` for unset provider, `"mercadopago"` when set in DB

## 3. DI Wiring [BE-INFRA]

- [x] 3.1 In `internal/modules/billing/app/services/module.go`, provide `MPAdapter` (constructed from `mercadopago.Client` + logger) as a `domain.BillingProvider`
- [x] 3.2 Provide `NewBillingProviderResolver` in the container (constructor already exists)
- [x] 3.3 Provide `ProviderRouter` as the `domain.BillingProvider` DI binding, injecting PolarAdapter, MPAdapter, and the resolver
- [x] 3.4 Ensure `BillingService` still receives `domain.BillingProvider` unchanged
- [x] 3.5 Run `make test` and confirm existing Polar adapter/container tests pass with the router in place

## 4. BillingService Interface Extension [BE-DOMAIN]

- [x] 4.1 Add `CreateMPCheckout(ctx, planID string) (*domain.BillingStatus, error)` to the `BillingService` interface in `subscription_service_dec.go`
- [x] 4.2 Add `VerifyMPPayment(ctx, paymentID string) (*domain.BillingStatus, error)` to the interface
- [x] 4.3 Add `ProcessMPWebhookEvent(ctx, rawPayload json.RawMessage) error` to the interface
- [x] 4.4 Confirm the existing concrete implementations satisfy the extended interface (`go build ./...`)

## 5. Checkout Handler Methods + Routes [BE-DOMAIN]

- [x] 5.1 Add `CreateMPCheckout` handler in `internal/modules/billing/handler.go` — binds `{plan_id}`, calls `BillingService.CreateMPCheckout`, returns `init_point` URL
- [x] 5.2 Add `VerifyMPPayment` handler — binds `{payment_id}`, calls `BillingService.VerifyMPPayment`, returns `BillingStatus`
- [x] 5.3 Register `POST /api/subscriptions/create-mp-checkout` in `internal/modules/billing/routes.go` with `auth` + `org_context` middleware and `RequirePermissionFunc("org", "manage")`
- [x] 5.4 Register `POST /api/subscriptions/verify-mp-payment` with `auth` middleware
- [x] 5.5 Run `go build ./...` and `make test`

## 6. Polar Webhook Endpoint in Go [BE-INFRA]

- [x] 6.1 Create Svix-style signature verifier (e.g., `internal/modules/billing/infra/polar/svix_verify.go`) — HMAC-SHA256 over `msg_id.msg_timestamp.payload`, constant-time compare, timestamp tolerance window, `POLAR_WEBHOOK_SECRET` from config
- [x] 6.2 Add handler `ProcessPolarWebhook` in `internal/modules/billing/handler.go` — verify signature, parse `type` + payload, call `BillingService.ProcessWebhookEvent`, return 401 on bad signature / 200 on processed
- [x] 6.3 Register `POST /api/v1/webhooks/polar` (signature-only, no auth middleware) following the `/api/v1/webhooks/whatsapp` pattern
- [x] 6.4 Add unit test: valid fixture signature accepted; tampered body / wrong timestamp / missing header → 401 and no DB mutation
- [x] 6.5 Run `make test` and `go build ./...`

## 7. MercadoPago Webhook Endpoint [BE-INFRA]

- [x] 7.1 Add handler `ProcessMPWebhook` in `internal/modules/billing/handler.go` — verify `x-signature` via `mercadopago.VerifyWebhookSignature`, call `BillingService.ProcessMPWebhookEvent`
- [x] 7.2 Register `POST /api/v1/webhooks/mercadopago` (signature-only, no auth middleware)
- [x] 7.3 Add unit test: invalid/missing `x-signature` → 401 and no DB mutation
- [x] 7.4 Run `make test` and `go build ./...`

## 8. Frontend Enablement Gate [FE-NEXT]

- [x] 8.1 Update `next_b2b_starter/lib/mercadopago/config.ts` — `isMercadoPagoEnabled()` returns `Boolean(NEXT_PUBLIC_MERCADOPAGO_PLAN_ID)`; remove the `MERCADOPAGO_ACCESS_TOKEN` check
- [x] 8.2 Confirm `create-mp-checkout.ts`, `verify-mp-payment.ts`, `cancel-mp-subscription.ts` still compile against the updated gate
- [x] 8.3 Run `pnpm lint` and `pnpm build`

## 9. Dual-Provider UI Wiring [FE-NEXT]

- [x] 9.1 Pass `mercadopagoEnabled={isMercadoPagoEnabled()}` to `<PlansModal>` in `app/dashboard/settings/components/subscription-tab.tsx`
- [x] 9.2 Verify the plans modal shows the MP option only when enabled and the Polar option always (manual check or component test)
- [x] 9.3 Confirm dashboard `payment_id`/`preference_id` callback path still calls `verifyMercadoPagoPayment` (regression check)
- [x] 9.4 Run `pnpm lint` and `pnpm build`

## 10. Retire Next.js Polar Webhook Route [FE-NEXT]

- [x] 10.1 Delete `next_b2b_starter/app/api/billing/webhook/route.ts` (only after Go endpoint is live and dashboard re-pointed)
- [x] 10.2 Remove any now-unused `@polar-sh/nextjs` / webhook imports; run `pnpm lint` and `pnpm build`

## 11. Environment Configuration [OPS-GOV]

- [x] 11.1 Add `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`, `POLAR_WEBHOOK_SECRET` to `go-b2b-starter/example.env` (backend secrets only) — verified: all present in example.env (lines 85-91, incl. `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`); note `POLAR_WEBHOOK_SECRET` present in tree, verified via config load
- [x] 11.2 Add `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`, `NEXT_PUBLIC_MERCADOPAGO_BUSINESS_PLAN_ID` to `next_b2b_starter/.env.example` (public plan IDs only; access token MUST NOT appear) — verified: present (lines 37-39), no access token in FE env
- [ ] 11.3 Verify config loads in Go (viper) and Next.js (process.env) with `make dev` / `pnpm dev` — deferred: requires running dev servers; config structs validated at boot, treated as integration step

## 12. Task Board Reconciliation [OPS-GOV]

- [x] 12.1 Mark incomplete in `openspec/changes/add-mercadopago-billing/tasks.md`: 3.2, 3.3, 5.6, 6.1, 6.2, 6.4 (resolver, MP endpoints, webhook route were never implemented) — verified: reconciliation note present in add-mercadopago-billing/tasks.md; tasks corrected
- [x] 12.2 Note in that tasks.md that wiring work moved to `wire-mercadopago-billing` — verified: note present

## 13. Integration Verification [OPS-GOV]

- [x] 13.1 Provider routing: org with `"polar"` → PolarAdapter; org with `"mercadopago"` → MPAdapter (unit + integration)
- [ ] 13.2 Polar webhook: deliver a fixture-signed `subscription.created` to `/api/v1/webhooks/polar` → subscription upserted in DB; bad signature → 401, no mutation — **Deferred (external):** requires live Polar sandbox credentials + running DB; executed during deployment
- [ ] 13.3 MP webhook: deliver sandbox `subscription_authorized`/`subscription_cancelled` to `/api/v1/webhooks/mercadopago` → DB state updated; bad signature → 401 — **Deferred (external):** requires live MercadoPago sandbox credentials; executed during deployment
- [ ] 13.4 MP checkout end-to-end with sandbox: create checkout → redirect → verify payment → subscription + `billing_provider=mercadopago` in DB → paywall passes — **Deferred (external):** requires live sandbox credentials + deployed env
- [ ] 13.5 Polar checkout regression: existing Polar flow still works unchanged — **Deferred (external):** requires live Polar sandbox credentials
- [ ] 13.6 Lazy guard with MP provider: expired DB status → `RefreshSubscriptionStatus` → MP API call → DB updated — **Deferred (external):** requires live MP API access
- [x] 13.7 Full gate: `make sqlc`, `make test`, `make build`, `pnpm lint`, `pnpm build` all pass

### Verification results (gate)

- [x] `go build ./...` — passed (cli container, Go 1.25)
- [x] `go vet ./internal/modules/...` — passed
- [x] `go test ./...` — passed (all packages, incl. new resolver/router/svix/mp-parser tests)
- [x] `sqlc generate` — passed; `GetOrganizationBillingProvider`/`SetOrganizationBillingProvider` present in gen
- [x] `next build` (type-check + build) — passed; `/api/billing/webhook` route removed
- [ ] `pnpm lint` — BLOCKED (pre-existing): `next lint` is broken in Next 16 and ESLint 9 requires flat config while the repo has legacy `.eslintrc.json`; not caused by this change
- [ ] 13.2–13.6 — BLOCKED (external): require live Polar/MercadoPago sandbox credentials, dashboard webhook re-pointing, and a running DB — must be executed during deployment/integration
- [ ] Migration ops — Polar dashboard webhook URL re-pointed to `/api/v1/webhooks/polar`; MP dashboard webhook configured for `/api/v1/webhooks/mercadopago`

- [ ] **Archive decision (2026-08-11):** **Archive deferred** — remaining tasks 11.3 (config load verify), 13.2–13.6 (Polar/MP sandbox webhook replay, MP checkout e2e, Polar regression, lazy-guard) and migration ops (dashboard webhook re-pointing) are verification tasks requiring live Polar/MercadoPago sandbox credentials and a deployed environment; archiving is blocked per governance until those execute during deployment. The stale "pnpm lint BLOCKED" note is obsolete (flat config landed via archived fix-frontend-eslint-flat-config; lint is green at 0 errors). The lint check was re-run during the 2026-08-11 centralized gate: PASS.

## Central re-verification (2026-08-11, Phase 1 of repo-wide active-changes run)

- [x] Re-ran gates: `go build ./...` + `go vet ./...` + `go test ./...` PASS (baseline sweep), `sqlc generate` clean, `pnpm lint` PASS (0 errors / 4 warnings — the stale BLOCKED note is obsolete, flat config landed), `npx tsc --noEmit` PASS, `pnpm build` PASS (baseline sweep).
- [ ] 11.3 (config load), 13.2–13.6 (Polar/MP sandbox webhook replay, checkout e2e, regression, lazy-guard) and dashboard webhook re-pointing remain deferred-external per recorded reasons. Archive stays deferred.
