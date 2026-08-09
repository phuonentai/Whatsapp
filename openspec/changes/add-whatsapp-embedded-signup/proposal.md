## Why

WhatsApp provisioning is manual today: an admin copies `phone_number_id`, `webhook_secret`, and `verify_token` from the Meta dashboard into the settings form (`PUT /api/v1/whatsapp/config`), then separately subscribes webhooks in Meta. Provisioning takes 24–48h, forces the client's mobile WhatsApp Business app to be discarded, and drives high drop-off. Meta's Embedded Signup + Coexistence Framework replaces this with a self-serve flow: the client keeps their mobile app, contacts, and chat history, and the platform gains Cloud API access via a Meta-hosted popup, followed by server-side token exchange, webhook auto-registration, and test-echo validation. Target first-value (active dual-channel connection) in under 3 minutes from first click. This change scopes the self-serve flow only — no workflow engine (Temporal explicitly rejected), no new infrastructure.

## What Changes

- **Migration `000020`** (repo head is `000018`; the in-progress agent change takes `000019`): new `whatsapp.signup_flows` table — `(organization_id UNIQUE, status, step, error_code, retry_count, metadata JSONB, created_at, updated_at)`. No transient OAuth tokens are stored; only the permanent Meta system-user token persists, in the existing `whatsapp_configs.access_token` column (already masked on read)
- **Graph API integration** `internal/modules/whatsapp/infra/graphapi/`: code→user-token exchange, `me` + WABA/phone resolution, system-user creation (`business_management`), `subscribed_apps` + `/{app_id}/subscriptions` registration, test-message send — behind a two-tier circuit breaker (mirrors the Stytch adapter pattern) with a mockable interface for tests
- **SignupService**: linear, idempotent orchestrator (`exchanging → registering → verifying → connected | failed`) with exponential backoff (3 retries); server-side generation of `webhook_secret`/`verify_token` (crypto/rand, ≥32 chars); terminal failures auto-create a high-priority ticket in the existing tickets module (8h SLA per `DefaultSLASeconds`) with the error code and webhook log reference
- **New API surface**: `GET /api/v1/whatsapp/signup/meta-config`, `POST /api/v1/whatsapp/signup/exchange`, `GET /api/v1/whatsapp/signup/status` — all behind the existing `auth` + `org_context` + `subscription` middleware with `org:manage` permission
- **Webhook ingress delta**: messages with `origin.type = "echo"` (phone-app-sent, coexistence mirror) are NOT published as inbound `whatsapp.message.received`; a new `whatsapp.message.echo` event persists them as `direction='outbound'` CRM messages, idempotent on `(organization_id, whatsapp_message_id)` — the inbox shows mobile-sent messages
- **Frontend**: primary "Connect WhatsApp" button in `whatsapp-config-section.tsx` loads the Meta Business JS SDK, calls `FB.login(config_id, response_type:'code')`, posts the code to `/signup/exchange`, and renders a 4-state micro-status (connecting → verifying → webhooks → live); failure shows a support CTA. Manual paste form retained under an "Advanced" disclosure as the repair/reconnect path
- **Config**: new env vars `WHATSAPP_APP_ID`, `WHATSAPP_APP_SECRET`, `WHATSAPP_SIGNUP_CONFIG_ID`; one-time external setup of the Meta Business app + embedded-signup config (see Assumptions)

## Capabilities

### New Capabilities

- `whatsapp-embedded-signup`: self-serve Meta Embedded Signup — signup flow state machine, Graph API token exchange, system-user provisioning, webhook auto-registration, test-echo validation, secret generation, HITL escalation via the tickets module

### Modified Capabilities

- `whatsapp-webhook-ingress`: echo-message handling — coexistence phone-app mirrors are persisted as outbound CRM messages, never published as inbound
- `whatsapp-config-frontend`: embedded-signup connect flow + micro-status; manual form becomes an advanced-mode repair path

## Impact

- **Go backend**: migration `000020`; SQLC queries in `query/whatsapp.sql` (regenerated); `infra/graphapi/` client; SignupService + state machine; handlers/routes in the whatsapp module; echo detection in `webhook_service.go`; CRM listener for `whatsapp.message.echo`; DI wiring in `internal/modules/whatsapp/cmd/init.go`
- **Database**: one table (`whatsapp.signup_flows`), org-scoped via Stytch `organization_id` FK pattern; no credential tables
- **Frontend**: `whatsapp-config-section.tsx` + `use-whatsapp-signup-*` hook/repository additions under `next_b2b_starter/lib/`
- **Dependencies**: none new (no Temporal, no AWS, no Redis, no ticketing SaaS — HITL reuses the existing tickets module)
- **Auth**: no Stytch flow changes; `org:manage` gate reuses existing middleware. Only the Meta system-user token is stored (masked on read); transient OAuth codes/user tokens live in request memory only
- **Ops**: `make migrateup` in the standard flow; Graph API calls carry the existing circuit-breaker semantics; unit tests in `make test`
- **Rollback**: Git — revert the change (migration, module, routes, DI, FE). DB — `000020.down.sql` drops `signup_flows`. Stytch tenant policy state is unaffected (no auth/RBAC changes), so no Stytch-side rollback is required
- **Non-Goals**: no Temporal/DAG engine (linear pipeline; `signup_flows` is the documented future seam); no OTP interception — the OTP screen runs inside Meta's popup and per-attempt retry counts are not observable by this platform; no day-10 inactivity guardrail for Meta's 13-day coexistence rule (deferred to a follow-up needing a scheduler); no `smb_message_echoes` double-subscription — echoes arrive as `messages` with `origin.type='echo'`; no catalog/business-profile features (unavailable under coexistence); no encryption-at-rest for `access_token` in v1 (masked-on-read only); **rejects any local credential storage beyond the existing Meta platform-token precedent — Stytch remains the sole identity authority**

## Assumptions

- **Meta app pre-configuration (external, manual)**: a Meta Business app of type Business with the WhatsApp product, an embedded-signup config, and `business_management` permission must exist; the app's "Valid OAuth Redirect URIs" must include the platform webhook URL. Not verifiable from this repo
- **`redirect_uri` matching rules** for the server-side code exchange (empty vs. site-root value) confirmed during implementation against Meta behavior
- **Echo delivery shape**: coexistence echoes arrive under the subscribed `messages` field with `origin.type='echo'`; `from` is the customer, `to` is the business number
- **Coexistence availability**: requires the client's phone to run the WhatsApp Business app with the number active at signup time; otherwise Embedded Signup still works (OTP path, API-only), and the flow records `coexistence: false` in metadata — no server-side OTP handling either way
- **Test echo viability**: Cloud API accepts a message to the org's own registered number without template approval (Meta quickstart pattern)
- **KPIs unverifiable from code**: TTV < 3 min, churn 35%→<3%, BPO 0.75h→0.02h are business estimates, not code-verifiable
- **Tickets module**: exists (migration `000017`, `TicketsRoutes`, domain with `PriorityHigh` 8h SLA, FE `ticket-repository.ts`); if the module is feature-disabled at runtime (`ErrTicketModuleDisabled`), signup failures are logged and surfaced via `/signup/status` instead
