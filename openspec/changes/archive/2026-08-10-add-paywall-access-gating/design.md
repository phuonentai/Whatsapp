## Context

The `paywall` package (`go-b2b-starter/internal/modules/paywall/`) is an adapter-pattern access gate, analogous to the `auth` package. It does not manage subscriptions — the billing module owns lifecycle and webhooks and exposes a `paywall.SubscriptionStatusProvider` adapter (`internal/modules/billing/infra/adapters/status_provider.go`). The paywall middleware only reads subscription status and either blocks or passes requests.

Routing stack per protected module (e.g. `cognitive/routes.go`):

```
auth → org_context → features.EntitlementMiddleware (403)
    → paywall RequireActiveSubscription (402) → credit guard (402)
```

Named middlewares are registered via `paywall.RegisterNamedMiddlewares` and consumed through the server's `MiddlewareResolver.Get("subscription")` in six modules: cognitive, documents, whatsapp, agent, campaigns, instagram.

## Goals / Non-Goals

**Goals:**
- Capture the implemented, test-verified paywall behavior as a living spec.
- Lock the 402 contract, status state machine, and lazy-guard recovery as normative requirements.
- Make the verification gate able to check paywall behavior.

**Non-Goals:**
- No code changes to `paywall`, the billing adapter, or any routes.
- No hardening of `parseStatusFromReason` (fragile reason-string matching) — out of scope, flagged as a risk.
- No subscription lifecycle / webhook behavior (owned by the billing module).
- No changes to `feature-gating`, `billing-provider-ux`, or `ai-usage-metering` specs.

## Decisions

### D1: New capability `paywall`, not a delta to existing specs

Paywall is the coarse "are you paying" gate; `feature-gating` is the fine "what plan unlocks" gate. They compose in the same route but answer different questions with different status codes (402 vs 403). Billing copy (`billing-provider-ux`) and credit ledger (`ai-usage-metering`) are downstream consumers, not the gate.

*Alternative rejected:* folding into `feature-gating` would conflate subscription state with plan-to-feature mapping and blur the 402/403 boundary.

### D2: Spec is purely descriptive (no behavior change)

The implementation already matches desired behavior and has e2e coverage elsewhere. This change adds normative requirements only.

### D3: Blocking middleware flow captured as normative sequence

The following flow in `middleware.go` `RequireActiveSubscription` becomes spec requirements:
1. OPTIONS → pass (CORS preflight).
2. Missing org in context → 500 `configuration_error` (misconfigured route; `RequireOrganization` must run first).
3. `GetSubscriptionStatus` error → 402 `subscription_required`, status `none`.
4. DB status inactive but status != `none` → `RefreshSubscriptionStatus` (lazy guard: heal missed webhook, grant if provider reports active, log occurrence).
5. Still inactive → 402 with status-specific body.
6. Active → set `SubscriptionStatus` in Gin context, pass.

### D4: 402 response contract is status-specific

```
status    error code               message
past_due  payment_failed           payment failed, update payment method
canceled  subscription_canceled    resubscribe to continue
unpaid    payment_required         update payment method
default   subscription_inactive    generic message / reason override
```

Body: `{ error, message, upgrade_url?, status? }` with `upgrade_url` defaulting to `/billing`.

### D5: Two middleware modes

- `paywall` (`RequireActiveSubscription`) — blocking. Also registered under legacy alias `subscription`.
- `paywall_optional` (`OptionalSubscriptionStatus`) — sets status in context when available, never blocks. Legacy alias `subscription_optional`.

### D6: Adapter contract separates fast path from refresh path

- `GetSubscriptionStatus` — local DB only, no external calls, for request hot path.
- `RefreshSubscriptionStatus` — provider sync, lazy-guard path.
- Status derivation in the billing adapter: `HasActiveSubscription=true` → `active`; reason `"no active subscription found"` → `none`; else `parseStatusFromReason`.

## Risks / Trade-offs

- [Status derivation via `parseStatusFromReason` relies on fragile substring matching of the human-readable `Reason` string] → Out of scope for this change; recorded in tasks.md as deferred hardening candidate. Spec describes the adapter contract at the interface level, not the string-parsing internals.
- [Lazy guard logs via `fmt.Printf` to stdout] → Monitoring signal, not a correctness issue; noted but not specified.
- [Legacy aliases `subscription`/`subscription_optional` risk divergence] → Spec requires both names to resolve to the same behavior; deprecation left to a future change.

## Migration Plan

N/A — specs only. No runtime, DB, or API change. Rollback = delete the change directory / do not archive.

## Open Questions

- Whether to later harden `parseStatusFromReason` and how statuses should flow structurally (typed field vs parsed reason). Deferred.
