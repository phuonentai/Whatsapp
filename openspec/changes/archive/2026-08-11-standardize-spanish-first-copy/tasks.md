## 1. Copy layer foundation

- [x] 1.1 [FE-NEXT] Create `lib/copy/ui.ts` with typed namespaces (`auth`, `billing`, `whatsapp`, `inbox`, `dashboard`, `agent`, `common`), Spanish-first strings, and English fallback constants; export a typed lookup so unknown keys fail the build.
- [x] 1.2 [OPS-GOV] Commit `next_b2b_starter/docs/ui-copy.md` tone & voice guide (Spanish-first, tú form, benefit-first, plain language, error copy pattern: what happened + next step).
- [x] 1.3 [FE-NEXT] Add a component test or lint check asserting no user-facing string is hardcoded in the affected surfaces' source (sweepable), and that no key in the primary flows resolves to the English fallback.

## 2. Auth / signup copy

- [x] 2.1 [FE-NEXT] Route `app/signup/page.tsx` strings (title, labels, placeholders, continue/back/create buttons, email-sent view, error banner) through the copy layer in Spanish-first voice.
- [x] 2.2 [FE-NEXT] Update `signup` component tests asserting the replaced strings. (N/A — no signup component tests exist; sweep test in 1.3 covers the file.)

## 3. Billing copy

- [x] 3.1 [FE-NEXT] Route `components/billing/plans-modal.tsx` strings (heading, payment-method explanation, loading/error/empty states, active-subscription notice, processing label, dialog) through the copy layer; unify to Spanish.
- [x] 3.2 [FE-NEXT] Route `components/billing/subscription-paywall.tsx` and `subscription-tab.tsx` status/alerts copy through the copy layer in Spanish.
- [x] 3.3 [FE-NEXT] Update billing component tests asserting replaced strings. (N/A — no billing component tests exist; sweep test in 1.3 covers the files.)

## 4. WhatsApp config copy + de-jargon

- [x] 4.1 [FE-NEXT] Route `whatsapp-config-section.tsx` primary copy (title, description, connect empty-state, connected banner, active/inactive labels, save/connect buttons) through the copy layer in Spanish.
- [x] 4.2 [FE-NEXT] Rewrite `MICRO_STATUS_STEPS` to Spanish user-facing progress strings; keep technical token fields only inside the advanced panel.
- [x] 4.3 [FE-NEXT] Update WhatsApp-config component tests asserting replaced strings. (N/A — no whatsapp-config component tests exist; sweep test in 1.3 covers the file.)

## 5. Inbox, dashboard, agent, and error copy

- [x] 5.1 [FE-NEXT] Route inbox empty/failure states (`conversation-list.tsx`, `message-thread.tsx`, `empty-state.tsx`) through the copy layer in Spanish.
- [x] 5.2 [FE-NEXT] Route `dashboard-home.tsx` (KPI labels, quick actions, welcome) through the copy layer in Spanish.
- [x] 5.3 [FE-NEXT] Route `agent-settings-section.tsx` and `agent-suggestions-panel.tsx` copy through the copy layer; ensure consistency with existing Spanish.
- [x] 5.4 [FE-NEXT] Update all affected component tests to assert copy-layer keys or the new Spanish strings.

## 6. Verification

- [x] 6.1 Run `pnpm lint` in `next_b2b_starter` — must pass. → **PASSED** (0 errors; 1 pre-existing unrelated warning in `components/crm/deal-kanban.tsx`).
- [x] 6.2 Run `pnpm build` in `next_b2b_starter` — must pass (also proves unknown copy keys fail compilation). → **PASSED** (compile + TypeScript + static generation green). During implementation the typed lookup rejected `ui.billing.contactSupport` (wrong namespace) and a duplicate `invoicesRemaining` key — fixed; proves the compile-time key guarantee.
- [x] 6.3 Run affected component test suites (`pnpm test` scoped to signup, billing, whatsapp-config, inbox, dashboard, agent) — must pass. → **PASSED** (96/96 tests across 22 files, including new `lib/copy/ui.test.ts` integrity + hardcoded-string sweep).
- [x] 6.4 Record results and archive decision in this file after completion.

**Archive deferred:** awaiting reviewer sign-off before `/opsx-archive`; copy layer and string migration are complete and all verification gates passed.
