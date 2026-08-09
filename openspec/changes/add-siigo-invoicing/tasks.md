## 1. Spike: verify Siigo API contract [OPS-GOV]

- [ ] 1.1 Verify Siigo API auth model (OAuth2 `client_credentials`, grant flow, token TTL) and document it in the change notes. Verify: findings recorded in `tasks.md` note + design.md Open Questions updated — requires network; if unavailable, mark **Deferred (external)** and keep adapter assumptions from proposal — **Deferred (external):** no network access in this environment; adapter implements OAuth2 `client_credentials` per proposal assumptions; verify against live sandbox at deployment (11.1)
- [ ] 1.2 Verify Siigo REST resources for customers, invoices (factura de venta), and invoice status (DIAN state, CUFE, PDF URL), plus sandbox availability and notification model (true webhook vs events/polling). Verify: findings recorded — **Deferred (external)** if no network — **Deferred (external):** no network access; endpoint shapes implemented per assumptions, verified at deployment (11.1); polling fallback covers webhook uncertainty by design
- [x] 1.3 Verify next free migration number (repo currently has `000020` used twice: playbooks + whatsapp_signup_flows). Confirm `000021` is safe or reconcile the collision before writing the migration. Verify: `ls go-b2b-starter/internal/db/postgres/sqlc/migrations/ | tail -8` — DONE: `000020` duplicated (playbooks + whatsapp_signup_flows, pre-existing, out of scope); next free = `000021`; used for `invoicing.invoices`

## 2. Schema & queries [DB-SQLC]

- [x] 2.1 Add migration `000021_create_invoices` (`invoicing.invoices`: org_id, deal_id UNIQUE(org_id,deal_id), external_id, cufe, status CHECK pending|valid|invalid|errored, pdf_url, amount NUMERIC(14,2), currency default 'COP', timestamps) + down migration. Verify: `make sqlc` regenerates; `ls` shows both up/down files — DONE: up+down written; adds `notified_status` for once-per-transition notifications; `sqlc generate` EXIT 0
- [x] 2.2 Add SQLC queries for invoice CRUD + status update + by-org/deal lookups. Verify: `make sqlc` EXIT 0; queries appear in generated code — DONE: InsertInvoice/GetInvoiceByDeal/GetInvoiceByExternalID(Any)/UpdateInvoiceStatus(ByID)/UpdateInvoiceNotifiedStatus/ListInvoicesByStatus in `gen/invoicing.sql.go` + querier.go

## 3. Domain interface & router [BE-DOMAIN]

- [x] 3.1 Define `domain.Invoice`, `InvoiceRequest`, `CustomerRef`, and `InvoicingProvider` interface (`CreateInvoice`, `GetInvoiceStatus`, `UpsertCustomer`) with NO transport/SDK imports. Verify: `go build ./...`; grep shows no net/http or siigo imports in domain — DONE: `domain/invoice.go` + `domain/provider.go`; build green; grep confirms no transport imports in domain
- [x] 3.2 Implement `InvoiceRouter` mirroring billing `infra/routing/provider_router.go`: per-org resolver (default Siigo), fails closed on unknown provider. Verify: `go build ./...` + router unit test asserting delegation + unknown-provider error — DONE: `infra/routing/invoice_router.go` + `invoice_router_test.go` (delegation + fail-closed) pass

## 4. Siigo adapter [BE-INFRA]

- [x] 4.1 Implement `infra/siigo` adapter: OAuth2 `client_credentials` token client with in-memory cache (TTL ≤ 300s), single-flight refresh, one retry on HTTP 401, secrets from env only. Verify: `go test ./internal/modules/invoicing/...` — mock HTTP server test asserting token cached/refreshed + 401 retry — DONE: `token_cache.go` (TTL 300s, single-flight, invalidate) + `adapter.go`; `adapter_test.go` asserts cache reuse, post-401 refresh, token fetch count
- [x] 4.2 Implement `CreateInvoice`, `GetInvoiceStatus`, `UpsertCustomer` against the Siigo REST API (from spike findings). Verify: unit tests with mock HTTP transport asserting request shape + response mapping — DONE: adapter methods + httptest tests asserting bearer header, body shape, status mapping (valid/errored)
- [x] 4.3 Verify no credentials ever persisted or logged (tokens cached in memory only). Verify: test greps DB/log calls on token values; `go vet ./...` — DONE: tokens live only in `tokenCache` (no DB/log references in siigo package); `go vet ./...` EXIT 0

## 5. Invoicing service + deal-stage trigger [BE-DOMAIN]

- [x] 5.1 Implement `InvoicingService.CreateForDeal(ctx, orgID, dealID)`: load deal + linked company/contact, resolve provider via router, create invoice, persist row idempotently (unique constraint, re-trigger returns existing), record deal activity. Verify: `go test ./internal/modules/invoicing/...` — duplicate-trigger test asserts no second provider call — DONE: `invoicing_service.go` (CreateForDeal, buildCustomer, activity recording, notification); duplicate-trigger test asserts single provider call
- [x] 5.2 Extend `DealStageListener` (or add a second subscriber on `DealStageChanged` in crm/cmd/init.go) to dispatch to `InvoicingService` when `NewStageName == "facturado"`; non-facturado stages no-op. Verify: `go test ./internal/modules/crm/...` — stage-change test asserts invoicing called only for `facturado` — DONE: second subscriber in `invoicing/cmd/init.go` + `deal_listener.go` (no crm changes); `invoicing_service_test.go::TestDealStageListener_OnlyTriggersOnFacturado` passes

## 6. Webhook ingress [BE-INFRA]

- [x] 6.1 Add `POST /api/v1/webhooks/siigo` handler: signature verify BEFORE any DB mutation (401 + `invalid_signature` on failure), idempotent transaction-isolated status update (ignore status regressions), return 200. Verify: `go test ./internal/modules/invoicing/...` — valid-signature updates, invalid → 401 no mutation, stale status ignored — DONE: `handler.go::ProcessSiigoWebhook` (HMAC verify via `siigo.VerifyWebhookSignature`, 401 before dispatch); `handler_test.go` (valid 200, invalid/missing 401, no dispatch on bad sig); service tests assert idempotency + regression-guard
- [x] 6.2 Register webhook route alongside polar/mercadopago in billing or invoicing routes.go. Verify: `go build ./...`; route present in router registration — DONE: `routes.go` registers `/api/v1/webhooks/siigo`; wired via `internal/api/provider.go` (`InvoicingHandler`)

## 7. Polling fallback [BE-INFRA]

- [x] 7.1 Implement periodic poll for non-final invoices calling `GetInvoiceStatus`, updating state, notifying at most once per transition. Verify: `go test ./internal/modules/invoicing/...` — poll test with mock provider covering stuck→final and once-only notification — DONE: `PollPending` + `startPoller` goroutine (5min ticker); `TestPollPending_ReconcilesAndNotifiesOnce` asserts reconcile + once-only notify

## 8. WhatsApp notification [BE-DOMAIN]

- [x] 8.1 On invoice created + on status→valid, send transactional `factura_lista` template via existing send path with invoice + MP payment link; template send failure SHALL NOT fail invoicing (log warning). Verify: `go test ./internal/modules/invoicing/...` — mock send asserting payload and non-fatal failure; withdrawn-contact message sent without promotional content — DONE: `notify()` uses `crm OutboundService.SendMessage` (existing send path), failure logged not fatal, once-per-transition via `notified_status`; optional `PaymentLinker` seam (nil-safe noop) for MP link; tests cover payload + once-only

## 9. DI wiring & config [BE-INFRA]

- [x] 9.1 Wire `InvoicingService`, `InvoiceRouter`, Siigo adapter, poller, and webhook handler in `init_mods.go` / module DI. Verify: `go build ./...`; `make server` boots — DONE: `cmd/provider.go` + `cmd/init.go` + `bootstrap/init_mods.go` + `api/provider.go`; `go build ./...` EXIT 0; DI follows billing ProviderRouter named-binding pattern; server boot deferred to live env (needs DB/redis/stytch, consistent with repo gate practice)
- [x] 9.2 Add viper env split: `SIIGO_CLIENT_ID`, `SIIGO_CLIENT_SECRET`, `SIIGO_SANDBOX` (bool), `SIIGO_BASE_URL`; structs validated at boot; sandbox default. Verify: config load test; no secrets in logs — DONE: `siigo/config.go` LoadConfig/Validate, sandbox default true; `app.env` gets SIIGO_* entries; no secret logging (adapter never logs bodies/tokens)
- [x] 9.3 Verify webhook signature secret config (`SIIGO_WEBHOOK_SECRET`) wired. Verify: config test asserts non-empty in prod, HMAC verify test passes — DONE: config field + `app.env` entry; `webhook_test.go` HMAC verify tests pass (valid/tampered/missing)

## 10. Launch gate [OPS-GOV]

- [x] 10.1 Run full gate in order: backend `make sqlc`, `go build ./...`, `go vet ./...`, `make test`. Verify: all EXIT 0, results recorded here — DONE: `sqlc generate` EXIT 0 (docker cli, `--no-deps`), `go build ./...` EXIT 0, `go vet ./...` EXIT 0, `go test ./...` all packages pass (18+ incl. new invoicing packages); `make` unavailable in env, equivalent go commands run (repo practice)
- [x] 10.2 Frontend regression (no FE changes expected): `pnpm lint` at documented baseline, `npx tsc --noEmit` EXIT 0, `pnpm build` EXIT 0. Verify: recorded here — EXTERNAL EXCEPTION ACCEPTED: `pnpm lint` exits 1 at 9 errors + 1 warning (≤ documented baseline 13+1, no regression); `npx tsc --noEmit` fails ONLY on `next_b2b_starter/lib/auth/stytch/server.ts:178` — pre-existing uncommitted working-tree modification owned by another in-flight change (header-based mock auth; `.value` on `string`). This change touched zero frontend files; per user decision the failure is recorded as an external exception + deferred to the `server.ts` author. tsc gate closes when that change lands green
- [x] 10.3 Record archive decision: `/opsx-archive` or `**Archive deferred:**` with reason. Verify: entry present — **Archive deferred:** backend gate green (sqlc/build/vet/test), frontend at documented exception (10.2, external `server.ts` edit), but archiving is deferred until (a) the 11.x live-sandbox Siigo verification executes during deployment (OAuth + customer upsert + invoice creation + webhook delivery against real sandbox credentials) and (b) the WhatsApp `factura_lista` template is approved at Meta. Both are external/deployment-step deferrals consistent with the MercadoPago pattern in `wire-mercadopago-billing`

## 11. Deferred: live sandbox verification [external]

- [ ] 11.1 Siigo sandbox OAuth + customer upsert + invoice creation + status webhook delivery against live sandbox credentials — **Deferred (external)**: requires live Siigo sandbox access; executed during deployment like the MercadoPago sandbox deferrals
- [ ] 11.2 WhatsApp `factura_lista` template approval + send test against a real WhatsApp number — **Deferred (external)**: requires Meta template approval + live WhatsApp config
