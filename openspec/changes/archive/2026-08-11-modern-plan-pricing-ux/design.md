## Context

`PlansModal` renders a card grid from products fetched via `/api/billing/products` (`useProductsQuery`). `PolarPlan` carries `interval` (month/year), `includedSeats`, `includedInvoices`, `benefits`, `badge`, `metadata`. The subscription tab already renders invoice-usage and AI-credit meters from `useAiUsageQuery`. Dashboard redirects post-checkout to `settings?view=subscription&payment_verified=true` (or `payment_error=true`) but the subscription tab does not acknowledge either param.

## Goals / Non-Goals

**Goals:**
- Comparison UX across plans (quantities and AI credits legible).
- Monthly/annual toggle when both intervals exist.
- Post-checkout success/error feedback.
- All copy Spanish-first via the copy layer.

**Non-Goals:**
- No Polar catalog/pricing changes.
- No new providers; no payment-instrument storage.
- No changes to `paywall` gating or `ai-usage-metering` ledger.

## Decisions

1. **Comparison grid + cards.** Keep cards as the primary choice surface; add a comparison table below/alongside that rows up included seats, invoices, and AI credits. Reuse `PolarPlan.metadata.ai_credits_max` for the AI-credit row; if absent for a plan, render "—".
2. **Interval toggle when data supports it.** If the product set contains both `month` and `year` intervals for the same product family, render a toggle that re-filters `useProductsQuery` output into the selected interval. Single-interval catalogs render without a toggle (no dead control).
3. **Checkout outcome feedback.** In `subscription-tab.tsx`, consume `payment_verified` → success banner ("Pago confirmado — tu plan está activo"); `payment_error` → error banner with retry link to the plans modal. Clear the param after acknowledgement to avoid repeat banners.
4. **Extract `PlanCard`** into its own component for reuse and testability; add `PlansComparison` component.
5. **AI credits line item** mirrors the meter in the subscription tab so the "buy" view and the "you're using it" view speak the same language (credits used / included per period).
6. **Copy layer** namespace `billing` extended; no new backend.

## Risks / Trade-offs

- **Metadata gaps:** some plans may lack `ai_credits_max`; render "—" rather than inventing numbers.
- **Interval toggle adds state complexity** to the modal; mitigations: derive interval options from fetched data, never hardcode, and disable toggle with zero options rather than hiding data.
- **Repeated success banner** if param not cleared; mitigation: `router.replace` after acknowledging.
- **Dependency risk:** copy layer from `standardize-spanish-first-copy` must land first.
