## 1. Database: Migration and SQLC Generation

- [x] 1.1 Create migration `000010_create_whatsapp_crm_schema.up.sql` with `whatsapp` and `crm` schemas, tables `whatsapp.whatsapp_configs`, `whatsapp.webhook_logs`, `crm.contacts`, `crm.conversations`, `crm.messages` (all scoped by `organization_id`, with triggers and indexes)
- [x] 1.2 Create migration `000010_create_whatsapp_crm_schema.down.sql` dropping the new schemas
- [x] 1.3 Create `internal/db/postgres/sqlc/query/whatsapp.sql` with SQLC-annotated queries for `whatsapp_configs` (GetByPhoneNumberID, Create, Update) and `webhook_logs` (Insert)
- [x] 1.4 Create `internal/db/postgres/sqlc/query/crm.sql` with SQLC-annotated queries for contacts (UpsertByPhone, GetByID), conversations (GetActiveByContact, Create, UpdateLastMessageAt), and messages (Insert, GetByWhatsAppID)
- [x] 1.5 Run `make sqlc` to generate Go code

## 2. CRM Module: Domain Layer

- [x] 2.1 Create `internal/modules/crm/domain/contact.go` — `Contact` entity struct with validation
- [x] 2.2 Create `internal/modules/crm/domain/conversation.go` — `Conversation` entity, `ConversationStatus` type, 24h window method
- [x] 2.3 Create `internal/modules/crm/domain/message.go` — `Message` entity, `MessageDirection` and `MessageType` types
- [x] 2.4 Create `internal/modules/crm/domain/repository.go` — `ContactRepository`, `ConversationRepository`, `MessageRepository` interfaces
- [x] 2.5 Create `internal/modules/crm/domain/errors.go` — domain error sentinels

## 3. CRM Module: Infrastructure Layer

- [x] 3.1 Create `internal/modules/crm/infra/repositories/contact_repository.go` — SQLC-backed implementation with `mapToDomain` for contacts
- [x] 3.2 Create `internal/modules/crm/infra/repositories/conversation_repository.go` — SQLC-backed implementation with 24h active window logic
- [x] 3.3 Create `internal/modules/crm/infra/repositories/message_repository.go` — SQLC-backed implementation with idempotency check on `whatsapp_message_id`

## 4. WhatsApp Module: Domain Layer

- [x] 4.1 Create `internal/modules/whatsapp/domain/config.go` — `WhatsAppConfig` entity (phone_number_id, webhook_secret, verify_token, org_id)
- [x] 4.2 Create `internal/modules/whatsapp/domain/errors.go` — domain error sentinels
- [x] 4.3 Create `internal/modules/whatsapp/domain/events/message_received.go` — `MessageReceived` event struct and constructor (carries org_id, message_sid, from, to, type, content, timestamp, raw_payload)
- [x] 4.4 Create `internal/modules/whatsapp/domain/repository.go` — `ConfigRepository` and `WebhookLogRepository` interfaces

## 5. WhatsApp Module: Infrastructure Layer

- [x] 5.1 Create `internal/modules/whatsapp/infra/repositories/config_repository.go` — SQLC-backed, `GetByPhoneNumberID`
- [x] 5.2 Create `internal/modules/whatsapp/infra/repositories/webhook_log_repository.go` — SQLC-backed, `Insert`

## 6. Shared Utilities

- [x] 6.1 Create `pkg/whatsapp/signature.go` — `VerifySignature(secret string, body []byte, signatureHeader string) error` using HMAC-SHA256 constant-time comparison
- [x] 6.2 Create `pkg/whatsapp/signature_test.go` — table-driven tests for signature verification
- [x] 6.3 Create `pkg/whatsapp/phone.go` — `CanonicalizeE164(raw string) (string, error)` with Colombian `^\+573\d{9}$` validation, logging non-matching numbers
- [x] 6.4 Create `pkg/whatsapp/phone_test.go` — table-driven tests for E.164 canonicalization

## 7. WhatsApp Module: Webhook Handler and Routes

- [x] 7.1 Create `internal/modules/whatsapp/app/services/webhook_service.go` — `WebhookService` interface and implementation: verify signature, resolve org, parse payload, insert webhook log, publish `MessageReceived` event
- [x] 7.2 Create `internal/modules/whatsapp/handler.go` — Gin handler for `POST /api/v1/webhooks/whatsapp` (with Swagger annotations) and `GET /api/v1/webhooks/whatsapp` (verification challenge)
- [x] 7.3 Create `internal/modules/whatsapp/routes.go` — route registration (webhook endpoint does NOT use standard auth middleware; signature verification is inline)

## 8. CRM Module: Event Subscriber and Service

- [x] 8.1 Create `internal/modules/crm/app/services/crm_service.go` — `CRMService` interface and implementation: `ProcessInboundMessage(ctx, event) error` with canonicalize → upsert contact → resolve/create conversation → insert message flow
- [x] 8.2 Create `internal/modules/crm/app/services/message_listener.go` — `MessageListener` interface and implementation: subscribes to `whatsapp.message.received`, delegates to `CRMService`

## 9. Module Wiring: DI and Bootstrap

- [x] 9.1 Create `internal/modules/crm/module.go` — Dig registration for `CRMService`, `MessageListener`
- [x] 9.2 Create `internal/modules/crm/provider.go` — Dig registration for handler/routes (placeholder for future CRUD handlers)
- [x] 9.3 Create `internal/modules/crm/cmd/init.go` — Module `Init` function: register dependencies + subscribe to `whatsapp.message.received` via `container.Invoke`
- [x] 9.4 Create `internal/modules/whatsapp/module.go` — Dig registration for `WebhookService`
- [x] 9.5 Create `internal/modules/whatsapp/provider.go` — Dig registration for handler and routes
- [x] 9.6 Create `internal/modules/whatsapp/cmd/init.go` — Module `Init` function: register dependencies
- [x] 9.7 Register 5 new repositories in `internal/db/inject.go` (whatsapp.ConfigRepository, whatsapp.WebhookLogRepository, crm.ContactRepository, crm.ConversationRepository, crm.MessageRepository)
- [x] 9.8 Add module init calls in `internal/bootstrap/init_mods.go`: `whatsapp.Init(container)` then `crm.Init(container)` in Phase 3 (after documents, before cognitive)

## 10. Verification

- [x] 10.1 Run `make sqlc` to verify generated code compiles
- [x] 10.2 Run `make build` to verify the full binary compiles
- [x] 10.3 Run `make test` to verify all existing and new tests pass
- [x] 10.4 Run `make migrateup` to verify migration applies cleanly
- [x] 10.5 Run `make migratedown` to verify migration rolls back cleanly
