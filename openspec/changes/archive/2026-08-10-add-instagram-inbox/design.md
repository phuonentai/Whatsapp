## Context

The platform ships a WhatsApp-only messaging pipeline: Meta Cloud API webhook ingress (`POST /api/v1/webhooks/whatsapp`) verifies `x-hub-signature-256`, resolves the org by `phone_number_id`, and writes webhook log + outbox events in one transaction; the outbox dispatcher fans `whatsapp.message.received` out to the CRM module (contact/conversation/message persistence), the agent module (AI suggestions), and send handlers route outbound sends through a circuit-breakered Graph API client. The inbox UI at `/dashboard/inbox` polls `GET /crm/conversaciones` + `GET /crm/conversaciones/:id/mensajes` every 5s.

The CRM schema is WhatsApp-couplet: `crm.messages.whatsapp_message_id` (dedup key), `crm.contacts.phone_number NOT NULL UNIQUE`, no channel discriminator on conversations/messages. `whatsapp.whatsapp_configs` is one-row-per-org (`organization_id UNIQUE`), so a second Meta channel cannot share it. Migration head is `000030`.

Target: Instagram DMs (Instagram Graph API product `messaging`) for IG-first small commerce, in the same inbox.

## Goals / Non-Goals

**Goals:**
- Unified inbox: WhatsApp + Instagram conversations in one queue, filterable by channel
- Full pipeline parity: webhook ingress (HMAC, outbox, dedup, replay, echo), config API with masking, outbound send, circuit breakers
- Generalize the CRM data model so a third channel (e.g., Messenger) is additive
- IG-first onboarding: manual config form + token expiry awareness (no scheduler infra)

**Non-Goals:**
- Instagram Embedded Signup (Meta SDK popup flow) — manual config v1, embedded signup later reusing the `whatsapp.signup_flows` seam
- Automatic token refresh — no background scheduler; manual `POST /config/refresh` + <7-day expiry warning
- Media re-hosting — attachment URLs stored as received; Meta URLs expire (follow-up: download + re-host)
- Conversation backfill via `GET /{ig_user_id}/conversations`
- IG campaigns/broadcasts, Messenger channel
- Renaming agent/campaign tables (`agent_suggestions.whatsapp_message_id`, `campaign_segments.whatsapp_message_id`) — they store provider ids as plain copies with no FK to `crm.messages`; renaming adds churn without correctness value

## Decisions

### D1. Generalize CRM tables in-place (channel column + provider_message_id rename)

Add `channel` to `crm.messages` and `crm.conversations`; rename `whatsapp_message_id` → `provider_message_id`; unique dedup index becomes `(organization_id, channel, provider_message_id)`; contacts gain nullable `phone_number` (partial unique), `instagram_user_id` (partial unique), `instagram_username`; `source` CHECK gains `instagram`.

- **Why over parallel IG tables**: the inbox must show both channels in one queue; parallel tables would force channel routing logic and duplicated query/join code forever. `organization_id UNIQUE` on `whatsapp_configs` forces a separate `instagram_configs` table regardless — so only the CRM entity layer is generalized.
- **Alternative considered**: parallel `ig_conversations`/`ig_messages` tables — rejected (unified inbox impossible without a union view; duplicated services).
- **Ripple (verified)**: rename touches `query/crm.sql` (CreateMessage, InsertMessageIdempotent, GetMessageByWhatsAppID, UpdateMessageStatus), `domain/message.go` JSON field, `crm_service.go` log field, frontend `message.model.ts`. Agent/campaign tables keep their own `whatsapp_message_id` columns untouched.

### D2. New `whatsapp.instagram_configs` — one row per org, `ig_user_id` UNIQUE

Mirrors `whatsapp_configs` + `token_expires_at`. `ig_user_id` (the business IG user id, i.e., webhook `recipient.id`) is the org-resolution key, so it must be globally unique. `ig_username`, `fb_page_id` are display/auxiliary. Secrets stay in the same masked-on-read pattern.

### D3. IG webhook org resolution by `recipient.id`, not phone_number_id

IG DMs are app-level webhook subscriptions: the payload has no `phone_number_id`-style metadata; each `value` carries `sender.id` and `recipient.id`. We resolve org by `value.recipient.id` → `instagram_configs.ig_user_id WHERE is_active = true`.

- **Why**: with one `ig_user_id` per org, `recipient.id` is a stable business-owned identifier, unlike the customer `sender.id` (IG-scoped, per-user).
- **Alternative considered**: verify-token-based resolution — rejected: Meta uses one verify token per app webhook; per-org tokens would be indistinguishable at handshake time. Platform-level `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` env var + per-config `verify_token` as secondary accept, mirroring the embedded-signup env precedent.
- **Consequence**: signature verification must look up the config (org) before computing HMAC — same order as WhatsApp (resolve → verify).

### D4. New `whatsapp.instagram_webhook_logs` table (parallel, not shared)

Same shape as `whatsapp.webhook_logs` with `ig_user_id` replacing `phone_number_id`, including `delivery_key` dedup and the raw-payload replay path.

- **Why not generalize `webhook_logs`**: `phone_number_id` + its unique dedup index are baked into whatsapp repo/SQLC queries and the replay handler; renaming buys nothing since the two channels never share logs. Parallel table keeps the WhatsApp surface untouched and the replay logic symmetric (mirrored service method).

### D5. Channel-routed outbound send

`POST /crm/conversaciones/:id/mensajes` reads the conversation's `channel` and dispatches to either the WhatsApp Cloud API client (unchanged) or the new IG client (`POST /{ig_user_id}/messages`, body `{recipient: {id}, message: {text}}`). New event type `instagram.message.send` mirrors `whatsapp.message.send`; the CRM send handler subscribes to both and picks the client by channel. Reuses `pkg/whatsapp/signature.go` (HMAC) and the `pkg/whatsapp` circuit breaker with a mockable `graphapi`-style interface for IG.

### D6. IG contact identity: `instagram_user_id` key + async profile backfill

IG webhooks carry only scoped ids — no username, no avatar. On first contact the CRM listener creates the contact with `phone_number = NULL` and enqueues an async `GetIGUser` (`fields=username,profile_picture_url`) via the outbox; transient failures retry with backoff, permanent failures dead-letter with `instagram_username` NULL. Display falls back to the scoped id.

### D7. Echo handling mirrors WhatsApp

`message.is_echo = true` (e.g., sent from Meta Business Suite) → `instagram.message.echo` → persisted as `direction = 'outbound'`, idempotent on the same key. Never published as inbound.

### D8. Frontend: channel tabs + channel-aware rendering

Inbox keeps one route; a `?channel=all|whatsapp|instagram` query param drives the repository call (`listConversations({channel})`). IG items show username + avatar + channel badge; delivery ticks render only for WhatsApp. Settings: new `instagram-config-section.tsx` cloned from `whatsapp-config-section.tsx` minus the Meta SDK login (manual form), plus token expiry warning + refresh button; overview gains an Instagram card.

## Risks / Trade-offs

- **IG webhook payload shape drift (legacy `entry[].messaging` vs `changes[].value`)**: Meta has migrated IG DMs to the changes shape; if a legacy shape arrives the parser must handle both → Parser accepts both shapes; delta spec notes the assumption; failure logs to `instagram_webhook_logs` for operator review
- **Token expiry (~60 days)**: expired token kills ingress+outbound silently → Expiry surfaced in config GET; <7-day warning in UI; manual refresh endpoint; documented reconnection path
- **Per-org token means per-org backfill calls**: username backfill needs the org's token at dispatch time; if token rotated mid-flight, backfill retries then dead-letters with NULL username → Acceptable degradation (display falls back to scoped id)
- **`provider_message_id` rename risk**: SQLC regen + frontend model rename could miss a reference → `make sqlc`, `make test`, `pnpm build` in verification; grep audit for `whatsapp_message_id` in CRM paths in tasks
- **IG 24h messaging window**: Meta enforces; out-of-window sends may reject → Mirror WhatsApp's soft `outside_24h_window` warning; hard enforcement is Meta-side
- **Media URL expiry**: IG attachment URLs expire quickly; storing as-received yields broken previews later → Non-goal v1; documented follow-up (download + re-host, mirrors WhatsApp media handling evolution)
- **Migration collision**: in-flight changes (`add-whatsapp-campaigns` owns `000029`) → Migration takes next free number (`000033`); verify head before authoring
- **App-level webhook = shared verify token**: orgs share the platform token; a misconfigured org could not be fingerprinted at handshake → Handshake accepts env token OR any active config's verify_token; org-specific correctness enforced later at message level

## Migration Plan

1. Author `000033_add_instagram_schema.up.sql`/`.down.sql` (generalization + two new tables)
2. Deploy order: migration → backend (module + CRM changes) → frontend. Old backend code after migration? WhatsApp paths are additive-safe: renames are internal to SQLC-generated code shipped in the same release; keep migration + code in one deploy
3. **Rollback**: Git revert of the change + `000033.down.sql` (restores `whatsapp_message_id`, re-adds NOT NULL phone constraint after data scrub, drops Instagram tables). Stytch tenant policy state unaffected — no auth/RBAC changes, so no Stytch-side rollback
4. Post-deploy: verify webhook handshake in Meta dashboard, send test DM, check `instagram_webhook_logs` + inbox rendering

## Open Questions

- Confirm Meta delivers IG DM webhooks in the `changes[].value` shape for the client's app configuration (assumption until first live delivery)
- Whether the org's IG token (obtained via Facebook Login with `instagram_manage_messages`) is exchangeable via `fb_exchange_token` — if not, refresh returns an explicit error and the operator reconnects manually
- Decide follow-up ownership for media re-hosting and IG embedded signup (both deliberately deferred)
