## Context

The WhatsApp module currently handles inbound-only flow: webhook → signature verification → event publish → CRM storage. There is zero outbound capability and no UI for viewing conversations. The settings panel stores only webhook credentials (`phone_number_id`, `webhook_secret`, `verify_token`) — missing the WABA ID and access token needed to call the WhatsApp Cloud API for sending messages.

This design covers three integrated changes:
1. Backend API to expose stored conversations and messages
2. WhatsApp Cloud API HTTP client for sending outbound messages
3. Frontend inbox page and settings panel additions

The architecture follows the existing Clean Architecture pattern (domain → app → infra) used by both the CRM and WhatsApp modules.

## Goals / Non-Goals

**Goals:**
- Expose paginated conversation list and message thread via REST API (`/crm/conversaciones`)
- Enable sending text replies from the inbox UI via WhatsApp Cloud API
- Add WABA ID, permanent access token, API version, and graph API URL to WhatsApp config
- Display webhook callback URL and webhook health in settings
- Add "Inbox" navigation entry to the sidebar
- Conversation status management (close, archive) from the inbox

**Non-Goals:**
- Real-time message delivery via WebSocket (uses polling via TanStack Query refetch)
- Media/file upload for outbound messages (text-only replies initially)
- WhatsApp template message support (24-hour window is tracked but not enforced)
- Notification sounds or desktop push notifications
- Bulk messaging or broadcast lists
- Multi-channel abstraction (WhatsApp is the only channel)
- Stytch integration changes (no auth model changes needed)

## Decisions

### Decision 1: Reuse existing CRM repositories for conversation/message API

**Choice**: Add new handler methods to `internal/modules/crm/` using existing `ConversationRepository` and `MessageRepository` interfaces.

**Alternatives considered:**
- A) New standalone inbox module → adds unnecessary module boilerplate when the CRM already owns these entities
- B) Expose via WhatsApp module → mixes concerns (WhatsApp module handles webhook ingress, CRM module owns the stored data)

**Rationale**: The CRM module already owns contacts, conversations, and messages. Adding list/read endpoints there follows the Single Responsibility principle. The existing repos have `ListByConversation`, `GetByID`, and `ListActiveByOrganization` methods that just need HTTP handlers.

### Decision 2: WhatsApp Cloud API client as a shared utility package

**Choice**: Create `pkg/whatsapp/client.go` as a standalone HTTP client (no domain layer dependency), used by a new `app/services/outbound_service.go` in the CRM module.

**Alternatives considered:**
- A) Put client in `internal/modules/whatsapp/` → blocks the CRM from calling it without a WhatsApp → CRM direction dependency
- B) Use a third-party Go WhatsApp library → adds dependency risk; the Cloud API is a simple REST API

**Rationale**: The WhatsApp Cloud API is a simple REST interface (`POST /{version}/{phone-number-id}/messages`). A lightweight `pkg/` client keeps it reusable and testable. The outbound service in CRM owns the business logic (look up config, build message payload, call client, persist outbound message).

```
┌──────────────────────────────────────────────┐
│  CRM Module                                  │
│  ┌──────────────────────────────────────┐    │
│  │  app/services/outbound_service.go    │    │
│  │  - Look up WhatsApp config by org-id │    │
│  │  - Build message payload             │    │
│  │  - Call pkg/whatsapp client          │    │
│  │  - Persist sent message              │    │
│  └──────────────┬───────────────────────┘    │
│                 │                             │
│                 ▼                             │
│  pkg/whatsapp/client.go                      │
│  - POST graph.facebook.com/{v}/{phone}/msgs  │
│  - Bearer token auth                         │
│  - Circuit breaker                           │
│  - Response parsing                          │
└──────────────────────────────────────────────┘
```

### Decision 3: Polling with TanStack Query for inbox updates

**Choice**: Use 5-second `refetchInterval` on TanStack Query for conversation list and message thread, with `staleTime: 5000`.

**Alternatives considered:**
- A) Server-Sent Events (SSE) → requires a persistent connection architecture not yet in the codebase; overkill for v1
- B) WebSocket → same infrastructure gap; better suited for a follow-up change

**Rationale**: The codebase already uses TanStack Query extensively. Polling is simple, reliable, and appropriate for a B2B SaaS inbox (chat volume is moderate). The 5s interval can be tuned via environment variable later.

### Decision 4: Access token stored in PostgreSQL (encrypted at rest)

**Choice**: Store the WhatsApp permanent access token in the `whatsapp_configs` table, masked on read (like webhook_secret). Not encrypt at application layer.

**Rationale**: This follows the existing pattern for `webhook_secret` and `verify_token` — stored in plaintext in the DB column but masked in API responses. Adding application-layer encryption would be a separate change. The token is scoped to the WhatsApp Business Account and can be rotated if compromised. This is acceptable for initial delivery given the target audience (B2B SaaS with admin-only settings access).

### Decision 5: Inbox page layout — two-pane master/detail

**Choice**: Side-by-side conversation list (left pane) and message thread (right pane) on desktop, collapsing to stack on mobile.

```
Desktop (>1024px):
┌──────────────┬──────────────────────────┐
│ Conversation │  Message Thread          │
│ List         │  ┌────────────────────┐  │
│ ┌──────────┐ │  │ Contact: +57...    │  │
│ │+57300... │ │  │ Status: active     │  │
│ ├──────────┤ │  ├────────────────────┤  │
│ │+57301... │ │  │ 12:30 - hello      │  │
│ ├──────────┤ │  │ 12:31 - hi there!  │  │
│ │+57302... │ │  │                    │  │
│ └──────────┘ │  │ [Reply input]      │  │
│              │  └────────────────────┘  │
└──────────────┴──────────────────────────┘
```

**Rationale**: Master/detail is the standard pattern for messaging UIs (WhatsApp, iMessage, Slack, Intercom). The sidebar layout has `lg:pl-64` for the main sidebar, so the inbox internal layout uses its own flex split.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Access token exposure** via API response if masking fails | Token compromise → unauthorized message sending | Config service already masks on Get; add a dedicated `accessTokenMasked` check in tests |
| **WhatsApp Cloud API rate limits** (250 messages/second per phone number) | Deliverability throttling | Circuit breaker on client; queue not implemented in v1 (document as known limitation) |
| **Polling load on database** with many concurrent users refreshing | Increased DB query load | TanStack Query deduplication; `staleTime` prevents redundant fetches; pagination limits row counts |
| **Outbound message failure without delivery feedback** | User doesn't know if message was sent | Persist message as `status: "sent"` immediately; webhook status callbacks (future) update to `delivered`/`read` |
| **Access token rotation** by Meta requires manual update in settings | Outbound messages fail silently | Show "last webhook received" timestamp in settings as proxy health indicator; token expiry is years, not days |

## Open Questions

1. **Should we enforce the 24-hour messaging window?** The `Conversation.IsWithin24HourWindow()` method exists but is unused. For v1, we allow sending and show a warning if the window is closed. Template messages (for outside 24h window) are out of scope.

2. **Should webhook health be a separate endpoint or inline in the config response?** Inline in config response is simpler for v1. A dedicated metrics endpoint could come later.

3. **Do we need message read status tracking?** The WhatsApp Cloud API provides delivery status webhooks (`sent`, `delivered`, `read`, `failed`). The webhook parser already detects `statuses` but doesn't process them. This is deferred to a follow-up change.
