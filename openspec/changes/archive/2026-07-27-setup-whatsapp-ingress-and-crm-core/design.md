## Context

This project uses Clean Architecture with modular monolith structure in the Go B2B starter. Existing patterns include:

- **Layered modules**: `domain/` → `app/services/` → `infra/repositories/` → handlers
- **Dependency injection**: uber-go/dig container, repositories registered in `internal/db/inject.go`, modules wired in `internal/bootstrap/init_mods.go`
- **Event bus**: In-memory `eventbus.InMemoryEventBus` with concurrent goroutine-based subscribers, dot-notation event names (`document.uploaded`, `invoice.created`)
- **Database**: PostgreSQL via SQLC (pgx/v5), incremental migrations with `created_at`/`updated_at` triggers, `organization_id` FK to `organizations.organizations(id) ON DELETE CASCADE`
- **Auth**: Stytch JWT middleware with `RequireAuth` + `RequireOrganization` + permission guards — but webhooks need alternative signature-based validation
- **Webhook precedent**: `internal/platform/polar/webhook.go` provides standalone HMAC-SHA256 verification (Standard Webhooks format)

## Goals / Non-Goals

**Goals:**
- Receive WhatsApp Cloud API webhook payloads at `POST /api/v1/webhooks/whatsapp`
- Validate HMAC-SHA256 signatures against a per-organization webhook secret
- Resolve `organization_id` from the webhook payload's `phone_number_id` via a configs lookup table
- Publish a structured `MessageReceived` event to the platform eventbus
- Persist contacts, conversations, and messages in CRM tables scoped by `organization_id`
- Decouple ingress (whatsapp module) from storage (crm module) via eventbus subscribers

**Non-Goals:**
- LLM agent processing or automated responses to messages
- Frontend UI components for CRM or WhatsApp configuration
- CRUD API endpoints for managing contacts, conversations, or messages
- WhatsApp Cloud API message sending (outbound)
- Media download or storage (URLs are stored, not fetched)
- Multi-country E.164 support — Colombian numbers only for MVP
- Persistent message queue (in-memory eventbus is sufficient for MVP)

## Decisions

### Decision 1: Event-driven module decoupling via eventbus

**Choice**: WhatsApp module publishes `MessageReceived` events; CRM module subscribes and persists. WhatsApp does NOT depend on CRM repositories or services.

**Rationale**: This mirrors the existing `documents → cognitive` pattern where `cognitive/cmd/init.go` imports `documents/domain/events` and subscribes in its Init function. The webhook handler completes fast (under 50ms) and returns 200 before CRM processing begins. Future policy layers (OPA/Rego) can be inserted as eventbus middleware or as additional subscribers without modifying either module.

**Alternative considered**: Direct service call from WhatsApp handler to CRM service. Rejected because it couples the modules, increases webhook latency, and makes policy injection harder.

### Decision 2: Organization resolution via `whatsapp.whatsapp_configs`

**Choice**: A dedicated `whatsapp.whatsapp_configs` table maps `phone_number_id` (from WhatsApp webhook metadata) to `organization_id` and stores the `webhook_secret` per org.

**Rationale**: WhatsApp Cloud API webhooks include `entry[].changes[].value.metadata.phone_number_id` in every payload. This value uniquely identifies the business phone number. By looking it up in a configs table, we can resolve the organization before the Stytch auth middleware runs. The webhook secret is also stored per-org rather than as a single global env var, enabling multi-tenant deployment.

**Alternative considered**: Single `WHATSAPP_WEBHOOK_SECRET` env var with org derived from the `from` phone number. Rejected because it doesn't support multi-tenant (multiple business phone numbers, one per org).

### Decision 3: Webhook signature verification as standalone function

**Choice**: HMAC-SHA256 verification lives in `pkg/whatsapp/signature.go` as a standalone function, called inline in the webhook handler. Not a Gin middleware.

**Rationale**: The existing `polar/webhook.go` uses the same pattern — standalone function. The webhook route cannot use the standard auth middleware (no Stytch JWT), and the signature verification needs access to the raw body bytes before Gin parses JSON. A Gin middleware would need to buffer and re-read the body, which is possible but adds complexity for a single endpoint.

**Alternative considered**: Gin middleware with `c.GetRawData()`. Rejected because it's overengineered for one endpoint.

### Decision 4: Conversation matching by contact + 24-hour window

**Choice**: When a message arrives, find the most recent `active` conversation for the contact whose `last_message_at` is within 24 hours. If none exists, create a new conversation.

**Rationale**: WhatsApp Cloud API uses 24-hour customer service windows. This heuristic matches WhatsApp's session model without relying on WhatsApp's `conversation.id` field (which is not reliably present in all webhook event types).

**Alternative considered**: Always create new conversation per message. Rejected because it fragments conversation history.

### Decision 5: Colombian E.164 only for MVP

**Choice**: Validate sender phone numbers against `^\+573\d{9}$` (Colombian mobile: country code 57, prefix 3, 9 digits). Non-matching numbers are logged at WARN level and still processed (not rejected).

**Rationale**: The MVP targets Colombian businesses. Rejecting non-Colombian numbers would cause WhatsApp to retry indefinitely for valid international messages. Logging allows operators to detect unexpected traffic without data loss.

**Alternative considered**: Reject with HTTP 400. Rejected because WhatsApp retries on non-2xx responses, creating noise.

### Decision 6: CRM module is data-layer only (no handlers)

**Choice**: The CRM module exposes domain entities, repository interfaces, a service layer, and an event subscriber. No HTTP handlers or routes.

**Rationale**: CRM CRUD operations (listing contacts, searching conversations) are future scope. Building the data model and subscriber first establishes the foundation without overbuilding. The module follows the same layered structure as other modules, just omitting `handler.go` and `routes.go`.

### Decision 7: Idempotency via UNIQUE constraint on `whatsapp_message_id`

**Choice**: `crm.messages` has `UNIQUE(organization_id, whatsapp_message_id)` to prevent duplicate message insertion on webhook retries.

**Rationale**: WhatsApp may deliver the same webhook multiple times. The event subscriber checks for existing messages by `whatsapp_message_id` (the WhatsApp-assigned `wamid`) and silently skips duplicates.

### Decision 8: Webhook logs table for audit and replay

**Choice**: `whatsapp.webhook_logs` stores the raw webhook payload synchronously before event publishing.

**Rationale**: Since the eventbus subscriber runs asynchronously and the webhook already returned 200, a CRM processing failure would lose the message with no way to replay. The webhook log provides an audit trail and replay source. Logs are written synchronously (inside the handler, before event publish), so they're atomic with the webhook receipt.

## Risks / Trade-offs

- **[Message loss on subscriber failure]** → Mitigated by `whatsapp.webhook_logs` sync write. If CRM subscriber fails, the raw payload is preserved for replay. Still, no automatic retry — manual or future reconciliation needed.

- **[In-memory eventbus restarts]** → All pending events are lost on server restart. Acceptable for MVP since webhook logs preserve the raw data. A persistent queue (Redis Streams, NATS) should be considered before production.

- **[No outbound messaging]** → The `messages` table has a `direction` column supporting both `inbound` and `outbound`, but outbound sending (WhatsApp Cloud API POST) is out of scope. The schema is forward-compatible.

- **[Conversation matching heuristic]** → The 24-hour active window is a heuristic, not exact. Edge cases (e.g., messages exactly at the 24h boundary) may create a new conversation instead of appending to a recently-expired one. Acceptable trade-off for MVP.

- **[Single-country E.164]** → Only Colombian numbers pass canonicalization. International senders' messages are stored with unvalidated phone numbers. Migration to multi-country regex is a simple config change.

## Open Questions

- Should the `whatsapp.whatsapp_configs` table be populated via a seed migration or a future admin UI? (For MVP, manual DB insert is sufficient.)
- What is the exact WhatsApp HMAC header format? (`x-hub-signature-256: sha256=<hex>` is Meta's standard, but confirmation needed.)
- Should `crm.messages.content` be TEXT or could it hold large payloads? (TEXT is sufficient for WhatsApp's 4096-char text message limit.)
