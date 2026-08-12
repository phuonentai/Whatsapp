## Context

The frontend resolves subscription state exclusively through the Polar SDK: `resolveCurrentSubscription` → `getActiveSubscription` (`lib/polar/subscription.ts`) lists Polar subscriptions by the Stytch org id. The backend's provider-agnostic `GET /api/subscriptions/status` (reads the local DB, which both providers update via webhook/verify) exists but is unused by the FE. Consequences for MP orgs: `isActive` is always false, the dashboard "Subscribe now" alert and FirstRunChecklist billing step never clear, and `subscription-paywall`'s auto-redirect can never fire. Two smaller seams: `plans-modal.handleMPCheckout` sends the Polar plan id to `createMercadoPagoCheckout`, bypassing the env-mapped MP plan id (`NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID`), and the dashboard callback ignores `preapproval_id`, which MP returns when a preapproval is authorized before the first payment is captured.

## Goals / Non-Goals

**Goals:**
- MP orgs resolve `isActive` correctly and clear paywall/alerts
- MP checkout uses the configured MP plan id; preapproval-only returns land on the subscription view without a false error
- Remove the dead `MP_UNCONFIGURED` branch in `subscription-tab`

**Non-Goals:**
- Replacing the Polar SDK path, backend status endpoint behavior, or copy (already Spanish-first)
- New billing pages or a billing redesign

## Decisions

### 1. Backend status as the MP fallback in `resolveCurrentSubscription`

When `isMercadoPagoEnabled()` is true and the Polar path resolves inactive (including `POLAR_UNCONFIGURED`, `CUSTOMER_NOT_FOUND`, `NO_ACTIVE_SUBSCRIPTION`), call `GET /api/subscriptions/status` with the session JWT (same pattern as `verify-payment.ts`). Map:
- `has_active_subscription=true` → `isActive=true`, `status="active"`, `reason` cleared; Polar-only fields (`subscription`, `usage`) stay `null` — downstream code is null-safe (`plans-modal` `currentProductId ?? null`, `subscription-tab` guards on `state?.subscription?.`).
- inactive + reason `subscription status: past_due` → `status="past_due"`, `reason="NO_ACTIVE_SUBSCRIPTION"`, which drives the existing dunning alert in `dashboard-layout.tsx` (`deriveSubscriptionUiState`).
- inactive + `no active subscription found` → `status="none"`, `reason="NO_ACTIVE_SUBSCRIPTION"`.
- Backend unreachable → keep the Polar result (graceful degradation).

**Alternatives considered:**
- Backend-first for all orgs: cleaner single source but drops Polar-only enrichments (meter usage, trial dates, plan) that the tab displays — rejected; Polar stays primary, backend is the MP fallback.
- MP SDK subscription lookup from the FE: requires MP public-key/credential surface in the FE — rejected (secrets stay backend-only per the env split).

### 2. Env plan id wins in the MP checkout action

`create-mp-checkout.ts` resolves `plan_id` as `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` first, then `params.planId`, then `"default"`. This matches the design intent ("MP plans are mapped by env plan IDs") and stops sending Polar product ids to the MP preapproval API.

**Alternatives considered:**
- Map Polar plan id → MP plan id in the modal: needs a catalog the FE doesn't have — rejected.

### 3. `preapproval_id` callback routing

`app/dashboard/page.tsx` adds `preapproval_id` to the searchParams contract; when present without `payment_id`, redirect to `/dashboard/settings?view=subscription` with no banner. State settles via the `subscription_authorized` webhook or a later payment; `verifyMercadoPagoPayment` needs a real payment id and must not be called with a preapproval id.

### 4. Wire `MP_UNCONFIGURED`

`subscription-tab.tsx` already treats `MP_UNCONFIGURED` as "billing not configured"; `resolveCurrentSubscription` now produces it when MP is enabled but `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` is unset, so the amber "config required" card renders truthfully for MP-first deployments.

### 5. Layout modal gets the MP option; MP-first CTA ordering

The layout-level `PlansModal` is the dashboard's main "Subscribe now" surface and currently omits `mercadopagoEnabled` (defaults `false`), so MP deployments dead-end there. It receives the flag and renders the MP checkout option; when MP is enabled and the Polar action is known to be unconfigured (server action surfaces "Polar billing is not configured."), the MP CTA becomes primary and Polar is demoted, so an MP-only client never hits a failing primary button.

### 6. Provider-accurate cancel/resume semantics

Backend `CancelMPSubscription` PUTs `status: cancelled` (immediate) and stores `CancelAtPeriodEnd: false`, while the shared dialog copy promises end-of-period access — an MP client loses access immediately despite the promise. Under MP: resume routes to the MP path (the current code calls Polar's `cancelSubscription`, which errors with "No active subscription to update."), and the dialog body states cancellation is immediate. Polar behavior is untouched.

### 7. Post-checkout hygiene

- `payment_verified`/`payment_error` are cleared after the banner renders via `history.replaceState`, not only on dismissal — refresh/back-nav no longer re-shows stale banners.
- `use-subscription-query` refetches on window focus and after the checkout callback (passive `staleTime` for display, active refetch for truth) so a payment made in another tab shows up without "Actualizar estado".
- String-typed numeric metadata (`"500"`) is coerced in `plan-card` and `create-checkout`; without coercion, configured quotas silently render as "—" and checkouts forward no quota.
- Touched dashboard billing strings move to the existing Spanish-first copy layer (`ui.billing`), matching the settings tab voice.

## Risks / Trade-offs

- **[Risk] Backend status endpoint requires `resource:view` permission and org context** → The admin owner has it; non-owners already short-circuit via `INSUFFICIENT_PERMISSIONS` before the fallback runs.
- **[Risk] Null Polar snapshot degrades tab metrics (renews date, usage bars) for MP orgs** → Accepted: subscription-tab already guards nulls; a follow-up can surface MP period dates from the backend status if needed.
- **[Risk] MP_UNCONFIGURED fires for orgs that actually pay via Polar while MP is globally enabled** → The backend status is authoritative; an org with a Polar-active row reports active before the unconfigured branch is evaluated.
- **[Risk] Detecting "Polar unconfigured" from the FE** → The server action error string is the only signal today; the env split keeps Polar credentials backend-only, so the FE checks MP-enablement and treats a failing Polar action as the unconfigured case (fallback to MP CTA).

## Migration Plan

1. `current-subscription.ts` fallback + status/reason mapping; unit-test the mapping.
2. `create-mp-checkout.ts` plan-id precedence; `dashboard/page.tsx` preapproval routing; `subscription-tab` MP_UNCONFIGURED wiring.
3. Layout modal `mercadopagoEnabled` + CTA ordering; cancel/resume branching + copy; return URL pass-through.
4. Post-checkout hygiene: param cleanup, focus refetch, metadata coercion, copy-layer routing.
5. `pnpm lint`, `pnpm build`, targeted component tests.
6. Rollback: revert FE files; no backend or Stytch impact.
