## 1. Database — WhatsApp config schema extension

- [x] 1.1 [DB-SQLC] Create migration to add `waba_id VARCHAR(100)`, `access_token VARCHAR(500)`, `api_version VARCHAR(20) DEFAULT 'v21.0'`, `graph_api_url VARCHAR(255) DEFAULT 'https://graph.facebook.com'` columns to `whatsapp.whatsapp_configs`
- [x] 1.2 [DB-SQLC] Update `whatsapp.sql` queries: add new columns to `CreateWhatsAppConfig`, `UpdateWhatsAppConfig`, and all SELECT queries
- [x] 1.3 [DB-SQLC] Run `make sqlc` to regenerate Go code and verify compilation

## 2. Backend — WhatsApp config domain & service update

- [x] 2.1 [BE-DOMAIN] Add `WABAID`, `AccessToken`, `APIVersion`, `GraphAPIURL` fields to `domain/config.go` WhatsAppConfig struct
- [x] 2.2 [BE-DOMAIN] Update `domain/config.go` `Validate()` to not require `WABAID` and `AccessToken` (optional for inbound-only setups)
- [x] 2.3 [BE-INFRA] Update `infra/repositories/config_repository.go` to handle new fields (map SQLC result to domain struct)
- [x] 2.4 [BE-DOMAIN] Update `app/services/config_service.go`: mask `access_token` in `GetConfig` response, handle partial update for new fields in `updateConfig`, pass new fields in `createConfig`
- [x] 2.5 [BE-INFRA] Update `handler.go` `UpsertConfigRequest` struct to include `waba_id`, `access_token`, `api_version`, `graph_api_url`
- [x] 2.6 [BE-INFRA] Add a `GET /api/v1/whatsapp/config/health` endpoint that returns webhook log counts (total, last 24h, last 7d, last error) for the org

## 3. Backend — WhatsApp Cloud API HTTP client

- [x] 3.1 [BE-INFRA] Create `pkg/whatsapp/client.go` with `Client` struct: `SendTextMessage(ctx, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (messageID string, err error)`
- [x] 3.2 [BE-INFRA] Add circuit breaker to client (threshold: 5 failures, timeout: 10s, half-open probe: 2 requests, cooldown: 30s) following the org's resilience protocol pattern
- [x] 3.3 [BE-INFRA] Add unit tests for `client_test.go`: successful send, API error response, circuit breaker open/close transitions, timeout handling

## 4. Backend — CRM conversation & message API

- [x] 4.1 [DB-SQLC] Add SQLC queries: `ListConversationsByOrganization`, `GetConversationByID`, `UpdateConversationStatus`, `ListMessagesByConversation`, `GetMessageByWhatsAppID` (only `ListMessagesByConversation` and `GetMessageByWhatsAppID` already exist in repo — verify and add missing ones)
- [x] 4.2 [DB-SQLC] Add `GetConversationByID` SQLC query with org scoping (already existed)
- [x] 4.3 [DB-SQLC] Add `ListConversationsByOrganization` SQLC query with status filter, pagination, ordered by `last_message_at DESC`
- [x] 4.4 [DB-SQLC] Add `UpdateConversationStatus` SQLC query (already existed)
- [x] 4.5 [DB-SQLC] Add `ListMessagesByConversation` and `GetMessageByWhatsAppID` SQLC queries with org scoping (already existed)
- [x] 4.6 [DB-SQLC] Run `make sqlc` to regenerate Go code
- [x] 4.7 [BE-DOMAIN] Add missing repository interface methods to `domain/repository.go`: `ListConversationsByOrganization`, `GetConversationByID` (if not present), `UpdateStatus`
- [x] 4.8 [BE-INFRA] Create `internal/modules/crm/app/services/conversation_service.go` with `ListConversations`, `GetConversation`, `UpdateStatus`, `ListMessages` methods, each verifying organization ownership
- [x] 4.9 [BE-INFRA] Create `internal/modules/crm/handler/conversation_handler.go` with HTTP handlers for `ListConversations`, `ListMessages`, `UpdateStatus`
- [x] 4.10 [BE-INFRA] Add routes to `internal/modules/crm/routes.go`: `GET /crm/conversaciones`, `GET /crm/conversaciones/:id/mensajes`, `PATCH /crm/conversaciones/:id/status` under the existing CRM middleware chain

## 5. Backend — Outbound send service

- [x] 5.1 [BE-DOMAIN] Create `internal/modules/crm/app/services/outbound_service.go` with `SendMessage(ctx, orgID, convID int32, content string) (*domain.Message, error)` that: looks up WhatsApp config, validates `is_active` and `access_token`, looks up conversation contact phone, calls `pkg/whatsapp.Client.SendTextMessage`, persists outbound message with `whatsapp_message_id` from API response
- [x] 5.2 [BE-INFRA] Create `internal/modules/crm/handler/outbound_handler.go` with `HandleSendMessage` HTTP handler
- [x] 5.3 [BE-INFRA] Add route `POST /crm/conversaciones/:id/mensajes` to `internal/modules/crm/routes.go` with rate limiting (10 requests per 10 seconds per user)
- [x] 5.4 [BE-INFRA] Wire `OutboundService` and `WhatsAppConfigRepository` into CRM module DI (`module.go`, `provider.go`)

## 6. Frontend — API client layer

- [x] 6.1 [FE-NEXT] Create `lib/models/conversation.model.ts` with `Conversation`, `ConversationStatus`, and `ConversationListResponse` TypeScript interfaces
- [x] 6.2 [FE-NEXT] Add `MessageDirection`, `MessageStatus` to `lib/models/crm.model.ts` (or create `lib/models/message.model.ts` if preferred)
- [x] 6.3 [FE-NEXT] Create `lib/api/api/repositories/conversation-repository.ts` with `listConversations`, `getConversation`, `listMessages`, `updateStatus`, `sendMessage` methods
- [x] 6.4 [FE-NEXT] Create `lib/api/api/dto/conversation.dto.ts` with DTO types matching backend API responses
- [x] 6.5 [FE-NEXT] Add `conversations` and `messages` query keys to `lib/hooks/queries/query-keys.ts`
- [x] 6.6 [FE-NEXT] Create `lib/hooks/queries/use-conversations-query.ts` TanStack Query hook (5s refetch interval, 5s stale time)
- [x] 6.7 [FE-NEXT] Create `lib/hooks/queries/use-messages-query.ts` TanStack Query hook for message thread (5s refetch interval, enabled only when conversation is selected)
- [x] 6.8 [FE-NEXT] Create `lib/hooks/mutations/use-send-message.ts` mutation hook for outbound send
- [x] 6.9 [FE-NEXT] Create `lib/hooks/mutations/use-update-conversation-status.ts` mutation hook

## 7. Frontend — WhatsApp config form additions

- [x] 7.1 [FE-NEXT] Update `lib/models/whatsapp-config.model.ts` to add `wabaId`, `accessToken`, `apiVersion`, `graphApiUrl` fields
- [x] 7.2 [FE-NEXT] Update `lib/api/api/repositories/whatsapp-config-repository.ts` to include new fields in request/response
- [x] 7.3 [FE-NEXT] Update `app/dashboard/settings/components/whatsapp-config-section.tsx`: add WABA ID text input, access token password input (masked), API version input, graph API URL input
- [x] 7.4 [FE-NEXT] Add webhook callback URL display field with copy-to-clipboard button (uses `window.location.origin + "/api/v1/webhooks/whatsapp"`)
- [x] 7.5 [FE-NEXT] Add webhook health indicator using `GET /api/v1/whatsapp/config/health` — show green (recent success), yellow (no recent), or gray (not configured)

## 8. Frontend — Inbox page

- [x] 8.1 [FE-NEXT] Create `app/dashboard/inbox/page.tsx` — inbox page with master/detail layout (conversation list on left, message thread on right)
- [x] 8.2 [FE-NEXT] Create `app/dashboard/inbox/layout.tsx` with metadata title "Inbox | AP Cash"
- [x] 8.3 [FE-NEXT] Create `app/dashboard/inbox/components/conversation-list.tsx` — scrollable list of conversations with contact name/phone, last message preview (60 chars), relative timestamp, status badge, filter tabs (all/active/closed/archived)
- [x] 8.4 [FE-NEXT] Create `app/dashboard/inbox/components/message-thread.tsx` — message bubble list (inbound left-aligned gray, outbound right-aligned blue), media placeholder for non-text types, timestamp labels
- [x] 8.5 [FE-NEXT] Create `app/dashboard/inbox/components/reply-input.tsx` — text input with Send button, disabled when no conversation selected, loading state during send
- [x] 8.6 [FE-NEXT] Create `app/dashboard/inbox/components/conversation-header.tsx` — shows contact info, conversation status badge, close/reopen action button
- [x] 8.7 [FE-NEXT] Add empty state component for inbox with link to settings
- [x] 8.8 [FE-NEXT] Add skeleton loading states for conversation list and message thread

## 9. Frontend — Sidebar and navigation

- [x] 9.1 [FE-NEXT] Add "Inbox" entry to `components/layout/sidebar.tsx` `mainNavigation` array with `MessageCircle` icon from lucide-react, permission: `org:manage`
- [x] 9.2 [FE-NEXT] Ensure `/dashboard/inbox` redirects to `/dashboard` if user lacks `org:manage` permission (extend the permission guard pattern from layout.tsx)
