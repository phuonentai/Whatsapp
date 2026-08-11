## 1. Wizard extension

- [x] 1.1 [FE-NEXT] Extend `use-signup-flow.ts` with a `business` step (WhatsApp readiness, business goal) between `organization` and submit; extend step machine and `canContinue*` guards.
- [x] 1.2 [FE-NEXT] Add `components/onboarding/business-context-step.tsx` (or inline step in `app/signup/page.tsx`) rendering the prompts and persisting answers to localStorage.
- [x] 1.3 [FE-NEXT] Add copy keys under `lib/copy` namespace `onboarding` for the new step, labels, and helper text.
- [x] 1.4 [FE-NEXT] Update signup component tests to cover the new step and to assert the Stytch payload still excludes `owner_password`.

## 2. First-run checklist

- [x] 2.1 [FE-NEXT] Create `components/onboarding/first-run-checklist.tsx` with steps (connect WhatsApp, choose a plan, meet the assistant, open the inbox) and status derivation from `useWhatsAppConfigQuery`, `subscriptionState`, and inbox visit state.
- [x] 2.2 [FE-NEXT] Mount the checklist in `app/dashboard/components/dashboard-home.tsx` when completion < 100%; link each step to its surface.
- [x] 2.3 [FE-NEXT] Add component tests for checklist completion transitions (connected → done, subscribed → done, all → hidden).

## 3. Assistant introduction

- [x] 3.1 [FE-NEXT] Create a dismissible "Meet your assistant" panel component for new orgs explaining the copilot (drafts replies, human approves) and knowledge base, linking to agent settings; persist dismissal client-side.
- [x] 3.2 [FE-NEXT] Mount the intro on the dashboard for organizations that have not dismissed it.
- [x] 3.3 [FE-NEXT] Add component tests for render + dismiss persistence.

## 4. Verification

- [x] 4.1 Run `pnpm lint` in `next_b2b_starter` — must pass.
  - **Result (2026-08-11): PASS.** `eslint .` → 0 errors. Single pre-existing warning in `components/crm/deal-kanban.tsx:173` (unrelated to this change).
- [x] 4.2 Run `pnpm build` in `next_b2b_starter` — must pass.
  - **Result (2026-08-11): PASS.** `next build` completed; 15/15 static pages generated, all routes compiled including `/signup` and `/dashboard`.
- [x] 4.3 Run affected component test suites (signup, dashboard-home, checklist) — must pass.
  - **Result (2026-08-11): PASS.** Full suite `pnpm test` → 26 files / 110 tests passed. Affected suites: `app/signup/page.test.tsx` (3), `components/onboarding/first-run-checklist.test.tsx` (6), `components/onboarding/assistant-intro.test.tsx` (3), `app/dashboard/components/dashboard-home.test.tsx` (4), `lib/api/api/repositories/signup-repository.test.ts` (1), `lib/copy/ui.test.ts` (14).
- [x] 4.4 Confirm no `owner_password` in any wizard payload path (grep + signup test).
  - **Result (2026-08-11): PASS.** `grep -rn "owner_password"` over signup page, `use-signup-flow.ts`, onboarding components, storage helpers, and `signup-repository.ts` returns matches only in the negative test assertions of `app/signup/page.test.tsx`. Repository-level test asserts the `/auth/signup` DTO equals exactly `{ org_display_name, owner_email, owner_name, industry }` with no password key.
- [x] 4.5 Record results and archive decision in this file after completion.
  - **Recorded (2026-08-11).** All verification commands pass; change eligible for archiving.
