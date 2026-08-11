# Add paywall access gating spec

## Why

The paywall module (`go-b2b-starter/internal/modules/paywall`, ~716 LOC) enforces commercial access: a subscription status state machine, a 402 Payment Required gate, lazy-guard recovery for missed provider webhooks, and non-blocking status exposure. It protects AI and WhatsApp routes across six modules (cognitive, documents, whatsapp, agent, campaigns, instagram). Yet it has **no living spec** — the verification gate cannot check it, and behavior is only enforceable by code inspection.

Adjacent specs cover edges, not the core:

- `feature-gating` covers plan→feature flags and HTTP 403 ("what your plan unlocks").
- `billing-provider-ux` covers inactive-paywall copy rendering only.
- `ai-usage-metering` covers the credit ledger, which runs *after* paywall passes.

None defines paywall access-gating behavior itself. The commercial money gate is unblueprinted.

## What Changes

Add a new living capability spec `paywall` capturing the implemented, test-verified behavior:

- Subscription status state machine: `active`/`trialing` → access; `past_due`/`canceled`/`unpaid`/`none` → 402.
- Blocking middleware flow: OPTIONS skip, missing-org 500, no-subscription 402, lazy-guard provider refresh, status-specific 402 codes, subscription context propagation.
- Two modes: `RequireActiveSubscription` (blocking) and `OptionalSubscriptionStatus` (non-blocking), including legacy aliases.
- Billing adapter contract: local-DB fast path (`GetSubscriptionStatus`) + provider refresh path (`RefreshSubscriptionStatus`).
- 402 error contract: `subscription_required` / `payment_failed` / `subscription_canceled` / `payment_required` / `subscription_inactive`, with `upgrade_url` (default `/billing`).
- Swiss-cheese routing: protected routes use blocking mode; billing/settings/profile/webhooks are not gated.

Specs only. No application code, tests, routes, or infrastructure changes.

## Capabilities

### New Capabilities
- `paywall`: subscription-based commercial access gating — status derivation, 402 gate, lazy-guard recovery, non-blocking status exposure, and the billing adapter contract.

### Modified Capabilities

None. No existing spec requirements change.

## Impact

- **Specs only.** No application code, test code, or infrastructure changes. The implementation already matches the specified behavior.
- **Architecture note:** paywall is the coarse "are you paying" gate; `feature-gating` is the fine "what plan unlocks" gate; the AI credit guard is usage enforcement. Route order per protected module: `auth → org_context → feature middleware (403) → paywall (402) → credit guard (402)`.
- **Dev workflow:** none — this change only adds OpenSpec coverage so the verification gate can check paywall behavior.
