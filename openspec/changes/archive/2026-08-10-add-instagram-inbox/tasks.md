## 1. DB Migration [DB-SQLC]

- [x] Confirm migration head (`ls internal/db/postgres/sqlc/migrations/ | tail`), verify no `000033` exists; author `000033_add_instagram_schema.up.sql`: add `channel` CHECK (whatsapp|instagram) DEFAULT 'whatsapp' to `crm.messages` + `crm.conversations`; `ALTER TABLE crm.messages RENAME whatsapp_message_id TO provider_message_id`; replace dedup unique index with `(organization_id, channel, provider_message_id)`; add partial unique `(organization_id, contact_id, channel) WHERE status='active'` on conversations; make `crm.contacts.phone_number` nullable, replace unique constraint with partial index `WHERE phone_number IS NOT NULL`; add `instagram_user_id` + `instagram_username` columns + partial unique on `instagram_user_id`; extend `source` CHECK with 'instagram'
- [x] Author `000033_add_instagram_schema.up.sql` part 2: create `whatsapp.instagram_configs` (organization_id UNIQUE, ig_user_id UNIQUE, ig_username, fb_page_id, access_token, token_expires_at, webhook_secret, verify_token, api_version DEFAULT 'v21.0', graph_api_url DEFAULT 'https://graph.facebook.com', is_active, metadata, timestamps) and `whatsapp.instagram_webhook_logs` (ig_user_id, status, event_type, raw_headers, raw_body, error_message, processed_at, created_at, delivery_key + unique partial index (ig_user_id, delivery_key))
- [x] Author `000033_add_instagram_schema.down.sql`: drop Instagram tables, restore `whatsapp_message_id` name + original indexes, restore phone NOT NULL (after NULL scrub), revert CHECKs
- [x] Update `query/crm.sql`: rename `whatsapp_message_id` refs → `provider_message_id` in CreateMessage/InsertMessageIdempotent/GetMessageByWhatsAppID/UpdateMessageStatus; ON CONFLICT target `(organization_id, channel, provider_message_id)`; add `channel` to insert column lists; add `?channel=` filter to ListConversationsByOrganization (CASE WHEN $5::text = '' THEN TRUE ELSE c.channel = $5::text END); include `channel` + contact IG fields in SELECT
- [x] Author `query/instagram.sql`: config CRUD + GetByIGUserID + toggle + update token (upsert with partial-update semantics), webhook log insert-with-outbox + dedup lookup + replay list, backfill state queries (reuse patterns from whatsapp queries)
- [x] Run `make sqlc`; fix generated code references; run `make test` — both must pass

## 2. Instagram Domain + Events [BE-DOMAIN]

- [x] Create `internal/modules/instagram/domain/`: `config.go` (InstagramConfig, mask helpers), `webhook_log.go`, `errors.go` (instagram_not_configured, instagram_no_access_token, instagram_api_error, instagram_token_refresh_failed, unknown_ig_user, ig_user_id_conflict, config_not_found), `repository.go` interface
- [x] Create `internal/modules/instagram/domain/events/`: `message_received.go` (`instagram.message.received`; fields: OrganizationID, FromIGUserID, ToIGUserID, MessageID/mid, MessageType, Content, MediaURLs, Timestamp, Channel='instagram'), `message_echo.go`, `message_send.go`
- [x] Register outbox codecs for the three event types in `cmd/init.go` (mirror whatsapp `registerOutboxCodecs`); wire DI module + provider + routes registration in `internal/api/provider.go` (public webhooks + `/instagram` group with auth→org_context→subscription→`org:manage`)
- [x] Verify `make build` + `make test` pass after module scaffold

## 3. Instagram Graph Client Infra [BE-INFRA]

- [x] Create `infra/graphapi/ig_client.go`: interface + HTTP impl with Bearer auth — `SendTextMessage(ctx, token, baseURL, apiVersion, igUserID, recipientID, text)` → `POST /{v}/{igUserID}/messages` body `{"recipient":{"id":...},"message":{"text":...}}`; `GetIGUser(ctx, token, baseURL, apiVersion, igUserID)` → `GET /{v}/{id}?fields=username,profile_picture_url`; `RefreshToken(ctx, appID, appSecret, baseURL, apiVersion, token)` → `GET /oauth/access_token?grant_type=fb_exchange_token&...`; wrap all calls in circuit breaker (5, 10s, 2); mockable interface
- [x] Add env config `INSTAGRAM_APP_ID`, `INSTAGRAM_APP_SECRET`, `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` to graphapi/config (or new config struct); wire into DI
- [x] Unit tests with mock HTTP server: correct URL/headers/body per spec scenarios, breaker opens after 5 failures + half-open probe, token refresh request shape; `make test` passes

## 4. Instagram Config Service + API [BE-DOMAIN]

- [x] Implement `config_service.go`: GetConfig (masked response + token_expires_at + token_expiry_warning when <7d), UpsertConfig (partial update, preserve masked/empty secrets, 409 ig_user_id_conflict), ToggleConfig, RefreshToken (calls IG client RefreshToken, persists new token+expiry, 502 on failure without modifying stored token)
- [x] Implement handlers + routes: `GET/PUT /api/v1/instagram/config`, `PATCH /api/v1/instagram/config/toggle`, `POST /api/v1/instagram/config/refresh` behind org:manage; tests for masking, conflict, refresh success/failure (mock client); `make test` passes
- [x] Webhook health endpoint reuse: expose recent `instagram_webhook_logs` activity for the settings health indicator (mirror whatsapp config logs pattern)

## 5. Instagram Webhook Ingress [BE-INFRA]

- [x] Implement `webhook_service.go`: `VerifyChallenge` (env `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` OR any active config's verify_token), `ProcessWebhook` — parse `entry[].changes[].value` (accept legacy `entry[].messaging` shape too), resolve org by `value.recipient.id` → config (is_active) else 404 `unknown_ig_user`, HMAC via `pkg/whatsapp/signature.go` against resolved config webhook_secret else 401 `invalid_signature`, detect `is_echo` → echo event, build outbox events, persist log + outbox in one tx, delivery_key dedup, replay from raw payload
- [x] Handlers + routes: `GET/POST /api/v1/webhooks/instagram` public (no auth); verification test + signature-validation tests (valid/invalid HMAC, constant-time, malformed header) + org-resolution tests (known/inactive/unknown ig_user_id) + dedup + echo routing, all with mock graph client; `make test` passes

## 6. CRM Generalization [BE-DOMAIN]

- [x] Generalize `crm_service.go` `persistMessage` path: channel-aware contact upsert (WhatsApp by phone; Instagram by instagram_user_id, phone NULL, source='instagram'), conversation resolution per `(org, contact_id, channel)`, message insert with channel + provider_message_id; activity subject/type per channel; grep-audit remaining `whatsapp_message_id` refs in crm (domain/message.go JSON field rename → `provider_message_id`; crm_service log fields)
- [x] Create `InstagramMessageListener` + `InstagramEchoListener` (mirror message_listener.go/echo_listener.go) consuming `instagram.message.received`/`instagram.message.echo`; register in `crm/cmd/init.go`; enqueue async `GetIGUser` backfill (outbox event `instagram.profile_backfill` with retry/dead-letter; consumer updates instagram_username/avatar_url)
- [x] Update `message_send_handler.go` + `outbound_service.go`: subscribe `instagram.message.send`; route by conversation.channel — IG path calls ig client SendTextMessage, persists sent/failed, `outside_24h_window` warning when `last_message_at` > 24h, 429 rate limit guard; keep WhatsApp path unchanged
- [x] Conversation service: accept + validate `channel` query param on list endpoint; messages response includes `channel` + `provider_message_id`; run `make sqlc`, `make test` (listener persistence, dedup per channel, echo, send routing with mock client)

## 7. Agent Pipeline Subscription [BE-DOMAIN]

- [x] In `agent/cmd/init.go` subscribe `instagram.message.received` to the same flow pipeline; verify provider id (IG mid) flows into suggestions/dedup (`GetPendingSuggestionByWhatsAppMessage` style lookup); `make test` passes

## 8. Frontend — Unified Inbox [FE-NEXT]

- [x] Models/DTOs: rename `whatsappMessageId` → `providerMessageId`, add `channel`, `contactUsername`, `avatarUrl` in `lib/models/conversation.model.ts` + `message.model.ts` + dto mappers
- [x] Repository + query: `conversation-repository.ts` `listConversations({channel})`; `use-conversations-query.ts` accepts channel param
- [x] Inbox page: channel tabs All/WhatsApp/Instagram synced to `?channel=` query param; per-channel empty state text ("No Instagram messages yet — connect Instagram in Settings to get started")
- [x] Components: `conversation-list.tsx` + `conversation-header.tsx` render IG username/avatar + channel badge, fallback display; `message-thread.tsx` render ticks only for WhatsApp channel
- [x] Run `pnpm lint` + `pnpm build`; update/extend component tests for channel tabs and IG rendering

## 9. Frontend — Instagram Settings [FE-NEXT]

- [x] `lib/api/api/repositories/instagram-config-repository.ts` (getConfig/upsertConfig/toggleConfig/refreshToken) + DTOs + hooks (`use-instagram-config-query`, `use-upsert-instagram-config`, `use-toggle-instagram-config`, `use-refresh-instagram-token`)
- [x] `app/dashboard/settings/components/instagram-config-section.tsx`: manual form (ig_user_id, ig_username, fb_page_id, access_token masked, token_expires_at + <7d warning, webhook_secret, verify_token, api_version, graph_api_url, is_active), webhook URL `.../webhooks/instagram` + copy button, webhook health indicator, refresh token button, loading/error/empty states
- [x] `settings-content.tsx`: register `view=instagram` in SettingsView union + permission check + render section; add "Instagram" overview card (value = ig_username || "Not connected", status, links to view)
- [x] Run `pnpm lint` + `pnpm build`; settings section tests

## 10. Verification Gate [OPS-GOV]

- [x] Run `make sqlc && make test` in `go-b2b-starter/` — all unit tests pass (includes new signature-validation, org-resolution, dedup, echo, send-routing, refresh tests)
- [x] Run `pnpm lint && pnpm build` in `next_b2b_starter/` — clean; run inbox/settings e2e subset (channel filter, IG thread render, settings save) with `pnpm test`
- [x] Grep audit: no stray `whatsapp_message_id`/`whatsappMessageId` references in CRM paths outside agent/campaign copies; record results in this file
