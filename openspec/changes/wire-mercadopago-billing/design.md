## Context

The `add-mercadopago-billing` change built the MercadoPago components (platform client, `MPAdapter`, webhook parser, MP checkout/webhook services, frontend server actions, plans-modal MP UI) but left every integration seam unwired. Verified current state:

- `ProviderRouter` exists in `infra/routing/provider_router.go` but is never registered in DI — `module.go` still provides `PolarAdapter` directly as `domain.BillingProvider`
- `billingProviderResolver.GetBillingProvider` (`infra/repositories/billing_provider_repository.go:18`) is a stub returning the constant `"polar"`; no SQLC query reads `organizations.billing_provider`
- `CreateMPCheckout`, `VerifyMPPayment`, `ProcessMPWebhookEvent` are implemented on the concrete `billingService` but absent from the `BillingService` interface (unreachable via DI)
- `billing/routes.go` registers only `GET /subscriptions/status` and `POST /subscriptions/verify-payment`; no MP endpoints, no webhook routes
- Polar webhooks arrive at Next.js `/api/billing/webhook` where `@polar-sh/nextjs` verifies the Svix-style signature, but the handler is a no-op TODO — nothing persists
- `isMercadoPagoEnabled()` gates on `MERCADOPAGO_ACCESS_TOKEN` (a backend secret) and `<PlansModal>` is rendered without the `mercadopagoEnabled` prop, so the MP option is invisible
- Precedent for Go webhook ingress exists: `/api/v1/webhooks/whatsapp` in the whatsapp module

The local DB is the system of record for subscription state; providers are payment rails. Paywall/feature gates read local DB only.

## Goals / Non-Goals

**Goals:**
- Make the MercadoPago provider actually reachable end-to-end: checkout endpoints, webhook ingress, provider routing
- Move webhook persistence into Go for both providers, with signature verification for each scheme
- Fix the frontend enablement gate so the MP checkout option is visible and keyed to public config
- Correct the `add-mercadopago-billing` task board to reflect reality
- Preserve all Polar.sh functionality and the lazy-guard/verify-payment synchronization model

**Non-Goals:**
- Replacing Polar.sh or altering its adapter/service code
- Building a recurring billing engine or adding new payment methods
- Changing the local subscription schema or paywall middleware
- Implementing `lib/mercadopago/plans.ts` — plans are already served by Go via `get-products`; MP plans are mapped by env plan IDs
- Replicating Stytch session state or credentials anywhere in the local DB (webhook endpoints are signature-only, no sessions)

## Decisions

### 1. Per-provider Go webhook endpoints over a single dispatch endpoint

**Chosen:** Two endpoints, matching the existing whatsapp precedent:

```
POST /api/v1/webhooks/polar         → Svix-style HMAC verify → ProcessWebhookEvent
POST /api/v1/webhooks/mercadopago   → HMAC x-signature verify → ProcessMPWebhookEvent
```

**Alternatives considered:**
- *Single Go endpoint with type dispatch*: chosen in `add-mercadopago-billing`'s design but contradicts both the repo precedent (whatsapp per-provider) and that change's own tasks.md (`POST /api/webhooks/mercadopago`). Rejected — per-provider endpoints keep each provider's verification self-contained and match the codebase pattern.
- *Keep Next.js as receiver/bridge*: Polar signature verification stays in `@polar-sh/nextjs` and forwards to Go. Rejected — keeps a dead-hop, duplicates secret config, and leaves a frontend route in the critical billing path.

Polar's signature scheme is Svix-style (verified in `@polar-sh/nextjs` dist code): headers `webhook-id`, `webhook-timestamp`, `webhook-signature`; signing input `msg_id.msg_timestamp.payload`; HMAC-SHA256 with the `whsec_` secret; constant-time comparison; timestamp tolerance. Reimplemented in Go with `crypto/hmac` (stdlib).

### 2. Resolver reads the DB; NULL means "polar"

**Chosen:** Two SQLC queries — `GetOrganizationBillingProvider` (returns the string; NULL → "polar" via `COALESCE`) and `SetOrganizationBillingProvider` (upsert). The existing `billingProviderResolver` implements `routing.BillingProviderResolver` against these queries instead of returning a constant.

**Alternatives considered:**
- *Keep the stub, default everywhere*: Rejected — the router could never route an org to MercadoPago, making the entire provider abstraction moot.

### 3. DI: ProviderRouter becomes the `domain.BillingProvider` binding

**Chosen:** In `app/services/module.go`, provide (in order): PolarAdapter → MPAdapter (new, built on the `platform/mercadopago` client) → resolver (already constructed, now real) → `ProviderRouter` as `domain.BillingProvider`. `NewBillingService` continues to receive `domain.BillingProvider` unchanged.

```
container
 ├─ polarpkg.Client ──────────────► PolarAdapter ──────────┐
 ├─ mercadopago.Client ───────────► MPAdapter ─────────────┤──► ProviderRouter ──► BillingService
 └─ sqlc.Store ───────────────────► billingProviderResolver┘        (domain.BillingProvider)
```

**Alternatives considered:**
- *Wire routing inside BillingService*: Rejected — couples the service to provider names; the router already implements the interface.

### 4. Extend `BillingService` with the MP methods already implemented

**Chosen:** Add `CreateMPCheckout(ctx, planID)`, `VerifyMPPayment(ctx, paymentID)`, `ProcessMPWebhookEvent(ctx, rawPayload)` to the interface in `subscription_service_dec.go`. The concrete implementations already exist; this only makes them reachable through DI.

### 5. Checkout routes mirror the Polar pattern

**Chosen:** `POST /api/subscriptions/create-mp-checkout` (auth middleware + `RequirePermissionFunc("org", "manage")`) and `POST /api/subscriptions/verify-mp-payment` (auth) — registered alongside the existing `/subscriptions/*` group. `external_reference` on the preapproval is the Stytch org ID (same as Polar's `external_customer_id`), resolved by the existing `OrganizationAdapter`.

**Alternatives considered:**
- *Verify-payment merged into the existing endpoint*: Rejected — the Polar endpoint is session-ID based and unchanged; MP polls `/v1/payments/{id}`, a different contract.

### 6. Frontend gate keys on public plan config; prop wired through

**Chosen:** `isMercadoPagoEnabled()` returns `Boolean(NEXT_PUBLIC_MERCADOPAGO_PLAN_ID)`. `subscription-tab.tsx` passes `mercadopagoEnabled` to `<PlansModal>`, surfacing the MP option (PSE / Nequi / Colombian card) alongside Polar (international card). The access token never appears in frontend env.

**Alternatives considered:**
- *Gate on the access token*: Rejected — forces the Go secret into the Next.js server env and conflates "backend has credentials" with "MP should be offered". The secret belongs to the Go backend only.

### 7. Env split: secrets backend-only, plan IDs public

**Chosen:**

| Env var | Where |
|---|---|
| `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL`, `MERCADOPAGO_WEBHOOK_SECRET`, `POLAR_WEBHOOK_SECRET` | Go `example.env` only |
| `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`, `NEXT_PUBLIC_MERCADOPAGO_BUSINESS_PLAN_ID`, `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` | Next.js `.env.example` only |

### 8. Retire the Next.js Polar route after Go endpoint is live

**Chosen:** Delete `app/api/billing/webhook/route.ts` (and its dependencies) once the Go endpoint deploys and the Polar dashboard URL is re-pointed. The ops ordering (deploy Go → re-point dashboard → remove Next.js route) avoids a window with no webhook delivery.

## Risks / Trade-offs

- **[Risk] Polar dashboard re-pointing window**: webhooks could be lost between Go deploy and dashboard URL change → Mitigation: deploy Go endpoint first, verify with a manual test webhook, then re-point; Next.js route stays live until re-point confirmed; rollback = re-point back to Next.js
- **[Risk] Svix scheme details differ (secret prefix, tolerance window)**: verification passes in staging but fails in production → Mitigation: unit tests with known-good signature fixtures; integration test against the live Polar account before cutover (see proposal Assumptions)
- **[Risk] MP webhook HMAC is currently implemented against `x-signature` parsing conventions that differ from MP's actual IPN headers** → Mitigation: integration test with sandbox webhook replay (tasks 10.x) before relying on it; MP checkout already has the verify-mp-payment polling fallback, so webhook failure degrades to polling, not outage
- **[Risk] DI ordering**: `ProviderRouter` requires both adapters + resolver; dig will fail fast if miswired → Mitigation: `make test` includes container build; regression test asserts `domain.BillingProvider` resolves and Polar path still delegates correctly
- **[Trade-off] Two webhook endpoints = two secrets to manage** → Accepted; each provider's verification is self-contained and mirrors the whatsapp pattern
- **[Trade-off] MP checkout UI becomes visible only when plan IDs are configured** → Intended; matches the Polar pattern where `NEXT_PUBLIC_POLAR_PRODUCT_ID` gates Polar checkout

## Migration Plan

1. Implement backend (queries, resolver, DI, interface, routes, webhook endpoints) — Polar behavior unchanged until the new endpoint exists
2. Run `make sqlc`, `make test`, `go build ./...`
3. Deploy backend; verify `/api/v1/webhooks/mercadopago` + `/api/v1/webhooks/polar` respond (signature reject on bad input)
4. Re-point Polar dashboard webhook URL to `/api/v1/webhooks/polar`; send test webhook; confirm subscription state updates
5. Configure MP dashboard webhook for `/api/v1/webhooks/mercadopago`
6. Frontend: fix gate + prop, env vars; `pnpm lint`, `pnpm build`
7. Remove Next.js `/api/billing/webhook` route
8. Reconcile `add-mercadopago-billing/tasks.md`

Rollback: revert this change (Git); re-point Polar webhook URL back to the Next.js app; no Stytch tenant policy state is touched; no local credentials introduced.

## Open Questions

- Whether Polar's live `webhook-secret` uses the standard `whsec_` prefix and the default Svix tolerance window — resolved during integration testing (see Assumptions)
- Whether MercadoPago sandbox IPN delivers `subscription_authorized` reliably, or whether the verify-mp-payment polling path remains the primary sync for MP checkouts — currently both paths are implemented; polling is the fallback
