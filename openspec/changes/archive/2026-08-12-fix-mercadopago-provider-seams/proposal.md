## Why

`wire-mercadopago-billing` is wired (DI router, endpoints, webhooks all registered and gated), but several seams break MP orgs end-to-end: (1) `VerifyMPPayment` fetches the subscription from the provider router, which resolves the org's not-yet-set `billing_provider` to **Polar** (`mp_checkout_service.go:139`); (2) the MP adapter persists **raw** MP statuses (`authorized`/`paused`/`cancelled`) on refresh (`mp_adapter.go` `GetSubscription`), which neither the SQL gate nor `GetBillingStatus` recognize; (3) MP subscriptions never carry an invoice quota, so `can_process_invoice` is always false (`invoice_count = 0`); (4) MP payment webhooks dispatch the **notification** id instead of the payment id to the client-payments handler (`process_mp_webhook_event_service.go:44`); (5) `CreateMPCheckout`/`CancelMPSubscription` read the org id from `ctx.Value("stytch_org_id")` on the request context, but the auth middleware stores it in Gin keys (`c.Set`) — the value is never present, so every checkout/cancel 500s with "organization context required" (`mp_checkout_service.go:13-16,59-62`); (6) MP cancellation builds a `Subscription` without `CurrentPeriodStart/End`, which the schema declares NOT NULL, so the local upsert fails and the canceled MP org keeps access; (7) the webhook group is mounted as `/api/v1/webhooks` while the module is already under the `/api` prefix, making the real paths `/api/api/v1/webhooks/{polar,mercadopago}` (`routes.go:62` × `api/provider.go:114`) — every dashboard-configured webhook URL 404s; (8) MP `x-signature` verification parses the timestamp but never enforces freshness and accepts a raw-header fallback, so replayed payloads pass (`mp_webhook_parser.go`). Separately, the backend hard-requires MP credentials at boot (`platform/mercadopago/cmd/init.go` → `LoadConfig().Validate()` → panic in `bootstrap/init_mods.go:87`), so a Polar-only deployment cannot start.

## What Changes

- `mp_checkout_service.go:139` — `VerifyMPPayment` SHALL fetch the subscription from `s.mpProvider` (the MP adapter), not the router/Polar.
- `mp_adapter.go` `GetSubscription` — SHALL map the preapproval status through `MapMPStatus` before storing, and SHALL read `metadata.invoice_count_max` (set at checkout) into the subscription `Metadata` so sync/verify paths derive a nonzero quota.
- `mp_adapter.go` `CreateCheckoutSession` — SHALL attach `metadata.invoice_count_max` to the preapproval body, resolved per plan from new config values.
- `platform/mercadopago/config.go` — add `MERCADOPAGO_CHECKOUT_INVOICE_COUNT` / `MERCADOPAGO_BUSINESS_INVOICE_COUNT` (default `0`); document both in `example.env`.
- `mp_webhook_parser.go` — `ParseSubscriptionEventData` SHALL extract `data.metadata.invoice_count_max` into product metadata so the shared `handleSubscriptionUpsert` seeds a nonzero quota; `ParsePaymentEventData` SHALL tolerate string payment ids.
- `process_mp_webhook_event_service.go:44` — payment events SHALL dispatch `data.id` (the payment id) to `paymentEventHandler`, not the notification id; update the unit tests that currently assert the wrong id.
- `platform/mercadopago/cmd/init.go` + `billing/app/services/module.go` — MP SHALL become optional: when `MERCADOPAGO_ACCESS_TOKEN` is unset, skip config/client/adapter registration and make the named `mercadopago` binding optional in `billingServiceParams`; MP service methods SHALL return a clear "MP not configured" error instead of panicking at boot.
- `mp_checkout_service.go:13-16,59-62` — `CreateMPCheckout`/`CancelMPSubscription` SHALL resolve the organization id from the Gin context (`c.Get` keys populated by `RequireOrganization`), not `ctx.Value` on the request context; the org id SHALL be passed explicitly into the service methods.
- `mp_checkout_service.go:83-92` — MP cancellation SHALL persist locally: build the `Subscription` with `CurrentPeriodStart/End` derived from the existing row (or the preapproval's `next_payment_date`/`end_date`) so the NOT NULL upsert succeeds and the canceled status sticks; `subscription_cancelled` webhook handling SHALL tolerate absent dates.
- `billing/routes.go:62` — the webhook group SHALL be mounted as `/v1/webhooks` under the existing `/api` prefix (the whatsapp pattern), so `POST /api/v1/webhooks/polar` and `/api/v1/webhooks/mercadopago` resolve instead of `/api/api/v1/webhooks/*`.
- `mp_webhook_parser.go` — `VerifyWebhookSignature` SHALL enforce a timestamp freshness window and SHALL reject headers that are not in the `ts=,v1=` format (drop the raw-header fallback that permits replay).
- Docs — replace stray `POLAR_WEBHOOK_SECRET` references (`docs/README.md`, `docs/billing.md`) with the real `WEBHOOK_SECRET` that code and `example.env` use.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mercadopago-checkout`: `VerifyMPPayment` SHALL verify against the MercadoPago provider and SHALL create subscription + quota records with the mapped status and the plan's invoice quota (via preapproval metadata); checkout/cancel SHALL resolve the org id from the Gin context and cancellation SHALL persist a mapped `canceled` row with valid period bounds.
- `mercadopago-webhooks`: subscription events SHALL persist **mapped** statuses; payment events SHALL dispatch `data.id` to the client-payments handler (the current delta says payment events are ignored — this change extends that requirement with the client-payments dispatch that `add-client-payments` wired); the webhook endpoints SHALL be reachable at `/api/v1/webhooks/{polar,mercadopago}` and signature verification SHALL reject replays via timestamp freshness.
- `billing-provider-routing`: the MP adapter is now optional in DI; the router SHALL degrade to Polar-only when MP is unconfigured.

## Non-Goals

- Building a recurring billing engine or adding new payment methods.
- Altering Polar adapter behavior or the paywall middleware.
- MP trial handling (covered by `new-client-billing-lifecycle`).
- No local credential storage is introduced; secrets stay in backend env only.

## Rollback

- **Git**: revert the change; no migration required (no schema change; quota arrives via metadata, not new columns).
- **Stytch**: untouched — no tenant policy changes.
- **MercadoPago**: preapprovals created while the change is live keep their metadata; reverting only stops writing new metadata. No API-side rollback needed.
