# Add WhatsApp Template Messages

## Why

Meta's WhatsApp Cloud API only allows business-initiated messages inside the 24-hour customer-service window; outside that window, business-initiated messages MUST use Meta-approved message templates. The `whatsapp-outbound-send` spec explicitly defers template support — its "24-hour messaging window closed" scenario returns HTTP 200 + warning `outside_24h_window` and notes "(template message support not yet implemented)". This blocks every cold/business-initiated flow: daily supplier inquiries (capability `supplier-inquiries`), order confirmations, follow-ups, and campaign sends. This change delivers template message support end to end: an org-scoped template registry, submission to Meta, approval-status tracking, and a template send path through the existing Cloud API client.

## What Changes

- **NEW capability `whatsapp-templates`:** org-scoped template registry (`whatsapp.templates`), template lifecycle (`draft → submitted → approved | rejected | paused`, synced with Meta), submission to Meta via the Graph API, approval-status sync via the existing webhook ingress (`message_template_status_update` field), a manual status refresh endpoint, and a template send endpoint.
- **MODIFIED `whatsapp-outbound-send`:** the WhatsApp Cloud API HTTP client (`pkg/whatsapp/client.go`) gains a `SendTemplateMessage` method that posts a `type: "template"` payload; template sends are not subject to the 24-hour messaging window. Existing plain-text behavior stays additive and unchanged (the outside-window TEXT scenario keeps HTTP 200 + `outside_24h_window` warning).
- **Backend:** SQLC migration `000036_create_whatsapp_templates` (+ down), domain entity, repository, and service inside the existing `internal/modules/whatsapp/` module, API routes behind the existing `auth` + `org_context` + `subscription` middleware, RBAC `org:manage` (manage) / `org:view` (list/read).
- **Frontend:** template management page in Settings (create/edit draft, `{{N}}` param placeholders, status badges, Spanish copy) reusing `whatsapp-config-frontend` patterns, TanStack Query, shadcn/ui.
- **No BREAKING changes:** this change is purely additive; no existing endpoint, spec, or behavior is altered.

## Capabilities

### New Capabilities
- `whatsapp-templates`: org-scoped template registry, submission to Meta, approval-status sync via webhook, manual status refresh, and the template message send endpoint.

### Modified Capabilities
- `whatsapp-outbound-send`: the WhatsApp Cloud API HTTP client gains `SendTemplateMessage`; template sends bypass the 24-hour window check.

## Impact

- **Code:** migration pair `000036_*`; domain entity + state machine + repository + service under `internal/modules/whatsapp/`; new routes registered in the whatsapp module and `internal/api/provider.go`; `SendTemplateMessage` in `pkg/whatsapp/client.go`; template status-sync handler wired into the existing webhook ingress event routing; frontend templates page under `/dashboard/settings`.
- **API:** new `POST /api/whatsapp/templates`, `PATCH /api/whatsapp/templates/:id`, `DELETE /api/whatsapp/templates/:id`, `GET /api/whatsapp/templates`, `POST /api/whatsapp/templates/:id/submit`, `POST /api/whatsapp/templates/:id/sync`, and `POST /crm/conversaciones/:id/mensajes/template`.
- **Deps:** no new third-party dependencies; Meta Graph API via the existing `pkg/whatsapp` client, Stytch B2B via the existing auth middleware (JWKS-verified sessions, no new Stytch API contract usage).
- **Systems:** Meta WhatsApp Cloud API (message templates + `message_template_status_update` webhooks), existing durable outbox, existing `whatsapp.whatsapp_configs` for credentials, Stytch B2B for session validation and RBAC (`org:manage` / `org:view` unchanged).

## Non-Goals

- **No local credential storage:** template records and any new tables SHALL NOT store Meta `access_token`, `webhook_secret`, `verify_token`, or Stytch credentials; credentials continue to live only in `whatsapp.whatsapp_configs` (as today) and in Stytch, per the repo's Dual SSOT constitution.
- No template-creation wizard beyond submit + sync: no AI copy generation, no preview rendering, no category recommendation.
- No scheduled or bulk template sends in this change (daily supplier inquiries consume this capability via the separate `add-supplier-inquiry-agent` change).
- No catalog or interactive-list template messages; no Instagram template support.
- No changes to the existing text-message send path or its 24-hour-window behavior for TEXT.
- No changes to Stytch tenant policies or RBAC role definitions; new routes reuse existing roles and permissions.

## Rollback

- **Git state:** revert this change's touched files (code, routes, specs, frontend) and run the down migration `000036_create_whatsapp_templates.down.sql` to drop `whatsapp.templates`. Verify the revert with `make build` and `pnpm build` before closing the rollback.
- **Stytch tenant policy state:** no Stytch policy, role, or permission changes are introduced, so no Stytch tenant-policy rollback is required; the new routes sit behind the existing `auth` + `org_context` + `subscription` middleware and reuse existing Stytch B2B RBAC permissions (`org:manage`, `org:view`), which are unchanged.

## Assumptions

- Meta template approval is external and time-variable; approval latency cannot be controlled from this codebase (external service behavior).
- The webhook field name is `message_template_status_update` per Meta Cloud API documentation (external, not verifiable from this repo).
- The org's WhatsApp config (`access_token`, `phone_number_id`, `graph_api_url`, `api_version`) is the credential source for template submission and sends, as specified by `whatsapp-config-api`; no new credential storage is introduced.
- Local template status is the UI source of truth; Meta is the runtime authority for sendability — a send requires local `status = 'approved'` AND `is_active = true`, with the template existing at Meta (confirmed via stored `meta_template_id`).
- The existing `pkg/whatsapp/client.go` circuit-breaker semantics (5 failures / 10s window, half-open probe after 30s) extend to `SendTemplateMessage`.
- Contact PII in template bodies is discouraged but not hard-rejected; compliance guardrails (consent/opt-out) apply at send time per `whatsapp-compliance`.
