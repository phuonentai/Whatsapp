## Why

WhatsApp messages are strictly ingested today: the webhook verifies HMAC signatures, persists the message, and stops. No response handling exists — even though the outbound seam (`SendTextMessage` with a circuit breaker, used by the CRM `OutboundService`) is already in place. Sales teams expect context-aware AI assistance: draft replies, intent recognition, and guarded autonomy. This change introduces an agentic WhatsApp assistant with a **copilot mode** (AI drafts, human approves before any send) and a **guardrail-bounded autopilot mode**, governed by a deterministic in-domain `GuardrailService` (no external policy engines), with every side-effecting action audited and every LLM call metered. It also adds Ley 1581 (Colombian data protection) compliance: a consent state machine, data export, and anonymization.

## What Changes

- **New `agent` database schema (migration `000019`)**: `conversation_flows` (linear pipeline state), `agent_settings` (mode/tone/brand voice/autopilot window/timezone/kill switch/daily limit/consent template/guardrails JSONB), `agent_suggestions` (pending/approved/rejected/superseded drafts with source), `agent_actions` (append-only governance audit). `crm.contacts` gains `consent_status` + `consented_at` (Ley 1581)
- **Agent pipeline (`internal/modules/agent`)**: `AgentService` consuming `whatsapp.message.received` — idempotent contact/conversation resolution, consent state machine, consolidated metered LLM analysis (credit-gated), copilot suggestion creation with superseding, autopilot guarded sends with `autopilot_fallback`
- **Deterministic guardrails (`domain.GuardrailService` + `infra/guardrails`)**: kill switch, discount cap, forbidden terms, escalation terms, consent, send window, daily limit; escalate/draft always allowed; fail-safe deny on error. No OPA/Rego — transport-free by design
- **Approval flow**: `POST /api/agent/suggestions/:id/approve|reject`; approved sends use the existing CRM outbound service (circuit breaker + outbound message persistence); approvals anchor the Stytch `member_id`
- **Compliance API (Ley 1581)**: `GET /api/agent/compliance/export/:contactId` (PII-masked when consent withdrawn), `POST /api/agent/compliance/forget/:contactId` (idempotent anonymization)
- **More API**: `GET /api/agent/suggestions`, `GET/PUT /api/agent/settings`, `GET /api/agent/flows/:conversationId` (debug); RBAC via existing `org:manage`/`org:view`
- **Frontend**: suggestion draft chips with Approve/Edit/Reject in the inbox (`message-thread.tsx`), pending-count badge (`conversation-list.tsx`), agent settings panel beside `whatsapp-config-section.tsx`
- **Wiring fix**: the WhatsApp and CRM modules were never registered in the DI container (pre-existing runtime gap); `init_mods.go` now initializes whatsapp, crm, and agent modules, restoring webhook → eventbus → CRM/agent listeners

## Capabilities

### New Capabilities

- `whatsapp-agent`: agentic WhatsApp assistant behavior — pipeline ingestion, copilot drafts, guardrail-bounded autopilot, approval flow, consent state machine, flow lifecycle, compliance export/forget
- `agent-governance`: deterministic guardrail layer — rule table for `send_message`, escalate/draft always allowed, parameters-as-data, append-only action audit

### Modified Capabilities

- (none — inbound webhook ingress behavior is unchanged; the agent subscribes to the already-published event)

## Impact

- **Go backend**: migration `000019` (`agent` schema + contact consent columns); SQLC queries in `query/agent.sql` + regenerated code; new module `internal/modules/agent/` (domain, `infra/guardrails`, `infra/repositories/agent_repository.go`, `app/services` — `agent_service.go`, `compliance_service.go` — handler/routes/module/provider); DI wiring in `internal/db/inject.go` (agent repository) and `internal/bootstrap/init_mods.go` (whatsapp/crm/agent module init — fixes the pre-existing missing wiring); outbound reuses `crm.OutboundService`; AI metering via the existing metered LLM client + `BillingService.GetAiUsageStatus`
- **Database**: four agent tables + consent columns on `crm.contacts`, all scoped by `organization_id` (same Stytch-org FK pattern, no credentials stored)
- **Frontend**: inbox suggestion UI (`app/dashboard/inbox/components/`), settings panel (`app/dashboard/settings/components/`)
- **Dependencies**: none new (deterministic guardrails in Go; the earlier OPA/Rego dependency was dropped with the OPA design)
- **Config**: no new required env vars; per-org agent settings and guardrails are DB rows
- **Auth**: no changes to Stytch flows. Routes reuse `auth` + `org_context` + `subscription` middleware and `RequirePermissionFunc`; approvals record the acting Stytch `member_id`
- **Ops**: `go test ./internal/modules/agent/...` covers guardrails + pipeline; migration `000019` runs in the standard migrate flow
- **Rollback**: Git — revert the change (migration, module, routes, DI, FE). DB — run `000019.down.sql` dropping the `agent` schema and consent columns. Stytch tenant policy state is unaffected (no auth/RBAC changes); no local credentials are introduced anywhere
- **Non-Goals**: no DAG/workflow engine (linear pipeline; a future Temporal runtime is a separate decision); no OPA/Rego or external policy engines; no calendar integration; no unbounded L3 autonomy; no changes to the webhook ingress contract; no changes to the Stytch identity flow; **rejects any local credential storage — Stytch remains the sole identity authority; guardrails govern machine actions only**

## Assumptions

- **ai-usage-metering status**: migration `000018`, `metered_llm_client.go`, `credits/rates.go`, and `BillingService.GetAiUsageStatus` exist in code (verified in `get_ai_usage_service.go`), but the change lists 0/36 tasks complete and is not archived. This change depends on those pieces and does not re-implement them; if the metered client or usage-status API shifts, the analysis gate adapts
- **Consent template default**: the built-in default template text is a product/legal decision; orgs can configure their own via settings
- **Affirmative consent terms**: the deterministic list ("sí", "acepto", "autorizo", "ok", "claro", "dale", …) is a first-pass heuristic; false positives are possible and the audit trail records every `consent_grant`
- **FE anchoring**: the inbox components (`message-thread.tsx`, `reply-input.tsx`, `conversation-list.tsx`) and settings (`whatsapp-config-section.tsx`) exist as verified anchors; their exact internal layout will adapt during implementation
- **Guardrail defaults**: discount cap 10% and the escalation term list are starter defaults, tunable per org
