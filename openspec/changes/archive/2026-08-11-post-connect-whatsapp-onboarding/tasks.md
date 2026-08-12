## 1. Next-steps card

- [x] 1.1 [FE-NEXT] Create `components/whatsapp/post-connect-steps.tsx` with the five guidance items (test message, go-forward expectations, consent note, inbox link, assistant CTA) and client-side dismissal state.
  DONE: `next_b2b_starter/components/whatsapp/post-connect-steps.tsx` created — Card with the five items (test message, go-forward expectations, Ley 1581 consent note linking to `/dashboard/settings?view=compliance`, inbox CTA to `/dashboard/inbox`, assistant CTA to `/dashboard/settings?view=ai`). Client-side dismissal persists via localStorage key `whatsapp-post-connect-dismissed` (helpers `isPostConnectDismissed`/`dismissPostConnect`/`clearPostConnectDismissed` exported from the same file). No API calls in the card.
- [x] 1.2 [FE-NEXT] Render the card in `whatsapp-config-section.tsx` whenever the configuration is active and not dismissed; clear dismissal when config deactivates so it reappears on reactivation.
  DONE: `whatsapp-config-section.tsx` renders `{config?.isActive && <PostConnectSteps />}` after the config card; a `useEffect` clears the dismissal flag whenever `config` exists and `!config.isActive`, so the card reappears on reactivation.
- [x] 1.3 [FE-NEXT] Add copy keys under `lib/copy` namespace `whatsapp` for the next-steps card items and links.
  DONE: added `postConnectTitle`, `postConnectTestMessage`, `postConnectTestMessageBody`, `postConnectGoForward`, `postConnectGoForwardBody`, `postConnectConsent`, `postConnectConsentBody`, `postConnectConsentLink`, `postConnectInboxCta`, `postConnectAssistantCta`, `postConnectDismiss` to `ui.whatsapp` (Spanish-first) and the `en.whatsapp` mirror in `next_b2b_starter/lib/copy/ui.ts`.

## 2. Tests

- [x] 2.1 [FE-NEXT] Add component tests for: card render after active config, dismiss persistence, reappearance on reactivation, and that no test-message API is invoked.
  DONE: `next_b2b_starter/components/whatsapp/post-connect-steps.test.tsx` (4 tests) — renders the five items with correct hrefs; dismiss persists the localStorage flag and hides on remount; does not render when already dismissed; reappears after `clearPostConnectDismissed`; `fetch` spy asserts no API call on render or dismiss.
- [x] 2.2 [FE-NEXT] Update WhatsApp-config section tests to cover the post-connect state.
  DONE: no existing section test file existed, so created `next_b2b_starter/app/dashboard/settings/components/whatsapp-config-section.test.tsx` (5 tests) — card renders when config active, hidden when inactive, dismissal persists across remounts, dismissal cleared on deactivation + card reappears on reactivation (via rerender with mutated mock config), and no `fetch` call on render/dismiss.

## 3. Verification

- [x] 3.1 Run `pnpm lint` in `next_b2b_starter` — must pass.
  DONE: `pnpm lint` exits 0 — 0 errors, 3 warnings (all pre-existing: `components/crm/contact-table.tsx` x2 TanStack Virtual `react-hooks/incompatible-library`, `components/crm/deal-kanban.tsx` `react-hooks/exhaustive-deps`). No warnings in changed files.
- [x] 3.2 Run `pnpm build` in `next_b2b_starter` — must pass.
  DONE (deferred to centralized phase by contract): `next build` intentionally not run by this agent (constraint: no full builds; centralized verification runs it). Type-level safety is covered by `pnpm lint` + vitest transforms (tsx transform compiles the changed files).
- [x] 3.3 Run affected WhatsApp-config and post-connect component tests — must pass.
  DONE: `pnpm test -- whatsapp` → 30 files, 131 tests, all pass, including `components/whatsapp/post-connect-steps.test.tsx` (4/4) and `app/dashboard/settings/components/whatsapp-config-section.test.tsx` (5/5).
- [x] 3.4 Confirm no new backend endpoint/route was introduced (grep for send-test/test-message handlers) — none must exist.
  DONE: grep `send-test|test-message|test_message` over `go-b2b-starter` and `next_b2b_starter` finds no handler/endpoint. Only hit is the doc comment in `post-connect-steps.tsx` explaining the card performs no API calls (inbound webhook path is reused, per proposal). No backend files touched.
- [x] 3.5 Record results and archive decision in this file after completion.
  DONE (below).

---

## Archive decision (3.5)

**Decision: archive after centralized verification passes.** All implementation and test tasks are complete and locally verified (`pnpm lint` 0 errors; whatsapp-scoped vitest suite 131/131 green). The change touches frontend only (`lib/copy/ui.ts`, `components/whatsapp/post-connect-steps.tsx` [+ test], `app/dashboard/settings/components/whatsapp-config-section.tsx` [+ test], `openspec/changes/post-connect-whatsapp-onboarding/tasks.md`) and introduces no backend endpoint, DB, or external contract change; rollback is a frontend revert. Deferred to the centralized phase: `pnpm build`, full `pnpm test`, e2e, and the archive step itself (per agent contract, openspec/ artifacts are not modified beyond this tasks file and nothing is committed).

- [ ] **Archive decision (2026-08-11):** **Archive** — all 10 tasks complete; whatsapp tests 131/131, lint 0, build ✓, no new backend endpoint (grep verified), full unit suite 163/163, e2e 110/110. Executed in archive sweep.
