# New-Client Billing Lifecycle: Trial + Dunning Baseline — Design

## Context

New organizations are bootstrapped (`member_service_impl.go`, `BootstrapOrganizationWithOwner`) with auth org, local org, owner member, and admin role — and **no subscription row**: `GetQuotaStatus` returns no row, `GetBillingStatus` reports a nil-error `none`, and the first gated request gets 402 `subscription_inactive`. There is no trial and no late-payment process: `past_due`/`unpaid` orgs get a 402 whose default `upgrade_url` is `/billing` (a route that does not exist — verified: no FE `/billing` app; `/dashboard/settings` exists), and the frontend dunning alert routes them into creating a **new** checkout instead of updating the payment method on the existing subscription.

This revision incorporates both council verdicts (`VERDICT.md`: initial REJECTED 2026-08-12, and re-review REJECTED 2026-08-12). All required design changes are folded in; premises were re-verified against the codebase.

**Verified premises (codebase):**

- `GetBillingStatus` swallows ALL `GetQuotaStatus` errors and returns a nil-error inactive status with `Reason: "no active subscription found"` (`get_billing_status_service.go:14-23`). `RefreshSubscriptionStatus` swallows the no-subscription case the same way.
- The billing repository already translates `sql.ErrNoRows` → `domain.ErrSubscriptionNotFound` (`subscription_repository.go:126-127`); `domain.QuotaStatus` carries `CurrentPeriodEnd` (`domain/types.go:43`, mapped at `subscription_repository.go:195`).
- The billing `/status` handler consumes the sentinel with **direct equality** (`handler.go:64`: `err == domain.ErrSubscriptionNotFound` → 200 `none`; other errors → 500) — the service MUST return the sentinel unwrapped.
- `GetQuotaStatus` is a **LEFT JOIN** (`subscription_billing.sql:116-135`) with zeroed quota defaults; it errors via `sql.ErrNoRows` only when the org has no subscription row, and also on genuine DB failures.
- The paywall middleware error branch maps ANY provider error to 402 `subscription_required` (`middleware.go:97-109`) — no classification; the `StatusNone` path falls into `buildErrorResponse`'s default → 402 `subscription_inactive`.
- **The lazy guard fires only when `!IsActive && Status != StatusNone`** (`middleware.go`). `paywall.IsActiveStatus("trialing")` is `true` and the adapter reports any `HasActiveSubscription=true` status as `StatusActive` — so **the lazy guard can never fire for a `trialing` org today**. The living paywall spec's "Lazy guard refreshes status on boundary" requirement names the "`trialing` boundary state" explicitly — today the spec and code disagree.
- `paywall.ErrNoSubscription` already exists (`paywall/errors.go`).
- Both `subscription_billing.subscriptions` and `quota_tracking` have `organization_id INT NOT NULL UNIQUE` (migration 000004) — `ON CONFLICT (organization_id)` is valid. `quota_tracking` real columns: `invoice_count`, `max_seats`, `ai_credits_max`, `period_start`, `period_end`, `last_synced_at` (**no `invoice_count_max`**). `subscriptions` has NOT NULL `external_customer_id`, `subscription_id` (UNIQUE), `product_id` — trial rows need synthetic placeholders.
- `ListActiveSubscriptions` filters `status='active'` only (no trial-expiry visibility).
- `IsGracePeriod = (status == "past_due")` with features parsed as `isActive || isGracePeriod` (`billing_provider.go`).
- FE raw-402 renders verified: `whatsapp-config-section.tsx` (~293-301) destructive Alert with `error.message` + retry; first-run checklist has `connectWhatsApp` as step 1 and `choosePlan` as step 2; WhatsApp mgmt routes gated by the subscription middleware + `org:manage`. Sibling `fix-billing-ux-provider-branching` provides `deriveSubscriptionUiState` in `dashboard-layout.tsx`.
- Neither the Polar nor the Mercado Pago adapter exposes a payment-method-update/portal capability (verified; the domain `BillingProvider` interface has no such method).

## Goals / Non-Goals

**Goals:**

- Config-gated trial seeding at signup so new orgs get real trial access instead of an immediate 402; seeding is idempotent and module-boundary-clean.
- **Enforced trial expiry at the status boundary** — local-only trials fail closed; provider trials heal via the lazy guard.
- Explicit free-tier (`none`) semantics documented.
- Correct first-hit 402 contract: `subscription_required` for no-subscription orgs (via the existing sentinel), HTTP 500 for DB failures — never a misleading 402.
- Onboarding surfaces render upgrade prompts on 402 instead of raw errors.
- Dunning UX paths: real upgrade target, payment-method-update distinct from resubscribe (provider-appropriate), grace messaging — with a `billing-provider-ux` delta spec.
- Trial-expiry population observable (global + org-scoped queries).

**Non-Goals:**

- Full dunning engine (scheduled retry/expiry job, dunning email campaign) — deferred, designed here.
- Trial auto-conversion with card capture (provider-side).
- Mercado Pago in-place payment-method update — deferred, documented; MP orgs get honest UX.
- Status-set changes, paywall middleware semantics beyond the minimal error-classification branch, credential storage, Stytch API changes.

## Decisions

### D1 — Error-classification contract (council #1 / #4)

**Service layer** (`get_billing_status_service.go`, `refresh_subscription_status_service.go`):

- `GetBillingStatus`: the repository already returns `domain.ErrSubscriptionNotFound` for missing rows — the service SHALL propagate it **unwrapped** (the `/status` handler uses `err == domain.ErrSubscriptionNotFound`, direct equality) and propagate all other errors wrapped. Remove the nil-error `none` swallow.
- `RefreshSubscriptionStatus`: same classification — propagate `domain.ErrSubscriptionNotFound` instead of fabricating a nil-error `none`.

**Adapter layer** (`infra/adapters/status_provider.go`): replace the reason-string match with sentinel logic — translate `domain.ErrSubscriptionNotFound` to `paywall.ErrNoSubscription` (already defined in `paywall/errors.go`); all other errors propagate unchanged.

**Middleware layer** (`internal/modules/paywall/middleware.go`): the provider-error branch classifies — `errors.Is(err, paywall.ErrNoSubscription)` → 402 `subscription_required`, status `none`, `upgrade_url`; any other error → HTTP 500 with a non-subscription body (never 402). This is a minimal, explicit change to the middleware error branch; the original "no middleware semantics change" claim is corrected.

**Regression pin:** `GET /api/subscriptions/status` for a no-subscription org continues to return 200 `none` (the handler's existing sentinel branch); a test asserts this after the service change.

### D2 — Idempotent trial seeding with module boundary (council #2 / re-review #3, #5)

New envs `TRIAL_ENABLED` (default `false`) and `TRIAL_DAYS` (default `14`). Trial seeding flows through a narrow DI-injected domain interface so the organizations module never imports billing repositories:

- **`TrialSeeder` domain interface** (in the organizations module domain): `SeedTrial(ctx, organizationID int32, trialEnd time.Time) error`, implemented by billing infrastructure and injected via dig. This keeps module boundaries per governance (organizations → billing via interface, not imports).

The implementation uses two new idempotent SQLC queries:

- `InsertLocalTrial :one` — `INSERT INTO subscription_billing.subscriptions (organization_id, external_customer_id, subscription_id, subscription_status, product_id, product_name, plan_name, current_period_start, current_period_end, metadata)` with synthetic placeholders for the NOT NULL columns: `external_customer_id = 'local-trial'`, `subscription_id = 'local-trial-' || organization_id` (unique per org), `product_id = 'trial'`; `subscription_status = 'trialing'`, `current_period_start = now()`, `current_period_end = $trialEnd`, `metadata = '{}'`. `ON CONFLICT (organization_id) DO NOTHING` — never overwrites an existing row (provider row wins; bootstrap retry cannot duplicate).
- `InsertLocalTrialQuota :one` — `INSERT INTO subscription_billing.quota_tracking (organization_id, invoice_count, period_start, period_end) VALUES ($1, 0, now(), $trialEnd) ON CONFLICT (organization_id) DO NOTHING` — real columns only (`max_seats`/`ai_credits_max` default NULL); **no quota granted** (`invoice_count = 0` count-down system → `can_process_invoice = false` while `trialing` passes the paywall).

Both inserts run within the bootstrap flow's existing rollback stack (both idempotent, so a partial failure + retry is safe).

### D3 — Real upgrade target for dunning responses

`DefaultMiddlewareConfig.UpgradeURL` becomes `/dashboard/settings?view=subscription` (the existing subscription tab), replacing the dead `/billing`. Callers may still override via `MiddlewareConfig`.

### D4 — Payment-method-update paths (council #3, `billing-provider-ux` delta)

Provider-appropriate, distinct from resubscribing:

- **Polar:** the `past_due`/`unpaid` alert gains an "Update payment method" CTA using the Polar customer portal URL when the subscription snapshot exposes one (open question — confirmed during implementation); when no portal URL is available, fall back to the plans modal. The settings subscription tab gains grace-period copy (`IsGracePeriod` from the entitlement).
- **Mercado Pago: in-place payment-method update is explicitly OUT OF SCOPE.** The MP adapter exposes no card/portal-update capability (verified). The MP dunning alert therefore does NOT present a new-checkout CTA as a payment-method update; it shows honest copy (provider auto-retry messaging; explicit statement that in-app payment-method update is not yet available for MP; resubscribe labeled as creating a new subscription). The limitation is documented for the follow-up.

### D5 — Onboarding 402 surfacing

- `whatsapp-config-section.tsx`: detect 402 and render an upgrade prompt (open plans modal / billing subscription view) instead of the raw "active subscription is required" destructive alert.
- First-run checklist: surface the plan choice (`choosePlan`) before the paywalled step (`connectWhatsApp`) when the subscription is inactive.

### D6 — Trial expiry is enforced at the status boundary (re-review #1, HIGH)

**Expiry classification (service, no background job):** in `GetBillingStatus`, after obtaining the quota status, classify `quotaStatus.SubscriptionStatus == "trialing" && time.Now().After(quotaStatus.CurrentPeriodEnd)` as expired: `HasActiveSubscription = false`, `Reason: "trial expired"`. `CurrentPeriodEnd` is already present on `domain.QuotaStatus` (verified).

**Consequence — the lazy guard now fires at the trial boundary:** an expired trial is an inactive-but-not-`none` status, which is exactly the lazy guard's trigger condition (`!IsActive && Status != StatusNone`). This makes the living paywall spec's "Lazy guard refreshes status on boundary" requirement (which names the "`trialing` boundary state") actually implementable:

- **Provider-backed trial auto-converted:** `RefreshSubscriptionStatus` → `SyncSubscriptionFromPolar` → provider reports active → access granted, status healed to active.
- **Local-only trial (no provider subscription):** fail closed → 402 `subscription_required` (via the `ErrSubscriptionNotFound`/`paywall.ErrNoSubscription` classification). No indefinite access.

**Synthetic-customer guard in the refresh path:** before `SyncSubscriptionFromPolar`, detect synthetic local-trial rows (`subscription_id LIKE 'local-trial-%'`); skip the provider call entirely and fail closed by returning `domain.ErrSubscriptionNotFound` — the provider is NEVER called with the synthetic `external_customer_id = 'local-trial'`. Unit test asserts no provider call is made for synthetic rows.

**Corrected claim:** the earlier design said "the lazy guard refreshes an un-expired trial" — false (the guard can never fire while trialing evaluates as active). Until the boundary is crossed, the trial is active by design; enforcement happens AT the boundary via the classification above. D6 and Risks reflect the real mechanism.

### D7 — Trial-expiry observability (re-review #5)

Two SQLC queries:

- `ListExpiredTrials :many` — `WHERE subscription_status = 'trialing' AND current_period_end < now()` across organizations, ordered by org (monitoring / deferred scanner).
- `ListExpiredTrialByOrg :one` — the same predicate scoped to a single `organization_id` (tenant-safe per-org variant; at most one row since `organization_id` is UNIQUE).

### D8 — 2026 dunning reference (follow-up scope)

Deferred engine, designed now: provider-side retry cadence is the first line (Polar/MP auto-retry failed payments); the product layer adds (a) a scheduled job scanning `past_due` and trialing-expired orgs (`ListActiveSubscriptions`, `ListQuotasNearLimit`, `ListExpiredTrials`), (b) transactional dunning emails at day 1/3/7 using the existing SMTP env config, (c) a downgrade-to-free transition after a grace window, (d) trial-expiry nudges, (e) MP in-place payment-method update. Each is a separate future change; this change ships the UX paths, the enforced trial boundary, and the observability queries.

## Risks / Trade-offs

- **[Risk] Local-only trial rows diverge from the provider (no provider-side trial)** → Mitigation: `TRIAL_ENABLED` defaults off; docs recommend pairing with provider-side product trials; the expiry boundary fails closed (never indefinite access); `ON CONFLICT DO NOTHING` keeps provider rows authoritative.
- **[Risk] Expiry boundary time-source skew** (app clock vs provider) → Mitigation: the boundary is the fail-closed first line; the provider refresh is authoritative for provider-backed trials; `current_period_end` comes from the DB row written at seed time.
- **[Risk] Changing `UpgradeURL` default alters every 402 body** → Mitigation: the new default is a real route; callers may still override via `MiddlewareConfig`.
- **[Risk] Middleware error-branch change could 500 on unexpected provider errors** → Mitigation: classification is explicit (sentinel → 402, other → 500); matches the billing handler's existing behavior; unit tests cover both branches and the `/status` 200-`none` regression pin.
- **[Risk] Polar portal URL may be absent from the SDK snapshot** → Mitigation: fallback to the plans modal; assumption recorded.
- **[Risk] MP orgs have no in-app PM-update path** → Mitigation: honest, documented UX; provider auto-retry copy; follow-up scoped.
- **[Risk] Organizations → billing coupling in bootstrap** → Mitigation: narrow `TrialSeeder` domain interface, DI-injected; organizations module never imports billing repositories.

## Migration Plan

1. Service + adapter + middleware classification (unwrapped sentinel pin + `/status` regression test).
2. `TRIAL_ENABLED`/`TRIAL_DAYS` config + `TrialSeeder` interface + `InsertLocalTrial`/`InsertLocalTrialQuota` + bootstrap wiring; documented in `example.env`.
3. Trial-expiry classification in `GetBillingStatus` + synthetic-customer guard in `RefreshSubscriptionStatus`.
4. `UpgradeURL` default change; FE alert CTA (Polar portal / MP honest copy) + grace copy; onboarding 402 prompts; checklist ordering.
5. `ListExpiredTrials` (global) + `ListExpiredTrialByOrg` (org-scoped) queries.
6. `make test` (backend), `pnpm lint` + targeted tests (FE).
7. Optional cleanup task: delete seeded trial rows for orgs created during the window on rollback.
8. Rollback: revert + unset `TRIAL_ENABLED`; no migration; no Stytch/provider impact.

## Open Questions

- Whether the Polar customer portal URL is exposed by the SDK snapshot used by the billing module — confirm during implementation; fallback is the plans modal.
- Provider-backed trial expiry semantics (what does Polar report for an unconverted expired trial?) — the provider refresh is the healing mechanism; the local boundary is the fail-closed first line; record findings during E2E.

## Testing Strategy

- Go unit: `GetBillingStatus` classification (`ErrNoRows` → `ErrSubscriptionNotFound` unwrapped; other error propagates; expired-trial classification); `/status` handler 200-`none` regression; middleware classification (sentinel → 402 `subscription_required` + status `none`; other error → 500, never 402); adapter sentinel translation; trial seeding idempotency (retry → single row; provider row not overwritten); **trial-expiry boundary** — (a) local-only trial expired → denied, (b) provider trial expired then auto-converted → lazy guard heals to active, (c) refresh skips the provider call and fails closed on synthetic `local-trial` customer IDs; `ListExpiredTrials`/`ListExpiredTrialByOrg` queries.
- FE component: 402 renders upgrade CTA in `whatsapp-config-section`; checklist ordering with paywalled step; `past_due` alert renders Polar portal CTA when available and honest MP copy otherwise; grace copy in the subscription tab.
- E2E (where feasible): new org with `TRIAL_ENABLED=true` passes the paywall; trial expiry → 402 for local-only org; no-subscription org gets `subscription_required`; simulated DB failure gets 500.
