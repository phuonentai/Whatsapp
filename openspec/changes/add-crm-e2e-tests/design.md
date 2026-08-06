## Context

The CRM module has ~30 API endpoints and a full React UI (contacts, companies, deals, pipelines, activities, tags) but zero tests. The frontend uses Next.js 16 + React 19 + TanStack Query. The backend is Go + Gin + PostgreSQL with Stytch auth. No test infrastructure exists for either frontend or backend.

## Goals / Non-Goals

**Goals:**
- Browser-level E2E tests for all CRM entities via Playwright
- Tests run against a real Go backend and real PostgreSQL
- Mock auth to avoid Stytch dependency in tests
- Feature gating tested across Free/Pro/Enterprise tiers
- Permission enforcement tested across admin/manager/member roles
- GitLab CI pipeline integration

**Non-Goals:**
- No changes to Go backend or Next.js application code (except mock auth middleware)
- No unit or integration tests (pure E2E only)
- No visual regression or screenshot diffing
- No mobile browser testing (Chromium only)

## Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|----------|--------|------------------------|-----------|
| Test runner | Playwright + TypeScript | Cypress, Selenium, Vitest + MSW | Industry standard for modern E2E. TypeScript matches frontend. Built-in test runner, auto-wait, network interception. No extra dependencies. |
| Auth strategy | Mock Stytch via `X-Test-Org-ID` header | Seed org + real Stytch test tokens, Stub all auth middleware | Avoids Stytch as a test dependency. No rate limits, no network calls, instant. Middleware checks for header before Stytch validation — zero changes to production code path. |
| Feature gating | 3 seeded orgs (Free/Pro/Enterprise) | Dynamic feature flag overrides via API | Tests real entitlement middleware with real plan data. More realistic. Single test seed script. |
| Page Objects | One per CRM entity | Direct DOM queries in tests | Standard Playwright pattern. Encapsulates selectors and interactions. Makes tests readable and maintainable. |
| Test location | `next_b2b_starter/e2e/` | Separate repo, Go module | Colocated with frontend. Uses same tsconfig, node_modules, lint rules. |
| CI | GitLab CI stage | GitHub Actions, separate pipeline | Matches existing `.gitlab-ci.yml`. Spins up test DB, runs migrations, starts Go API + Next.js, executes Playwright. |

### Mock Auth Design

```
Request ──► Auth Middleware
              │
              ├── X-Test-Org-ID present? ──► MockSession{
              │     OrganizationID: header_value,
              │     AccountID: derived from header,
              │     Role: derived from header
              │   }
              │
              └── No header ──► Normal Stytch JWT validation (production path)
```

The middleware is configured via environment variable `AUTH_MOCK_ENABLED=true` — only enabled in test mode. The `X-Test-Org-ID` header encodes the org slug (e.g., `test-org-free`, `test-org-pro`, `test-org-enterprise`, `test-org-rbac`). The middleware looks up the org and account from the seeded DB.

### Test Data Seeding

**global-setup.ts** runs once before all tests:

| Org | Plan | Accounts |
|-----|------|----------|
| `test-org-free` | Free | 1 admin |
| `test-org-pro` | Pro | 1 admin |
| `test-org-enterprise` | Enterprise | 1 admin |
| `test-org-rbac` | Pro | 1 admin, 1 manager, 1 member |

Each test case seeds only the data it needs via API calls through the authenticated session.

### Test File Structure

```
next_b2b_starter/e2e/
├── playwright.config.ts
├── global-setup.ts              # DB seed, auth setup
├── fixtures/
│   └── auth.ts                  # Login via mock header, storage state
├── page-objects/
│   ├── login.page.ts
│   ├── contacts.page.ts
│   ├── companies.page.ts
│   ├── deals-kanban.page.ts
│   ├── pipelines.page.ts
│   ├── activities.page.ts
│   └── tags.page.ts
├── helpers/
│   ├── api.ts                   # Raw fetch wrapper for data seeding
│   └── wait.ts                  # Wait utilities
└── specs/
    ├── contacts.spec.ts
    ├── companies.spec.ts
    ├── deals.spec.ts
    ├── pipelines.spec.ts
    ├── activities.spec.ts
    ├── tags.spec.ts
    ├── feature-gating.spec.ts
    └── cross-entity.spec.ts
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mock auth diverges from real auth | Tests pass but auth broken in prod | Mock is additive middleware only. Production path unchanged. Smoke test manually before release. |
| Flaky tests from async UI (TanStack Query, optimistic updates) | False failures | Use Playwright's `waitForResponse` and `waitForSelector` instead of arbitrary timeouts. Retry flaky tests (Playwright built-in). |
| Test data collision between parallel tests | Cross-test contamination | Each test creates its own data. Use unique prefixes/randomized names. Run serially (no worker parallelization for now). |
| CI runtime too long | Slow pipeline | Playwright + browser binary ~150MB. Cache with GitLab CI cache. Estimate ~5 min for full suite. |
| Playwright version drift with browser | Breaking changes | Pin Playwright version in package.json. Use `npx playwright install --with-deps` in CI. |

### Open Questions

- Should we add a `make test-e2e` target in the Go Makefile or frontend package.json? → Frontend `package.json` script preferred.
- Should we test drag-and-drop deal stage movement via Playwright? → Yes, but use keyboard + click simulation rather than HTML5 drag API (known flaky in Playwright).
