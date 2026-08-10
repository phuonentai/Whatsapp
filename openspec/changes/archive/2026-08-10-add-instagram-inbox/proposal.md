## Why

Small-commerce sellers in LatAm are often Instagram-first: discovery and customer chat happen in IG DMs before any website or WhatsApp contact. Today the platform supports WhatsApp messaging only, forcing IG-first merchants to run a second inbox (Meta Business Suite) or lose sales conversations. Adding Instagram DM support through the Meta Instagram Messaging API (Instagram Graph API product `messaging`) gives the unified inbox parity with WhatsApp using the same proven pipeline: webhook ingress → outbox → CRM persistence → inbox UI.

## What Changes

- **Migration `000033`** (repo head is `000030`): generalizes CRM to multi-channel and adds Instagram tables.
  - `crm.messages`: add `channel` (CHECK `whatsapp|instagram`, default `whatsapp`); **BREAKING** rename `whatsapp_message_id` → `provider_message_id`; replace dedup unique index with `(organization_id, channel, provider_message_id)`
  - `crm.conversations`: add `channel` (same CHECK); unique `(organization_id, contact_id, channel)`
  - `crm.contacts`: make `phone_number` nullable (partial unique index `WHERE phone_number IS NOT NULL`); add `instagram_user_id` + `instagram_username` (partial unique on `instagram_user_id`); extend `source` CHECK with `instagram`
  - New `whatsapp.instagram_configs`: `organization_id` UNIQUE, `ig_user_id` UNIQUE (webhook org-resolution key), `ig_username`, `fb_page_id`, `access_token`, `token_expires_at`, `webhook_secret`, `verify_token`, `api_version`, `graph_api_url`, `is_active`, `metadata`, timestamps
  - New `whatsapp.instagram_webhook_logs`: mirrors `whatsapp.webhook_logs` with `ig_user_id` in place of `phone_number_id`, including `delivery_key` dedup index
  - No FK/type changes to `agent.agent_suggestions` or `campaign_segments` — they store the provider message id as a copy (verified: no FK on `whatsapp_message_id`), only new write paths supply the IG mid
- **New Go module** `internal/modules/instagram/` mirroring the whatsapp module: config service (GET/PUT/PATCH + token refresh), webhook service (HMAC + org resolution by `recipient.id` → `ig_user_id`, echo via `is_echo`, outbox + log in one transaction, delivery dedup, replay), Graph API client behind circuit breaker (threshold 5, timeout 10s, half-open probe 2), outbox codec registration
- **New API surface**: public `GET/POST /api/v1/webhooks/instagram` (hub challenge + ingress); authed `/api/v1/instagram/config` GET/PUT/PATCH/toggle and `POST /api/v1/instagram/config/refresh` behind `auth` → `org_context` → `subscription` + `org:manage`
- **CRM generalization**: conversation/message list endpoints accept `?channel=`; `POST /crm/conversaciones/:id/mensajes` routes by conversation `channel` to WhatsApp or Instagram client; `persistMessage` becomes channel-aware (IG contacts keyed by `instagram_user_id`, phone NULL); new Instagram message/echo listeners; async username+avatar backfill via `GetIGUser` with retry
- **Agent**: subscribe `instagram.message.received` so AI suggestions work on IG threads (channel-agnostic service)
- **Frontend**: unified inbox with channel tabs (All / WhatsApp / Instagram) on `/dashboard/inbox`; IG contact shows username + avatar + channel icon; WhatsApp delivery ticks only on WhatsApp threads; new Instagram settings section (manual form: ig_user_id, ig_username, fb_page_id, access_token, token_expires_at with <7-day expiry warning, webhook_secret, verify_token, api_version, graph_api_url, is_active, webhook callback URL, health indicator, refresh button); settings overview gains an "Instagram" card
- **Config**: new env vars `INSTAGRAM_APP_ID`, `INSTAGRAM_APP_SECRET` (token refresh only), `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` (platform-level hub verify — IG app webhooks are app-scoped, not per-org)

## Capabilities

### New Capabilities

- `instagram-messaging`: Instagram DM integration — config API with masking/token expiry, webhook ingress with org resolution by `ig_user_id`, outbound send via Instagram Graph API, provider resilience, settings frontend (config section + overview card)

### Modified Capabilities

- `crm-conversation-api`: conversation/message list endpoints gain `channel` filter; outbound send routes by conversation channel
- `crm-core-data`: message/conversation/contact schema generalized to multi-channel (`channel`, `provider_message_id` rename, nullable phone, IG contact fields)
- `inbox-ui`: unified inbox with channel tabs and channel-specific contact rendering
- `whatsapp-config-frontend`: settings overview adds the Instagram card alongside the WhatsApp "Messaging" card
- `whatsapp-agent`: agent suggestion pipeline also consumes `instagram.message.received`

## Impact

- **Go backend**: migration `000033` (+down); SQLC regen (`query/crm.sql`, `query/instagram.sql` new); new `internal/modules/instagram/` (domain, app, infra, cmd, routes, handler); CRM service/query channel-awareness; agent subscription; DI wiring; env vars
- **Database**: three generalized CRM columns/indexes, two new `whatsapp.` tables; org-scoped via Stytch `organization_id` FK pattern; **no credential tables** — IG access token stored in `instagram_configs.access_token` (masked on read), same precedent as WhatsApp; Stytch remains sole identity/RBAC authority (no auth flow changes, `org:manage` gate reused)
- **Frontend**: `app/dashboard/inbox/` components, `lib/models/conversation.model.ts` + `message.model.ts` (rename `whatsappMessageId` → `providerMessageId`), `conversation-repository.ts` channel param, settings `instagram-config-section.tsx` + repo/hooks
- **Dependencies**: none new (reuses outbox, eventbus, circuit breaker, HMAC `pkg/whatsapp/signature.go`, GRPC none)
- **Ops**: `make migrateup`; unit tests in `make test`; e2e via existing frontend test infra
- **Rollback**: Git — revert the change (migration, module, routes, DI, FE). DB — `000033.down.sql` restores column name/indexes and drops Instagram tables. Stytch tenant policy state unaffected (no auth/RBAC changes), so no Stytch-side rollback required
- **Non-Goals**: no local credential storage beyond the existing Meta platform-token precedent (`instagram_configs.access_token`, masked on read; Stytch remains the sole identity authority); no Instagram Embedded Signup (manual config form only in v1); no automatic token refresh (manual refresh endpoint + expiry warning; no scheduler infra — same precedent as the deferred WhatsApp coexistence guardrail); no media re-hosting (attachment URLs stored as received; Meta URLs expire — recorded risk, follow-up); no conversation backfill via `GET /{ig_user_id}/conversations`; no IG campaigns/broadcasts

## Assumptions

- **Meta app pre-configuration (external, manual)**: a Meta Business app with the Instagram product + `instagram_manage_messages` and `instagram_basic` permissions, and an app-level webhook subscription for field `messages` pointing at `https://<domain>/api/v1/webhooks/instagram`, must exist. Platform-level `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` must match the token entered in Meta's webhook settings. Not verifiable from this repo.
- **IG webhook payload shape**: DM webhooks arrive as `entry[].changes[]` with `field = "messages"`; `value` contains `sender.id` (IG-scoped ID), `recipient.id` (business IG user ID), `timestamp`, `messages[]` with `mid`, `message.is_echo`, `text`, and `attachments[]`. Echo messages carry `is_echo = true`. If Meta delivers the legacy `entry[].messaging` shape instead, the parser needs adjustment — unverifiable until first live delivery.
- **Token refresh viability**: `GET /oauth/access_token?grant_type=fb_exchange_token&client_id=...&client_secret=...&fb_exchange_token=<current>` returns a new long-lived token (~60 days) for Instagram Graph API tokens obtained via Facebook Login with the above permissions. If the token was not obtained through Facebook Login, refresh may fail and the operator reconnects manually.
- **IG 24-hour window**: Meta enforces a 24h business-messaging window; out-of-window sends may be rejected or fail silently. We mirror WhatsApp's soft warning (`outside_24h_window`); hard enforcement is Meta-side.
- **Rate limits**: Instagram Messaging API rate limits (~200 messages/hour per IG user in the business) are Meta-enforced; the existing 10-messages/10-seconds endpoint guard remains the platform's own limit.
