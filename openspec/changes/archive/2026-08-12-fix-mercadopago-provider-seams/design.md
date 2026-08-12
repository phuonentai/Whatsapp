## Context

`wire-mercadopago-billing` shipped the MP surface: `ProviderRouter` as the unnamed `domain.BillingProvider`, DB-backed provider resolution, checkout/verify/cancel endpoints, per-provider webhook ingress with signature verification, and the frontend gate keyed on `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`. The remaining seams break MP orgs at runtime:

- `VerifyMPPayment` (`mp_checkout_service.go:139`) fetches the subscription through the router, which resolves the org's `billing_provider` (still unset at verify time) to **Polar** — MP verification cannot persist a subscription.
- `mp_adapter.GetSubscription` stores raw MP statuses (`authorized`, `paused`, `cancelled`); neither the SQL gate nor `GetBillingStatus` recognize them, so refreshed MP orgs lock out.
- MP subscriptions carry no `invoice_count_max`, so `can_process_invoice` is always false (quota 0).
- Payment webhooks dispatch the notification id instead of `data.id` (`process_mp_webhook_event_service.go:44`).
- MP config is a hard boot dependency (`platform/mercadopago/cmd/init.go` → `LoadConfig().Validate()`), so Polar-only deployments panic in `bootstrap/init_mods.go:87`.
- `CreateMPCheckout`/`CancelMPSubscription` read `ctx.Value("stytch_org_id")` from the request context, which the auth middleware never populates (it uses Gin keys) — every checkout/cancel 500s (`mp_checkout_service.go:13-16,59-62`).
- MP cancellation and `subscription_cancelled` webhooks build rows without period bounds against a NOT NULL schema — the local cancel never persists, so canceled MP orgs keep access.
- The webhook group mounts `/api/v1/webhooks` under the already-prefixed `/api` mount, so the real paths are `/api/api/v1/webhooks/*` — advertised URLs 404 (`routes.go:62`).
- MP `x-signature` verification never checks timestamp freshness and accepts a raw-header fallback — signed payloads replay indefinitely.

## Goals / Non-Goals

**Goals:**
- MP orgs activate end-to-end: verify → mapped status + nonzero quota persisted → paywall passes
- Payment events reach client-payments with the correct payment id
- Backend boots without MP credentials (Polar-only degradation)

**Non-Goals:**
- Recurring billing engine, new payment methods, Polar changes, schema changes
- Trial handling for MP (see `new-client-billing-lifecycle`)

## Decisions

### 1. Verify fetches from the MP adapter directly

`VerifyMPPayment` uses `s.mpProvider.GetSubscription` instead of the router. The router is correct for orgs whose provider is already recorded (refresh path), but at verify time the provider is set only **after** the fetch (`SetBillingProvider` runs last), so the router resolves `""` → Polar. Direct MP access removes the ordering dependency.

**Alternatives considered:**
- Set `billing_provider` before fetching: would route correctly but mutates org state before payment verification succeeds — rejected (a failed verify would leave the org pointing at MP).

### 2. Statuses mapped at the adapter boundary

`mp_adapter.GetSubscription` maps the preapproval status via `MapMPStatus` before returning, matching the webhook path (`ProcessMPWebhookEvent` already maps). Local DB therefore only ever holds canonical statuses (`active`, `past_due`, `canceled`, `pending`).

**Alternatives considered:**
- Map at the SQL/service layer: would scatter MP-specific knowledge through generic code — rejected.

### 3. MP quota via preapproval metadata + env config

MP plans carry no product metadata (unlike Polar's `invoice_count`), so the quota is attached to the preapproval at creation: `CreateCheckoutSession` writes `metadata.invoice_count_max` from new config values `MERCADOPAGO_CHECKOUT_INVOICE_COUNT` / `MERCADOPAGO_BUSINESS_INVOICE_COUNT` (default `0`), and `GetSubscription` / `ParseSubscriptionEventData` read it back. `VerifyMPPayment` and `SyncSubscriptionFromPolar` already consume `Metadata["invoice_count_max"]` unchanged.

**Alternatives considered:**
- Fixed default quota: hides misconfiguration and makes plans indistinguishable — rejected.
- New quota table keyed by plan: heavier than needed for two env-mapped plans — rejected for now; revisit if the plan catalog grows.
- Reuse Polar product metadata: MP and Polar plan ids differ, no shared catalog — rejected.

### 4. Payment events dispatch `data.id`

For MP IPN payment notifications, `data.id` is the payment id; the top-level `id` is the notification id. `ProcessMPWebhookEvent` parses `payload.Data` via `ParsePaymentEventData` and passes that payment id to `paymentEventHandler.HandlePaymentEvent`. `ParsePaymentEventData` gains a string-id fallback (MP may deliver numeric or string ids). Unit tests asserting the notification id are updated.

### 5. MP becomes optional at boot

`platform/mercadopago/cmd/init.go` skips config/client/adapter registration when `MERCADOPAGO_ACCESS_TOKEN` is empty; `billingServiceParams.MPProvider` becomes `optional:"true"`; `CreateMPCheckout`/`VerifyMPPayment`/`CancelMPSubscription` return a clear `mercadopago not configured` error when the adapter is absent; the router stays Polar-only.

**Alternatives considered:**
- Keep the hard requirement: simplest, but Polar-only deploys (or sandboxes without MP credentials) cannot boot — rejected for parity with the FE's `isMercadoPagoEnabled()` gating.

### 6. Org id comes from the Gin context, not the request context

`RequireOrganization` (`auth/middleware.go:355`) stores the org id with `c.Set` (Gin keys); the service reads `ctx.Value("stytch_org_id")` on `c.Request.Context()`, which is a different store, so the value is always nil and every MP checkout/cancel 500s. The handler SHALL read `c.GetString("stytch_org_id")` and pass it explicitly into the service method (explicit parameter beats a magic context key).

**Alternatives considered:**
- Copying Gin keys into the request context in a middleware: adds an undocumented convention and touches the shared auth path for every module — rejected.
- Reading `c.Get` inside the service: services should not depend on Gin — rejected; the handler extracts and passes.

### 7. MP cancel persists with derived period bounds

The schema requires non-null `current_period_start/end`; MP cancel and `subscription_cancelled` webhooks may not carry dates. The local row keeps its existing period bounds (or derives them from `next_payment_date`/`end_date` when the row is fresh), so the upsert succeeds, the status flips to `canceled`, and the paywall denies. This also makes the `subscription_cancelled` webhook idempotent and stops the retry-forever 500 loop.

### 8. Webhook group mounted under the existing prefix

`routes.go:62` registers `router.Group("/api/v1/webhooks")` while the billing module is already mounted under `server.ApiPrefix = "/api"` (`api/provider.go:114`), yielding `/api/api/v1/webhooks/*`. The whatsapp module mounts `/v1/webhooks` under the same prefix and resolves at `/api/v1/webhooks/whatsapp` — the billing group SHALL match that pattern. Ops impact: the dashboard webhook URLs stay `/api/v1/webhooks/{polar,mercadopago}` as already configured.

### 9. Signature freshness instead of raw-header fallback

MP's `x-signature` is normally `ts=<unix>,v1=<hmac>`. The current verifier parses `ts` but never checks age, and if the header lacks the format it treats the whole value as the raw HMAC — so any correctly-signed old payload replays forever. The verifier SHALL require the `ts=,v1=` format, check `|now - ts| <= 300s`, and constant-time-compare the computed HMAC against `v1`.

## Risks / Trade-offs

- **[Risk] MP sandbox IPN quirks (status names, metadata casing)** → Mitigation: unit tests for `MapMPStatus` + metadata parse; the verify-mp-payment polling path remains the fallback if webhooks misbehave.
- **[Risk] Optional DI binding ripples into mock constructors/tests** → Mitigation: keep the named binding registered when configured (current tests unchanged); only the unconfigured path exercises the nil branch.
- **[Risk] Quota defaults to 0 until envs are set** → Mitigation: documented in `example.env`; a zero quota yields quota-blocked (not paywall-blocked) which is loud and unambiguous.

## Migration Plan

1. Config fields + `example.env`; adapter metadata write/read; status mapping.
2. `VerifyMPPayment` provider swap; webhook payment-id dispatch + tests.
3. Optional-boot wiring; `go build ./...`, `go test ./...`.
4. Deploy; sandbox replay of `subscription_authorized` + a payment event; then enable envs.
5. Rollback: revert the change; no migration (no schema change); MP preapprovals already created keep their metadata harmlessly.

## Open Questions

- Whether MP sandbox preapproval metadata round-trips through the search API — verified during integration (task 13.x in the wire change is still deferred).
