## Why

The MVP story — "a Colombian SME runs the whole sale inside WhatsApp" — has a payment gap: the siigo invoicing flow already sends the customer a WhatsApp message with an invoice link and (optionally) a MercadoPago payment link, but that link path is an inert seam (`noopPaymentLinker`), so the platform generates no tracked payment state, no revenue, and no confirmation loop. Colombia is a mobile-wallet economy (Nequi ~20M+ users, Daviplata ~15M+, PSE as standard); closing the sale — including payment — inside WhatsApp is the killer flow for SMBs. MercadoPago already supports Colombian rails (PSE/Nequi); one-shot payment links ship fast on the existing adapter. The platform monetizes with a commission markup on top of the payment amount (MercadoPago rail; future Nequi/Daviplata direct rails are fee-free and out of scope).

## What Changes

- **New `payments` module (client-facing, one-shot)**: `internal/modules/payments` — creates a one-shot MercadoPago payment preference (link) for a deal/invoice amount, tracks payment state in a new `client_payments` table, and records platform commission.
- **Real PaymentLinker seam wired**: replace `noopPaymentLinker` (invoicing) with an implementation that creates a MercadoPago payment preference; the invoice WhatsApp notification message then carries a real, tracked payment link.
- **Webhook dispatch extended**: `ProcessMPWebhookEvent`'s `payment_created`/`payment_updated`/`payment_approved` branch no longer ignores payment events — it dispatches them to the payments module, which verifies via `GET /v1/payments/{id}` (polling fallback), marks the payment paid, records a deal activity, and sends a WhatsApp confirmation message to the contact. Subscription events (`subscription_*`) are untouched.
- **Commission model**: platform commission is a configurable percentage added to the payment preference amount (default from environment config, overridable per org later). The customer pays amount + commission; the SME receives the full amount. Payment events continue to be ignored for the subscription billing path.
- **Idempotency**: webhook-driven payment state transitions are transaction-isolated state checks (payment events carry an event/payment id; no double-apply).

## Capabilities

### New Capabilities
- `client-payments`: one-shot, customer-facing MercadoPago payment links tied to deals/invoices — link creation, payment state machine (pending → paid/failed/expired), commission markup, WhatsApp confirmation on payment, deal activity recording.

### Modified Capabilities
- `mercadopago-webhooks` (delta of active change `wire-mercadopago-billing`): requirement "payment events SHALL be ignored" changes to "payment events SHALL be dispatched to the client-payments module for verification" — subscription event handling unchanged. The delta spec is amended via this change so the folded living spec is correct when `wire-mercadopago-billing` archives.

## Impact

- **Backend (go-b2b-starter)**: new `internal/modules/payments` module (domain → app → infra); new migration `client_payments` table + SQLC queries; `billing/infra/mercadopago/mp_adapter.go` gains one-shot preference methods; `billing/app/services/process_mp_webhook_event_service.go` payment branch dispatches to payments service; `invoicing/app/services/invoicing_service.go` PaymentLinker seam wired to a real implementation.
- **MercadoPago API surface used**: `POST /checkout/preferences` (creates link, `init_point`) and `GET /v1/payments/{id}` (verification fallback). Existing `x-signature` webhook verification reused — no new ingress.
- **Frontend (next_b2b_starter)**: none (auto flow only; no manual trigger UI in this change).
- **Ops**: MercadoPago dashboard must emit payment events to `/api/v1/webhooks/mercadopago` (same endpoint as today); optional WhatsApp `pago_recibido` transactional template is a deployment-time nicety — free-form text confirmation works without it, matching the existing invoicing precedent.
- **Config**: new env key for the commission percentage (backend only).

## Non-Goals

- **No local credential storage**: payment rails and identities remain external; `client_payments` stores only MercadoPago preference/payment ids and Stytch org foreign keys, never tokens, card data, or wallet credentials. Local PostgreSQL stores no passwords, MFA tokens, or session tokens (Stytch remains runtime SSOT for identity/RBAC).
- No Nequi/Daviplata direct API integrations (future fee-free rails, own proposal).
- No WhatsApp Flows (native in-chat payment UI).
- No recurring billing / subscriptions for client payments — one-shot payment links only.
- No automatic deal-stage move on payment (pipeline stages are per-org custom; confirmation is activity + WhatsApp message only).
- No manual payment-link trigger endpoint or frontend UI — auto flow from invoicing only.
- No changes to the subscription billing path (Polar/MP preapproval, paywall, verify-payment polling) beyond webhook payment-event dispatch.

## Rollback

- **Git state**: revert the change's commits; re-apply the `noopPaymentLinker` wiring; migration rollback removes `client_payments`.
- **MercadoPago state**: no tenant-policy changes on Stytch; MP dashboard webhook config is unchanged (same endpoint, same signature verification) — only event handling behavior changes server-side; no MP-side rollback required.
