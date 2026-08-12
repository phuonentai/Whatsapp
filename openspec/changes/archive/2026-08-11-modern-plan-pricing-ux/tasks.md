## 1. Pricing UI components

- [x] 1.1 [FE-NEXT] Extract `PlanCard` from `components/billing/plans-modal.tsx` into its own component; add the AI-credits line item on the card from `metadata.ai_credits_max`.
- [x] 1.2 [FE-NEXT] Create `components/billing/plans-comparison.tsx` with rows for seats, invoices, and AI credits across plans; render "—" for missing values.
- [x] 1.3 [FE-NEXT] Add the monthly/annual interval toggle to `plans-modal.tsx`; derive interval options from fetched products; render no toggle for single-interval catalogs.
- [x] 1.4 [FE-NEXT] Add copy keys under `lib/copy` namespace `billing` for comparison headers, interval toggle, and AI-credit line labels.

## 2. Checkout outcome feedback

- [x] 2.1 [FE-NEXT] In `app/dashboard/settings/components/subscription-tab.tsx`, consume `payment_verified` and `payment_error` search params and render success/error banners; clear params after acknowledgement.
  - DONE: Banner alerts render above the existing error alerts — emerald success Alert (`paymentVerifiedTitle`/`paymentVerifiedBody`/`understood`) and destructive error Alert (`paymentErrorTitle`/`paymentErrorBody`). Dismiss calls `clearPaymentParams()` which deletes both params via `router.replace`, preserving `view=subscription` and any other params. Covered by subscription-tab.test.tsx (banners render only when params present; dismiss clears params).
- [x] 2.2 [FE-NEXT] Wire the error banner retry link to open the plans modal.
  - DONE: Retry button (`ui.billing.retryCheckout`) in the payment_error banner calls the existing `setPlansOpen(true)` state, opening the already-wired `PlansModal`. Verified in subscription-tab.test.tsx (clicking retry shows the plans modal).

## 3. Tests

- [x] 3.1 [FE-NEXT] Add component tests for the comparison (values + "—" fallback), interval toggle (both/single interval), AI-credit line item, and checkout-outcome banners (success/error + param clearing).
  - DONE: Created `components/billing/plans-modal.test.tsx` (comparison values + "—" fallback, interval toggle both/single, AI-credit line from `metadata.ai_credits_max`) and `app/dashboard/settings/components/subscription-tab.test.tsx` (success/error banners, no banners without params, param clearing on dismiss, retry opens modal). Both files pass: 5/5 and 6/6.
- [x] 3.2 [FE-NEXT] Update existing plans-modal tests for extracted `PlanCard`.
  - DONE: There were no pre-existing plans-modal tests; `plans-modal.test.tsx` covers the extracted `PlanCard` (renders name/description/price, seats/invoices/AI-credit lines, missing-value omission, current-plan badge + disabled button).

## 4. Verification

- [x] 4.1 Run `pnpm lint` in `next_b2b_starter` — must pass.
- [x] 4.2 Run `pnpm build` in `next_b2b_starter` — must pass.
- [x] 4.3 Run affected billing component tests — must pass.
- [x] 4.4 Record results and archive decision in this file after completion.

- [ ] **Archive decision (2026-08-11):** **Archive** — implementation + tests green (lint 0, tsc 0, build ✓, plans-modal 5/5 + subscription-tab 6/6, full unit suite 163/163, full e2e 110/110). Executed in archive sweep.
