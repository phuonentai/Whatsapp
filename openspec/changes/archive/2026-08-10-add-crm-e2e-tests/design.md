## Context

The CRM module has ~30 API endpoints and a full React UI (contacts, companies, deals, pipelines, activities, tags) but zero tests. The frontend uses Next.js 16 + React 19 + TanStack Query. The backend is Go + Gin + PostgreSQL with Stytch auth. No test infrastructure exists for either frontend or backend.

## Goals / Non-Goals

**Goals:**
- Browser-level E2E tests for all CRM entities via Playwright
- Tests run against a real Go backend and real PostgreSQL
- Mock auth to avoid Stytch dependency in tests
- Feature gating tested across Free/Pro/Enterprise tiers
- Permission enforcement tested across admin/manager/member roles
- Inbound WhatsApp message ingestion simulated end-to-end (signed webhook → eventbus → `crm.messages` → inbox UI)
- GitLab CI pipeline integration

**Non-Goals:**
- No changes to Next.js application code
- No unit or integration tests (pure E2E only)
- No visual regression or screenshot diffing
- No mobile browser testing (Chromium only)
- No real outbound WhatsApp sends — outbound requires a live WhatsApp access token and outbound API call; tests assert on inbound simulation and the sending UI state only
- No change to the webhook request-processing pipeline — except correcting the error-to-HTTP-status mapping (`ErrInvalidSignature` → 401, `ErrUnknownPhoneNumber` → 404) in `handler.go`, required by the existing `whatsapp-webhook-ingress` living spec that the e2e specs assert against

## Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|----------|--------|------------------------|-----------|
| Test runner | Playwright + TypeScript | Cypress, Selenium, Vitest + MSW | Industry standard for modern E2E. TypeScript matches frontend. Built-in test runner, auto-wait, network interception. No extra dependencies. |
| Auth strategy | Mock Stytch via `X-Test-Org-ID` header | Seed org + real Stytch test tokens, Stub all auth middleware | Avoids Stytch as a test dependency. No rate limits, no network calls, instant. Middleware checks for header before Stytch validation — zero changes to production code path. |
| Feature gating | 3 seeded orgs (Free/Pro/Enterprise) | Dynamic feature flag overrides via API | Tests real entitlement middleware with real plan data. More realistic. Single test seed script. |
| Page Objects | One per CRM entity | Direct DOM queries in tests | Standard Playwright pattern. Encapsulates selectors and interactions. Makes tests readable and maintainable. |
| Test location | `next_b2b_starter/e2e/` | Separate repo, Go module | Colocated with frontend. Uses same tsconfig, node_modules, lint rules. |
| CI | GitLab CI stage | GitHub Actions, separate pipeline | Matches existing `.gitlab-ci.yml`. Spins up test DB, runs migrations, starts Go API + Next.js, executes Playwright. |
| WhatsApp simulation | Forge HMAC-SHA256 and POST to the real webhook endpoint | Test-only webhook endpoint, mock eventbus publisher | Exercises the production path untouched: real signature validation, org resolution, `webhook_logs`, eventbus, and `crm.messages` persistence. No backend code changes needed. |

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

### WhatsApp Webhook Simulation Design

Inbound messages are simulated against the **real** `POST /api/v1/webhooks/whatsapp` endpoint — no test-only endpoint. The webhook is unauthenticated; the `x-hub-signature-256` HMAC *is* the auth, so the test helper can forge valid signatures with the org's seeded `webhook_secret`.

```
PUT /api/v1/whatsapp/config            ──► seeds whatsapp.whatsapp_configs for the org
  (X-Test-Org-ID, org:manage)             (phone_number_id, webhook_secret, verify_token)

helpers/whatsapp.ts
  buildPayload()  ──► Cloud API shape: entry[].changes[].value.metadata.phone_number_id,
                      value.messages[].id/from/type/text.body/timestamp
  signPayload()   ──► "sha256=" + hex(HMAC-SHA256(rawBody, webhook_secret))

POST /api/v1/webhooks/whatsapp (signed)
        │
        ▼
  whatsapp.webhook_logs (status=received/failed)  →  eventbus whatsapp.message.received
        │
        ▼
  crm.messages (idempotent on whatsapp_message_id) → /dashboard/inbox UI renders it
```

The inbox UI polls every 5s (`refetchInterval: 5000`), so specs use `waitForResponse`/`waitForSelector` rather than fixed sleeps. A seeded config is created once per org in global setup (or per-test where a unique `phone_number_id` is needed).

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
│   ├── tags.page.ts
│   └── inbox.page.ts               # Conversation list, thread, reply input, status tabs
├── helpers/
│   ├── api.ts                   # Raw fetch wrapper for data seeding
│   ├── wait.ts                  # Wait utilities
│   └── whatsapp.ts              # Cloud API payload builder + HMAC-SHA256 signer + webhook POST
└── specs/
    ├── contacts.spec.ts
    ├── companies.spec.ts
    ├── deals.spec.ts
    ├── pipelines.spec.ts
    ├── activities.spec.ts
    ├── tags.spec.ts
    ├── feature-gating.spec.ts
    ├── cross-entity.spec.ts
    └── whatsapp-inbox.spec.ts
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mock auth diverges from real auth | Tests pass but auth broken in prod | Mock is additive middleware only. Production path unchanged. Smoke test manually before release. |
| Flaky tests from async UI (TanStack Query, optimistic updates) | False failures | Use Playwright's `waitForResponse` and `waitForSelector` instead of arbitrary timeouts. Retry flaky tests (Playwright built-in). |
| Test data collision between parallel tests | Cross-test contamination | Each test creates its own data. Use unique prefixes/randomized names. Run serially (no worker parallelization for now). |
| CI runtime too long | Slow pipeline | Playwright + browser binary ~150MB. Cache with GitLab CI cache. Estimate ~5 min for full suite. |
| Playwright version drift with browser | Breaking changes | Pin Playwright version in package.json. Use `npx playwright install --with-deps` in CI. |
| WhatsApp sim tests flake on async eventbus delivery | False failures | UI polls every 5s; use `waitForResponse`/polling on `/crm/conversaciones` and `/crm/conversaciones/:id/mensajes` instead of fixed sleeps. Unique `phone_number_id` + `whatsapp_message_id` per test. |
| Agent pipeline (`add-agentic-whatsapp-assistant`) also subscribes to `whatsapp.message.received` | Simulated messages may trigger AI pipeline when that change lands | Scope assertions to the seeded org's own conversations; run suite serially; document interference risk in that change. |
| Outbound sends impossible in test env | Cannot E2E-test reply sending | Non-goal: no real WhatsApp access token. Assert reply-input sending state only. |

### Open Questions

- Should we add a `make test-e2e` target in the Go Makefile or frontend package.json? → Frontend `package.json` script preferred.
- Should we test drag-and-drop deal stage movement via Playwright? → Yes, but use keyboard + click simulation rather than HTML5 drag API (known flaky in Playwright).
