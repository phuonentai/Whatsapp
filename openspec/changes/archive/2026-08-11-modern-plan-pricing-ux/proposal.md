## Why

The plans modal is a bare grid of cards with no comparison, no billing-interval toggle, no per-plan quantity clarity (seats, invoices, AI credits), and payment verification redirects to the subscription tab with no success feedback. For a Colombian SME buying via PSE/Nequi or card, the purchase decision and confirmation are the highest-stakes moments in the product. Modern SaaS (2026) pricing UX surfaces a comparison, makes quantity and credits legible, and confirms payment outcome.

## What Changes

- Add a plan comparison experience to the plans modal: feature rows across plans (seats, invoices, AI credits) in addition to cards, with a monthly/annual billing-interval toggle where the provider exposes both intervals.
- Make AI credits a first-class, visible line item on plan cards and comparison rows, sourced from Polar product metadata (`ai_credits_max`) — mirroring the existing AI credits meter in the subscription tab.
- Add post-checkout success and failure feedback on the subscription tab for the existing `payment_verified` / `payment_error` query params, which today redirect with no UI acknowledgement.
- Clarify payment-method and regional copy (Polar international card vs MercadoPago PSE/Nequi/Colombian card) within the comparison surface.
- Copy resolves through the typed copy layer from `standardize-spanish-first-copy` (Spanish-first).

## Capabilities

### New Capabilities
- `plan-pricing-ux`: plan comparison, billing-interval toggle, AI-credit line items, and checkout-outcome feedback on billing surfaces.

### Modified Capabilities
- (none) — pricing display changes live in the new capability; `ai-usage-metering` (the ledger contract) and `paywall` (subscription gating) are unchanged.

## Impact

- Frontend: `components/billing/plans-modal.tsx`, `components/billing/plans-comparison.tsx` (new), `components/billing/plan-card.tsx` (new/extracted), `app/dashboard/settings/components/subscription-tab.tsx` (checkout-outcome feedback), `lib/polar/plans.ts` (interval grouping helpers), copy layer additions.
- Backend: none. Plan data already flows from `/api/billing/products` (Polar product metadata includes `ai_credits_max`); no ledger, quota, or provider changes.
- Dependencies: typed copy layer from `standardize-spanish-first-copy`.
- Payment data handling: no card/token data touches this surface; checkout continues through Polar and MercadoPago, and no local credential or payment-instrument storage is introduced.
- Rollback: revert the frontend commit in Git; no Stytch tenant policy state or provider configuration is changed, so no external rollback applies.
- Non-Goals: no pricing/plan catalog changes on Polar, no new payment providers, no changes to the `paywall` 402 gating, no changes to the `ai-usage-metering` ledger contract, no local storage of payment instruments or credentials.
