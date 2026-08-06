## Context

The WhatsApp module already has:
- A `whatsapp.whatsapp_configs` database table with per-organization WhatsApp Business API configuration (phone number ID, webhook secret, verify token, app ID, active toggle)
- A `ConfigRepository` with `GetByPhoneNumberID`, `GetByOrganizationID`, `Create`, and `Update` methods
- A `WebhookService` handling inbound webhook verification, signature validation, and event publishing
- Public webhook routes: `GET /api/v1/webhooks/whatsapp` (verification) and `POST /api/v1/webhooks/whatsapp` (payload)

What's missing:
- No management API endpoints exist — configs must be inserted/updated manually via SQL
- The WhatsApp routes are not registered in `internal/api/provider.go`, meaning even the webhook endpoints are unreachable in production
- The `VerifyChallenge` method in `WebhookService` does not check `hub.verify_token` against the stored config — it accepts any non-empty token
- The frontend has no UI for viewing or managing WhatsApp configuration

The frontend already has a workspace settings page at `/dashboard/settings` with a view stack pattern (`?view=profile|members|subscription`) that we'll extend.

## Goals / Non-Goals

**Goals:**
- Add backend API endpoints for an organization to get, upsert, and toggle its WhatsApp configuration
- Gate these endpoints with the existing auth middleware chain (auth + org_context + subscription) and `org:manage` permission
- Wire WhatsApp routes into `api/provider.go` so they're actually reachable
- Fix `VerifyChallenge` to validate `hub.verify_token` against the stored config
- Add a frontend admin UI integrated into the settings page with a new `whatsapp` view
- Mask secret fields (webhook_secret, verify_token) on read; accept them in full on write

**Non-Goals:**
- Multi-phone-per-org support (the DB schema enforces one config per org via UNIQUE on `organization_id`)
- Outbound message sending (separate feature, out of scope)
- Webhook analytics or message statistics (future scope)
- A separate admin route outside the settings page
- A new permission (`org:manage` is reused)
- react-hook-form integration (stays consistent with existing `useState` + `Input` pattern)
- Hard delete of configs (soft delete via `is_active` toggle)

## Decisions

### 1. Separate ConfigService from WebhookService

**Decision:** Create a new `ConfigService` interface and implementation in `app/services/config_service.go`.

**Rationale:** `WebhookService` handles inbound webhook processing — signature verification, payload parsing, event publishing. `ConfigService` handles CRUD management — get, upsert, toggle. These are distinct concerns. Keeping them separate avoids bloating `WebhookService` and follows the single-responsibility principle.

**Alternatives considered:**
- Extending `WebhookService` with management methods: rejected because it mixes auth-gated management with public webhook logic, making testing and DI harder.
- Putting CRUD directly in the handler: rejected because business logic (upsert logic, validation, masking) belongs in the service layer.

### 2. Management routes under auth middleware, webhook routes stay public

**Decision:** The existing public webhook routes (`GET/POST /api/v1/webhooks/whatsapp`) remain unchanged (no auth middleware — webhooks come from Meta). New management routes use the standard auth middleware stack.

```
# Public (existing, unchanged)
GET  /api/v1/webhooks/whatsapp          → HandleVerification
POST /api/v1/webhooks/whatsapp          → HandleWebhook

# Auth-gated (new)
GET   /api/v1/whatsapp/config           → GetConfig
PUT   /api/v1/whatsapp/config           → UpsertConfig
PATCH /api/v1/whatsapp/config/toggle    → ToggleConfig
```

**Rationale:** The webhook endpoint receives requests from Meta's servers, not authenticated users. Auth is cryptographic via `X-Hub-Signature-256`. The management endpoints are user-facing and must be protected.

### 3. Reuse `org:manage` permission

**Decision:** Gate all management endpoints on `auth.RequirePermissionFunc("org", "manage")` — the same permission used by member management and subscription management.

**Rationale:** Configuring WhatsApp is an organization-level operation. Creating a new `whatsapp:manage` permission adds complexity with no clear benefit at this stage. The existing permission model maps cleanly — org admins manage org-level configuration.

### 4. Frontend: Settings page view stack pattern

**Decision:** Add `"whatsapp"` to the `SettingsView` union type in `settings-content.tsx`, add one entry to `overviewSections`, and one `case "whatsapp"` branch in `renderDetailContent()`.

**Rationale:** The view stack pattern (`?view=profile|members|subscription`) is well-established and keeps the settings page cohesive. A separate route would fracture the management experience and require new layout wrappers.

### 5. No new repository methods needed

**Decision:** The existing `ConfigRepository` methods (`GetByOrganizationID`, `Create`, `Update`) are sufficient. No `List` or `Delete` methods are needed.

**Rationale:** One config per org means `GetByOrganizationID` serves as both "get" and "list." The `is_active` column provides soft-delete — toggling it via `Update` is sufficient. No pagination or search needed for a single-record-per-org domain.

### 6. Secrets masked on read, accepted in full on write

**Decision:** The `GET` response masks `webhook_secret` and `verify_token` (e.g., `"whsec_****abcd"`). The `PUT` request accepts them in full. On `PUT`, if a secret field is empty or masked, the existing stored value is preserved (partial update via `COALESCE` in SQLC).

**Rationale:** Users shouldn't re-read secrets from the API, but they should be able to update them. The `COALESCE` pattern in the existing `UpdateWhatsAppConfig` SQLC query already supports this: `COALESCE($N, column)` preserves the existing value when the input is NULL. The service layer adds a masking check to distinguish "keep existing" from "update to new value."

### 7. Fix VerifyChallenge to validate against stored config

**Decision:** The `VerifyChallenge` method in `WebhookService` currently accepts any non-empty token. It must be updated to look up the config by the `hub.verify_token` query param and validate it.

**Rationale:** The existing spec (`whatsapp-webhook-ingress`) already requires this behavior ("the system SHALL compare `hub.verify_token` against the configured verify token for the resolved organization"). The current implementation has a gap. The challenge endpoint cannot resolve by phone_number_id (it's not in the query params), so we iterate over active configs to find a matching verify_token. This is acceptable because verify_tokens are user-defined and should be unique per Meta's requirements.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| **Concurrent config edits**: Two admins editing the config simultaneously could result in lost updates | Acceptable for MVP (single org, few admins). If needed later, add optimistic locking via `updated_at` checks. |
| **Verify_token collision across orgs**: If two orgs use the same verify_token, the GET challenge endpoint can't disambiguate which org is being verified | Meta requires verify_tokens to be unique within an app. Add a UNIQUE constraint on `verify_token` in a follow-up if collisions become an issue. For MVP, iterate all active configs and return the first match. |
| **Route registration ordering**: If `api.Init()` runs before WhatsApp module init, the container won't have `WhatsAppRoutes` provided | The existing `init_mods.go` already initializes WhatsApp before API (#20 vs #23). This ordering is preserved. |
| **Breaking change to VerifyChallenge**: Users who relied on the "accept any token" behavior might have webhook verifications that silently break | The implementation was already wrong per spec. Any working Meta webhook subscription would have used a correctly configured verify_token. Risk is low. |
