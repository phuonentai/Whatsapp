# Tasks: add-supplier-inquiry-agent

## 1. Schema: procurement tables [DB-SQLC]

- [x] 1.1 Add migration `000037_create_procurement_schema` (up: create `procurement` schema with `suppliers`, `products`, `inquiry_runs`, `inquiry_recipients`, `inquiry_responses`, `orders`; composite tenant FKs `(organization_id, ...)` referencing `crm.contacts`; status CHECK constraints; unique `(recipient_id, raw_message_id)` on responses and `(run_id, supplier_id)` on orders; unique `(organization_id, nit)` on suppliers and `(organization_id, sku)` on products; indexes `inquiry_recipients(organization_id, contact_id)`, `inquiry_recipients(run_id)`, `inquiry_runs(organization_id, status)`, `suppliers(organization_id)`, `products(organization_id)` per design D15; down: drop schema), following the `000019_create_agent_schema` style. `suppliers.contact_id` uses `ON DELETE RESTRICT` per design D2. **Note:** the design assumed `000036` was free, but `000036_create_whatsapp_templates` (sibling change) took it; the migration is `000037` (per the design's "next free numbers are 000036+"). Verify: `ls internal/db/postgres/sqlc/migrations/000037_*` shows both up and down files — PASS.
- [x] 1.2 Add SQLC queries for supplier/product CRUD and run/recipient/response lifecycle in `internal/db/postgres/sqlc/query/procurement.sql`; regenerate via `docker compose run --no-deps cli sqlc generate`. Verify: generated `gen/procurement.sql.go` exists and `go build ./...` — PASS.
- [x] 1.3 Add a migration integrity test asserting the `procurement` schema is created by `000037` and fully dropped by its down migration (mirror existing migration test conventions). Verify: `go test ./internal/db/...` — PASS (also `-tags integration` full suite PASS; harness list extended with `000034`–`000037`).

## 2. Domain entities and state machines [BE-DOMAIN]

- [x] 2.1 Implement domain entities in `internal/modules/procurement/domain/`: `Supplier`, `Product`, `InquiryRun` (status machine `draft|sending|awaiting_responses|completed|partially_answered|failed|escalated|cancelled`, `source = 'manual'`, nullable `schedule_ref`), `InquiryRecipient` (`pending|sent|answered|timed_out|failed`, `followup_count`), `InquiryResponse` (extracted JSONB, confidence, requires_human). No Stytch SDK or transport imports in domain (constitution rule). Verify: `go build ./...` — PASS.
- [x] 2.2 Implement the run lifecycle transitions (`draft → sending → awaiting_responses → completed | partially_answered`, plus `failed`, `escalated`, `cancelled` reachable per spec) and the escalation-always-allowed rule mirroring agent-governance. Verify: `go test ./internal/modules/procurement/domain/...` — PASS.
- [x] 2.3 Unit tests: valid transitions, invalid transitions rejected, kill switch cancels in-progress runs, escalation allowed under any settings including kill switch. Verify: `go test ./internal/modules/procurement/...` — PASS.

## 3. Repositories [BE-DOMAIN]

- [x] 3.1 Implement `SupplierRepository` and `ProductRepository` (org-scoped CRUD backed by SQLC; create-supplier transaction also creates/links the `crm.contacts` row with `tipo_documento = 'NIT'`, `numero_documento`, `consent_status = 'granted'`, `consented_at` set, and audits `supplier_created` + `consent_grant` with `basis = 'org_declared'` and the acting `stytch_member_id`; a duplicate `(organization_id, nit)` insert is rejected by the unique index → 400 Spanish, no duplicate contact). Verify: `go build ./...`; `go test ./internal/modules/procurement/...` — PASS.
- [x] 3.2 Implement `InquiryRunRepository`/`InquiryRecipientRepository`/`InquiryResponseRepository` with transaction-isolated state checks: recipient `pending → sent` and `pending|sent → answered` transitions guarded by conditional UPDATEs (idempotent under redelivery), response insert idempotent on `(recipient_id, raw_message_id)`. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 3.3 Repository tests: cross-tenant access rejected (composite FK), duplicate response insert is a no-op, redelivered send does not double-transition. Verify: `go test ./internal/modules/procurement/...` — PASS (integration: `TestProcurementCrossTenantContactRejected`, `TestProcurementDuplicateNitRejected`, `TestProcurementRedeliveredSendNoDoubleTransition`, `TestProcurementDuplicateResponseNoOp`, `TestProcurementDownMigrationDropsSchema`).

## 4. Inquiry drafting (metered LLM) [BE-DOMAIN]

- [x] 4.1 Implement `InquiryDraftingService`: one metered LLM call per supplier via `internal/platform/llm` (contract `{"message": "..."}`), Spanish prompt with supplier display name (business-identity allowlist, D11), product names × quantities from the run request, and availability/price/lead-time asks; prompt contains no contact PII (documents/phones never included). Verify: `go build ./...`; `go test ./internal/modules/procurement/...` — PASS.
- [x] 4.2 Credits exhausted path: run → `escalated`, zero unmetered LLM invocations (mock metered client asserting no call on exhaustion). Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 4.3 Signature-validation and mock-fallback tests for the drafting LLM boundary: valid JSON parse; malformed JSON → run `escalated` with audit; metered client records tokens (org id in LLM context); LLM 5xx/timeout → run `escalated` via mock fallback (no partial sends). Verify: `go test ./internal/modules/procurement/...` — PASS.

## 5. Fan-out via durable outbox [BE-INFRA]

- [x] 5.1 Implement the run-send orchestration: enqueue exactly one outbox event per recipient (event type `procurement.inquiry_send`), consumed by a handler that resolves the org WhatsApp config and sends via `pkg/whatsapp/client.go`; recipient `pending → sent` with provider message id; run → `awaiting_responses` when all sends succeed; retry/backoff and dead-letter come from the existing dispatcher. The handler re-validates run/recipient state transaction-isolated inside the claim (run not `cancelled`/`escalated`, recipient still `pending`, kill switch off) before invoking the client; a blocked dispatch is audited with the reason and no send occurs (D14). `SendFanOut` transitions draft→sending and enqueues all events in ONE transaction. Verify: `go build ./...`; `go test ./internal/modules/procurement/...` — PASS.
- [x] 5.2 Rate-limit pacing: a per-organization token bucket (capacity 10, refill 1/1s) shared by all dispatcher workers paces fan-out AND order confirmations at most 10 messages per 10 seconds per organization (no burst over the limit). Verify: `go test ./internal/modules/procurement/...` — pacing test with a fake clock passes; a concurrent-worker test does not exceed the burst — PASS.
- [x] 5.3 Idempotent send handler: re-dispatched event after success does not send twice (transaction-isolated recipient state check); all-sends-fail run → `failed`; `outside_24h_window` warning recorded (audit) per whatsapp-outbound-send without failing the send. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 5.4 Mock-fallback execution tests for the WhatsApp boundary: fake client success, 5xx failure → retry, circuit-open → immediate error + dead-letter after max attempts (dispatcher behavior), and signature-validation of outbox event payloads. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 5.5 Observability counters (D16): increments for draft/extraction/summary attempts, extraction escalations, sends succeeded/retried/dead-lettered per event type, order placements, kill-switch/consent blocks. Verify: `go test ./internal/modules/procurement/...` — counter assertions with a fake metrics sink — PASS.

## 6. Reply extraction [BE-INFRA]

- [x] 6.1 Implement the procurement subscriber on `whatsapp.message.received` (independent of CRM/agent listeners): resolve sender → active run recipient (run `sending`/`awaiting_responses`, recipient awaiting reply); non-recipient messages return without LLM calls; subscriber errors return to the eventbus without crashing it. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 6.2 Implement `InquiryReplyExtractionService`: exactly one metered LLM call per eligible reply, PII masked per whatsapp-compliance before the call (no PII in the prompt; inbound content only), JSON contract `{"items": [...], "resumen": ..., "requiere_humano": ...}`; persist response, recipient → `answered`. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 6.3 `requiere_humano` semantics (low confidence, negotiation mentions, price ranges, "depende") → recipient + run `escalated`, no auto-quote anywhere; credits exhausted → escalation without unmetered call. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 6.4 Idempotency + signature-validation tests: redelivered message → no second extraction or response row; malformed extraction JSON → escalation with audit; mock LLM fallback on 5xx/timeout escalates without persisting a fabricated response. Verify: `go test ./internal/modules/procurement/...` — PASS.

## 7. Aggregation and ranking [BE-DOMAIN]

- [x] 7.1 Implement the board query: per-supplier recipient status + extracted items, deterministic sort (availability desc, unit price asc, lead time asc; unpriced/unanswered sort last; ties by recipient id). Verify: `go test ./internal/modules/procurement/...` — ranking tests pass — PASS.
- [x] 7.2 Optional metered board summary: one LLM call, tokens recorded; credits exhausted → board returned without summary, no LLM call. Verify: `go test ./internal/modules/procurement/...` — PASS.
- [x] 7.3 Lazy timeout reconciliation (D12): on any board/run-status read of a run in `awaiting_responses`, transition recipients `sent` with `sent_at` older than the 24h window to `timed_out` and re-evaluate `completed`/`partially_answered`, transaction-isolated and idempotent. Verify: `go test ./internal/modules/procurement/...` — read-time reconciliation and no-repeat tests pass — PASS.

## 8. API handlers + RBAC [BE-DOMAIN]

- [x] 8.1 Add routes under `/api/procurement/...` (`/suppliers`, `/products`, `/runs`, `/runs/:id`, `/runs/:id/send`, `/runs/:id/orders`) behind `auth` + `org_context` + `subscription` middleware, `org:manage` on writes/approvals and `org:view` on reads, Spanish-first error messages, 401/403/400 mapping (402 reserved for the ai-usage credit guard). Verify: `go build ./...`; `go vet ./...` — PASS.
- [x] 8.2 Handler tests with mocked services: RBAC denials (403 Spanish), validation errors (400 Spanish), run creation triggers drafting, order placement guardrail checks. Verify: `go test ./internal/modules/procurement/...` — PASS.

## 9. Order placement + CRM integration [BE-DOMAIN]

- [x] 9.1 Implement `OrderPlacementService` (D10/D13): requires recipient `answered` with `requiere_humano = false` (or explicit `org:manage` override, audited with Stytch `member_id`); one transaction inserts the `procurement.orders` marker (UNIQUE `(run_id, supplier_id)`), enqueues `procurement.order_confirm_send` in the durable outbox, creates a negocio in the org's default pipeline (`es_predeterminado`, `estado = 'abierto'`, `moneda = 'COP'`) + an activity on the supplier contact timeline, and audits `order_placed`. Kill switch and consent `withdrawn` block the send at placement (order/negocio/actividad still recorded, block audited) and are re-checked at dispatch time (D14). Retried POSTs are idempotent (existing order returned, no second send/deal). Verify: `go build ./...`; `go test ./internal/modules/procurement/...` — PASS.
- [x] 9.2 Tests: order without answered response rejected (400 Spanish, no send/deal); override path works; consent withdrawn blocks send; kill switch blocks send at placement and at dispatch; retried POST returns the existing order without a second send or deal; deal + actividad + order + audit rows created atomically; no customer-facing quote emitted anywhere. Verify: `go test ./internal/modules/procurement/...` — PASS.

## 10. whatsapp-agent skip integration [BE-INFRA]

- [x] 10.1 Agent subscriber skip check: before analysis, the agent pipeline resolves — tenant-scoped by `(organization_id, contact_id)`/phone from the event — whether the message sender is an active inquiry-run recipient; if so it returns without analysis/suggestions and without creating a conversation flow or suggestion, while the procurement subscriber processes the message independently; webhook response never blocked. Fail-safe: on lookup errors the agent processes normally. Verify: `go build ./...`; `go test ./internal/modules/agent/... ./internal/modules/procurement/...` — PASS.
- [x] 10.2 Signature-validation and mock-fallback tests: inbound message event processed by both subscribers — agent skips, procurement extracts exactly once; redelivered event idempotent on both sides; eventbus dispatch of the skip path returns nil error. Verify: `go test ./internal/modules/agent/... ./internal/modules/procurement/...` — PASS.

## 11. Frontend [FE-NEXT]

- [x] 11.1 Suppliers page: CRUD table + create/edit dialog (Spanish copy under `lib/copy/ui.ts` + `en` mirror), org-scoped list. Verify: `pnpm lint`; `npx tsc --noEmit` — PASS.
- [x] 11.2 Products page: CRUD table + create/edit dialog, deactivate action. Verify: `pnpm lint`; `npx tsc --noEmit` — PASS.
- [x] 11.3 Inquiry-run creation wizard: select products (name × quantity) + suppliers + free-text note, submit to `POST /api/procurement/runs`, show run status (incl. `escalated`). Verify: `pnpm lint`; `npx tsc --noEmit` — PASS.
- [x] 11.4 Comparison board + approval queue: board renders ranked per-supplier comparison from `GET /api/procurement/runs/:id`; approval queue submits `POST /api/procurement/runs/:id/orders` with override option (TanStack Query, shadcn/ui, Spanish). Verify: `pnpm lint`; `npx tsc --noEmit` — PASS.
- [x] 11.5 Component tests: supplier/product forms, wizard submit payload, board ranking render, order approval flow with override. Verify: `pnpm exec vitest run components/procurement` — PASS (9/9).

## 12. Verification gate [OPS-GOV]

- [x] 12.1 Backend gate: `docker compose run --no-deps cli sqlc generate` (PASS, exit 0) && `go build ./...` (PASS) && `go vet ./...` (PASS) && `go test ./internal/modules/procurement/... ./internal/modules/agent/...` (PASS) && `go test -tags integration ./internal/db/postgres/sqlc/integration/` (PASS, full suite incl. `TestProcurement*`). Results recorded above.
- [x] 12.2 Frontend gate: `pnpm lint` (0 errors, 4 pre-existing warnings — matches baseline) && `npx tsc --noEmit` (PASS). Results recorded above.
- [x] 12.3 Component tests gate: `pnpm exec vitest run components/procurement` — PASS (9/9).
- [x] 12.4 Record results and archive decision in this file — see **Verification record** below.

## Verification record (2026-08-11, gate run)

- **Backend**: sqlc generate PASS · `go build ./...` PASS · `go vet ./...` PASS · `go test ./internal/modules/procurement/... ./internal/modules/agent/...` PASS (all packages) · `go test -tags integration ./internal/db/postgres/sqlc/integration/` PASS (postgres container, full suite).
- **Frontend**: `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `npx tsc --noEmit` PASS · `pnpm exec vitest run components/procurement` PASS (9/9).
- **Deviations from the design**: (1) migration renumbered `000036 → 000037` because `000036_create_whatsapp_templates` (sibling change) occupies the number; (2) supplier creation requires a `phone` (the CRM contact `phone_number` is `NOT NULL` and messaging targets the WhatsApp number); (3) the run's requested product ids are not persisted, so the approval dialog matches extracted item names to the org product catalog before placing an order.
- **Archive decision:** deferred — the change depends on `add-whatsapp-template-messages` for cold-outbound mitigation (design risk section) and the council verdict required design changes that were folded into the artifacts; a council re-review and `/opsx-archive` should run before declaring the change complete. Non-verification gap only.

## Verification record (2026-08-12, repo-wide active-changes run)

- [x] Full implementation verified in tree (code was implemented by a parallel session with stale checkboxes; this record reconciles reality):
  - **1.x Schema/SQLC**: migration `000037_create_procurement_schema` (up 237 lines + down) with suppliers/products/inquiry_runs/inquiry_recipients/inquiry_responses/orders/audit_log, composite tenant FKs, ON DELETE RESTRICT, unique `(org,nit)`/`(org,sku)`/`(recipient_id,raw_message_id)`/`(run_id,supplier_id)`, D15 index set, updated_at triggers. `procurement.sql` queries + regen (gen/procurement.sql.go). Migration integrity test `TestProcurementDownMigrationDropsSchema` PASS (integration, 23s).
  - **2.x Domain**: entities + run/recipient state machines + escalation-always-allowed; domain tests PASS.
  - **3.x Repos**: supplier/product (contact+NIT+consent in one tx, org-declared consent audit), run/recipient/response with transaction-isolated guards; idempotency tests PASS.
  - **4.x Drafting**: one metered LLM call per supplier, credits-exhausted → escalated with zero unmetered calls; drafting tests PASS.
  - **5.x Fan-out**: outbox `procurement.inquiry_send` per recipient, dispatch-time re-validation (kill switch/run state), token-bucket pacer (10/10s), idempotent send, mock-fallback tests, metrics counters; send handler tests (7) + pacer tests (2) + metrics tests (2) PASS.
  - **6.x Extraction**: independent `whatsapp.message.received` subscriber, one metered extraction per eligible reply, requiere_humano → escalation, idempotent on (recipient_id, raw_message_id); subscriber tests (4) PASS.
  - **7.x Board**: deterministic ranking (availability desc → price asc → lead time asc), optional metered summary, lazy 24h timeout reconciliation; board tests (3) PASS.
  - **8.x Routes**: `/api/procurement/{suppliers,products,runs,runs/:id,runs/:id/send,runs/:id/orders}` behind auth+org_context+subscription, org:manage writes / org:view reads, Spanish errors; handler tests (7) PASS. **Gap closed this session**: routes registered in `internal/api/provider.go` (ProcurementRoutes added to moduleRoutes + RegisterRoutes).
  - **9.x Orders**: one-tx placement (orders marker UNIQUE (run_id,supplier_id) + order_confirm_send outbox + negocio es_predeterminado/abierto/COP + actividad + order_placed audit), override path, consent/kill-switch blocks at placement AND dispatch, idempotent retry; order tests (5) PASS.
  - **10.x Agent skip**: agent subscriber consumes optional `ActiveInquiryChecker` (provided by procurement module); TestShouldSkipInquiry PASS; agent+procurement suite green.
  - **11.x Frontend**: procurement page (`/dashboard/procurement`), suppliers/products managers, run wizard, board + approval; wired into sidebar nav ("Proveedores", org:manage). **Gaps fixed this session**: label htmlFor/id associations (a11y + testability) in suppliers/products managers; test fixture gained display_name/phone_number; 9/9 component tests PASS.
  - **12.x Gates**: `docker compose run --no-deps cli sqlc generate` PASS (done during implementation); `go build ./...` PASS; `go vet ./...` PASS; `go test ./...` PASS (full repo); `npx tsc --noEmit` PASS; `pnpm lint` PASS (0 errors / 4 pre-existing warnings); `pnpm build` PASS; `pnpm exec vitest run components/procurement` PASS (9/9).
- [ ] **Archive decision:** all 40 tasks complete and locally verified; archive deferred to the centralized repo-wide archive phase so the deltas fold into living specs together with sibling changes (supplier-inquiries new, whatsapp-agent modified) — not a verification blocker.

## Phase 0 baseline checkpoint
 (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state.
