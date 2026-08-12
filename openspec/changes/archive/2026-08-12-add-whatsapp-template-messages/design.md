# Add WhatsApp Template Messages — Design

## Context

Meta's WhatsApp Cloud API only permits business-initiated messages inside the 24-hour customer-service window; outside it, business-initiated messages MUST use Meta-approved message templates. The `whatsapp-outbound-send` spec explicitly defers template support, which blocks cold/business-initiated flows (supplier inquiries, order confirmations, follow-ups, campaigns). This change adds an org-scoped template registry, submission to Meta, approval-status sync via webhook, and a template send path.

Repo facts grounding this design:

- Migrations live at `go-b2b-starter/internal/db/postgres/sqlc/migrations/`; latest is `000035_add_campaign_message`, so new migration is `000036_create_whatsapp_templates` (+ down), under the existing `whatsapp` schema (created by `000010_create_whatsapp_crm_schema`).
- An existing module `internal/modules/whatsapp/` already hosts domain (`domain/config.go`, `domain/repository.go`, `domain/events/`), `infra/repositories/`, `infra/graphapi/`, `app/services/`, plus handler/routes/module/provider and `cmd/init.go` DI wiring.
- The Cloud API client `pkg/whatsapp/client.go` already implements Bearer auth, configurable base URL, circuit breaker, and `SendTextMessage`; its circuit breaker opens after 5 consecutive 5xx within 10s and half-opens after 30s.
- Outbound text send endpoint `POST /crm/conversaciones/:id/mensajes` is registered in `internal/modules/crm/routes.go`; the new template send endpoint is a sibling route.
- Org WhatsApp credentials (`access_token`, `phone_number_id`, `waba_id`, `graph_api_url`, `api_version`) live in `whatsapp.whatsapp_configs` per `whatsapp-config-api`; webhook ingress (`POST /api/v1/webhooks/whatsapp`) validates `x-hub-signature-256`, resolves org by `phone_number_id`, and dispatches durable outbox events per `whatsapp-webhook-ingress`.
- Stytch B2B is the runtime SSOT for identity/RBAC; `org:manage` / `org:view` are defined in `internal/modules/auth/rbac.go` (`PermOrgManage`, `PermOrgView`).

## Goals / Non-Goals

Goals:

- Org-scoped template registry with a lifecycle (`draft → submitted → approved | rejected | paused`) kept in sync with Meta.
- Template sends that work outside the 24-hour window using only approved templates.
- Approval status kept current via webhook with idempotent, transaction-isolated updates.
- Keep the existing text send path and its outside-window warning behavior untouched.
- Spanish-first user-facing copy and validation errors.

Non-Goals:

- No local credential storage: no new tables or fields hold Meta or Stytch credentials.
- No AI copy generation, no preview rendering, no scheduled sends, no catalog/interactive-list/Instagram templates.
- No Stytch policy or RBAC role changes; no changes to the text send endpoint contract.

## Decisions

### D1 — Local-first template registry with Meta sync (vs Meta-only CRUD)

Local PostgreSQL (`whatsapp.templates`) is the UI source of truth, with Meta as the runtime authority for sendability.

- Rationale: offline-capable UX (orgs can author and edit drafts before any Meta interaction), org-scoped multi-tenant isolation via `organization_id` FK, and a hard send gate that checks local `status = 'approved'` AND `is_active` before contacting Meta. Meta-only CRUD would couple every list/edit to Graph API latency and failure modes and would make approval-state gating fragile.
- Alternatives considered: (a) Meta-only CRUD — rejected (latency, no offline drafts, harder org isolation); (b) full local replica with no Meta sync — rejected (Meta must remain the approval authority; local-only status would drift).
- Invariant: local status is updated only by explicit user action (draft edits, submit) or by Meta-sourced events (webhook status update, manual refresh). Sends require local `approved`; Meta confirms existence via `meta_template_id`.

### D2 — Dedicated template send endpoint (vs extending the text endpoint)

New `POST /crm/conversaciones/:id/mensajes/template` accepts `{"template_id": ..., "params": ["..."]}`.

- Rationale: the existing `POST /crm/conversaciones/:id/mensajes` contract (text payload, outside-window warning) is stable and covered by existing tests; extending it would complicate its validation, request shape, and 24h-window behavior. A dedicated endpoint keeps the change additive and lets template sends bypass the window check without touching TEXT semantics.
- Alternatives considered: adding a `type: "template"` switch to the text endpoint — rejected (contract churn, ambiguous error semantics, risk to the additive constraint).

### D3 — Approval-status sync via webhook plus manual refresh (vs polling)

The existing webhook ingress SHALL route Meta's `message_template_status_update` field to a template status-sync handler; `POST /api/whatsapp/templates/:id/sync` provides manual reconciliation.

- Rationale: webhooks give near-real-time approval state with no polling load; the durable outbox already provides retry/backoff and the ingress already validates `x-hub-signature-256` and resolves the org. A manual refresh endpoint covers webhook delivery gaps (Meta retries, outages) without adding a poller.
- Alternatives considered: periodic polling of `GET .../message_templates` — rejected (extra Meta API load, delayed state, no benefit over webhooks given the existing ingress).

### D4 — Parameter validation strategy

`param_count` is computed from `{{N}}` placeholders in the body at create/update time; sends require `len(params) == param_count`; submission builds Meta body components from the same placeholders.

- Rationale: a single source of truth (the body) keeps local/Meta drift detectable — a mismatch between `param_count` and Meta's component count is a sync error, not a validation error. Strict count matching at send time prevents malformed Cloud API payloads (Meta rejects wrong parameter counts).
- Alternatives considered: free-form params without count checks — rejected (Meta API errors at send time are worse than a local 400); client-side-only validation — rejected (backend must enforce the contract).
- Soft compliance note: template bodies containing obvious contact PII (E.164 phone, email) are warned about at authoring time (Spanish copy) but not hard-rejected, per the brief; consent/opt-out guardrails apply at send time per `whatsapp-compliance`.

### D5 — Extend `internal/modules/whatsapp/` (vs new `internal/modules/whatsapptemplates/`)

Templates live in the existing whatsapp module: new `domain/template.go` (entity + state machine), `domain/template_repository.go` (interface), `infra/repositories/template_repo.go` (SQLC-backed), `app/services/template_service.go`, and routes/handler additions.

- Rationale: the feature is inseparable from the whatsapp module's existing pieces — org WhatsApp config credentials, `infra/graphapi`, webhook ingress hooks, and DI wiring in `cmd/init.go`. A separate module would need cross-module access to whatsapp config and webhook routing, duplicating DI and creating coupling.
- Alternatives considered: new `internal/modules/whatsapptemplates/` module — rejected (duplicate config access and webhook wiring; the brief permits either, repo evidence favors the existing module).

### D6 — Webhook field routing in the existing ingress

The ingress already validates signature, resolves org, logs raw payload, and enqueues outbox events. This change adds a `message_template_status_update` branch: when the payload's `entry[].changes[].field` is `message_template_status_update`, the handler resolves the template by `meta_template_id` + resolved org and applies the status update transaction-isolated and idempotently (re-applying the same status is a no-op; unknown `meta_template_id` is logged and ignored, HTTP 200).

- Rationale: reuses the existing signature validation, org resolution, raw logging, and durable dispatch; keeps `whatsapp.message.received` handling untouched.
- Alternatives considered: a separate webhook endpoint for template events — rejected (Meta delivers one webhook URL; splitting requires Meta-side configuration changes and duplicate signature/org logic).

### D7 — Stytch boundary and RBAC mapping

No new Stytch API contract usage. New routes sit behind the existing `auth` + `org_context` + `subscription` middleware chain; writes (create/update/delete/submit/sync/send) require the Stytch B2B `org:manage` permission, listing requires `org:view`, mapped to the local `auth.RequirePermission("org", "manage")` / `("org", "view")` checks. Governance invariants: Go domain models (`internal/modules/whatsapp/domain/template.go`) MUST NOT import Stytch SDKs or transport packages; auth adapters implement domain interface abstractions; DB operations triggered by Meta webhooks (which carry the same transactional discipline as Stytch webhooks per the constitution) MUST be idempotent using transaction-isolated state checks. No credentials are stored in template tables.

## Risks / Trade-offs

- [Risk] Meta approval latency blocks cold sends → Mitigation: draft workflow plus status badges and a manual refresh endpoint; orgs author templates ahead of need.
- [Risk] Webhook delivery gaps leave local status stale → Mitigation: manual `POST /api/whatsapp/templates/:id/sync`; durable outbox retry/backoff on the ingress side.
- [Risk] Template body drift between local and Meta → Mitigation: store `meta_template_id`; sync on mismatch; param count checked at send time; refresh reconciles status.
- [Risk] Spam/compliance (Ley 1581) → Mitigation: reuse consent/opt-out guardrails at send time per `whatsapp-compliance`; soft PII warning at authoring time; template bodies are org-authored content.
- [Risk] Circuit breaker shared with text sends → Mitigation: `SendTemplateMessage` uses the same breaker; a Meta outage blocks both paths consistently (existing semantics), which is acceptable and observable.
- Trade-off: local-first registry means two sources of truth (local + Meta) — accepted because local gates sends and Meta stays authoritative for approval.

## Migration Plan

Additive migration `000036_create_whatsapp_templates` (+ `000036_create_whatsapp_templates.down.sql`) creating `whatsapp.templates` with columns `id`, `organization_id` (FK), `name`, `category`, `language`, `body`, `param_count`, `status`, `meta_template_id` (nullable), `rejection_reason` (nullable), `is_active`, `created_at`, `updated_at`, and a unique constraint on `(organization_id, name, language)`. Deploy order: run `make migrateup` → deploy backend (`make build`) → deploy frontend (`pnpm build`) → archive the change via `openspec archive`. No backfill needed (registry starts empty). Rollback: `make migratedown` + revert the change's files (see proposal Rollback).

## Open Questions

- Should a `rejected` template be resubmittable directly via `POST /api/whatsapp/templates/:id/submit` after editing, or must it return to `draft` first? (Proposal: allow submit from `draft` and `rejected`; final decision at implementation.)
- Does Meta's submission for this org's categories require additional component types (e.g., header/footer) beyond the body component? (Registry currently stores a single body; extension point: future `components` JSON column.)
- Should the soft PII warning become a hard validation rule if Meta starts rejecting such templates at scale? (Currently a design-level warning, not a spec requirement.)
