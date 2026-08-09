## Why

The `add-mercadopago-billing` change built MercadoPago as a second billing provider but left every integration seam unwired: the provider router is never registered in DI, the org→provider resolver is a hardcoded stub returning `"polar"`, the MercadoPago checkout/webhook services are unreachable (no interface methods, no HTTP routes), the Polar webhook receiver is a signature-verifying no-op that persists nothing, and the frontend enablement gate keys off a backend secret while the MP checkout option is never surfaced in the UI. The result is a large volume of dead code that cannot run, and a billing system whose only real synchronization path is polling.

## What Changes

- **Resolver un-stubbed**: SQLC queries `GetOrganizationBillingProvider` / `SetOrganizationBillingProvider` added; the `billingProviderResolver` implementation reads `organizations.billing_provider` (NULL → `"polar"`) instead of returning a constant
- **DI wiring**: `MPAdapter` (MercadoPago platform client) and the resolver are provided in the container; `ProviderRouter` is registered as the `domain.BillingProvider` implementation, delegating per-org to PolarAdapter or MPAdapter
- **Interface extension**: `CreateMPCheckout`, `VerifyMPPayment`, `ProcessMPWebhookEvent` added to the `BillingService` interface (already implemented on the concrete service, previously unreachable)
- **Checkout endpoints**: `POST /api/subscriptions/create-mp-checkout` (auth + `org:manage` permission) and `POST /api/subscriptions/verify-mp-payment` (auth) with handler methods, mirroring the existing Polar verify pattern
- **Webhook ingress moved to Go (per-provider)**: `POST /api/v1/webhooks/polar` (Svix-style HMAC-SHA256 signature verification, `webhook-id`/`webhook-timestamp`/`webhook-signature` headers) → existing `ProcessWebhookEvent`; `POST /api/v1/webhooks/mercadopago` (existing HMAC `VerifyWebhookSignature`) → existing `ProcessMPWebhookEvent`. The Next.js `/api/billing/webhook` route (no-op handler) is retired. Polar dashboard webhook URL is re-pointed at the Go backend (ops step)
- **Frontend enablement fixed**: `isMercadoPagoEnabled()` gates on public `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID` instead of the backend secret; `mercadopagoEnabled` prop wired through `subscription-tab.tsx` so the MP checkout option is actually visible; `lib/mercadopago/plans.ts` is dropped (plans already served by Go `get-products`; MP plans mapped via env plan IDs)
- **Environment split corrected**: server secrets (`MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`, `POLAR_WEBHOOK_SECRET`) added to backend `example.env` only; public `NEXT_PUBLIC_MERCADOPAGO_*` plan IDs added to frontend `.env.example` only — the access token is never exposed to the frontend
- **Task board reconciliation**: `add-mercadopago-billing/tasks.md` checkboxes corrected to match reality (3.2, 3.3, 5.6, 6.1, 6.2, 6.4 marked incomplete)

## Capabilities

### New Capabilities

- `mercadopago-webhooks`: Go webhook ingress for MercadoPago IPN and Polar events — per-provider endpoints under `/api/v1/webhooks/`, signature verification for both schemes (HMAC `x-signature` for MP, Svix-style HMAC for Polar), and dispatch to the existing `ProcessMPWebhookEvent` / `ProcessWebhookEvent` services

### Modified Capabilities

- `billing-provider-routing` (delta of `add-mercadopago-billing`): requirement changed from "resolver exists" to "resolver reads `organizations.billing_provider` from the local DB (NULL defaults to `polar`)"; `ProviderRouter` registered as the `domain.BillingProvider` DI binding
- `mercadopago-checkout` (delta of `add-mercadopago-billing`): checkout endpoints become reachable (routes + handler methods), `BillingService` exposes the MP methods, and frontend enablement is keyed to public plan-ID config with the provider option surfaced in the plans modal

## Impact

- **Go backend**: SQLC queries in `query/organizations.sql` + regenerated code; `billing_provider_repository.go` real implementation; `app/services/module.go` DI registration (MPAdapter + resolver + ProviderRouter); `BillingService` interface in `subscription_service_dec.go`; new handler methods and route registrations in `internal/modules/billing`; new Polar signature verifier and webhook endpoints under `internal/modules/billing` (mirroring the existing `/api/v1/webhooks/whatsapp` pattern). Existing Polar adapter, services, and paywall middleware untouched
- **Frontend**: `lib/mercadopago/config.ts` gate fix; `subscription-tab.tsx` prop wiring; `app/api/billing/webhook/route.ts` retired. Existing server actions and `dashboard/page.tsx` callback unchanged
- **Database**: no schema change — `billing_provider` column already exists (migration 000015); only new SQLC query models
- **Dependencies**: none new (Go `crypto/hmac` stdlib for Svix verification)
- **Config**: backend `example.env` gains `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`, `POLAR_WEBHOOK_SECRET`; frontend `.env.example` gains `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`, `NEXT_PUBLIC_MERCADOPAGO_BUSINESS_PLAN_ID`. `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` retained
- **Auth**: no changes. Checkout endpoints reuse existing `auth` middleware + `RequirePermissionFunc("org", "manage")`; webhook endpoints are signature-only (no session)
- **Ops**: Polar dashboard webhook URL re-pointed from the Next.js app to the Go backend (`/api/v1/webhooks/polar`); MP dashboard webhook configured for `/api/v1/webhooks/mercadopago`
- **Rollback**: Git — revert the change (routes, DI, interface). Webhooks — re-point Polar dashboard URL back to the Next.js route. Stytch tenant policy state is unaffected (no auth changes); no local credentials are introduced anywhere
- **Non-Goals**: Not replacing Polar.sh; not building a recurring billing engine; no new payment methods beyond MP/Polar; not implementing `lib/mercadopago/plans.ts` (plans served by Go); not changing the lazy-guard/verify-payment synchronization model for Polar

## Assumptions

- **Polar dashboard webhook destination**: the Polar dashboard currently delivers webhooks to the Next.js app at `/api/billing/webhook`. Re-pointing to the Go backend is an external ops step performed after the Go endpoint deploys; the exact current destination URL cannot be verified from this repository
- **Svix signature scheme**: Polar's `validateEvent` (from `@polar-sh/sdk/webhooks`) verifies the `webhook-id` / `webhook-timestamp` / `webhook-signature` headers using the Svix HMAC-SHA256 scheme (`whsec_` secret, `msg_id.msg_timestamp.payload` signing input, constant-time comparison, timestamp tolerance). Verified against the installed `@polar-sh/nextjs` dist code (headers + `validateEvent`), but the precise secret-key prefix and tolerance window follow Svix conventions that must be confirmed against the live Polar account during integration testing
- **MP plan IDs**: `preapproval_plan` records exist in the MercadoPago dashboard for the plan IDs referenced by `NEXT_PUBLIC_MERCADOPAGO_*` env vars — created externally, not verifiable from this repo
