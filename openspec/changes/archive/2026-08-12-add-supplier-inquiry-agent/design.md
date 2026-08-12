# Design: add-supplier-inquiry-agent

## Context

Colombian SMBs sell without holding inventory; the daily ritual of messaging suppliers over WhatsApp to check availability, price, and lead time is manual, one chat at a time. The platform already ships the ingredients: WhatsApp outbound/inbound (Cloud API), the metered AI agent pipeline, a durable message outbox, and the CRM. This change adds the procurement capability as a new `procurement` module, keeping the existing building blocks untouched except for one additive skip path in the whatsapp-agent inbound trigger.

Verified facts (premise validation, 2026-08-11):
- Migrations head: `000035_add_campaign_message` — next free numbers are `000036+`.
- `pkg/whatsapp/client.go` exists (circuit-breakered client); `whatsapp-outbound-send` defines the 10 msgs/10s rate limit and the `outside_24h_window` warning (template support not yet implemented).
- Durable outbox (`internal/platform/outbox` + `durable-message-pipeline` spec): outbox table, dispatcher, retry/backoff, dead-letter, `FOR UPDATE SKIP LOCKED`, idempotent handlers.
- `agent.conversation_flows` is the existing flow-row pattern (`running`/`awaiting_human`/`succeeded`/`failed`/`cancelled`); the agent pipeline subscribes to the same `whatsapp.message.received` events as the CRM listeners, resolves contact/conversation idempotently, and never blocks the webhook HTTP response.
- `crm.contacts` supports `tipo_documento` (CHECK: CC/NIT/CE/TI/PP), `numero_documento`, and the Ley 1581 consent state machine (`none`/`requested`/`granted`/`withdrawn`) — a supplier fits the existing contact model, including NIT.
- `pipeline-management` auto-seeds a default "Pipeline de Ventas" (`es_predeterminado`); `deal-management` stores negocios with `estado = 'abierto'` and `moneda = 'COP'`.
- Middleware `auth`, `org_context`, `subscription` are registered; Stytch B2B provides roles `org:manage`/`org:view`; the Dual SSOT constitution forbids local credential storage and requires idempotent, transaction-isolated DB operations for webhook-triggered work.
- FE is `next_b2b_starter/` (Next.js 16 app router, TanStack Query, shadcn/ui, Spanish-first copy).
- Clean Architecture enforcement (Dual SSOT constitution): the `procurement` Go domain SHALL NOT import Stytch SDKs or external transport packages; infrastructure adapters SHALL implement domain interface abstractions (e.g., an `InquirySender` domain port implemented by the `pkg/whatsapp` adapter, a metered-LLM domain port implemented by the `internal/platform/llm` adapter); DB operations triggered by webhooks SHALL be idempotent using transaction-isolated state checks.

## Goals / Non-Goals

**Goals:**
- Suppliers and products registered org-scoped, with a supplier being a CRM contact (NIT via `tipo_documento`).
- Inquiry runs: one metered LLM draft per supplier, durable fan-out via the outbox, structured reply extraction, deterministic aggregation board, human-approved order sends.
- Compliance parity: Ley 1581 consent at supplier creation, PII masking before LLM calls, `org:manage`/`org:view` RBAC, Spanish-first errors, metered AI everywhere.
- No double-processing of supplier replies: the agent pipeline skips analysis for messages consumed by an active inquiry run.

**Non-Goals:**
- No local credential storage; no customer-facing auto-quoting; no Siigo purchase orders; no template messages (separate change); no WhatsApp-native intent routing; no scheduled runs or follow-up automation (separate change); no agent-pipeline internals changes beyond the documented skip path.

## Decisions

### D1: A supplier IS a CRM contact (contact + role context), not a separate entity

`procurement.suppliers` stores `organization_id`, `contact_id` (FK `crm.contacts`), `nit`, `delivery_days`, `min_order_amount`, `notes`, `is_active`. Rationale: contacts already carry NIT (`tipo_documento`/`numero_documento`), the consent state machine, compliance export/forget, conversation history, and the outbound message link — a separate supplier entity would duplicate identity, PII, and compliance handling. Alternative (standalone `procurement.suppliers` with duplicated profile fields) rejected: splits the identity, doubles consent/compliance surface, and breaks message→supplier linkage. Consent at creation is an org-declared grant — basis, audit fields, and the NIT persona-jurídica rationale are in D11.

### D2: `suppliers.contact_id` uses `ON DELETE RESTRICT`

Procurement history (inquiry responses, orders) references suppliers; a cascade would silently destroy inquiry/order audit history. Deletion is deliberately frictionful: deactivate via `is_active = false` instead. Alternative (CASCADE) rejected for audit integrity.

### D3: One metered LLM call per supplier for drafting

Each run drafts one personalized Spanish message per supplier (greeting with supplier name, products with quantities, availability/price/lead-time ask), contract `{"message": "..."}`. The greeting name is the supplier display name, admitted through the `whatsapp-compliance` masking decorator via a business-identity allowlist limited to `procurement.suppliers` contacts with `tipo_documento = 'NIT'` (persona jurídica — corporate identity, not natural-person PII; documents and phone numbers are always masked); non-NIT supplier contacts fall back to the `[NOMBRE]` placeholder (D11). Rationale: personalization needs the supplier name and the per-supplier product list; per-supplier calls are parallelizable and bounded by run size. Alternatives: one call per run producing all messages (brittle JSON, weaker personalization) rejected; zero-call templates rejected (no template infra in this change). Credits exhausted → run `escalated`, no unmetered call (mirrors the ai-usage-metering escalation convention).

### D4: One metered LLM extraction call per replied message

Inbound replies from active recipients run exactly one extraction call (PII masked per `whatsapp-compliance` before the call) with a single JSON contract (`items[]`, `resumen`, `requiere_humano`). Rationale: cheapest, simplest, testable; `requiere_humano` (low confidence, negotiation mentions, price ranges, "depende") routes to escalation with no auto-quote. Alternative (multi-step classification then extraction) rejected: doubles metered cost and latency for no correctness gain.

### D5: Deterministic ranking + optional metered summary on the board

The board sort is deterministic: availability descending, unit price ascending, lead time ascending — stable, unit-testable, identical across refreshes. The natural-language summary is a separate optional metered call, skipped when credits are exhausted. Alternative (pure-LLM ranking) rejected: nondeterministic, credits per view, opaque ordering.

### D6: Durable-outbox fan-out instead of synchronous sends

Run send enqueues one outbox event per supplier; the dispatcher drives sends through the circuit-breakered client with retry/backoff and dead-letter, and fan-out paces at 10 msgs/10s via a per-organization token bucket (capacity 10, refill 1/1s) shared by all dispatcher workers (D16). Send handlers re-validate run/recipient state transaction-isolated inside the dispatch claim before invoking the client, so already-enqueued events never fire after the kill switch flips (D14). Rationale: durability across crashes, retry semantics, and rate-limit pacing for free, matching the existing dispatcher contract. Alternative (synchronous sends in the request path) rejected: couples run creation to send latency, loses retry durability, risks exceeding the burst limit.

### D7: Procurement is a new independent subscriber; the agent pipeline gets a skip check

Procurement subscribes to `whatsapp.message.received` exactly like the CRM listener. To prevent double-processing, the agent subscriber skips analysis for messages whose sender is an active inquiry-run recipient. Rationale: no coupling between the two pipelines, no shared state machine, webhook response never blocked, both handlers idempotent. Alternative (threading procurement through the agent pipeline) rejected: entangles unrelated flows and violates the independent-subscriber pattern the CRM listener already uses.

### D8: Run lifecycle as a flow row, governance mirrors agent-governance

`procurement.inquiry_runs` is the flow row (statuses `draft`/`sending`/`awaiting_responses`/`completed`/`partially_answered`/`failed`/`escalated`/`cancelled`, `source = 'manual'`, nullable `schedule_ref` for the future scheduling change). Escalation is always allowed; the kill switch cancels in-progress runs and stops pending sends. Rationale: same governance invariants as `agent.conversation_flows`, with `escalated`/`cancelled` reachable from any in-progress state.

### D9: RBAC and middleware unchanged

All `/api/procurement/...` routes sit behind `auth` + `org_context` + `subscription`; `org:manage` for writes/approvals (suppliers, products, run create/send, order placement), `org:view` for reads. No Stytch policy changes; Spanish-first error messages; supplier export/forget reuse the existing agent compliance endpoints.

### D10: Order placement creates a negocio + actividad and sends plain text

`POST /api/procurement/runs/:id/orders` requires an `answered` response with `requiere_humano = false` (or an explicit `org:manage` override). Placement is one transaction: insert the `procurement.orders` marker (UNIQUE `(run_id, supplier_id)`), enqueue the pre-composed Spanish order-confirmation text as a durable outbox event (`procurement.order_confirm_send`), create a negocio in the default pipeline (`estado = 'abierto'`, `moneda = 'COP'`) plus an activity on the supplier contact timeline, and audit `order_placed` (D13). Because the send is enqueued, not executed, a send failure never orphans a deal and a deal failure never sends a confirmation. Kill switch and consent `withdrawn` block at placement time and are re-checked at dispatch time (D14); a blocked send is audited with the reason while the order/negocio/actividad remain recorded. Rationale: templates are a separate change; the plain-text confirmation keeps this change self-contained while the negocio/actividad keep procurement on the CRM timeline.

### D11: Compliance boundaries — org-declared consent and business-identity allowlist

Suppliers are NIT legal entities (`persona jurídica`); their commercial identity (trade name, NIT) is not personal data under Ley 1581, which protects natural persons. Two audited consequences. (1) Consent at supplier creation is an **org-declared grant**: the organization, as data controller, asserts consent; the platform records `consent_status = 'granted'` with `consented_at` set and audits `consent_grant` with `basis = 'org_declared'` and the acting `stytch_member_id` — a documented departure from the subject-initiated `none → requested → granted` pipeline, valid only for NIT contacts; non-NIT (natural-person) contacts keep the full `whatsapp-compliance` pipeline. (2) The drafting prompt may contain the supplier display name via a **business-identity allowlist** on the masking decorator, scoped to `procurement.suppliers` NIT contacts; documents and phone numbers are always masked, and non-NIT contacts use `[NOMBRE]`. Rationale: without the allowlist, greetings would carry the `[NOMBRE]` placeholder and break D3; without the org-declared basis, the `granted` insert would silently contradict the consent state machine's intent. Alternative (treat supplier data as natural-person personal data) rejected: misapplies Ley 1581 to legal entities and breaks every greeting.

### D12: Lazy timeout terminates unanswered runs

Recipients transition `sent → timed_out` lazily on read: whenever the board (`GET /api/procurement/runs/:id`) or a run-status query observes a run in `awaiting_responses` with a recipient `sent` whose `sent_at` is older than the response window (fixed 24h default), the read path transitions that recipient to `timed_out` and re-evaluates the run to `completed` (all answered) or `partially_answered` (some answered, rest `timed_out`/`failed`) — idempotent, transaction-isolated, no scheduler. Rationale: the run lifecycle needs a terminating mechanism and scheduling/follow-up automation is explicitly out of scope, so a deterministic read-time reconciliation is the cheapest correct mechanism. Alternative (scheduled job) rejected: requires new infra in a change that defers scheduling.

### D13: Order placement atomicity and idempotency

One transaction inserts the `procurement.orders` marker (UNIQUE `(run_id, supplier_id)`), enqueues `procurement.order_confirm_send` in the durable outbox, creates the negocio + actividad, and audits `order_placed`. A retried POST hits the UNIQUE marker and returns the existing order — no second send, no duplicate deal. Rationale: the confirmation must never be sent without a deal (or vice versa), and double-click/retry on the approval queue must be safe. Alternative (synchronous send then DB commit) rejected: partial-failure window and duplicate-deal risk.

### D14: Dispatch-time state re-validation

Every procurement outbox handler (`procurement.inquiry_send`, `procurement.order_confirm_send`) re-validates state transaction-isolated inside the dispatch claim before invoking the WhatsApp client: run not `cancelled`/`escalated`, recipient still `pending` (inquiry sends), order not already sent, kill switch off, and (for order confirmations) consent not `withdrawn`. On any guard failure the event completes without sending and the block is audited with the reason (`kill_switch`, `consent_withdrawn`, `run_cancelled`). Rationale: kill-switch enablement and webhook/event redelivery race with already-enqueued events; enqueue-time checks alone cannot prevent a send after the kill switch flips. Alternative (enqueue-time checks only) rejected: violates the D8 kill-switch guarantee.

### D15: Schema indexes and uniqueness

Migration `000036` adds, beyond the composite tenant FKs and status CHECKs: unique `(organization_id, nit)` on `suppliers` (one supplier per NIT per org — also prevents duplicate contacts), unique `(organization_id, sku)` on `products`, unique `(recipient_id, raw_message_id)` on `inquiry_responses`, unique `(run_id, supplier_id)` on `orders`, plus indexes `inquiry_recipients(organization_id, contact_id)` (the hot inbound lookup shared by the procurement subscriber and the agent skip check), `inquiry_recipients(run_id)`, `inquiry_runs(organization_id, status)`, `suppliers(organization_id)`, `products(organization_id)`. Rationale: every inbound WhatsApp message now performs the active-recipient lookup; an unindexed scan on a growing table would degrade webhook latency.

### D16: Pacing gate and observability

Fan-out pacing is an in-process per-organization token bucket (capacity 10, refill 1 token/1s → 10 msgs/10s) shared by all dispatcher workers and keyed by `organization_id`; order confirmations share the same gate so approval bursts cannot exceed the WhatsApp limit either. Observability: the LLM paths already meter into the ai-usage ledger; procurement additionally increments counters for draft/extraction/summary attempts, extraction escalations, sends succeeded/retried/dead-lettered (per event type), order placements, and kill-switch/consent blocks, wired into the existing metrics surface. Rationale: per-worker counting would overshoot the burst under concurrency; without counters, extraction escalation and dead-letter rates are invisible in production.

## Risks / Trade-offs

- **Supplier replies with half the SKU list** -> extraction marks `requiere_humano`; the board shows partial items and the run can end `partially_answered`; nothing auto-quotes.
- **Meta 24h window blocks cold outbound** -> depends on `add-whatsapp-template-messages` (implement first); the fan-out records the `outside_24h_window` warning per `whatsapp-outbound-send` and still marks the recipient sent.
- **LLM price hallucination** -> prices are never quoted to customers automatically; orders require human approval; `requiere_humano` escalates ambiguity, negotiation, and price ranges.
- **Low-confidence extraction** -> `requiere_humano` flags the recipient and run `escalated`; no auto-quote anywhere.
- **Outbox double-dispatch / webhook redelivery** -> idempotent handlers with transaction-isolated state checks (recipient `sent`/`answered` transitions guarded; responses unique per `(recipient_id, raw_message_id)`), consistent with the constitution's webhook-idempotency rule.
- **Duplicate order POST / approval retry** -> `procurement.orders` UNIQUE `(run_id, supplier_id)` (D13); a retried placement returns the existing order without a second send or a duplicate deal.
- **Rate-limit burst on fan-out** -> fan-out paces at 10 msgs/10s per organization; burst never exceeded.
- **Cross-tenant access** -> composite tenant FKs `(organization_id, ...)` on procurement rows referencing `crm.contacts`, mirroring `crm-core-data`.
- **Kill switch mid-run** -> run transitions `cancelled`; send handlers re-validate run state inside the dispatch claim (D14) so already-enqueued events are not sent; audited as `skip` with reason `kill_switch`.
- **Consent withdrawn for a supplier** -> order confirmation send blocked (deal/actividad still recorded); the supplier's Habeas Data rights flow through the existing compliance endpoints.
- **Masking/consent compliance drift** -> the business-identity allowlist is limited to NIT supplier contacts and the org-declared consent grant is audited with `basis = 'org_declared'` (D11); natural-person contacts keep the full `whatsapp-compliance` masking and consent pipeline.
- **LLM cost growth** -> exactly one metered call per draft, per reply, and optional per board view; exhausted credits escalate instead of burning unmetered tokens.

## Migration Plan

1. Migrations `000036_create_procurement_schema` (up/down): create `procurement` schema with `suppliers`, `products`, `inquiry_runs`, `inquiry_recipients`, `inquiry_responses`, `orders` (composite tenant FKs `(organization_id, ...)`, status CHECKs, unique `(recipient_id, raw_message_id)` on responses and `(run_id, supplier_id)` on orders, unique `(organization_id, nit)` on suppliers and `(organization_id, sku)` on products, and the index set in D15); down drops the schema. Additive — safe to deploy.
6. Rollback note: before reverting, an operator MUST drain or cancel in-flight `procurement.inquiry_send` / `procurement.order_confirm_send` outbox events (delete pending events or let them dead-letter after handlers are removed) so no message is stranded mid-send.
2. SQLC queries + regen (`docker compose run --no-deps cli sqlc generate`).
3. Backend module `internal/modules/procurement/` (domain → app → infrastructure) + agent subscriber skip check; deploy with the migrations.
4. Frontend pages/wizard/board/queue behind feature usage; Spanish copy.
5. Ordering note: the whatsapp-agent skip path is safe without the template change; cold-outbound template mitigation belongs to `add-whatsapp-template-messages` (implement first).

## Open Questions

- Should one supplier ever map to multiple contacts (e.g., several reps per supplier)? This change is one supplier = one contact; multi-contact suppliers can be added later.
- Unanswered-recipient follow-up cadence: `followup_count` is reserved but undriven — the scheduled-runs change owns any automation. The lazy timeout window (D12) is a fixed 24h default in this change; making it configurable belongs to the settings surface (later change).
- Should the order confirmation eventually use an approved template once `add-whatsapp-template-messages` lands? Out of scope here; the plain-text path stays.
