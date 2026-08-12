# New-Client Billing Lifecycle: Trial + Dunning Baseline

## Why

A brand-new org (`BootstrapOrganizationWithOwner`, `member_service_impl.go`) is created with **no subscription row, no trial, and no explicit free-plan state**: `GetQuotaStatus` errors, `GetBillingStatus` reports `none`, and the paywall 402s on the first gated call. There is also no late-payment process: `past_due` orgs are only hit with a 402 body ("update your payment method") whose `upgrade_url` defaults to `/billing` — a route that does not exist — and the frontend dunning alert routes the user into creating a **new** checkout instead of recovering the existing subscription. Modern 2026 SaaS (Stripe/Polar/Paddle) treats trials and dunning as first-class: trials auto-convert, failed payments get a retry schedule + dunning emails + a payment-method-update path, then a grace period before downgrade. This change establishes the trial and dunning baseline for new clients.

## What Changes

- **Trial seeding (optional, config-gated)**: when `TRIAL_ENABLED=true` (default off) and `TRIAL_DAYS` (default 14) are set, bootstrap SHALL create a local subscription row with status `trialing` and `current_period_end = now + TRIAL_DAYS` for the new org, plus a `quota_tracking` row granting **no quota** (`invoice_count = 0`, invoice processing blocked) while the paywall evaluates `trialing` as active. Seeding is idempotent (`ON CONFLICT (organization_id) DO NOTHING`) and goes through a narrow DI-injected `TrialSeeder` domain interface (organizations module never imports billing repositories directly). A retried bootstrap cannot duplicate rows, and a provider subscription that already exists is never overwritten by a trial row.
- **Trial expiry is enforced (no background job)**: the status path classifies `trialing AND now() > current_period_end` as expired (inactive) using the `current_period_end` already returned by `GetQuotaStatus`. This makes the paywall lazy guard fire at the trial boundary (the living paywall spec's "trialing boundary state" scenario): a provider-backed trial that auto-converted heals to active via provider refresh; a local-only trial fails closed to 402 `subscription_required` — no indefinite, invisible access. The refresh path never calls the provider with the synthetic `local-trial` customer ID (guard + fail-closed).
- **Free-plan state made explicit**: document (design + spec) that `none` is the free tier; no code change to gating, which already 402s with `subscription_required` and returns `StatusNone`.
- **Correct first-hit 402 contract**: `GetBillingStatus` currently swallows every `GetQuotaStatus` error and returns a nil-error `none` (`get_billing_status_service.go:14-23`), so the true no-subscription case falls into the generic `subscription_inactive` branch (contradicting the spec's `subscription_required`) and a transient DB failure becomes a 402 instead of a 500. The service SHALL surface the repository's `domain.ErrSubscriptionNotFound` sentinel **unwrapped** (the billing `/status` handler compares with direct equality) and propagate all other errors; the paywall middleware SHALL classify (sentinel → 402 `subscription_required` with status `none`; any other error → HTTP 500). This reuses the existing `ErrSubscriptionNotFound`/`paywall.ErrNoSubscription` pattern and requires a minimal, explicit middleware error-classification branch.
- **Onboarding surfaces surface upgrade prompts on 402**: `whatsapp-config-section.tsx` renders the raw 402 message as a destructive alert; onboarding SHALL recognize 402 and render an upgrade prompt (open the plans modal / billing tab) instead of the raw error, and the FirstRunChecklist step-1 (connect WhatsApp) deadlock SHALL be broken by surfacing the plan choice before the paywalled step.
- **Dunning paths**:
  - Paywall `upgrade_url` for `past_due`/`unpaid` SHALL point at a real route (`/dashboard/settings?view=subscription`), not the dead `/billing`.
  - Frontend `past_due`/`unpaid` alerts SHALL offer a payment-method-update path **distinct from resubscribing**: Polar orgs get the customer portal URL CTA when the subscription snapshot exposes one; **Mercado Pago in-place payment-method update is explicitly out of scope** in this change — MP orgs get honest copy (no misleading new-checkout CTA; resubscribe is labeled as creating a new subscription; provider auto-retry messaging) and the limitation is documented for a follow-up. These are specified in a new `billing-provider-ux` delta spec.
  - Grace messaging: reuse the existing `IsGracePeriod` entitlement state (features stay readable) and surface it in the settings subscription tab.
- **Comparison & follow-up design**: `design.md` documents the 2026 SaaS dunning/trial reference (retry cadence day 1/3/7, dunning emails, grace → downgrade, trial auto-convert) and scopes the follow-up work (scheduled retry/expiry job, transactional dunning emails, MP in-place payment-method update) as explicit non-goals of this change. Trial-expiry and trial-expired populations are observable via new queries (global monitoring + org-scoped variants).

## Revision 1 (council, 2026-08-12)

Addressed the six items from the first council verdict: error-classification contract (sentinel → 402, other → 500), idempotent trial seeding with synthetic NOT NULL placeholders, provider-appropriate payment-method update (Polar portal; MP out-of-scope with honest UX), LEFT JOIN correction, trial-expiry observability query, and aligned trial spec wording.

## Revision 2 (council re-review, 2026-08-12)

Addressed the re-review verdict on the revised design:

1. **[HIGH] Trial-expiry boundary fixed:** the status path classifies `trialing` with `now() > current_period_end` as expired (inactive), so the paywall lazy guard fires at the trial boundary — local-only trials fail closed to 402 `subscription_required`; provider trials auto-convert/heal via refresh. The false "lazy guard refreshes an un-expired trial" claim is deleted, and the refresh path guards against calling the provider with the synthetic `local-trial` customer ID (skip + fail-closed).
2. **[MED] `billing-provider-ux` delta spec added** (payment-method-update path distinct from resubscribe; grace copy) matching the declared Modified Capability.
3. **[MED] `InsertLocalTrialQuota` corrected** to the real `quota_tracking` columns (`invoice_count`, `max_seats`, `ai_credits_max`; the phantom `invoice_count_max` is removed).
4. **[MED] Sentinel contract pinned:** the service returns `domain.ErrSubscriptionNotFound` **unwrapped** (the `/status` handler uses direct equality); a 200-`none` regression test is added.
5. **[LOW] `ListExpiredTrials` aligned** (global monitoring + org-scoped `:one` variants; tenant-safe per-org wording) and trial seeding goes through a DI-injected `TrialSeeder` interface across the organizations→billing module boundary.

## Capabilities

### New Capabilities

- `new-client-billing`: trial seeding on signup (config-gated), enforced trial expiry, explicit free-plan (`none`) state, correct no-subscription 402 code, and dunning UX paths (payment-method update, grace messaging, real upgrade target) for new clients.

### Modified Capabilities

- `paywall`: `past_due`/`unpaid` 402 responses SHALL carry an upgrade URL that resolves to a real page; a no-subscription org SHALL receive `subscription_required` (not the generic inactive code) and DB errors SHALL surface as 500.
- `billing-provider-ux`: `past_due`/`unpaid` states SHALL surface a payment-method-update path distinct from subscribing anew, and grace-period state SHALL surface provider-appropriate copy; onboarding surfaces SHALL render an upgrade prompt on 402 instead of a raw error.

## Non-Goals

- A full dunning engine (scheduled retry/expiry job, dunning email campaign) — explicitly deferred; this change ships the UX paths, the enforced trial boundary, and the design.
- Trial auto-conversion with card capture (Polar handles trial→paid conversion on its side once a product has a trial configured; this change only makes the local state honor it — via the expiry boundary + lazy guard).
- Mercado Pago in-place payment-method update (updating the card on an existing MP subscription) — explicitly deferred; MP orgs receive honest, documented UX in this change.
- Changing the status set or paywall middleware semantics beyond the minimal error-classification branch.
- No local credential storage; no Stytch API contract changes.

## Rollback

- **Git**: revert; trial rows are only created when `TRIAL_ENABLED` is set, so reverting plus unsetting the flag is sufficient; an optional task deletes seeded trial rows for orgs created during the window.
- **Stytch**: untouched — no tenant policy changes.
- **Providers**: no Polar/MP API changes; dunning is purely local UX/config.

## Assumptions

- The Polar customer portal URL is exposed by the SDK subscription snapshot used by the billing module (to be confirmed during implementation; fallback is the plans modal — open question in design).
- MP auto-retry semantics (provider-side retry cadence) follow MP platform defaults; our copy reflects this without asserting exact MP API behavior.
- Provider-backed trial expiry semantics (does Polar report an unconverted expired trial as `canceled`/`inactive`?) follow the provider webhook contract; the lazy guard + provider refresh is the healing mechanism, with the local boundary as the fail-closed first line.
