## Context

The siigo invoicing flow already sends the deal's contact a WhatsApp message containing the invoice link and, via the optional `PaymentLinker` seam, a MercadoPago payment link — but the seam is a `noopPaymentLinker` (invoicing_service.go:302), so no payment is ever created or tracked. Verified current state:

- `PaymentLinker` interface (invoicing/app/services/invoicing_service.go:37): `PaymentLink(ctx, orgID int32) (string, error)` — **too narrow**: no amount, no deal reference. A real implementation must create a preference priced at the invoice amount and persist state tied to the deal.
- MP platform client (`internal/platform/mercadopago/client.go:35`) is a generic HTTP wrapper (`Get`/`Put`/`Post`); the billing adapter already demonstrates `POST /preapproval` and `GET /v1/payments/{id}` (`mp_adapter.go:45,167`) with status mapping (`approved` → `succeeded`, rejected/cancelled → `failed`, else `pending`).
- MP webhook ingress `POST /api/v1/webhooks/mercadopago` (signature-verified) dispatches to `ProcessMPWebhookEvent` (billing/app/services/process_mp_webhook_event_service.go), which currently **ignores** `payment_created`/`payment_updated`/`payment_approved` (line 47) — a requirement this change amends (delta `mercadopago-webhooks`).
- Send path: `crm OutboundService.SendMessage(ctx, orgID, convID, content)` — free-form text, used by invoicing `notify()` (invoicing_service.go:277). No Meta template infra needed for MVP (matches invoicing precedent).
- Invoicing DI is dig-based (`invoicing/app/services/module.go`), with `PaymentLinker` provided as `optional:"true"`.
- Migrations: latest is `000030`; next is `000031`.
- Deal stages are per-pipeline custom (crm `pipeline.go`) — no hardcoded "pagado" stage; deal status enum has `ganado` (deal.go:12). Payment confirmation therefore records a system activity, not a stage move.

## Goals / Non-Goals

**Goals:**
- Create tracked, one-shot MercadoPago payment links (Checkout Preferences API) for invoices, priced at amount + platform commission.
- Persist payment state locally (`client_payments`), keyed by MP preference/payment ids, referenced to org (Stytch org FK) + deal.
- Extend `ProcessMPWebhookEvent` payment branch to dispatch payment events to the payments module, verified against `GET /v1/payments/{id}` before mutation, idempotently.
- Confirm payment to the contact inside WhatsApp + record deal activity on payment.
- Wire the real `PaymentLinker` into invoicing so the invoice message carries a real, tracked link; link failure never fails invoicing.

**Non-Goals:**
- Nequi/Daviplata direct API integrations (future, fee-free rails — own proposal).
- WhatsApp Flows (native in-chat payment UI).
- Recurring/subscription client payments; any change to the subscription billing path beyond webhook dispatch.
- Auto deal-stage move on payment (custom pipelines); manual payment-link trigger endpoint or frontend UI.
- Commission per-org config UI — env default only.

## Decisions

### D1. New `payments` module, not an extension of `billing`
`internal/modules/payments` (domain → app → infra), Clean Architecture as mandated. Billing = platform revenue (subscriptions); payments = customer-facing one-shot links. Distinct state machines; keeps `domain.BillingProvider` and the subscription paywall untouched.
- *Alternative rejected*: adding client-payment methods to `domain.BillingProvider` — pollutes the subscription contract and forces the paywall path to know about customer payments.

### D2. Payment preference via Checkout Preferences API, tracked in `client_payments`
`POST /checkout/preferences` with `items[0]` (title, quantity 1, unit_price = amount × (1 + commission)), `external_reference` = deal id (mirrors billing's external-reference pattern), `back_url` reused from billing config. Response `init_point` = the link sent in WhatsApp. Payment id arrives later via webhook/polling.
- *Alternative rejected*: MercadoPago "payment links" (storefront) API — no programmatic status correlation guarantees in the current SDK surface; preferences + `GET /v1/payments/{id}` reuses the exact pattern already proven in `mp_adapter.GetCheckoutSession`.

### D3. `client_payments` table (migration `000031_create_client_payments`)
```
id              bigserial PK
organization_id int        NOT NULL (FK organizations, Stytch org FK domain)
deal_id         int        NOT NULL (FK deals)
invoice_id      int        NULL     (FK invoices, when link came from invoicing)
amount_cop      bigint     NOT NULL (base amount, SME receivable)
commission_cop  bigint     NOT NULL DEFAULT 0
currency        text       NOT NULL DEFAULT 'COP'
status          text       NOT NULL DEFAULT 'pending'  -- pending|paid|failed|expired
mp_preference_id text     UNIQUE NULL
mp_payment_id   text       UNIQUE NULL
paid_at         timestamptz NULL
created_at      timestamptz NOT NULL DEFAULT now()
```
SQLC queries: `CreateClientPayment`, `GetClientPaymentByPreferenceID`, `GetClientPaymentByPaymentID`, `UpdatePaymentStatus` (idempotent: `WHERE status <> 'paid'` guarded update inside transaction). Status transitions: pending → paid | failed | expired; terminal states never mutate.

### D4. Extend `PaymentLinker` seam instead of bypassing it
Change interface to `PaymentLink(ctx, orgID, dealID int32, amountCOP int64) (string, error)`. Invoicing `notify()` computes amount from the invoice and calls the seam; the new payments module provides the real implementation through the same dig `optional:"true"` binding (module.go pattern), so invoicing stays decoupled. `noopPaymentLinker` remains as the fallback when the payments module isn't wired.
- *Alternative rejected*: invoicing calls payments service directly — couples modules; the seam exists precisely to avoid this.

### D5. Webhook dispatch via injected handler interface in billing
`ProcessMPWebhookEvent`'s payment branch calls an injected `PaymentEventHandler` interface (method `HandlePaymentEvent(ctx, eventType, paymentID string) error`), implemented by the payments module. Billing's dig `module.go` provides the handler; payments module must not import billing (one-way dependency). Subscription branches untouched.
- Fallback: verification failure leaves payment `pending`; the same verification is retried on subsequent events (event re-delivery) — no polling loop required at MVP.

### D6. Verification before mutation + idempotency
`HandlePaymentEvent` on `payment_approved`: load by `mp_payment_id` → if absent, load by `mp_preference_id` from event data → `GET /v1/payments/{id}` verify `approved` → guarded transition inside a transaction (state check + update atomic). Duplicate events apply at most once. Payment events for untracked payments are acknowledged and dropped (matches billing precedent).

### D7. Commission as env-configured percentage
`PAYMENTS_COMMISSION_RATE` (decimal, default 0.0), applied at preference creation. `commission_cop = round(amount × rate)`; `unit_price = amount + commission_cop`. Recorded on the row; zero rate = exact amount. Nequi/Daviplata rails (future) are fee-free by default — the rate key is per-rail by design.

### D8. Confirmation via existing send path, non-fatal
On `paid`: resolve contact + active conversation (same helpers invoicing `notify()` uses — `GetByID` deal → `GetActiveByContact` conv), `OutboundService.SendMessage` free-form confirmation, `recordActivity` system activity "Pago recibido". Send failure = log warn, payment stays `paid`. Withdrawn contacts still receive the confirmation (transactional, no promo content — invoicing precedent, spec `client-payments`).

## Risks / Trade-offs

- **[Risk] MP preferences API returns `init_point`; webhook delivery of `payment_approved` for preferences is not guaranteed in all sandbox configs** → Mitigation: `GET /v1/payments/{id}` verification path is the source of truth; events are the trigger, not the authority. Sandbox replay verification deferred to deployment (external credentials), matching `wire-mercadopago-billing` 13.x pattern.
- **[Risk] `external_reference` collisions between client payments and subscription preapprovals** → Mitigation: namespaced reference (e.g. `deal:<id>`); lookup keyed by MP preference/payment ids, not external_reference.
- **[Risk] Commission rounding / MP fee structure** → Mitigation: commission computed and persisted in COP at creation; MP's own processing fee is the SME's cost, separate from platform commission (documented in spec, not modeled in DB).
- **[Risk] Billing module depends on payments handler → DI cycle** → Mitigation: handler is an interface in billing, implementation in payments; payments never imports billing. Dig binds at compose root.
- **[Risk] Duplicate/out-of-order events** → Mitigation: transaction-isolated guarded transition + terminal-state immutability; out-of-order `payment_created` after `payment_approved` is a no-op.
- **[Trade-off] No `pago_recibido` Meta template** → free-form text confirmation works within the 24h window; template approval deferred to ops, not a blocker.

## Migration Plan

1. Migration `000031_create_client_payments` up/down + SQLC queries (`make sqlc`).
2. Payments module: domain types → app services (`CreateLink`, `HandlePaymentEvent`) → infra adapter (`CreateClientPaymentPreference`, `VerifyPayment` reusing `/v1/payments/{id}` + status mapping pattern).
3. Extend `PaymentLinker` interface + invoicing `notify()`; wire real implementation in invoicing dig module.
4. Billing: inject `PaymentEventHandler`; payment branch dispatches.
5. Env: `PAYMENTS_COMMISSION_RATE` documented in backend config.
6. Verify: `make sqlc && go build ./... && go vet ./... && make test`; unit tests: preference creation payload (commission math), webhook dispatch + idempotency, confirmation send non-fatal, seam fallback.
7. Deferred (external): live MP sandbox replay of `payment_approved` → `/api/v1/webhooks/mercadopago`, end-to-end preference → pay → webhook → paid; recorded in tasks.md.

**Rollback:** revert commits (seam falls back to noop, webhook branch returns to ignore) + `000031` down migration. No Stytch tenant policy changes; no MP dashboard config change (same endpoint, same signature) — server-side behavior only.

## Open Questions

- Commission per-org override — needed before Nequi/Daviplata rails, or env default forever? (deferred, not blocking)
- Should `paid` transition attempt a deal-stage move when a pipeline has a "pagado"/"cerrado ganado" stage? (deferred — pipelines custom)
