## Context

WhatsApp messages are strictly inbound today: the webhook verifies HMAC signatures, logs to `whatsapp.webhook_logs`, publishes `whatsapp.message.received`, and the CRM listener persists the message. The outbound seam already exists — `pkg/whatsapp/client.go` has `SendTextMessage` plus `ClientWithBreaker` (threshold 5 / cooldown / half-open probes) and the CRM `OutboundService` already uses it while persisting `crm.messages` rows with `direction = 'outbound'`. The ai-usage-metering work is largely implemented: migration `000018` (`subscription_billing.ai_usage`, `ai_usage_events`), `internal/platform/llm/infra/metered_llm_client.go`, `internal/modules/billing/infra/credits/rates.go`, and `BillingService.GetAiUsageStatus`/`ai_assistant` fields in `billing_provider.go`. The eventbus is in-process and dispatches handlers in goroutines, so subscriber work never blocks the webhook HTTP response.

This change adds an agentic WhatsApp assistant with two autonomy modes:
- **Copilot (default)**: the agent analyzes inbound messages with metered LLM calls, produces reply drafts as *pending suggestions*, and a human rep approves/edits/rejects before anything is sent.
- **Autopilot**: the agent may send directly, but only when a deterministic, in-domain `GuardrailService` allows it (kill switch, discount cap, forbidden terms, escalation terms, consent, send window, daily limit). Denied autopilot sends fall back to pending suggestions.

Every side-effecting action is evaluated by the guardrail layer and appended to an immutable `agent_actions` audit ledger. The conversation flow is a **linear pipeline** (analysis → decide → notify/send) per conversation — not a DAG; no workflow engine is introduced.

## Goals / Non-Goals

**Goals:**
- Copilot/autopilot modes with deterministic, tenant-editable guardrails (parameters-as-data from `agent_settings.guardrails`)
- **No autonomous sending in copilot mode**; in autopilot mode only guardrail-approved sends
- Escalation is ALWAYS allowed: automation can never trap a lead
- Ley 1581 compliance: consent state machine (none → requested → granted / withdrawn), consent template as first message, data export and anonymization (forget) endpoints
- Append-only audit of every governance evaluation, anchored to the approving Stytch member id and linked to the ai-usage ledger via `request_id`
- Metered analysis: every LLM call runs through the existing metered client, gated by AI credits
- Reuse the existing outbound seam (`crm.OutboundService` → `ClientWithBreaker`)
- Never block the webhook response; flows advance from the async eventbus subscriber and from API actions

**Non-Goals:**
- No DAG/workflow engine (the pipeline is linear; a future Temporal runtime is a separate decision)
- No OPA/Rego or any external policy engine — guardrails are deterministic Go evaluated in-domain ("no external policy engines", per the adopted design)
- No calendar integration (slot proposals are out of scope for this change)
- No full L3 autonomy (autopilot is guardrail-bounded and consent-gated)
- No changes to the webhook ingress contract, Stytch identity flows, or the feature-flag derivation model
- No local credential storage — Stytch remains the sole identity authority; guardrails govern machine actions only

## Decisions

### D1: `agent` schema — migration `000019`

Tables (all scoped by `organization_id` FK to `organizations.organizations`):

- `agent.conversation_flows` — one row per conversation run: `organization_id`, `conversation_id`, `contact_id`, `status` (`running`, `awaiting_human`, `succeeded`, `failed`, `cancelled`); index `(conversation_id, status)`. **Linear pipeline state only** — no nodes/edges (the earlier DAG design with `flow_nodes`/`flow_edges` was dropped).
- `agent.agent_settings` — one row per org (UNIQUE `organization_id`): `mode` (`copilot`|`autopilot`), `tone` (`formal`|`casual`), `brand_voice` TEXT, `autopilot_start`/`autopilot_end` TIME, `timezone` (default `America/Bogota`), `kill_switch` bool, `max_daily_messages` int (0 = unlimited), `consent_required` bool (default TRUE), `consent_template` TEXT, `guardrails` JSONB (`never.max_discount_percent`, `never.forbidden_terms[]`, `escalate.terms[]`).
- `agent.agent_suggestions` — `organization_id`, `conversation_id`, `contact_id`, `flow_id` (SET NULL), `type` (`reply`|`escalation`), `body`, `metadata` JSONB, `status` (`pending`|`approved`|`rejected`|`superseded`), `source` (`copilot`|`autopilot_fallback`|`escalation`), `approved_by_member_id` (Stytch `stytch_member_id`), `whatsapp_message_id`, `request_id`; index `(organization_id, status, created_at DESC)`.
- `agent.agent_actions` — append-only audit: `action`, `decision` (`allow`|`deny`|`skip`), `policy_input` JSONB, `reasons` JSONB, `approved_by_member_id`, `whatsapp_message_id`, `request_id`; index `(organization_id, created_at DESC)`. No UPDATE path.
- **Ley 1581**: `ALTER TABLE crm.contacts ADD consent_status (none|requested|granted|withdrawn) + consented_at` with a CHECK constraint.

Rationale: keeps the agent's state self-contained in one schema; `crm.contacts` gains only the consent columns (data still owned by CRM). Rollback via `000019.down.sql` (drops the `agent` schema + consent columns).

### D2: Linear pipeline — analysis → decide → notify/send

`AgentService.HandleMessageReceived` per inbound message event:
1. Skip non-text/empty messages.
2. Idempotency: a redelivered webhook with an existing pending suggestion for the same `whatsapp_message_id` is skipped.
3. Resolve contact + active conversation (idempotent upserts mirroring CRM patterns — the agent does not depend on the CRM listener's event ordering).
4. Load settings (defaults materialized on first read); kill switch → flow cancelled + `skip` audit.
5. **Consent state machine (Ley 1581)**, run before analysis:
   - `none` + `consent_required` → send consent template (outbound) → mark `requested` → audit `consent_request`; no further processing for this message.
   - `requested` + affirmative reply ("sí", "acepto", "autorizo", …) → mark `granted` → audit `consent_grant`; continue.
6. **Analysis**: consolidated metered LLM call (JSON `{intent, sentiment, suggested_reply}`); credits exhausted → escalation suggestion (`ai_credits_exhausted`) + flow `awaiting_human`; unparsable responses fall back to raw text.
7. **Decide + act**: copilot → supersede pending replies for the conversation → insert pending suggestion (source `copilot`). Autopilot → guardrail evaluation with `Autonomous: true`; allowed → send via `crm.OutboundService` + flow succeeded; denied → escalation on `escalation_match` else pending suggestion (source `autopilot_fallback`) + flow `awaiting_human`.

Flow status transitions: `running` → `succeeded` | `awaiting_human` | `cancelled` | `failed`.

Rationale vs the earlier DAG design: the L1/L2 scope never needed fan-out/join; a linear pipeline keeps state in one flow row and makes each decision auditable at one point. If parallel analysis branches are ever needed, a DAG can be introduced behind `AgentService` without changing the API.

### D3: Deterministic guardrails in-domain (no OPA/Rego)

`domain.GuardrailService` + `infra/guardrails/guardrail_service.go` implement deterministic rules with zero external engines:

- `escalate_human` and `generate_draft` → always allowed.
- `send_message`:
  - **Never rules (always apply, even to human approvals):** kill switch; discount cap (`max_discount_percent`, parsed from the draft); forbidden terms.
  - **Escalate rules:** matching terms (defaults: abogado, legal, garantía, demanda, superintendencia) deny autonomous sends.
  - **Autonomous-only checks (bypassed by human approval):** consent not granted (when `consent_required`), outside the autopilot send window (overnight windows supported, timezone-aware), daily limit reached.
- Fail-safe direction: unknown actions and evaluation errors → deny with `guardrail_error`.

The earlier OPA/Rego design was dropped: the concurrent design explicitly requires transport-free guardrails with no external policy engines; the policy table stays identical in behavior and is covered by Go unit tests.

### D4: Approval flow

`POST /api/agent/suggestions/:id/approve` (body may carry `edited_body`):
1. Load suggestion (must be `pending`).
2. Evaluate guardrails with `Autonomous: false` (kill switch + never/escalate rules still apply; window/consent/limit do not).
3. Audit the evaluation (`allow`/`deny` + reasons).
4. Denied → suggestion is REJECTED and a `DenialError` surfaces as HTTP 409.
5. Allowed → `crm.OutboundService.SendMessage` → suggestion `approved` with `approved_by_member_id` (Stytch member id from the auth identity) → flow succeeded.

`POST /api/agent/suggestions/:id/reject` → status `rejected` + audit `deny` (`human_rejection`).

### D5: Governance boundary — Stytch unchanged

Guardrails govern machine actions only. Stytch remains the sole authority for member identity, sessions, and RBAC; agent routes reuse `auth` + `org_context` + `subscription` middleware plus `RequirePermissionFunc("org","manage")` (approve/reject/settings/compliance) and `org:view` (list/debug). The approving member is stored as `stytch_member_id` — an accountability anchor, not a local identity. No passwords, tokens, or session data enter the `agent` schema.

### D6: Audit + metering linkage

Every guardrail evaluation appends an `agent_actions` row (policy input snapshot: action, draft, autonomous flag, approver, consent, contact, timestamp + reasons). LLM analysis runs through the metered client with `domain.WithOrgID`, recording into `subscription_billing.ai_usage_events`; the same `whatsapp_message_id` and `request_id` fields make it possible to reconstruct: which AI calls consumed which credits, what guardrails decided, who approved, and what message went out.

### D7: Compliance API (Ley 1581)

- `GET /api/agent/compliance/export/:contactId` → full data bundle (contact + conversations + messages); PII masked (`[ELIMINADO]`/`[ANONIMIZADO]`) when consent is withdrawn.
- `POST /api/agent/compliance/forget/:contactId` → idempotent anonymization of the contact (name/phone/email/document fields scrubbed, consent → `withdrawn`).

### D8: API surface

- `GET /api/agent/suggestions` — pending queue (drives the FE inbox).
- `POST /api/agent/suggestions/:id/approve|reject`
- `GET/PUT /api/agent/settings` — mode, tone, brand voice, autopilot window, timezone, kill switch, daily limit, consent template, guardrails (discount cap, forbidden/escalate terms).
- `GET /api/agent/flows/:conversationId` — active flow + pending suggestions (debug).
- Compliance export/forget (D7).

All in `internal/modules/agent/`, registered under the existing router group conventions with the same middleware chain as other CRM routes.

### D9: Frontend

- `app/dashboard/inbox/components/message-thread.tsx` — render pending `reply` suggestions as draft chips with Approve / Edit / Reject (Approve sends immediately; Edit prefills the reply input).
- `app/dashboard/inbox/components/conversation-list.tsx` — pending-count badge per conversation.
- `app/dashboard/settings/components/agent-settings-section.tsx` — mode toggle (copilot/autopilot), tone, brand voice, autopilot window, kill switch, daily limit, consent template, guardrails (adjacent to `whatsapp-config-section.tsx`).

### D10: Seam for future changes

The `AgentService` interface is the seam for future capabilities: durable timers (a Temporal-backed follow-up) or calendar integration slot behind it without changing the API surface.

## Risks / Trade-offs

- **[Fail-safe guardrails]** Guardrail evaluation errors produce deny (`guardrail_error`), never allow; audit failures are logged, not blocking (the send path still depends on the audit row being insertable — see next risk).
- **[Audit coupling]** The agent service fails closed when the audit insert fails (returns the error) — acceptable for v1 because every action must be explainable; switch to fire-and-forget + alerting if latency becomes an issue.
- **[Consent as first message]** New contacts always receive the consent template before any analysis — this delays first-touch AI replies for new leads, which is the intended Ley 1581 posture; orgs can disable `consent_required`.
- **[Autopilot blast radius]** Autopilot sends are bounded by kill switch, window, daily limit, consent, and never/escalate rules, but a wrong draft can still go out → the audit + supersede flow and the `autopilot_fallback` path limit damage; default mode is copilot.
- **[In-process eventbus]** Pipeline work runs in the eventbus handler goroutine (not the webhook response); heavy LLM latency under high volume is a listed future concern (worker pool or Temporal behind `AgentService`).
- **[ai-usage-metering drift]** That change is not archived; `BillingService.GetAiUsageStatus` is used (verified in `get_ai_usage_service.go`); if its API shifts, the analysis gate adapts.
- **Rollback**: Git revert of the change; `000019.down.sql` drops the `agent` schema and the contact consent columns. No Stytch tenant policy state exists for this feature (no auth/RBAC changes), so no Stytch-side rollback is required. No local credentials are ever stored.
