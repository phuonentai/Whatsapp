# Tasks: ux-error-recovery

## 1. Shared infrastructure

- [x] 1.1 Create `components/common/error-state.tsx` ([FE-NEXT]): ErrorState with title/description/onRetry/isRetrying props, accessible (role="alert", focusable retry button)
- [x] 1.2 Add unit test `components/common/error-state.test.tsx` ([FE-NEXT]): renders title, calls onRetry on click, shows retrying state. Verify: `pnpm test -- error-state`
- [x] 1.3 Create `hooks/use-error-recovery.ts` ([FE-NEXT]): small helper returning `{ isLoading, isError, retry, isRetrying }` from a TanStack query result to standardize the pattern. Verify: `pnpm lint`

## 2. Native dialog removal

- [x] 2.1 Replace `window.alert` in `components/billing/plans-modal.tsx:73,100` ([FE-NEXT]) with inline banner + `ConfirmDialog` for "cancel current subscription first". Verify: `pnpm lint`
- [x] 2.2 Replace `window.confirm` for role change + member removal in `app/dashboard/settings/components/member-list.tsx` ([FE-NEXT]) with `ConfirmDialog`; keep "last admin" inline error. Verify: `pnpm lint`
- [x] 2.3 Replace `window.confirm` in `app/dashboard/settings/components/compliance-section.tsx` forget flow ([FE-NEXT]) with `ConfirmDialog`. Verify: `pnpm lint`
- [x] 2.4 Grep for remaining `window.alert|window.confirm|window.prompt` in `app/` + `components/` ([FE-NEXT]) and confirm none remain in product flows. Verify: `rg -n "window\.(alert|confirm|prompt)" app components` returns only acceptable non-product sites

## 3. Send-failure UX (inbox)

- [x] 3.1 Update `app/dashboard/inbox/components/reply-input.tsx` ([FE-NEXT]): clear input only on successful send; on failure show `toast.error` and keep draft. Verify: `pnpm test -- reply-input`
- [x] 3.2 Extend `reply-input.test.tsx` ([FE-NEXT]) with failure scenario: rejected send keeps draft and shows toast. Verify: `pnpm test -- reply-input`
- [x] 3.3 Audit other await-mutation call sites ([FE-NEXT]) for unhandled rejections (grep `mutateAsync` in app/). Verify: `rg -n "mutateAsync" app components` and confirm each has try/catch or onError

## 4. Error/retry states across data views

- [x] 4.1 CRM lists: add error/retry branches to `components/crm/contact-table.tsx`, `company-table.tsx`, `deal-kanban.tsx` ([FE-NEXT]) using ErrorState + Spanish copy ("Error al cargar los contactos", "Reintentar"). Verify: `pnpm lint`
- [x] 4.2 Tickets: add error/retry to `components/tickets/ticket-list.tsx` and `ticket-detail.tsx` ([FE-NEXT]). Verify: `pnpm lint`
- [x] 4.3 Inbox: add error/retry to conversation list and message thread ([FE-NEXT]) (conversation-list.tsx, message-thread.tsx). Verify: `pnpm lint`
- [x] 4.4 Settings: align remaining sections to ErrorState where they still use raw text "Cargando..." ([FE-NEXT]). Verify: `pnpm lint`
- [x] 4.5 Dialog mutations: ensure failed create/update keeps dialog open with entered values and toasts Spanish error ([FE-NEXT]) — audit contact/company/deal dialogs. Verify: `pnpm test -- crm` (component tests pass)
- [x] 4.6 Verification sweep: grep for `isLoading` without `isError` in data views ([FE-NEXT]) and fix remaining views. Verify: `rg -l "isLoading" components app | xargs rg -L "isError"` shows no data-view matches

## 5. Unread indicators (inbox)

- [x] 5.1 Add per-conversation `lastSeenAt` tracking to inbox page state ([FE-NEXT]) (stores/ or page state). Verify: `pnpm lint`
- [x] 5.2 Render unread indicator (dot + count) in `conversation-list.tsx` when latest inbound message is newer than `lastSeenAt` ([FE-NEXT]), distinct from pending-suggestion badge. Verify: `pnpm lint`
- [x] 5.3 Clear indicator on conversation open and on successful reply ([FE-NEXT]). Verify: `pnpm lint`
- [x] 5.4 Add unit test for unread derivation logic ([FE-NEXT]). Verify: `pnpm test -- inbox`

## 6. Live-region announcements

- [x] 6.1 Add `role="log" aria-live="polite"` to message thread container ([FE-NEXT]). Verify: `pnpm lint`
- [x] 6.2 Add `aria-live="polite"` to knowledge chat assistant message container ([FE-NEXT]). Verify: `pnpm lint`
- [x] 6.3 Add `aria-live="polite"` announcement region to agent-suggestion panel ([FE-NEXT]). Verify: `pnpm lint`

## 7. Verification

- [x] 7.1 Run frontend unit tests ([FE-NEXT]). Verify: `pnpm test`
  - Result: PASS — 17 files, 63 tests.
- [x] 7.2 Run typecheck + lint ([FE-NEXT]). Verify: `pnpm lint` and `npx tsc --noEmit`
  - Result: PASS — tsc 0 errors; eslint 0 errors (1 pre-existing warning in deal-kanban.tsx:173).
- [x] 7.3 Run e2e smoke for inbox + CRM ([OPS-GOV]). Verify: `pnpm test:e2e` (inbox + crm specs pass)
  - Result: PASS — inbox-ui (19), contacts (4), companies, deals, pipelines, tags, activities, cross-entity (21 total, 3 flaky→passed), knowledge-base-ui, whatsapp-inbox, feature-gating (18) = 79 specs green.
  - Note: `settings-ui.spec.ts` cannot collect — pre-existing syntax error in `e2e/page-objects/settings.page.ts` (uncommitted Siigo methods appended outside the class; not part of this change).

**Archive deferred:** `settings-ui.spec.ts` e2e collection is blocked by a pre-existing syntax error in `e2e/page-objects/settings.page.ts` (uncommitted Siigo work outside this change). Run `/opsx-archive` after that file is fixed.
