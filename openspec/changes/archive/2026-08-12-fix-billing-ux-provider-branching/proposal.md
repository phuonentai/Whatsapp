## Why

The frontend subscription pipeline is Polar-only: `resolveCurrentSubscription` → `getActiveSubscription` queries the Polar SDK (`lib/polar/subscription.ts`), never the backend `/api/subscriptions/status` endpoint. A MercadoPago org therefore never resolves `isActive=true`: the dashboard "Subscribe now" alert and the FirstRunChecklist "choose a plan" step never clear, and the paywall redirect loop (`subscription-paywall.tsx` `router.replace("/dashboard")`) can never fire for a paying MP customer. Further provider-branching seams: the layout-level `PlansModal` (`dashboard-layout.tsx:195-199`) never receives `mercadopagoEnabled`, so the "Subscribe now" entry point renders no MP button at all; the Polar button is always the primary CTA (`plan-card.tsx:76-103`) even in MP-only deployments where it fails with "Polar billing is not configured."; `plans-modal` sends the **Polar** plan id to `create-mp-checkout` (MP needs its own env-mapped plan id); the dashboard callback ignores `preapproval_id` (MP returns it when a preapproval is authorized before the first payment is captured), so authorized-but-not-yet-paid customers get no verification path and no redirect; resume always calls Polar's `cancelSubscription` even under MP (`subscription-tab.tsx:554-560`); the MP cancellation dialog promises end-of-period access that MP's immediate `status: cancelled` PUT does not deliver; and the MP `back_url` default is `http://localhost:3000/dashboard` (`platform/mercadopago/config.go:31-32`) with no frontend-supplied return URL. Post-checkout UX also leaks: `payment_verified`/`payment_error` params are cleared only on banner dismissal, the subscription query never refetches on window focus or after checkout, and string-typed numeric product metadata (`"500"`) silently renders as "—".

## What Changes

- `lib/polar/current-subscription.ts` — when `isMercadoPagoEnabled()` and the Polar path resolves inactive (or `POLAR_UNCONFIGURED`), fall back to the backend `GET /api/subscriptions/status` (JWT bearer, same pattern as `verify-payment.ts`); map `has_active_subscription` and the reason-derived status (`past_due`, `canceled`, `unpaid`, `none`) into `SubscriptionGateState` (`isActive`, `status`, `reason`), leaving Polar-only fields (`subscription`, `usage`) null-safe. MP-active orgs then clear the paywall state; `NO_ACTIVE_SUBSCRIPTION` + `status: "past_due"` feeds the existing dunning alert in `dashboard-layout.tsx`.
- `lib/actions/billing/create-mp-checkout.ts:51` — prefer `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` over `params.planId` (env plan ids are the designed mapping; `params.planId` becomes a last resort before `"default"`).
- `app/dashboard/page.tsx` — accept `preapproval_id`; when present without a `payment_id`, redirect to `/dashboard/settings?view=subscription` (no error banner) and let the `subscription_authorized` webhook / later payment settle state.
- `app/dashboard/settings/components/subscription-tab.tsx:76` — align the dead `MP_UNCONFIGURED` branch: produce the reason when MP is enabled but its checkout plan id is unset, so the "billing not configured" card is truthful for MP-first deployments.
- `components/layout/dashboard-layout.tsx:195-199` — pass `mercadopagoEnabled` (from `isMercadoPagoEnabled()`) into the layout-level `PlansModal` so the "Subscribe now" entry point renders the MP checkout option.
- `components/plans-modal.tsx` / `plan-card.tsx` — when MP is enabled and Polar is unconfigured, the MP option SHALL be the primary CTA; the Polar button SHALL not be presented as the only or primary action in an MP-only deployment.
- `lib/actions/billing/create-mp-checkout.ts` — pass an explicit return URL (application origin + `/dashboard?view=subscription`) to the backend so MP `back_url` is never the `localhost` default in production.
- `app/dashboard/settings/components/subscription-tab.tsx:554-560` — resume (`cancel=false`) SHALL use the MP path (`resumeMPSubscription`) when MP is enabled, never Polar's `cancelSubscription`.
- Cancellation copy — the dialog body SHALL state provider-accurate timing: end-of-period for Polar, immediate for MP (`ui.billing.cancelDialogBody`/`scheduledToEndBody` branch on `mercadopagoEnabled`).
- `subscription-tab.tsx:125-131,401-433` — `payment_verified`/`payment_error` params SHALL be cleared after the banner renders (not only on dismissal), so refresh/back-nav does not re-show stale banners.
- `lib/hooks/queries/use-subscription-query.ts:27-38` — refetch on window focus and after a checkout callback so the tab reflects webhook/verify state without a manual "Actualizar estado".
- `plan-card.tsx:106-109` / `create-checkout.ts:73-90` — coerce string-typed numeric metadata (`"500"`) for `included_seats`/`included_invoices`/`ai_credits_max` so configured limits render instead of "—".

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `billing-provider-ux`: subscription state resolution SHALL be provider-aware — when MP is enabled and Polar reports no active subscription, the backend status endpoint SHALL be consulted before declaring the org inactive; the layout-level plans modal SHALL render the MP checkout option; cancellation copy and resume SHALL branch by provider; MP checkout SHALL return to the application origin.
- `plan-pricing-ux`: the MP checkout action SHALL use the configured public MP plan id (`NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID`) rather than the Polar plan id; the checkout callback SHALL handle `preapproval_id`; checkout result params SHALL be acknowledged once; product metadata SHALL tolerate string-typed numerics.

## Non-Goals

- Replacing the Polar SDK path or backend status endpoint behavior.
- New billing pages or redesigns; no new copy strings (existing hardcoded dashboard billing strings are routed through the existing Spanish-first copy layer where touched).
- No local credential storage; only the session JWT is reused for the backend call.

## Rollback

- **Git**: revert the frontend files; no migration.
- **Stytch**: untouched; the change only adds a server-side call authenticated with the existing session JWT.
- **Backend**: no backend change ships with this change, so no runtime rollback is required.
