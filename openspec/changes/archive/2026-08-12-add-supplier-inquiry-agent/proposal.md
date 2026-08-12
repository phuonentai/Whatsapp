# Proposal: add-supplier-inquiry-agent

## Why

Colombian stockless retailers ("venta sin inventario" — tiendas, papelerías, ferreterías, bazares) sell without holding stock: every day they manually message their suppliers over WhatsApp checking availability, price, and lead time, then quote customers from memory. The platform already has WhatsApp outbound/inbound, the AI agent pipeline, metered LLM, and CRM — this change adds the procurement capability: suppliers registry, product list, inquiry runs with AI-drafted messages, structured reply extraction, an aggregation board, and human-approved order sends.

## What Changes

- **New capability `supplier-inquiries`** (see `specs/supplier-inquiries/spec.md`): org-scoped suppliers registry (a supplier IS a CRM contact with NIT, consent recorded `granted` at creation), product/SKU catalog, inquiry runs (`POST /api/procurement/runs`) with one metered LLM draft per supplier, durable-outbox fan-out through the circuit-breakered WhatsApp client, structured reply extraction with a quote schema, deterministic aggregation board, human-approved order placement (CRM negocio + actividad), run lifecycle (kill switch, escalation always allowed), RBAC + compliance (export/forget reuse the existing agent endpoints), metered AI.
- **Modified `whatsapp-agent`** (see `specs/whatsapp-agent/spec.md`): messages consumed by an active supplier inquiry run recipient skip the sales-agent analysis pipeline — no double-processing, no conversation flow or suggestion created, procurement stays an independent subscriber to `whatsapp.message.received` and the webhook response is never blocked.
- **Backend**: new `procurement` module (domain → app → infrastructure) in `go-b2b-starter/internal/modules/procurement/`; SQLC migrations `000036+` (next after `000035`); API routes under `/api/procurement/...`; RBAC `org:manage` for writes/approvals and `org:view` for reads; all routes behind the existing `auth` + `org_context` + `subscription` middleware like other CRM routes; Spanish-first error messages. Order placements are atomic and idempotent (`procurement.orders` UNIQUE `(run_id, supplier_id)`, confirmation enqueued via the durable outbox in the same transaction); supplier consent is an audited org-declared grant (`basis = 'org_declared'`); unanswered recipients terminate via a lazy read-time timeout (24h window, no scheduler).
- **Frontend**: suppliers page, products page, inquiry-run creation wizard, comparison board, and approval queue in `next_b2b_starter/` (TanStack Query, shadcn/ui, Spanish copy).
- No **BREAKING** changes: everything is additive; the only existing-spec touch is the additive skip scenario on the whatsapp-agent inbound trigger.

## Capabilities

### New Capabilities

- `supplier-inquiries`: suppliers registry (contact-linked, consent granted at creation), product/SKU catalog, inquiry runs (AI-drafted Spanish messages, durable fan-out), structured reply extraction (quote schema), aggregation board with deterministic ranking, human-approved order sends, flow lifecycle (kill switch, escalation), compliance (Ley 1581 consent, PII masking, RBAC), metered AI.

### Modified Capabilities

- `whatsapp-agent`: the inbound-trigger requirement gains a skip path — a `whatsapp.message.received` event belonging to an active supplier inquiry run recipient skips analysis and suggestions (no flow or suggestion created), without blocking or duplicating the procurement subscriber's processing.

## Impact

- **Code**: `go-b2b-starter/` — migrations `000036+` (procurement schema up/down), SQLC queries + regen, new `internal/modules/procurement/` (domain/app/infrastructure), the agent subscriber skip check, tests. `next_b2b_starter/` — suppliers, products, run wizard, comparison board, approval queue + copy/tests.
- **API**: new `/api/procurement/suppliers`, `/api/procurement/products`, `/api/procurement/runs`, `/api/procurement/runs/:id`, `/api/procurement/runs/:id/orders`; no existing endpoint changes.
- **Dependencies**: none new — reuses `internal/platform/llm` (metered client + ai-usage ledger), `pkg/whatsapp/client.go` (circuit-breakered), the durable outbox, and the existing `auth`/`org_context`/`subscription` middleware. Auth/RBAC continue to reference the Stytch B2B API contracts (members, orgs, sessions, roles `org:manage`/`org:view`); no Stytch policy changes.
- **Systems**: metered LLM calls (one per supplier draft, one per replied inquiry message, optional board summary); WhatsApp outbound sends within the existing rate limit (10 msgs/10s via a per-org token bucket) and 24h-window semantics per `whatsapp-outbound-send`; new DB rows in the `procurement` schema (suppliers, products, runs, recipients, responses, orders) referencing existing `crm.contacts` (NIT via `tipo_documento`).

## Non-Goals

- **No local credential storage** — credentials remain solely in Stytch B2B and the existing encrypted WhatsApp config store; this change adds no credential-handling path and no new secret storage.
- No customer-facing auto-quoting — extracted supplier prices are never quoted to customers automatically; orders require human approval.
- No Siigo purchase-order integration in this change (the invoicing module is untouched).
- No WhatsApp template messages — order confirmations are plain text; template infrastructure is the separate `add-whatsapp-template-messages` change.
- No WhatsApp-native intent routing.
- No scheduled/automated inquiry runs — scheduling is the separate `add-scheduled-inquiry-runs` change; no follow-up automation (the `followup_count` column is reserved, not driven).
- No changes to the agent pipeline internals beyond the documented skip path; no new AI provider, model, or metering.

## Rollback

- **Git state**: revert the touched files — migrations `000036+` (up + down), SQLC queries + regen, `internal/modules/procurement/`, the whatsapp-agent subscriber skip check, the FE pages/components/copy/tests, and this change's artifacts. The down migrations drop the `procurement` schema (additive — safe). Before reverting, an operator MUST drain or cancel in-flight `procurement.inquiry_send` / `procurement.order_confirm_send` outbox events (delete pending events or let them dead-letter after handlers are removed) so no message is stranded mid-send.
- **Stytch tenant policy state**: no Stytch policy changes — all new routes reuse the existing `org:manage`/`org:view` roles behind existing middleware, and local tables store only `stytch_member_id`/`stytch_organization_id` foreign keys per the Dual SSOT constitution. No tenant-policy rollback is required.

## Assumptions

- Suppliers are represented as `crm.contacts` (verified: contacts support `tipo_documento` including NIT and `numero_documento`), so procurement history references a contact row and reuses the consent/compliance machinery.
- Every organization has a default pipeline (`es_predeterminado`, "Pipeline de Ventas") auto-seeded per `pipeline-management`; order placement creates the negocio there.
- The `add-whatsapp-template-messages` change lands first where cold outbound (outside the 24h window) matters; inquiry sends record the `outside_24h_window` warning per `whatsapp-outbound-send` regardless.
- `procurement.inquiry_runs` is the flow row for the run, mirroring the `agent.conversation_flows` pattern; escalation semantics mirror `agent-governance`.
