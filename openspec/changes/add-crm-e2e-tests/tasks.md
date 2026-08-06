## 1. Mock Auth Middleware (BE)

- [x] 1.1 [BE-INFRA] Add `AUTH_MOCK_ENABLED` env var and `X-Test-Org-ID` middleware that bypasses Stytch when header is present
- [x] 1.2 [BE-INFRA] Verify mock middleware returns 401 when header is missing but mock mode is enabled
- [x] 1.3 [BE-INFRA] Verify mock middleware passes through to real auth when mock mode is disabled

## 2. Playwright Setup (FE)

- [x] 2.1 [FE-NEXT] Install Playwright with TypeScript in `next_b2b_starter/`
- [x] 2.2 [FE-NEXT] Create `e2e/playwright.config.ts` with base URL from env, Chromium-only, CI-ready settings
- [x] 2.3 [FE-NEXT] Add `test:e2e` script to `package.json`
- [x] 2.4 [FE-NEXT] Create `e2e/global-setup.ts` that seeds test DB with 4 orgs and accounts
- [x] 2.5 [FE-NEXT] Create `e2e/fixtures/auth.ts` — mock auth helper that sets `X-Test-Org-ID` header and stores session

## 3. Test Helpers & Page Objects (FE)

- [x] 3.1 [FE-NEXT] Create `e2e/helpers/api.ts` — typed fetch wrapper for data seeding via API
- [x] 3.2 [FE-NEXT] Create `e2e/page-objects/login.page.ts` — login flow using mock auth
- [x] 3.3 [FE-NEXT] Create `e2e/page-objects/contacts.page.ts` — Contacts table, create/edit/delete/search/filter
- [x] 3.4 [FE-NEXT] Create `e2e/page-objects/companies.page.ts` — Companies table, create/edit/delete/search
- [x] 3.5 [FE-NEXT] Create `e2e/page-objects/deals-kanban.page.ts` — Kanban board, create/edit/delete/move stage
- [x] 3.6 [FE-NEXT] Create `e2e/page-objects/pipelines.page.ts` — Pipeline editor, create pipeline, manage stages
- [x] 3.7 [FE-NEXT] Create `e2e/page-objects/activities.page.ts` — Activity timeline, create note/call/task, filter
- [x] 3.8 [FE-NEXT] Create `e2e/page-objects/tags.page.ts` — Tag list, create/delete, tag/untag entities

## 4. Contacts E2E Specs (FE)

- [x] 4.1 [FE-NEXT] Write `specs/contacts.spec.ts` — create, view, update, delete contact
- [x] 4.2 [FE-NEXT] Add contact search and lead_status filter tests
- [x] 4.3 [FE-NEXT] Add duplicate phone and empty phone validation tests

## 5. Companies E2E Specs (FE)

- [x] 5.1 [FE-NEXT] Write `specs/companies.spec.ts` — create, view, update, delete company
- [x] 5.2 [FE-NEXT] Add company search by name/NIT test
- [x] 5.3 [FE-NEXT] Add duplicate company name validation test

## 6. Deals & Pipelines E2E Specs (FE)

- [x] 6.1 [FE-NEXT] Write `specs/deals.spec.ts` — create, view, update, delete deal on Kanban
- [x] 6.2 [FE-NEXT] Add deal stage movement test (drag-and-drop simulation via keyboard/click)
- [ ] 6.3 [FE-NEXT] Add deal status change (won/lost) test
- [ ] 6.4 [FE-NEXT] Add deal creation with linked contact and company test
- [x] 6.5 [FE-NEXT] Write `specs/pipelines.spec.ts` — view default pipeline, create pipeline, add/edit stages

## 7. Activities E2E Specs (FE)

- [x] 7.1 [FE-NEXT] Write `specs/activities.spec.ts` — create note, call, and task activities
- [ ] 7.2 [FE-NEXT] Add activity type filtering test
- [ ] 7.3 [FE-NEXT] Add activities filtered by contact and deal tests

## 8. Tags E2E Specs (FE)

- [x] 8.1 [FE-NEXT] Write `specs/tags.spec.ts` — create and delete tag
- [ ] 8.2 [FE-NEXT] Add tag contact and tag deal tests
- [ ] 8.3 [FE-NEXT] Add untag entity test
- [x] 8.4 [FE-NEXT] Add duplicate tag name validation test

## 9. Feature Gating E2E Specs (FE)

- [x] 9.1 [FE-NEXT] Write `specs/feature-gating.spec.ts` — Free plan hides Empresas, Negocios, Actividad tabs
- [x] 9.2 [FE-NEXT] Add Pro plan hides Etiquetas tab test
- [x] 9.3 [FE-NEXT] Add Enterprise plan shows all tabs test
- [x] 9.4 [FE-NEXT] Add API-level 403 test for gated endpoints

## 10. Cross-Entity Workflow Spec (FE)

- [x] 10.1 [FE-NEXT] Write `specs/cross-entity.spec.ts` — full workflow: create company → create contact linked to company → create deal for contact → tag deal → move stage → verify activity logged for stage change

## 11. CI Integration

- [x] 11.1 [BE-INFRA] Add `e2e` stage in `.gitlab-ci.yml` with test DB, migrations, Go server, Next.js, Playwright execution
- [x] 11.2 [BE-INFRA] Cache Playwright browser binaries in CI pipeline

## 12. Verification

- [ ] 12.1 [FE-NEXT] Run full Playwright suite locally against test environment and verify all 40+ tests pass
- [ ] 12.2 [BE-INFRA] Verify mock auth middleware is gated by `AUTH_MOCK_ENABLED` and unreachable in production
- [ ] 12.3 Document test setup in `DEVELOPMENT.md` with `make test-e2e` target
