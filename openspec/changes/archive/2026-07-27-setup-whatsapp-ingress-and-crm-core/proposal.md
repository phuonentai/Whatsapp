## Why

The platform currently has no channel for receiving real-time messages from external messaging services. To power AI-driven conversational features (LLM agents, automated responses, notifications), we need a WhatsApp Cloud API ingress pipeline that validates, parses, and routes incoming messages into the CRM data layer — with clean module boundaries that support future policy injection (e.g., OPA/Rego) without modifying the pipeline.

## What Changes

- **New `whatsapp` module** (`internal/modules/whatsapp/`): Webhook endpoint at `POST /api/v1/webhooks/whatsapp` with HMAC-SHA256 signature validation, WhatsApp Cloud API JSON payload parsing, organization resolution via `phone_number_id` lookup, and `MessageReceived` event publishing to the platform eventbus.

- **New `crm` module** (`internal/modules/crm/`): Domain entities for Contact, Conversation, and Message, repository interfaces and SQLC-backed implementations, and an async event subscriber that processes `MessageReceived` events into persistent CRM records (contact upsert, conversation matching, message insertion with idempotency guards).

- **New database migrations**: `whatsapp.whatsapp_configs` (org-to-phone mapping + webhook secrets), `crm.contacts`, `crm.conversations`, and `crm.messages` tables — all scoped by `organization_id`.

- **Event-driven decoupling**: The webhook handler does minimal work (verify, resolve org, parse, publish event, return 200). CRM processing runs asynchronously via eventbus subscribers, keeping webhook latency low and enabling future OPA/Rego policy layers between the modules without code changes.

- **Colombian E.164 canonicalization**: Phone numbers are validated against the `^\+573\d{9}$` pattern as MVP scope; non-Colombian numbers are logged but not rejected.

- **No CRM CRUD handlers** in this change — the `crm` module is data-layer only at this stage.

## Capabilities

### New Capabilities

- `whatsapp-webhook-ingress`: WhatsApp Cloud API webhook reception with HMAC-SHA256 signature validation, `phone_number_id` to organization resolution, JSON payload parsing, and event publishing to the platform eventbus.

- `crm-core-data`: Contact, conversation, and message domain entities with organization-scoped repository interfaces and SQLC-backed implementations, including an async event subscriber that ingests `MessageReceived` events.

### Modified Capabilities

_None — this is a greenfield addition. No existing specs are modified._

## Impact

- **New modules**: `internal/modules/whatsapp/` and `internal/modules/crm/` — both following the existing Clean Architecture layered pattern (domain/app/infra/handler/routes/cmd).
- **New database schema**: Migration `000010` creating `whatsapp` and `crm` schemas with 4 new tables.
- **New SQLC queries**: `whatsapp.sql` and `crm.sql` query files.
- **DI registrations**: 4 new repository registrations in `internal/db/inject.go`; 2 new module init entries in `internal/bootstrap/init_mods.go`.
- **Shared utilities**: HMAC verification helper in `pkg/whatsapp/`.
- **No external dependencies**: Uses existing `eventbus`, `logger`, `httperr`, `sqlc.Store`, and `gin` — no new third-party packages.
