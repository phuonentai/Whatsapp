## Context

The application currently uses Polar.sh as its sole billing provider via the `BillingProvider` interface in `domain/repository.go`. Polar.sh (backed by Stripe) handles subscriptions, checkout, webhooks, and usage metering. However, it only exposes international credit/debit cards as payment methods. Colombian B2B customers rely on PSE (ACH bank transfers), Nequi (digital wallet), and local cards — none of which Polar.sh/Stripe supports.

The existing architecture is well-abstracted:
- `domain.BillingProvider` interface — 4 methods
- `domain.SubscriptionRepository` — local DB as source of truth
- `domain.OrganizationAdapter` — maps Stytch org IDs to external customer IDs
- Paywall middleware reads local DB, not the provider
- Feature gates read local DB subscription metadata

This design adds MercadoPago as a parallel provider without changing any of those abstractions.

## Goals / Non-Goals

**Goals:**
- Add MercadoPago as a second `BillingProvider` implementation alongside Polar.sh
- Build a provider router so organizations can use either provider
- Map MercadoPago's subscription API (`/preapproval`, `/preapproval_plan`) to existing domain types
- Add dual-provider checkout UI (Polar for cards, MercadoPago for PSE/Nequi/cards)
- Preserve all existing Polar.sh functionality untouched

**Non-Goals:**
- Replacing Polar.sh (it stays as-is)
- Building a custom recurring billing engine (MercadoPago's `/preapproval` handles this)
- Colombian tax handling (IVA, facturación electrónica)
- Meter/usage-based billing for MercadoPago (track locally or defer)
- Data migration for existing Polar subscribers
- Supporting Wompi, Bold, or other Colombian gateways (architecture enables them, implementation is scoped to MercadoPago only)

## Decisions

### 1. Provider Router pattern over service-level branching

**Chosen:** A new `ProviderRouter` struct implements `domain.BillingProvider` and delegates to the correct adapter based on the organization's `billing_provider` setting.

```
┌────────────────────────────────┐
│         BillingService         │  (unchanged, injects BillingProvider)
└──────────────┬─────────────────┘
               │
               ▼
┌────────────────────────────────┐
│        ProviderRouter          │  implements domain.BillingProvider
│                                │
│  Route(ctx, orgID) → adapter   │
│  ┌──────────────────────────┐  │
│  │ org.billing_provider     │  │
│  │ "polar"  → PolarAdapter  │  │
│  │ "mp"     → MPAdapter     │  │
│  └──────────────────────────┘  │
└───────┬────────────────────────┘
        │
   ┌────┴────┐
   ▼         ▼
PolarAdapter  MPAdapter
(unchanged)   (new)
```

**Alternatives considered:**
- *Service-level if/switch*: Embed routing in `billingService`. Rejected — couples the service to provider names and makes adding a third provider messier.
- *Separate BillingService per provider*: Inject `polarService` or `mpService` based on org. Rejected — duplicates service logic for no gain.

### 2. MercadoPago subscriptions via `/preapproval` with plans

**Chosen:** Use MercadoPago's `POST /preapproval` with a `preapproval_plan_id` to create subscriptions. Plans are created once in the MercadoPago dashboard (or via `POST /preapproval_plan` API). Each subscription links to a plan and auto-renews per the plan's `auto_recurring` settings.

MP preapproval statuses map to local DB:
- `pending` → pending (awaiting first payment)
- `authorized` → active
- `paused` → past_due
- `cancelled` → canceled

**Alternatives considered:**
- *Checkout Pro only (no subscription API)*: Create one-time preferences, manually schedule charges. Rejected — requires building a recurring billing engine, which is a non-goal.
- *Subscription Plans (no-code)*: Use MP's no-code subscription links. Rejected — no API control, can't sync state to our DB.

### 3. Webhook handling: normalize MP events to existing handlers

**Chosen:** New webhook parser file (`mp_webhook_parser.go`) that receives MP IPN notifications, normalizes them to `domain.SubscriptionEventData`, and calls the same `handleSubscriptionUpsert`, `handleSubscriptionCanceled` methods.

MP event → domain mapping:
- `subscription_authorized` → creates/updates subscription (like Polar's `subscription.created`)
- `subscription_cancelled` → sets status to canceled
- `payment_created` → for checkout verification, polls payment status

The webhook endpoint accepts both Polar and MP signatures. A dispatch function checks the `type` field or signature header to route to the correct parser.

**Alternatives considered:**
- *Separate webhook endpoints*: `/api/webhooks/polar` and `/api/webhooks/mercadopago`. Rejected — adds route clutter for what is essentially the same concept.
- *Frontend-only webhook receiver*: Like the current Next.js route for Polar. Rejected — Go backend should be the authority for database mutations.

### 4. Organization ↔ MP mapping via `external_reference`

**Chosen:** When creating an MP preapproval, set `external_reference` to the Stytch organization ID (same value used as Polar's `external_customer_id`). The existing `OrganizationAdapter.GetOrganizationIDByStytchOrgID()` resolves it during webhook processing.

**Alternatives considered:**
- *MP customer ID as the link*: Store MP's internal customer ID. Rejected — adds an extra lookup; `external_reference` is designed for this.

### 5. Meter/usage tracking for MP subscriptions

**Chosen:** Track invoice consumption locally in `quota_tracking` only. Skip `IngestMeterEvent` for MP (the `MPAdapter.IngestMeterEvent` method is a no-op). MercadoPago doesn't have a meter/event system analogous to Polar's.

**Alternatives considered:**
- *Use MP authorized_payments count as metering*: Query `/authorized_payments/search` periodically. Rejected — adds latency and complexity for something the local DB already tracks.
- *Send usage to Polar anyway*: Cross-provider metering. Rejected — confusing, two sources of truth.

### 6. Org-level provider selection stored in organization metadata

**Chosen:** Add `billing_provider` field to organization metadata (JSONB or a new column). Default to `"polar"` for backward compatibility. Set to `"mercadopago"` when an org completes their first MP checkout.

The provider is selected at checkout time — the user chooses "PSE/Nequi" (MP) or "International card" (Polar) in the plans modal.

**Alternatives considered:**
- *Hardcoded by region*: Auto-select based on org country. Rejected — some Colombian companies have international cards and prefer Polar.
- *Runtime-only (no persistence)*: Derive from which provider has a subscription. Rejected — ambiguous if both exist (unlikely but possible).

### 7. Frontend approach: Checkout Pro redirect (not embedded)

**Chosen:** Use MercadoPago Checkout Pro (hosted redirect). The backend creates a preference, returns `init_point` URL, and the frontend redirects. After payment, MP redirects back with query params; the frontend calls verify endpoint which polls `GET /v1/payments/{id}`.

**Alternatives considered:**
- *Checkout Bricks (embedded)*: Embed MP payment form in our UI. Rejected — more complex, requires PCI compliance considerations, doesn't support PSE as cleanly as Checkout Pro.
- *Checkout API (fully custom)*: Build our own payment form. Rejected — massive effort, PCI scope, reinventing PSE bank selection UI.

## Risks / Trade-offs

- **[Risk] MercadoPago API downtime blocks Colombian checkout** → Mitigation: Polar.sh remains available as fallback; checkout UI shows both options. If MP is down, users can still pay with international cards via Polar.
- **[Risk] Webhook delivery failures cause stale subscription state** → Mitigation: Same lazy guarding pattern as Polar — if DB says expired but subscription exists, `RefreshSubscriptionStatus` calls MP API to self-heal.
- **[Risk] No meter/usage events on MP** → Mitigation: `IngestMeterEvent` is a no-op for MP. Invoice counting is purely local via `DecrementInvoiceCount`. Acceptable since MP doesn't offer meter-based billing.
- **[Trade-off] Two providers = two webhook secrets, two API tokens, doubled config surface** → Accepted. Necessary cost of multi-provider. Each provider's config is isolated in its own platform package.
- **[Trade-off] Subscription lifecycle management is provider-specific** → Polar has its own subscription state machine; MP has `preapproval` statuses. Both map to the same local `subscription_status` field. This works because we treat the local DB as the system of record — the provider is just the payment rail.

## Open Questions

- **MP Go SDK vs raw HTTP**: There's a community `mercadopago-go` SDK but it lags behind the API. Recommend raw HTTP + structs for reliability, same as the current Polar adapter.
- **IVA 19% on invoices**: MercadoPago Checkout Pro does not handle tax as a Merchant of Record. When Colombian facturación electrónica is needed, this will be a separate change.
- **Migration path for existing Polar subscribers**: If an existing org wants to switch to PSE/Nequi, they'd need to cancel their Polar subscription and create a new MP one. This could be automated in a future change.
