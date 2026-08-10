## Context

The E2E suite is strong on happy-path CRUD and primary webhook error cases but misses spec-mandated webhook edge cases and surrounding-process guarantees. The failed-webhook-logging requirement in `whatsapp-webhook-ingress` is unsatisfiable today: `whatsapp.webhook_logs.organization_id` is `NOT NULL` so unresolvable-org failures cannot be logged, and `webhook_service.go` returns before ever writing a `failed` row. Surrounding processes (cross-org isolation, pagination, outbound reply persistence, the mock-auth guard, RBAC delete boundary) have zero coverage.

Current `ProcessWebhook` ordering in `internal/modules/whatsapp/app/services/webhook_service.go`:

```
extract phone_number_id ──► GetByPhoneNumberID (is_active=true filter)
      ──► VerifySignature ──► Insert webhook_log(status=received) ──► publish events
```

Config lookup precedes signature verification, so an invalid-signature failure on a known `phone_number_id` already has a resolved org id available for logging; only unknown-phone/missing-phone failures have no org.

## Goals / Non-Goals

**Goals:**
- Satisfy the living `whatsapp-webhook-ingress` "failed webhook still logged" requirement by writing `status='failed'` rows with `error_message` before error returns.
- Allow failed logs when org is unresolvable via a nullable `organization_id` (new migration `000024`).
- Add E2E + Go unit coverage for the webhook edge cases and surrounding processes listed in the proposal.
- No change to inbound message processing, event publishing, or the 401/404/400 HTTP mappings.

**Non-Goals:**
- No message-processing or event-semantics changes.
- No RBAC role/permission changes.
- No real outbound WhatsApp sends (live token not available); reply persistence asserted through the offline mock-id path in `outbound_service.go`.
- No pagination API changes — existing default `limit=20` (max 100) behavior asserted as-is.
- No visual regression or new browser targets.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| `organization_id` nullability | Migration `000024` drops `NOT NULL` | Unknown-phone failures have no org; spec still requires the failed log row. Rollback: `.down.sql` restores `NOT NULL` |
| Failed-log insertion point | In `ProcessWebhook`, per error path, before returning | Config lookup precedes signature check, so known-org failures can log with org id |
| Failed-log shape | `status='failed'`, `error_message`, nullable org, plus `event_type`/`phone_number_id` where available | Reuses existing `InsertWebhookLog` (already accepts error_message via `raw_headers`/`error_message` columns); no new query needed for insert |
| Stats visibility | `GetWebhookLogStatsByOrganization` groups by status | `failed` count assertable for known-org failures via existing `GET /api/v1/whatsapp/config/health` |
| E2E assertion for failed log | Invalid-HMAC + known phone → 401, then health stats show a `failed` status row | Avoids direct DB access from tests; exercises the real endpoint |
| Edge-case spec layout | New `e2e/specs/whatsapp-edge-cases.spec.ts` | Keeps existing `whatsapp-inbox.spec.ts` focused on the primary flow |
| Surrounding-process layout | New `e2e/specs/surrounding-processes.spec.ts` | Cross-org isolation, pagination, reply persistence, mock-auth guard, RBAC delete in one capability-focused file |
| Cross-org calls | `e2e/helpers/api.ts` already parameterizes org; switch per request | Verified: `apiRequest` takes `orgSlug`/`email` (used by `seedWhatsAppConfig`) |
| Manager RBAC test | Use `member-pro@test.com` (role `member`) for forbidden-path against `org:manage`-gated endpoints; note seed has no real manager role | Verified: `roleFromMockEmail` derives role from email prefix; mock grants non-admin explicit perms minus `org:manage` |
| Echo payload | Add `buildEchoTextPayload` adding `message.origin.type="echo"` | Echo messages must publish the echo event, never an inbound row; assert no inbound row for that message id |

### Failed-logging flow (post-change)

```
POST /api/v1/webhooks/whatsapp
        │
        ▼
ProcessWebhook:
  extract phone_number_id ── empty? ──► insert webhook_log(status='failed', org=NULL)
        │                                   ──► return err ──► handler 500 webhook_processing_failed
  GetByPhoneNumberID ── miss? ──► insert webhook_log(status='failed', org=NULL)
        │                            ──► return ErrUnknownPhoneNumber ──► handler 404 unknown_phone_number
        │ found (config)
  VerifySignature ── fail? ──► insert webhook_log(status='failed', org=config.OrganizationID)
        │                          ──► return ErrInvalidSignature ──► handler 401 invalid_signature
        │ pass
  Insert webhook_log(status='received')   [unchanged]
  publish echo/received events            [unchanged]
```

### E2E assertions added

| Test | Org | Assertion |
|---|---|---|
| Inactive config → 404 | pro | seed config, set `is_active=false` via config endpoint, deliver signed payload → 404 `unknown_phone_number` |
| Invalid verify_token → 403 | pro | GET handshake with wrong token → 403 `verification_failed` |
| Malformed JSON → 400 | pro | POST non-JSON body with valid signature → 400 `invalid_json` |
| Failed webhook logged | pro | invalid HMAC + known phone → 401, then health stats show a `failed` status row |
| direction=inbound | pro | deliver valid inbound → GET `conversaciones/:id/mensajes` → message `direction === "inbound"` |
| Echo not inbound | pro | deliver echo payload → message persists with `direction != "inbound"` (echo stored as outbound; verified `message_data.origin = echo`) |
| Cross-org isolation | pro creates / free lists | pro-created contact absent from free org contacts list (API-level) |
| Pagination | pro | seed 25 contacts → default limit returns 20; `?limit=25&offset=20` returns remainder |
| Reply persistence | pro | POST `conversaciones/:id/mensajes` → GET shows `direction="outbound"` row |
| Mock-auth guard | API | request without `X-Test-Org-ID` under `AUTH_MOCK_ENABLED` → 401 |
| RBAC boundary | pro member | member `GET /api/v1/whatsapp/config` (org:manage-gated) → 403. NOTE: mock auth grants member all perms except `org:manage`, so member *can* delete CRM entities — delete-boundary premise invalidated at implement time |

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|---|---|---|
| Nullable `organization_id` weakens FK integrity | Orphaned failed logs with NULL org | Only NULL when org is genuinely unresolvable; `ON DELETE CASCADE` unaffected for resolved rows |
| `make test-e2e` still flaky (2 pre-existing: whatsapp-inbox idempotency stall, deals:91 request-drop) | Verification gate blocked | Record as owned by other changes; this change's specs verified in isolation |
| Echo-payload shape drift from Meta | False failure | Echo asserted only as "no inbound row", tolerant of optional fields |
| Pagination seed volume under parallel runs | Test-DB accumulation | Unique phone prefix per run; `run_e2e.sh` already drops/recreates `saas_db_test` each run |
| Member delete permission unknown until implement | Wrong 403 assert | Verified at implement: mock auth grants member `contact:delete`; asserted the real boundary (`org:manage`-gated whatsapp config) instead |
| `InsertWebhookLog` inserts before config resolution needs nullable org handling in sqlc gen | Gen model changes ripple | Regen via `make sqlc`; `WebhookLog.OrganizationID` becomes nullable (pgtype) |
