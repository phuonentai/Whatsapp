## 1. Migration and SQLC layer [DB-SQLC]

- [x] 1.1 Create migration `000020_create_whatsapp_signup_flows.up.sql` (per design D1): `whatsapp.signup_flows` with `organization_id UNIQUE` FK to `organizations.organizations`, `status` with CHECK on `exchanging|registering|verifying|connected|failed`, `step`, `error_code`, `retry_count`, `metadata JSONB`, timestamps; create `000020_create_whatsapp_signup_flows.down.sql` dropping the table. Verify: `openspec validate` passes; both files inspected
- [x] 1.2 Add SQLC queries to `query/whatsapp.sql`: `UpsertWhatsAppSignupFlow`, `GetWhatsAppSignupFlowByOrganization`, `UpdateWhatsAppSignupFlowStatus` (status, step, error_code, retry_count). Verify: `make sqlc` regenerates `gen/whatsapp.sql.go` and `querier.go` without errors
- [x] 1.3 Run `make test` after regeneration to confirm no existing queries broke. Verify: `make test` passes for sqlc/db packages

## 2. Graph API client [BE-INFRA]

- [x] 2.1 Implement `internal/modules/whatsapp/infra/graphapi/` per design D3: `ExchangeCode`, `FetchMe`, `ResolveWABAAndNumbers`, `CreateSystemUser`, `SubscribeWABA`, `RegisterAppSubscriptions`, `SendTestMessage`; `Client` interface + real implementation; configurable `api_version`/`graph_api_url`; wrapped in the existing two-tier circuit-breaker semantics. Verify: `go build ./...`; `go vet ./...`
- [x] 2.2 Unit tests for the Graph client with a mock HTTP transport: successful exchanges, Meta error responses mapped to `error_code`, circuit-breaker trip/half-open behavior (mock fallback execution per AGENTS.md gate rules). Verify: `go test ./internal/modules/whatsapp/infra/graphapi/...`

## 3. Signup domain and orchestrator [BE-DOMAIN]

- [x] 3.1 Create domain models in `internal/modules/whatsapp/domain/`: `SignupFlow` (status/step/error_code/retry_count/metadata) with status enum + validation, `SignupService` interface (transport-free, no external imports). Verify: `go vet ./...`; interface has no external package imports
- [x] 3.2 Implement the orchestrator per design D2: single-flight upsert (`signup_in_progress`/`signup_already_connected`), stepwise execution `exchanging → registering → verifying → connected`, exponential backoff (3 retries), resume-at-step after code consumption, server-side `webhook_secret`/`verify_token` generation (crypto/rand ≥32 chars), config write (including `coexistence` in metadata), test-echo to own `business_phone`, masked responses. Verify: unit tests covering every state transition and the 409/400 paths; `go test ./internal/modules/whatsapp/...`
- [x] 3.3 Wire HITL escalation per design D9: terminal failure creates a high-priority ticket via the tickets module with `error_code` + step + retry count; falls back to logging when the tickets module is feature-disabled. Verify: unit test with mocked tickets service asserting ticket creation and the disabled-module fallback

## 4. Echo handling [BE-INFRA]

- [x] 4.1 Modify `webhook_service.go`: detect `origin.type = "echo"` per message entry, publish `whatsapp.message.echo` instead of `whatsapp.message.received`, publish nothing for non-message fields; non-echo entries unchanged. Verify: unit tests with recorded payload fixtures (echo, non-echo, no-origin)
- [x] 4.2 Add a CRM listener on `whatsapp.message.echo` (alongside the existing inbound listener) that persists `crm.messages` with `direction = 'outbound'`, idempotent on `(organization_id, whatsapp_message_id)` via `INSERT ... ON CONFLICT DO NOTHING` with a subsequent fetch on no-op. Verify: `go test ./internal/modules/crm/...` including a concurrent-delivery test asserting exactly one row

## 5. Handlers and routes [BE-INFRA]

- [x] 5.1 Implement handlers in the whatsapp module: `GET /api/v1/whatsapp/signup/meta-config`, `POST /api/v1/whatsapp/signup/exchange`, `GET /api/v1/whatsapp/signup/status`, each behind `auth` + `org_context` + `subscription` with `org:manage`; register routes in the module route registration (module already wired in `internal/api/provider.go`). Verify: handler tests assert 200/400/401/403/404/409/502 paths; `go build ./...`
- [x] 5.2 Add the three env vars `WHATSAPP_APP_ID`, `WHATSAPP_APP_SECRET`, `WHATSAPP_SIGNUP_CONFIG_ID` to env loading/validation, and use them in `meta-config` and the exchange flow. Verify: `make test` passes; env validation unit test covers missing-var failure

## 6. Frontend [FE-NEXT]

- [x] 6.1 Add `lib/api/api/repositories/whatsapp-signup-repository.ts` (+ DTO/model + `use-whatsapp-signup-*` hook/queries) following the existing repository pattern: `meta-config`, `exchange`, `status`. Verify: `pnpm lint`
- [x] 6.2 Rework `whatsapp-config-section.tsx` per specs: "Connect WhatsApp" primary button when no config; lazy-load Meta Business JS SDK; `FB.login` with `config_id`, `response_type:'code'`, `override_default_response_type:true`; submit code to the exchange endpoint; 4-state micro-status during flight; failure state with support CTA; existing manual form moved under an "Advanced" disclosure; connected summary shown after success. Verify: `pnpm lint`; `pnpm build`
- [x] 6.3 Status-polling recovery: on settings page load with an in-progress flow, query `GET /api/v1/whatsapp/signup/status` and render current state/step. Verify: `pnpm build`

## 7. Verification gate [OPS-GOV]

- [x] 7.1 Verification gate run and recorded:
  - `sqlc generate` (make unavailable in this env; sqlc binary v`/home/phuongbinhnguyentai/go/bin/sqlc`) — PASS (regenerated `gen/whatsapp.sql.go`, `gen/querier.go` with `Get/Upsert/UpdateWhatsAppSignupFlow`)
  - `go build ./...` — PASS; `go vet ./...` — PASS (exit 0)
  - `go test -count=1 ./internal/... ./pkg/...` (substitute for `make test`) — PASS, 21 packages, 0 failures
  - `pnpm build` — PASS
  - `pnpm lint` — **EXCEPTION (recorded):** baseline failures pre-exist on 6 files this change did not touch (`knowledge-content.tsx`, `modules-section.tsx`, `settings-content.tsx`, `plans-modal.tsx`, `auth-context.tsx`, `deal-kanban.tsx`; `react-hooks/set-state-in-effect`, 9 errors) — being resolved by the in-progress `fix-frontend-eslint-flat-config` change. `whatsapp-config-section.tsx` retains 1 instance of the pre-existing committed pattern (config→form sync effect); no new violations were introduced by this change. Verify: exit code noted per command above
- [ ] 7.2 Manual smoke checklist — **NOT RUN (blocked):** requires a live Meta Business app + embedded-signup config (Assumption in proposal) and a running stack; cannot be executed in this environment. Checklist: connect button renders; Meta popup completes; webhook subscription visible in Meta app dashboard; phone-app-sent message appears in inbox as outbound; manual form still saves. This is a verification-gap blocker for archive
- [ ] 7.3 Archive decision — **Archive deferred:** verification gap 7.2 (manual smoke) cannot run until a Meta app with embedded-signup config exists and the stack is exercised end-to-end
- [ ] 7.2 Manual smoke checklist (record results): connect button renders; Meta popup completes; webhook subscription visible in Meta app dashboard; a phone-app-sent message appears in the inbox as outbound; manual form still saves. Verify: all checklist items recorded in this file
- [ ] 7.3 Archive decision: either run `/opsx-archive` or record an explicit `**Archive deferred:** <reason>` entry. Verify: entry present in this file
