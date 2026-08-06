## Why

Polar.sh — the current billing provider — only exposes international credit/debit cards as payment methods (backed by Stripe). Colombian B2B customers predominantly pay via PSE (bank transfer) and Nequi (digital wallet), neither of which Polar.sh/Stripe supports. MercadoPago offers all Colombian payment rails (PSE, Nequi, cards, Efecty) plus a built-in subscription engine (`/preapproval` API) with recurring charges, retry logic, and hosted checkout — and is present across all of Latin America, making it the strategic pick for Colombia-first, LatAm-later expansion.

## What Changes

- **New platform layer**: MercadoPago HTTP client (`internal/platform/mercadopago/`) with Bearer token auth, config, and DI wiring, mirroring the existing Polar platform package
- **New `BillingProvider` adapter**: `MPAdapter` implements the existing `domain.BillingProvider` interface, mapping MercadoPago's subscription API (`/preapproval`, `/preapproval_plan`, `/authorized_payments`) to domain types
- **Provider router**: New `ProviderRouter` implementing `domain.BillingProvider` that delegates to the correct adapter (Polar or MercadoPago) per organization, with org-level provider selection stored in DB
- **New webhook parser**: MercadoPago IPN/webhook normalization layer mapping `subscription_authorized`, `subscription_cancelled`, `payment_created` events to existing domain event handlers
- **New checkout endpoints**: `POST /api/subscriptions/create-mp-checkout` (creates MP preference + returns redirect URL) and existing verify-payment pattern adapted for MP
- **Frontend SDK layer**: `lib/mercadopago/` replacing `lib/polar/` concepts — client, plans, subscription status, server actions for checkout and cancellation
- **Dual-provider checkout UI**: Plan selection modal updated to offer provider choice — "International card" (Polar) vs "PSE / Nequi / Colombian card" (MercadoPago)

## Capabilities

### New Capabilities

- `mercadopago-billing-provider`: MercadoPago platform client and `BillingProvider` adapter that handles subscription creation, recurring charge lifecycle, checkout sessions, and webhook processing via MercadoPago's `/preapproval` API
- `billing-provider-routing`: Provider-agnostic routing layer that selects the correct billing adapter (Polar or MercadoPago) per organization, enabling multi-provider coexistence without coupling services to specific providers
- `mercadopago-checkout`: Frontend and backend checkout flow for MercadoPago, including preference creation, hosted checkout redirect, payment verification polling, and dual-provider selection UI

### Modified Capabilities

None. Existing specs (`feature-gating`, `signup-stytch-compliance`) read billing state from the local database and are provider-agnostic by design. The `BillingService` interface, domain types, paywall middleware, and subscription/quotatracking tables remain unchanged.

## Impact

- **Go backend**: New files in `internal/platform/mercadopago/`, new adapter in `internal/modules/billing/infra/mercadopago/`, new provider router (likely `internal/modules/billing/infra/routing/`), new webhook parser, extended checkout handlers. Existing Polar integration files are **untouched**.
- **Database**: One new column or metadata field on `organizations` to store `billing_provider` preference (e.g., `"polar"` or `"mercadopago"`). Existing `subscription_billing` schema unchanged.
- **Frontend**: New `lib/mercadopago/` module, new server actions in `lib/actions/billing/`, modified `plans-modal.tsx` and `subscription-paywall.tsx` for provider selection. Existing Polar frontend code preserved.
- **Dependencies**: Go — `mercadopago-go` (community SDK) or raw HTTP client. Frontend — MercadoPago SDK JS or redirect-based checkout (no client SDK required for Checkout Pro redirect flow).
- **Config**: New env vars: `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL` (default `https://api.mercadopago.com`), `MERCADOPAGO_WEBHOOK_SECRET`, MercadoPago plan IDs. Polar config untouched.
- **Auth**: No changes. Organization mapping via `external_reference` field on MP preapprovals, same pattern as Polar's `external_customer_id`.
- **Non-Goals**: Not replacing Polar.sh. Not building a custom recurring billing engine (MercadoPago's `/preapproval` handles this). Not handling Colombian IVA/tax (MercadoPago checkout Pro does not handle tax as MoR — this is deferred to a future change).
