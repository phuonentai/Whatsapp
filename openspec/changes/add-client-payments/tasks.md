## 1. Database

- [ ] 1.1 [DB-SQLC] Add migration `000031_create_client_payments` (up/down): table `client_payments` (id, organization_id FK, deal_id FK, invoice_id FK nullable, amount_cop, commission_cop, currency default 'COP', status default 'pending' with check constraint, mp_preference_id unique nullable, mp_payment_id unique nullable, paid_at nullable, created_at). Verify: `ls internal/db/postgres/sqlc/migrations/000031_*` shows both files; SQL follows `000021_create_invoices` style
- [ ] 1.2 [DB-SQLC] Add SQLC queries: `CreateClientPayment`, `GetClientPaymentByPreferenceID`, `GetClientPaymentByPaymentID`, `UpdatePaymentStatus` (guarded: only transitions from `pending`, never mutates terminal states). Verify: `make sqlc` regenerates without errors; new query models compile in `go build ./...`

## 2. Payments domain + application layer

- [ ] 2.1 [BE-DOMAIN] Define `internal/modules/payments/domain`: `PaymentStatus` enum (pending/paid/failed/expired), `ClientPayment` struct, repository interface matching the 1.2 queries. No MP SDK or transport imports in domain. Verify: `go build ./...`
- [ ] 2.2 [BE-DOMAIN] Implement `PaymentsService.CreateLink(ctx, orgID, dealID, invoiceID, amountCOP)` — applies commission (`PAYMENTS_COMMISSION_RATE`, `commission_cop = round(amount × rate)`, `unit_price = amount + commission`), calls infra adapter, persists pending record. Verify: unit test asserts preference payload amount math incl. zero-rate exact-amount case; `go test ./internal/modules/payments/...`
- [ ] 2.3 [BE-DOMAIN] Implement `PaymentsService.HandlePaymentEvent(ctx, eventType, paymentID)` — on approval: load by `mp_payment_id` else by `mp_preference_id`, verify via adapter `VerifyPayment` (`GET /v1/payments/{id}` status mapping per `mp_adapter.go:203`), guarded transactional transition to `paid`, then WhatsApp confirmation + deal activity (both non-fatal, log warn on failure). Verify: unit tests — approved → paid + confirmation payload; verification failure → stays pending; duplicate event → single transition; untracked payment → acknowledged no-op; withdrawn contact still receives confirmation without promo content; `go test ./internal/modules/payments/...`
- [ ] 2.4 [BE-DOMAIN] Define `PaymentEventHandler` interface in payments domain (method `HandlePaymentEvent`) and export constructor for billing consumption. Verify: `go build ./...`

## 3. Infrastructure adapter

- [ ] 3.1 [BE-INFRA] Implement `internal/modules/payments/infra/mercadopago` adapter over the platform `mp.Client`: `CreateClientPaymentPreference(ctx, orgID, dealID, amountCOP, commissionCOP)` → `POST /checkout/preferences` (items unit_price = amount+commission, `external_reference = "deal:<id>"`, back_url reused from billing config); `VerifyPayment(ctx, paymentID)` reusing `GET /v1/payments/{id}` + status mapping pattern. Verify: unit tests with mocked transport — success returns init_point; non-201/200 → wrapped error; `go test ./internal/modules/payments/...`
- 3.2 [BE-INFRA] **No new webhook route** — reuse `POST /api/v1/webhooks/mercadopago` (existing `x-signature` verification). Verify: route unchanged, `go build ./...`

## 4. Invoicing seam wiring

- [ ] 4.1 [BE-DOMAIN] Extend `PaymentLinker` interface (invoicing_service.go:37) to `PaymentLink(ctx, orgID, dealID int32, amountCOP int64) (string, error)`; update `notify()` to pass invoice deal + amount and the `noopPaymentLinker` fallback. Verify: `go build ./...`; existing invoicing tests updated and green
- [ ] 4.2 [BE-INFRA] Provide real `PaymentLinker` implementation in invoicing dig module (`module.go` optional binding) backed by payments service; link creation failure logged, invoice notification sent without link (non-fatal). Verify: `go test ./internal/modules/invoicing/...` — mock asserts failure non-fatal; `go build ./...`

## 5. Billing webhook dispatch

- [ ] 5.1 [BE-INFRA] Inject `PaymentEventHandler` into billing service (optional dig binding); in `ProcessMPWebhookEvent` payment branch (process_mp_webhook_event_service.go:47) dispatch to handler instead of ignore. Subscription branches untouched. Verify: `go test ./internal/modules/billing/...` — mock handler receives payment events, subscription events unaffected; `go build ./...`
- [ ] 5.2 [BE-INFRA] Confirm DI compose root (billing module) wires payments handler without circular import (payments never imports billing). Verify: `go build ./...`; app boots (`make server` reaches ready state)

## 6. Configuration

- [ ] 6.1 [BE-INFRA] Add `PAYMENTS_COMMISSION_RATE` env var (decimal, default 0.0) to backend config with validation; document in SETUP/DEVELOPMENT env list. Verify: config tests pass; zero-rate default asserted

## 7. Verification gate

- [ ] 7.1 [OPS-GOV] Run full verification: `make sqlc && go build ./... && go vet ./... && make test`. Verify: all pass
- [ ] 7.2 [OPS-GOV] Deferred (external): live MP sandbox replay — create preference → sandbox pay → `payment_approved` to `/api/v1/webhooks/mercadopago` → row `paid` + WhatsApp confirmation; bad `x-signature` → 401. Verify: executed during deployment with sandbox credentials; recorded here when done
- [ ] 7.3 [OPS-GOV] Record archive decision: `/opsx-archive` or `**Archive deferred:** <reason>` entry. Verify: entry present in this file
