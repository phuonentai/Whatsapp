# Change Proposal: add-more-e2e-tests

## Why

The Playwright suite covers CRUD + webhook rendering (61 tests) but has zero coverage over three shipped, user-facing surfaces: inbox interactions (reply, status filter, quick replies, agent suggestions), the knowledge base (/dashboard/knowledge), and most of /dashboard/settings (invite member, modules, playbooks, profile, subscription, whatsapp config). These features ship untested, so regressions pass CI silently.

Beyond missing surfaces, existing e2e is almost entirely happy-path: no RBAC matrix, no empty-state, no error-path, and no edge-input coverage. The frontend additionally has **zero** unit/component test infrastructure — no vitest, no Testing Library — so component logic (form validation, table empty states, toggle persistence) has no fast feedback layer. CI (`add-ci-pipeline`) has no frontend test job because no frontend test command exists.

## What Changes

- Add `e2e/specs/inbox-ui.spec.ts` — reply send, empty-reply no-op, status filter, quick-reply insert, agent suggestion approve/reject.
- Add `e2e/specs/knowledge-base-ui.spec.ts` — PDF upload, non-PDF rejection, document list rendering, chat send, empty-chat no-op.
- Add `e2e/specs/settings-ui.spec.ts` — invite-member form, module toggle, playbooks section, profile update, subscription tab, whatsapp-config section, member list roles.
- Add `e2e/page-objects/inbox.page.ts` methods (reply, setStatusFilter, selectQuickReply, approveSuggestion, rejectSuggestion) and new `e2e/page-objects/knowledge.page.ts`, `e2e/page-objects/settings.page.ts`.
- Extend the above specs with negative/RBAC/empty/error/edge scenarios (~30–40 additional tests): non-admin RBAC matrix using existing seeds (`test-org-pro:member-pro@test.com`, `test-org-rbac` admin/manager/member), empty states, error toasts on failed mutations, long-text/unicode/duplicate-email edge inputs, double-submit guards.
- Add frontend component-test infrastructure: vitest + Testing Library + jsdom devDependencies, `vitest.config.ts`, test setup file, `pnpm test` script, and a shared `renderWithProviders` helper (QueryClientProvider wrapper, server-action mocks).
- Add ~20–30 component specs for forms/dialogs (contact/company/deal), tables (empty state + render), tag-picker, upgrade-banner, deal-kanban, confirm-dialog, inbox reply input, knowledge dropzone, settings module toggle.
- Add mock-auth-only BE endpoint `POST /api/agent/suggestions/seed` (gated by `AUTH_MOCK_ENABLED=true`) to seed pending suggestions for the approve/reject UI test without a live LLM pipeline.
- Make mock auth role-aware (`internal/modules/auth/middleware.go`, gated by `AUTH_MOCK_ENABLED`): derive the RBAC role from the seeded e2e email convention (`admin-*` / `member-*` / `manager-*`) so non-admin identities receive explicit permissions minus `org:manage` — required to exercise the RBAC delta scenarios. Admins keep the `*:*` wildcard, so existing e2e behavior is unchanged.
- Two bug fixes discovered while writing tests (both verified by the new tests): (1) `lib/api/api/repositories/module-repository.ts` did not unwrap the `{data: [...]}` envelope for `/modules` + `/modules/org`, crashing the Modules settings section; (2) a type-predicate compile error in `lib/auth/server-permissions.ts`.
- No other production code changes; otherwise test-only addition (devDependencies + config + specs).

## Capabilities

### New Capabilities

- `inbox-ui`: UI behaviour of the inbox — sending replies, status filtering, quick replies from playbook guiones, agent-suggestion approve/reject actions, and RBAC/empty/error behaviour of that surface.
- `knowledge-base-ui`: UI behaviour of the knowledge base — document upload (PDF-only), document list rendering, chat send, and empty/error states.
- `settings-ui`: UI behaviour of the settings dashboard — invite member, module toggling, playbooks, profile, subscription, whatsapp config, member roles, and RBAC restrictions on those controls.
- `fe-component-tests`: frontend unit/component test capability — vitest runner, Testing Library render harness, form validation and table rendering coverage for CRM, inbox, knowledge, and settings components.

### Modified Capabilities

(none)

## Impact

- `next_b2b_starter/e2e/` — 3 new spec files, 2 new page-object files, 1 extended page-object file; 3 specs extended with negative/RBAC scenarios.
- `next_b2b_starter/package.json` — vitest + @testing-library devDependencies, `test` script.
- `next_b2b_starter/vitest.config.ts`, `next_b2b_starter/e2e`/`tests` setup file (new).
- `next_b2b_starter/components/` — component spec files (`*.test.tsx`) colocated or under a test directory.
- `go-b2b-starter/internal/modules/agent/` — mock-gated `SeedPendingSuggestion` service method + `HandleSeedSuggestion` handler + `POST /api/agent/suggestions/seed` route (AUTH_MOCK_ENABLED only).
- `next_b2b_starter/e2e/playwright.config.ts` — unchanged (already parallel).
- Verification: `pnpm test` (vitest), `pnpm test:e2e` full suite (61 existing + ~18 UI + ~35 negative/RBAC), `npx tsc --noEmit` on e2e and component-test projects, `pnpm lint`.
- No auth flow or data persistence changes; Stytch contracts and local DB schema untouched.

## Assumptions

- Agent-suggestion approve/reject tests seed a pending suggestion via the mock-auth seed endpoint `POST /api/agent/suggestions/seed` (new, gated by `AUTH_MOCK_ENABLED`) rather than driving a live LLM/agent pipeline; LLM calls are flaky in e2e and metered. This endpoint was added because no HTTP create-suggestion path existed.
- Knowledge chat tests assert UI render (message appended to thread) and request delivery, not model response quality.
- Quick replies render only when playbook guiones exist for the test org; tests assert presence/absence accordingly.
- RBAC negative tests reuse existing seeds — `seed-e2e` already creates `test-org-pro:member-pro@test.com` (member) and `test-org-rbac` (admin-rbac / manager-rbac / member-rbac) — so no seed changes are required. Verify the exact claim in `go-b2b-starter/cmd/seed-e2e/main.go` before writing tests.
- Error-path tests stub failure via mock-auth API seeds and/or Playwright network interception; no live third-party failures are simulated.
- No `inbox-ui` / `knowledge-base-ui` / `settings-ui` / `fe-component-tests` capability specs exist today (verified: `openspec/specs/` has no matching names), so all are created fresh.
- vitest + Testing Library + jsdom versions are added as devDependencies and are compatible with React 19 / Next 16; exact versions pinned at implementation time against what `pnpm` resolves.

## Non-Goals

- No local credential storage anywhere: tests rely on mock-auth (`AUTH_MOCK_ENABLED` + `X-Test-Org-ID`), never on Stytch passwords, MFA tokens, or session tokens.
- No coverage for CSV import/export, MercadoPago billing, or Siigo invoicing — those changes are incomplete or un-archived and get their own e2e work.
- No accessibility (axe) scanning, cross-browser projects (webkit/firefox), or visual-regression/golden-image testing — out of scope for this change.
- No Go API handler tests (httptest) or expanded Go integration tests — frontend testing focus only. The BE exceptions are the mock-gated suggestion seed endpoint and the mock-auth RBAC role derivation (both active only under `AUTH_MOCK_ENABLED`).
- No production behavior changes: the only non-mock production code touched are the two discovered bug fixes (module catalog envelope unwrap + server-permissions type predicate), both restoring intended behavior.
