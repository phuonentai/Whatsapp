## 1. Provider-aware state resolution [FE-NEXT]

- [x] 1.1 In `lib/polar/current-subscription.ts`, when `isMercadoPagoEnabled()` and the Polar path resolves inactive (or `POLAR_UNCONFIGURED`), fetch `GET /api/subscriptions/status` with the session JWT (pattern from `verify-payment.ts`)
  - Gate: `fetchBackendSubscriptionStatus(session.session_jwt)` → `GET ${GO_BACKEND}/api/subscriptions/status` with `Authorization: Bearer <session_jwt>`; null on unreachable/!ok (graceful degradation). Verified by `lib/polar/current-subscription.test.ts` (fetch tests incl. JWT header assertion).
- [x] 1.2 Map the backend response into `SubscriptionGateState`: active → `isActive=true`, `status="active"`, reason cleared; reason `subscription status: past_due` → `status="past_due"` + `reason="NO_ACTIVE_SUBSCRIPTION"`; `no active subscription found` → `status="none"` + `reason="NO_ACTIVE_SUBSCRIPTION"`; backend unreachable → keep the Polar result
  - Gate: pure `applyBackendStatus()` mapper; unit tests cover active / past-due / none (case-insensitive) / indecisive-keeps-Polar.
- [x] 1.3 Produce `reason="MP_UNCONFIGURED"` when MP is enabled but `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` is unset (wires the existing branch in `subscription-tab.tsx:76`)
  - Gate: fallback branch sets `reason = "MP_UNCONFIGURED"` before the backend call when the env plan id is missing; `subscription-tab.tsx` now renders provider-accurate amber-card copy (`mpConfigRequired`/`mpConfigDesc`) for that reason.
- [x] 1.4 Add unit tests for the mapping (active, past-due, none, backend-unreachable)
  - Gate: `lib/polar/current-subscription.test.ts` — 4 mapper cases + 3 fetch cases (network error → null, non-OK → null, OK → parsed + bearer header). `pnpm exec vitest run lib/polar/current-subscription.test.ts` → 7/7 pass.

## 2. Checkout seams [FE-NEXT]

- [x] 2.1 `create-mp-checkout.ts` resolves `plan_id` as `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` first, then `params.planId`, then `"default"`
  - Gate: `create-mp-checkout.test.ts` — env > params > default precedence, asserting the request body `plan_id`.
- [x] 2.2 `app/dashboard/page.tsx`: add `preapproval_id` to the searchParams contract; when present without `payment_id`, redirect to `/dashboard/settings?view=subscription` with no banner
  - Gate: `app/dashboard/page.test.tsx` — preapproval-only → redirect no banner + no verify call; preapproval+payment → verify runs; payment-only → unchanged; no params → home renders.
- [x] 2.3 Add/adjust component tests: env plan id precedence; preapproval-only callback routing
  - Gate: `lib/actions/billing/create-mp-checkout.test.ts` (4 tests) + `app/dashboard/page.test.tsx` (4 tests) — all pass.

## 3. Verification [OPS-GOV]

- [x] 3.1 Run `pnpm lint` and `pnpm build` — all pass
  - Gate: `pnpm lint` → 0 errors / 4 pre-existing warnings (baseline). `npx tsc --noEmit` → clean for this change (only transient `proxy.ts`/`proxy.test.ts` errors from the concurrent EdgeSessionFixer auth-session change, resolved before the final build). `pnpm build` → `✓ Compiled successfully` on the final attempt; see the 6.1 gate record for the full sequence.
- [x] 3.2 Run targeted tests: `plans-modal`, `subscription-tab`, new state-mapping tests
  - Gate: `pnpm exec vitest run` (full) → 47 files / 243 tests pass, incl. plans-modal (8), subscription-tab (7), current-subscription (7), plan-metadata (3), create-mp-checkout (4), dashboard page (4).
- [x] 3.3 Record verification results and archive decision in `tasks.md`
  - Gate: this file.

## 4. Entry-point and provider-CTA fixes [FE-NEXT]

- [x] 4.1 Pass `mercadopagoEnabled` into the layout-level `PlansModal` (`dashboard-layout.tsx:195-199`) so the "Subscribe now" entry point renders the MP option
  - Gate: `mercadopagoEnabled={isMercadoPagoEnabled()}` added to the layout `<PlansModal>`.
- [x] 4.2 When MP is enabled and Polar is unconfigured, promote the MP CTA to primary and demote/conditionally render the Polar button (`plans-modal.tsx:256-257`, `plan-card.tsx:76-103`)
  - Gate: modal tracks Polar-unconfigured (initial state `reason === "POLAR_UNCONFIGURED"` + the "Polar billing is not configured." action error); `mpPrimary` flips MP CTA to dark primary and Polar to outline secondary. Plans-modal test asserts MP option renders when enabled (2 cards) with primary styling and Polar demoted.
- [x] 4.3 `create-mp-checkout.ts` passes an explicit return URL (app origin + subscription view) to the backend; assert the backend `back_url` uses it (document `MERCADOPAGO_BACK_URL` in `example.env`)
  - Gate: body now `{ plan_id, back_url: "${appOrigin}/dashboard/settings?view=subscription" }`; `create-mp-checkout.test.ts` asserts the back_url; `MERCADOPAGO_BACK_URL` documented in `next_b2b_starter/.env.example` (backend default = app origin + subscription view; backend `mp_adapter.CreateCheckoutSession` applies `back_url` from config per `go-b2b-starter/internal/modules/billing/infra/mercadopago/mp_adapter.go:41`).

## 5. Cancel/resume semantics + post-checkout UX [FE-NEXT]

- [x] 5.1 Resume under MP routes to the MP resume path; keep Polar `cancelSubscription` only for Polar (`subscription-tab.tsx:554-560`)
  - Gate: `handleUpdateCancellation` now branches `mercadopagoEnabled ? cancelMPSubscription({ subscriptionId }) : cancelSubscription({ cancelAtPeriodEnd: cancel })` — Polar's action is never called under MP (it errors "No active subscription to update."). Note: backend ships only `mp-cancel` (no dedicated MP resume endpoint), so the MP path is the MP subscription action; the resume button is unreachable for MP-served orgs (Group 1 resolves them with `subscription: null`).
- [x] 5.2 Cancellation dialog body branches on `mercadopagoEnabled`: "immediate" for MP, end-of-period for Polar (`ui.billing.cancelDialogBody`/`scheduledToEndBody`)
  - Gate: new copy keys `cancelDialogBodyMp` / `scheduledToEndBodyMp` / `headsUpBodyMp` (es + en mirrors); dialog body, scheduled-to-end alert, and heads-up alert branch on `mercadopagoEnabled`.
- [x] 5.3 Clear `payment_verified`/`payment_error` after banner render (not only on dismiss) via `history.replaceState` (`subscription-tab.tsx:125-131,401-433`); add a test that refresh does not re-show the banner
  - Gate: mount effect strips the params via `history.replaceState` (preserving other params) after the banner renders; `subscription-tab.test.tsx` "clears checkout params from the URL after the banner renders, so refresh does not re-show it" → pass.
- [x] 5.4 `use-subscription-query` refetches on window focus and after checkout callback (`staleTime` stays for passive display) (`lib/hooks/queries/use-subscription-query.ts:27-38`)
  - Gate: `refetchOnWindowFocus: true`; the subscription-tab mount effect also triggers `onRefresh()` once when a checkout outcome param is present (covers the post-callback refetch).
- [x] 5.5 Coerce string-typed numeric metadata in `plan-card.tsx:106-109` and `create-checkout.ts:73-90` (`included_seats`/`included_invoices`/`ai_credits_max`); add a test with `"500"`-style values
  - Gate: shared `coerceNumericMetadata` (`lib/polar/plan-metadata.ts`) applied in `server-products.ts` + `create-checkout.ts` (seats/invoices) and `plan-card.tsx` `getAiCredits` (ai_credits_max). Tests: `plan-metadata.test.ts` ("500" → 500, etc.) + plans-modal test renders "500" from `{ ai_credits_max: "500" }` in card and comparison. NOTE: `server-products.ts` also updated (plans modal renders from it — required by spec "plans modal and checkout SHALL coerce").
- [x] 5.6 Route the dashboard billing alert strings (inactive/subscribe/upgrade) through the existing Spanish-first copy layer where touched (`dashboard-layout.tsx:239-383`)
  - Gate: all `deriveSubscriptionUiState` alert titles/bodies/actions + `getInactiveDescription` strings now read `ui.billing.*` (es + en mirror); `tpl()` used for `{used}/{included}/{remaining}/{date}` interpolation.

## 6. Verification [OPS-GOV]

- [x] 6.1 Re-run `pnpm lint`, `pnpm build`, and targeted component tests — all pass
  - Gate: `pnpm lint` → 0 errors / 4 pre-existing warnings (baseline). `pnpm build` → `✓ Compiled successfully` (final attempt after EdgeSessionFixer landed the `proxy.ts` fix; first attempt was blocked by `proxy.ts:203` `k.kid` on `JsonWebKey` + 23 `proxy.test.ts` errors from that in-flight auth-session change). `npx tsc --noEmit` → **0 errors** at final gate (clean for every file; the transient `proxy.ts`/`proxy.test.ts` errors from the concurrent auth-session change were resolved by EdgeSessionFixer). Targeted + full vitest → 47 files / 243 tests pass, incl. all new suites below.
- [x] 6.2 Record verification results and archive decision in `tasks.md`
  - Gate: this file.

**Archive deferred:** centralized verification phase per repo practice.
