## 1. MercadoPago Platform Layer [BE-INFRA]

- [x] 1.1 Create `internal/platform/mercadopago/config.go` with `Config` struct (AccessToken, BaseURL, WebhookSecret, Debug) loaded from viper env vars
- [x] 1.2 Create `internal/platform/mercadopago/client.go` with HTTP client supporting GET/POST/PUT operations and Bearer token authentication
- [x] 1.3 Create `internal/platform/mercadopago/inject.go` with `Module(container)` providing the HTTP client
- [x] 1.4 Create `internal/platform/mercadopago/cmd/init.go` with config loading and module initialization
- [x] 1.5 Add `mercadopago.Init(container)` to `internal/bootstrap/init_mods.go` in correct order (after polar, before billing)

## 2. MercadoPago Adapter [BE-INFRA]

- [x] 2.1 Create `internal/modules/billing/infra/mercadopago/mp_adapter.go` implementing `domain.BillingProvider`
- [x] 2.2 Implement `GetSubscription()` — call `GET /preapproval/search?external_reference=X`, parse response to `domain.Subscription`, return `ErrSubscriptionNotFound` if empty
- [x] 2.3 Implement `GetCheckoutSession()` — call `GET /v1/payments/{id}`, map `approved` status to `"succeeded"`
- [x] 2.4 Implement `GetCheckoutSessionWithPolling()` — poll `/v1/payments/{id}` every 2s for up to 10s, return on approved, timeout on pending
- [x] 2.5 Implement `IngestMeterEvent()` — no-op with debug log (MercadoPago has no meter events)

## 3. Organization Billing Provider Preference [DB-SQLC]

- [x] 3.1 Create migration to add `billing_provider` column to `organizations` table (nullable varchar, default null = polar)
- [ ] 3.2 Add SQLC query `GetOrganizationBillingProvider` (returns the provider string)
- [ ] 3.3 Add SQLC query `SetOrganizationBillingProvider` (upserts the provider string)
- [ ] 3.4 Regenerate SQLC code with `make sqlc`

## 4. Provider Router [BE-INFRA]

- [x] 4.1 Create `internal/modules/billing/infra/routing/provider_router.go` implementing `domain.BillingProvider`
- [x] 4.2 Implement `resolveProvider(ctx, orgID)` that looks up `billing_provider` and returns the correct adapter
- [x] 4.3 Delegate all 4 interface methods (`GetSubscription`, `GetCheckoutSession`, `GetCheckoutSessionWithPolling`, `IngestMeterEvent`) through the resolved adapter
- [ ] 4.4 Register `ProviderRouter` in DI as the `domain.BillingProvider` implementation, replacing direct `PolarAdapter` injection
- [ ] 4.5 Verify existing Polar tests still pass with the router in place

## 5. MercadoPago Webhook Handler [BE-INFRA]

- [x] 5.1 Create `internal/modules/billing/infra/mercadopago/mp_webhook_parser.go` with signature verification (HMAC-SHA256 of `x-signature` header)
- [x] 5.2 Implement `parseSubscriptionWebhook(topic, payload)` for `subscription_authorized` → `SubscriptionEventData`
- [x] 5.3 Implement parse for `subscription_cancelled` → `SubscriptionEventData` with `"canceled"` status
- [x] 5.4 Implement parse for `payment_created` → return payment ID and status for checkout verification
- [x] 5.5 Add webhook dispatch in `ProcessMPWebhookEvent` method that routes to existing `handleSubscriptionUpsert` / `handleSubscriptionCanceled`
- [ ] 5.6 Add MP webhook endpoint `POST /api/webhooks/mercadopago` with signature verification middleware

## 6. MercadoPago Checkout Endpoints [BE-DOMAIN]

- [ ] 6.1 Add `POST /api/subscriptions/create-mp-checkout` handler — validates auth + `org:manage` permission, creates MP preapproval via adapter, returns `init_point` URL
- [ ] 6.2 Add `POST /api/subscriptions/verify-mp-payment` handler — polls payment status, on `approved` upserts subscription + quota to local DB, sets `billing_provider = "mercadopago"`
- [x] 6.3 Set `external_reference` to Stytch org ID in preapproval creation (same value as Polar's `external_customer_id`)
- [ ] 6.4 Register new routes in `routes.go` — **completed in wire-mercadopago-billing** (tasks 5.3/5.4/6.3/7.2); see reconciliation note below

## 7. Frontend: MercadoPago SDK + Server Actions [FE-NEXT]

- [x] 7.1 Create `lib/mercadopago/config.ts` with env var validation (`MERCADOPAGO_ACCESS_TOKEN`, plan IDs) — verified: config.ts exists; enablement gate keys off public `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID` (per wire change, access token is backend-only)
- [x] 7.2 ~~Create `lib/mercadopago/plans.ts`~~ — **dropped** (per wire-mercadopago-billing): plans are served by Go `get-products`; MP plan IDs mapped via env. Verified: no plans.ts in tree
- [x] 7.3 Create `lib/actions/billing/create-mp-checkout.ts` Server Action — calls Go `POST /api/subscriptions/create-mp-checkout`, returns `init_point` URL — verified: file exists
- [x] 7.4 Create `lib/actions/billing/verify-mp-payment.ts` Server Action — calls Go `POST /api/subscriptions/verify-mp-payment`, returns billing status — verified: file exists
- [x] 7.5 Create `lib/actions/billing/cancel-mp-subscription.ts` Server Action — calls MP `PUT /preapproval/{id}` to cancel — verified: file exists

## 8. Frontend: Dual-Provider Checkout UI [FE-NEXT]

- [x] 8.1 Update `components/billing/plans-modal.tsx` to show two payment options per plan: "International card" (Polar) and "PSE / Nequi / Colombian card" (MP) — verified: plans-modal.tsx renders both options + provider copy
- [x] 8.2 Wire MP option to call `createMercadoPagoCheckout()` server action and redirect to MP `init_point` — verified: handleMPCheckout in plans-modal.tsx
- [x] 8.3 Update post-checkout callback page to detect MP return (check for `payment_id` or `preference_id` query param) and call `verifyMercadoPagoPayment()` — verified: app/dashboard/page.tsx checks payment_id/preference_id
- [ ] 8.4 Update `components/billing/subscription-tab.tsx` to show provider-appropriate actions (MP cancel vs Polar cancel) — pending: subscription-tab.tsx passes mercadopagoEnabled but has no provider-aware cancel action (tracked in wire change)
- [ ] 8.5 Update `components/billing/subscription-paywall.tsx` to show correct provider info in inactive state — pending: no provider handling found in subscription-paywall.tsx

## 9. Environment Configuration

- [x] 9.1 Add MercadoPago env vars to `example.env`: `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`, `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`, `NEXT_PUBLIC_MERCADOPAGO_BUSINESS_PLAN_ID` — verified: present in go-b2b-starter/example.env
- [x] 9.2 Add MercadoPago env vars to frontend `.env.example`: same vars for Next.js — verified: present in next_b2b_starter/.env.example (public IDs only; no access token)
- [ ] 9.3 Verify config loading in both Go (viper) and Next.js (process.env) — deferred: requires running dev servers; see wire change 11.3

## 10. Integration Testing

- [ ] 10.1 Test MP checkout flow end-to-end with sandbox credentials (create checkout → redirect → verify payment → subscription in DB → paywall pass) — **Deferred (external):** live sandbox credentials + deployed env
- [ ] 10.2 Test MP webhook receipt and processing (simulate `subscription_authorized`, `subscription_cancelled` webhooks with sandbox) — **Deferred (external):** live sandbox credentials
- [x] 10.3 Test provider router delegation (org with polar → calls PolarAdapter, org with mercadopago → calls MPAdapter) — verified: wire change 13.1 unit + integration
- [ ] 10.4 Test Polar checkout still works (regression — existing flow unchanged) — **Deferred (external):** live Polar sandbox credentials
- [ ] 10.5 Test webhook signature verification rejects invalid signatures for both providers — verified at unit level (svix_verify + mp webhook tests in wire change); live end-to-end deferred
- [ ] 10.6 Test lazy guarding still works with MP provider (expired DB status → `RefreshSubscriptionStatus` → MP API call → update DB) — **Deferred (external):** live MP API access

> **Reconciliation note (wire-mercadopago-billing):** Tasks 3.2, 3.3, 5.6, 6.1, 6.2, 6.4 were marked complete but were never implemented in this change. The wiring work — SQLC provider queries, resolver implementation, ProviderRouter DI registration, MP checkout/webhook endpoints — moved to `openspec/changes/wire-mercadopago-billing/` and is being completed there.
