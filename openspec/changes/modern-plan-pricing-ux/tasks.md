## 1. Pricing UI components

- [ ] 1.1 [FE-NEXT] Extract `PlanCard` from `components/billing/plans-modal.tsx` into its own component; add the AI-credits line item on the card from `metadata.ai_credits_max`.
- [ ] 1.2 [FE-NEXT] Create `components/billing/plans-comparison.tsx` with rows for seats, invoices, and AI credits across plans; render "—" for missing values.
- [ ] 1.3 [FE-NEXT] Add the monthly/annual interval toggle to `plans-modal.tsx`; derive interval options from fetched products; render no toggle for single-interval catalogs.
- [ ] 1.4 [FE-NEXT] Add copy keys under `lib/copy` namespace `billing` for comparison headers, interval toggle, and AI-credit line labels.

## 2. Checkout outcome feedback

- [ ] 2.1 [FE-NEXT] In `app/dashboard/settings/components/subscription-tab.tsx`, consume `payment_verified` and `payment_error` search params and render success/error banners; clear params after acknowledgement.
- [ ] 2.2 [FE-NEXT] Wire the error banner retry link to open the plans modal.

## 3. Tests

- [ ] 3.1 [FE-NEXT] Add component tests for the comparison (values + "—" fallback), interval toggle (both/single interval), AI-credit line item, and checkout-outcome banners (success/error + param clearing).
- [ ] 3.2 [FE-NEXT] Update existing plans-modal tests for extracted `PlanCard`.

## 4. Verification

- [ ] 4.1 Run `pnpm lint` in `next_b2b_starter` — must pass.
- [ ] 4.2 Run `pnpm build` in `next_b2b_starter` — must pass.
- [ ] 4.3 Run affected billing component tests — must pass.
- [ ] 4.4 Record results and archive decision in this file after completion.
