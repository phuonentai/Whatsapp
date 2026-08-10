## Why

The WhatsApp integration currently handles inbound messages (webhook → CRM storage) but has zero outbound capability and no UI for viewing conversations. The settings panel for connecting to Meta is also incomplete — missing critical fields (WABA ID, permanent access token, callback URL display) needed for a working production setup. This change bridges the gap from "passive message logging" to a functional WhatsApp inbox.

## What Changes

- **New inbox page** at `/dashboard/inbox` with conversation list and message thread view
- **New backend API endpoints** for listing conversations (`GET /crm/conversaciones`) and messages (`GET /crm/conversaciones/:id/mensajes`) scoped to the organization
- **Settings panel expansion** — add WABA ID, permanent access token fields, callback URL display, and webhook health indicator to the WhatsApp config section
- **New navigation entry** — "Inbox" in the dashboard sidebar, permission-gated to `org:manage`
- **Outbound message reply API** — `POST /crm/conversaciones/:id/mensajes` to send text replies via WhatsApp Cloud API (graph.facebook.com)
- **WhatsApp Cloud API HTTP client** — a new `pkg/whatsapp/client.go` with send-message capability, token management, and circuit breaker

## Capabilities

### New Capabilities
- `whatsapp-inbox`: Frontend inbox page with conversation list, message thread view, and inline reply composer. Conversation status management (close, archive). Real-time-ish polling via TanStack Query refetch intervals.
- `crm-conversation-api`: Backend REST API for listing organization-scoped conversations with last message preview, listing messages within a conversation (paginated), and updating conversation status.
- `whatsapp-outbound-send`: Backend service and WhatsApp Cloud API HTTP client for sending text messages via the `POST /{phone-number-id}/messages` Graph API endpoint. Includes token-based auth, circuit breaker, and delivery callback support.

### Modified Capabilities
- `whatsapp-config-api`: Add `waba_id`, `access_token`, `api_version` (default `v21.0`), and `graph_api_url` (default `https://graph.facebook.com`) fields to the WhatsApp config domain entity, repository, SQL schema, and API response.
- `whatsapp-config-frontend`: Add WABA ID, permanent access token, and API version fields to the settings form. Display the webhook callback URL. Add a webhook health indicator showing recent webhook log status counts.

## Impact

- **Database**: Migration to add new columns to `whatsapp.whatsapp_configs` (`waba_id`, `access_token`, `api_version`, `graph_api_url`)
- **Backend modules**: New `pkg/whatsapp/client.go` (HTTP client), modified `internal/modules/whatsapp/domain/config.go`, `internal/modules/whatsapp/app/services/config_service.go`, new conversation and message handlers in `internal/modules/crm/`
- **Frontend**: New `app/dashboard/inbox/` page, modified `whatsapp-config-section.tsx`, modified `sidebar.tsx`, new hooks, queries, and repository files
- **No breaking changes** — existing config fields and APIs remain backward-compatible
