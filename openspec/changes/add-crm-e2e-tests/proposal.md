## Why

The CRM module — contacts, companies, deals, pipelines, activities, tags — has ~30 API endpoints and a full React UI, but zero tests at any layer. Regressions in the WhatsApp → CRM bridge, permission logic, or feature gating are invisible until they hit production. We need a browser-level E2E test suite that exercises the real Next.js frontend against a real Go backend to catch integration failures before deployment.

## What Changes

- Install Playwright with TypeScript in `next_b2b_starter/`
- Add test environment config (test DB, seeded orgs, mock auth)
- Create reusable fixtures: mock auth bypass, CRM data seeder, page objects
- Write E2E specs for all 6 CRM entities covering happy-path CRUD, search/filter, cross-entity workflows, error/edge cases, feature gating, and permission enforcement
- Simulate inbound WhatsApp Cloud API messages by forging HMAC-SHA256 signatures against the real webhook endpoint, and verify they render in the inbox UI
- Add GitLab CI job to run Playwright against test infrastructure

## Capabilities

### New Capabilities
- `crm-test-infrastructure`: Playwright project setup, configuration, mock auth middleware, test DB seeding, page object models, and shared helpers.
- `crm-contactos-e2e`: E2E tests for the Contacts list page, create/update/delete flows, search, and lead status filtering.
- `crm-empresas-e2e`: E2E tests for the Companies list page, create/update/delete flows, and search.
- `crm-negocios-e2e`: E2E tests for Deals Kanban board, create/update/delete, pipeline stage movement, and status transitions.
- `crm-pipelines-e2e`: E2E tests for Pipeline editor, creating pipelines, managing stages (add/edit).
- `crm-actividades-e2e`: E2E tests for Activity timeline, creating activities (notes/calls/tasks), and filtering by entity.
- `crm-etiquetas-e2e`: E2E tests for Tag management, creating/deleting tags, tagging and untagging entities.
- `crm-feature-gating-e2e`: E2E tests verifying that Pro/Enterprise feature-gated UI elements are properly hidden on lower-tier plans.
- `crm-whatsapp-e2e`: E2E tests simulating inbound WhatsApp Cloud API webhooks (HMAC-signed payloads) against the real `/api/v1/webhooks/whatsapp` endpoint and verifying the inbox UI renders conversations and messages. Covers signature rejection, unknown `phone_number_id`, duplicate-delivery idempotency, and the `hub.mode=subscribe` verification handshake.

### Modified Capabilities
- None

## Impact

- **Dependencies**: Playwright + browser binaries in CI
- **Environment**: Test PostgreSQL DB, 3 seeded orgs (Free/Pro/Enterprise), mock auth middleware
- **CI**: New `e2e` stage in `.gitlab-ci.yml` — spins up DB, runs migrations, starts Go + Next.js, executes Playwright
- **Developer workflow**: `pnpm exec playwright test` locally, new `make test-e2e` target
- **Non-goals**: No frontend application code changes. No visual regression testing. No real outbound WhatsApp sends (outbound requires a live WhatsApp access token and outbound API call — the test helper asserts on inbound simulation and the disabled/sending UI state only). No modification of the webhook endpoint's request handling — the real endpoint is exercised with forged signatures. Exception: the webhook handler's error-to-HTTP-status mapping is corrected (401 `invalid_signature`, 404 `unknown_phone_number`) to satisfy the existing `whatsapp-webhook-ingress` living spec, which the e2e tests assert against.
