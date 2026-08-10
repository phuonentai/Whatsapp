# Design: add-more-e2e-tests

## Context

Playwright suite (61 tests, 12 specs) in `next_b2b_starter/e2e/` covers CRUD, auth redirects, feature gating, admin-panel, and webhook-driven inbox rendering. Three shipped surfaces lack any coverage: inbox interactions, knowledge base, and most settings tabs. E2e is happy-path heavy: no RBAC, empty-state, or error-path coverage. The frontend has zero unit/component test tooling (no vitest/jest). Test stack: mock auth via `X-Test-Org-ID` header (`helpers/api.ts`), seeded orgs (`test-org-pro`, `test-org-free`, `test-org-enterprise`, `test-org-rbac`), Chromium-only config, fully parallel.

## Goals / Non-Goals

**Goals:**
- Cover inbox interaction loop: reply send, status filter, quick replies, agent-suggestion approve/reject.
- Cover knowledge base: PDF upload, non-PDF rejection, document list, chat send.
- Cover settings: invite member, module toggle, playbooks, profile, subscription, whatsapp config, member roles.
- Add negative/RBAC/empty/error/edge e2e coverage across the three surfaces using existing seed identities.
- Add a frontend component-test capability: vitest + Testing Library with a shared render harness.
- Keep tests deterministic: seed via mock-auth API, no live LLM/agent calls.
- Reuse existing page-object/fixture patterns; zero production code changes.

**Non-Goals:**
- No CSV import/export, billing, or Siigo e2e (separate changes, gates pending).
- No performance/reliability refactors (covered by `speed-up-e2e-tests`).
- No screenshot/golden-image testing, no axe accessibility scanning, no cross-browser projects.
- No Go API handler (httptest) or expanded Go integration tests.

## Decisions

**D1. Add mock-auth seed endpoint for agent suggestions.**
No HTTP create-suggestion path exists; the pipeline requires live LLM (placeholder key in `app.env`), and the `ai_credits_exhausted` escalation path is not seeded. Decision: add `POST /api/agent/suggestions/seed` (gated `AUTH_MOCK_ENABLED=true`, 404 otherwise) that calls a new `SeedPendingSuggestion` service method — resolves conversation, creates flow if absent, inserts a pending reply suggestion via the existing `InsertSuggestion` repo. Tests seed via this endpoint, then drive approve/reject through the UI.
*Alternatives rejected:* driving the real pipeline — nondeterministic, burns AI credits, placeholder key; dropping the approve/reject test — leaves spec unimplemented.

**D2. Extend `inbox.page.ts`, add `knowledge.page.ts` + `settings.page.ts`.**
Matches existing pattern (contacts/companies/deals each have a page object). Inbox page object already exists and is referenced by `whatsapp-inbox.spec.ts`; extend, don't duplicate.
*Alternative rejected:* inline locators in specs — duplicated selectors, harder to maintain.

**D3. Quick-reply tests gate on playbook data.**
`QuickReplies` renders only `pb.applied && pb.guiones`. Test seeds an applied playbook with a guion via API, asserts the pill renders, clicks it, asserts text lands in the reply input. Absence case: seed org without applied playbook → no pills.
*Alternative rejected:* asserting against existing seed data — brittle across seed changes.

**D4. Knowledge chat asserts UI render, not model output.**
Chat hits RAG backend; assertions target "message appended to thread + request reached API". No expectation on LLM answer content.
*Alternative rejected:* mocking model responses — couples test to LLM provider internals.

**D5. Settings tests run under `test-org-pro` (admin role).**
Invite member, module toggle, profile, subscription, whatsapp config all need ORG_MANAGE. `test-org-pro` seeded as admin (`admin-pro@test.com`), same as `api.ts` defaults.
*Alternative rejected:* per-feature org switching — complicates fixtures for no gain.

**D6. Component tests use vitest + Testing Library + jsdom.**
Add `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jsdom` as devDependencies; `vitest.config.ts` (react plugin, jsdom environment, setup file); `pnpm test` script. Runs headless-fast, no browser, no servers — the missing fast-feedback layer.
*Alternative rejected:* Playwright component testing (`@playwright/experimental-ct-react`) — reuses browser but couples component tests to the slow e2e runtime; vitest is the community-standard fast layer.

**D7. RBAC e2e reuses existing seeded identities — no seed changes.**
`go-b2b-starter/cmd/seed-e2e/main.go` already seeds `test-org-pro:member-pro@test.com` (member) and `test-org-rbac` (`admin-rbac@test.com` admin, `manager-rbac@test.com` + `member-rbac@test.com` member). Negative tests send `X-Test-Org-ID: <org>:<member-email>` and assert the admin-only control is hidden/disabled and privileged API returns 403.
*Alternative rejected:* inviting a fresh member per test then asserting — slower, and duplicates settings invite coverage already planned.

**D8. Component-test harness wraps providers and mocks server actions.**
Shared `renderWithProviders()` (test-utils) wraps components in `QueryClientProvider` (fresh client per test), memory router, and `vi.mock`s server actions (`lib/actions/`) and auth/user hooks. Assertions target rendered DOM + interaction outcomes, not network.
*Alternative rejected:* rendering raw components with real providers — TanStack Query refetches and server-action side effects make tests flaky.

**D9. Error-path e2e stubs failures, never live failures.**
Mutation-failure toasts (invite duplicate, upload failure, approve/reject failure) are driven by Playwright `page.route()` network interception returning 4xx/5xx, or by seeding an already-invalid state (duplicate email). No third-party outage simulation.
*Alternative rejected:* real invalid payloads only — often silently accepted server-side, yields false negatives.

**D10. Mock auth becomes role-aware so RBAC scenarios are reachable.**
Previously every `X-Test-Org-ID` identity got `Roles:[admin]` + `Permissions:["*:*"]`, so "member" identities saw full admin UI and the RBAC delta scenarios were untestable. Mock auth now derives the role from the seeded email convention (`admin-*`/`member-*`/`manager-*`) and grants non-admin identities the full permission vocabulary minus `org:manage`. Admins keep `*:*` (byte-identical behavior). Change is confined to the `AUTH_MOCK_ENABLED` branch.
*Alternative rejected:* rewriting the RBAC delta specs down to role-badge display assertions — gutted the security coverage this change exists to provide.

**D11. Discovered production bugs get minimal in-scope fixes.**
Writing the tests surfaced (1) `module-repository` not unwrapping the `{data:[...]}` envelope, crashing the Modules settings section, and (2) a type-predicate compile error in `server-permissions.ts`. Both fixed minimally and covered by the new tests. A third issue — `POST /auth/members` panicking on a nil Stytch client in mock env — is out of scope (organizations module) and recorded as a follow-up; the e2e covers UI-side RBAC instead.
*Note:* react-dropzone silently ignores non-accept files injected via `input[type=file]` (file-selector accept filtering), so the "Only PDF files are accepted" error message is asserted at component-test level (`document-upload.test.tsx`); the e2e asserts the non-PDF path triggers no upload request.

## Risks / Trade-offs

- **Agent-suggestion approve needs correct backend route contract** → verify route exists before writing test (`grep` `/api/agent/suggestions` in backend); if route differs, adjust spec to match.
- **Playbook seed shape may drift from `PlaybookGuionDto`** → read `playbook.dto.ts` + backend handler at implementation time; keep spec assertions at the UI-visible level (pill text, input value).
- **Settings module toggles may write plan-gated state** → feature-gating spec already proves plan gating works; settings test asserts toggle UI + persistence only, not billing outcome.
- **Full suite runtime grows (~61 → ~115)** → suite already fully parallel; new specs share pattern, no new serial bottlenecks. Component tests add no e2e runtime.
- **RBAC assertions can false-fail if a control is merely disabled vs hidden** → assert the UI contract the component actually implements (`toBeVisible` vs `toBeDisabled`) per control; verified per-surface at implementation.
- **vitest + React 19 / Next 16 compatibility** → pin devDependencies to versions `pnpm` resolves; `jsdom` may lag React 19 APIs — if so, fall back to happy-dom.
- **`:3001` port conflict with dev watcher stalls verification** → run suite with backend on `:8080` + next-server on `:3001`, dev watcher stopped (same constraint as `add-ci-pipeline`).
- **Component-test scope is large** → land infra + highest-value components first (dialogs, tables), then remaining components; partial coverage still ships capability.

## Migration Plan

Test-only change; no DB migration, no runtime config change, no Stytch tenant change. Rollback = revert spec/page-object/test/config files + `package.json` devDependency removal; nothing else affected.

## Open Questions

- ~~Exact backend route for seeding agent suggestions~~ → **Resolved:** `POST /api/agent/suggestions/seed` (AUTH_MOCK_ENABLED gated). Verify at apply that the running e2e backend was rebuilt with this endpoint before running suggestion tests.
- Whether `document-upload` `onUpload` posts to `/api/documents` directly — confirm endpoint exists before writing upload test.
- Whether server actions used by the component-test targets are importable/mockable without a Next runtime — resolve by mocking `lib/actions/*` module boundary (D8); if a component reaches into Next internals, test at the hook boundary instead.
- CI wiring of `pnpm test` into `add-ci-pipeline` — cross-change note recorded there; if that change archives first, CI coverage of component tests is deferred until a follow-up.
