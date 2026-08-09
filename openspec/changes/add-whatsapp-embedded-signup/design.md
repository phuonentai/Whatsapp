## Context

WhatsApp provisioning is manual today: `PUT /api/v1/whatsapp/config` (`internal/modules/whatsapp/app/services/config_service.go`) accepts `phone_number_id`, `business_phone`, `webhook_secret`, `verify_token` pasted from the Meta dashboard; the user then subscribes webhooks in Meta manually. The webhook ingress (`webhook_service.go`) verifies HMAC signatures, resolves the org via `phone_number_id`, logs to `whatsapp.webhook_logs`, and publishes `whatsapp.message.received`; the CRM listener persists inbound messages idempotently. The outbound seam exists: `pkg/whatsapp.ClientWithBreaker.SendTextMessage` with a two-tier circuit breaker. `whatsapp_configs` already carries `app_id`, `waba_id`, `access_token`, `api_version`, `graph_api_url` (migration `000014`) — the token-based outbound columns exist and are unused. The eventbus is in-process; subscriber work never blocks the webhook HTTP response.

The in-flight agent change (unmerged) takes migration `000019`; this change uses `000020`. The tickets module (migration `000017`, `TicketsRoutes`, domain `TicketStatus`/`TicketPriority`, 8h SLA for high priority) provides the HITL queue. The settings UI stack (`whatsapp-config-section.tsx`) is where the connect flow lands.

Meta Embedded Signup owns the interactive portion (WABA select/create, number select, OTP where needed) inside a Meta-hosted popup. Per-attempt OTP retry counts are not observable by this platform — only final success/error reaches the login callback. Coexistence is automatically offered by Meta when the selected number is an active WhatsApp Business app number on the client's phone; echoes of phone-app-sent messages arrive on the webhook as `messages` entries with `origin.type = "echo"`.

## Goals / Non-Goals

**Goals:**
- Self-serve connect: one click → Meta popup → server-side provisioning, target first-value (active dual-channel connection) in under 3 minutes
- Coexistence-first: client keeps the mobile WhatsApp Business app; phone-app-sent messages mirrored into the CRM inbox as outbound
- Webhook auto-registration with server-generated `webhook_secret`/`verify_token` (no manual pasting)
- Test-echo validation before reporting `connected`
- Deterministic, retryable state machine in Postgres (repo idiom; no workflow engine)
- HITL only on terminal failures, via the existing tickets module
- Zero new infrastructure dependencies

**Non-Goals:**
- No Temporal/DAG workflow engine (linear pipeline; `signup_flows` is the documented future seam)
- No OTP interception — OTP runs inside Meta's popup and is not observable server-side
- No day-10 inactivity guardrail for Meta's 13-day coexistence rule (deferred to a follow-up requiring a scheduler)
- No `smb_message_echoes` double-subscription — echoes arrive under `messages` with `origin.type='echo'`
- No catalog/business-profile features (unavailable under coexistence)
- No encryption-at-rest for the system-user token in v1 (masked-on-read only)
- No removal of the manual paste form (repair/reconnect path)
- No changes to Stytch identity flows; no local credential storage beyond the existing Meta platform-token precedent

## Decisions

### D1: `whatsapp.signup_flows` — migration `000020`

```
whatsapp.signup_flows
  id BIGSERIAL PK
  organization_id INT NOT NULL REFERENCES organizations.organizations(id) UNIQUE
  status VARCHAR(20) NOT NULL DEFAULT 'exchanging'  -- exchanging|registering|verifying|connected|failed
  step VARCHAR(40)                                  -- current sub-step for recovery
  error_code VARCHAR(100)
  retry_count INT NOT NULL DEFAULT 0
  metadata JSONB NOT NULL DEFAULT '{}'              -- waba_id, phone_number_id, coexistence, ticket_id
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

One row per org (unique constraint gives natural single-flight semantics). Chosen over columns on `whatsapp_configs` because the flow precedes a config existing and keeps provisioning metadata (retry state, error codes, ticket refs) out of the config row. `.down.sql` drops the table. This mirrors the `conversation_flows` seam pattern used by the agent change.

### D2: State machine + idempotency

```
exchanging ─▶ registering ─▶ verifying ─▶ connected
    │              │             │
    ▼              ▼             ▼
 failed ◀───────── all terminal states (error_code + retry_count)
```

- `POST /signup/exchange` is single-flight per org: `409 signup_in_progress` while a flow is mid-flight; `409 signup_already_connected` when already connected
- A consumed OAuth code cannot be reused; retries after code consumption resume at the recorded `step` using the persisted system-user token
- `verifying` = test echo to the org's own `business_phone`; success → `connected` and `is_active=true` on the config
- Terminal failure after 3 backoff retries → high-priority ticket via the tickets module service (fallback: log + surface `error_code` via `/signup/status` if the module is disabled)
- Alternative considered: a full event-sourced DAG — rejected as overkill for a linear flow; `signup_flows.step` retains enough recovery data

### D3: Graph API client (`infra/graphapi/`)

Stateless HTTP client in `internal/modules/whatsapp/infra/graphapi/` exposing: `ExchangeCode`, `FetchMe`, `ResolveWABAAndNumbers`, `CreateSystemUser`, `SubscribeWABA`, `RegisterAppSubscriptions`, `SendTestMessage`. Configurable `api_version`/`graph_api_url` (read from existing config defaults `v21.0`/`graph.facebook.com`). Defines a `Client` interface with the real implementation and a mock for tests. All outbound calls wrapped in the existing two-tier circuit-breaker semantics (threshold 5 / 10s / half-open probe 2) per governance — reused from the `pkg/whatsapp` breaker pattern rather than duplicating it. Alternative considered: calling Graph directly from the orchestrator — rejected; a transport seam is required for testability and matches the clean-architecture infra-adapter rule.

### D4: Secrets and token handling

- `webhook_secret` + `verify_token` generated server-side (crypto/rand, ≥32 chars) during registration; `verify_token` is passed to `/{app_id}/subscriptions`; both stored on the config row and only ever read back masked
- Transient OAuth code + user token live in request memory only; the permanent system-user token (created via `business_management`) is persisted in the existing `whatsapp_configs.access_token` column (VARCHAR(500), already masked on read in `config_service.go`)
- This respects the SSOT constitution: Stytch remains the sole identity authority; the only local credential is the Meta platform token, same precedent as today's manual flow

### D5: Echo handling (webhook ingress delta)

In `webhook_service.go`, per `messages[]` entry: if `origin.type == "echo"`, publish `whatsapp.message.echo` instead of `whatsapp.message.received`; the CRM listener persists `direction='outbound'` with the existing `(organization_id, whatsapp_message_id)` idempotency. Signature validation and org resolution are unchanged — echoes flow through the same endpoint with the same HMAC and `phone_number_id` resolution.

### D6: API surface

- `GET /api/v1/whatsapp/signup/meta-config` — bootstrap `{app_id, config_id, redirect_uri}` for the FE SDK
- `POST /api/v1/whatsapp/signup/exchange` — body `{code}`; orchestrates D2 flow
- `GET /api/v1/whatsapp/signup/status` — flow state + `error_code` for recovery UI
- All behind `auth` + `org_context` + `subscription` middleware with `org:manage`; registered in the whatsapp module routes (module already wired in `internal/api/provider.go`)

### D7: Frontend flow

`meta-config` fetch → lazy-load Meta Business JS SDK (`connect.facebook.net/en_US/sdk.js`) → `FB.login(cb, {config_id, response_type:'code', override_default_response_type:true})` → `POST /signup/exchange` → 4-state micro-status rendered client-side during the await (connecting → verifying → webhooks → live) → success shows connected summary with an "Advanced" disclosure containing the existing manual form for repair/reconnect. Failure state shows the returned `error_code` and a support CTA.

### D8: Env config

New env vars `WHATSAPP_APP_ID`, `WHATSAPP_APP_SECRET`, `WHATSAPP_SIGNUP_CONFIG_ID`. One-time external Meta app setup (Business app with WhatsApp product, embedded-signup config, `business_management` permission, Valid OAuth Redirect URIs) is an explicit Assumption in the proposal.

### D9: HITL escalation

Terminal signup failures create a high-priority ticket via the tickets module (8h SLA default per `DefaultSLASeconds`); subject includes `error_code` + org id; body includes step, retry count, and webhook log reference. If the tickets module is feature-disabled (`ErrTicketModuleDisabled`), the failure is logged and surfaced via `/signup/status` only.

## Risks / Trade-offs

- [Meta popup flow changes (OTP, coexistence dialogs) are opaque] → Mitigation: surface callback `error` params verbatim into `signup_flows.error_code`; HITL ticket includes the raw Meta error
- [Coexistence requires the client's phone app to be opened every 13 days] → Mitigation: deferred day-10 guardrail documented as Non-Goal; not part of this change
- [OAuth code is single-use; a mid-flow crash after exchange cannot be replayed from the same code] → Mitigation: resume at recorded `step` with the persisted system-user token
- [Meta app secrets in env (no secret manager)] → Mitigation: matches existing env-invariant pattern; never logged; masked config reads
- [Echo parsing depends on Meta payload shape] → Mitigation: unit tests with recorded payload fixtures; unknown `origin` values default to inbound (today's behavior)
- [Tickets module may be feature-disabled] → Mitigation: graceful fallback to log + status endpoint

## Migration Plan

1. `make migrateup` applies `000020` (additive; no backfill needed)
2. Deploy backend (Graph client, signup service, handlers, echo detection, CRM echo listener)
3. Deploy frontend (connect flow)
4. One-time external Meta app setup (Assumption) before the flow is exercised
5. Rollback: Git revert of the change; `000020.down.sql` drops `signup_flows`; no Stytch-side rollback required; failed signups leave no platform-side state beyond standard app subscription records

## Open Questions

- Exact `redirect_uri` value that satisfies Meta's code exchange for the embedded popup (empty vs. site root) — confirm during implementation against the Meta app config
- Whether `smb_message_echoes` subscription is ever needed in addition to `messages` for full mirror coverage — default is `messages` + `statuses` only
