## Why

The product reads and responds to WhatsApp chats (inbound pipeline, copilot, metered AI) but never surfaces the intelligence it derives back to the rep. A rep opening a conversation sees raw history with no summary, no intent, no key facts, and no sense that the assistant is building context. "Reading chats and extracting context" — the promise of the AI-first position — is invisible. Surfacing a per-conversation AI context panel turns the copilot from a reply-drafter into a memory layer.

## What Changes

- Add a backend endpoint `GET /api/agent/conversations/:id/context` that returns AI-generated conversation context: summary, detected intent/topic, and key facts, derived from the conversation's message history and the contact record.
- Context generation SHALL run through the existing metered LLM client (`ai-usage-metering`) so every generation is recorded in the AI usage ledger, and SHALL apply PII masking before any LLM call per `whatsapp-compliance`.
- Cache generated context in a new `agent.conversation_contexts` table, regenerated when the conversation has new messages since the generation cursor.
- Consent gating (Ley 1581): when an org requires consent and the contact has not granted it, the endpoint SHALL return structural context only (message counts, dates, channel) without AI analysis.
- Frontend: a context panel in the message thread, a contact-intelligence summary, and an "assistant is learning" state while no context exists yet.
- Copy resolves through the typed copy layer from `standardize-spanish-first-copy` (Spanish-first).

## Capabilities

### New Capabilities
- `ai-context-intelligence`: the conversation context endpoint, context cache, and the context panel UX that surfaces extracted conversation context to reps.

### Modified Capabilities
- `whatsapp-agent`: the agent pipeline gains the conversation-context requirement (metered generation, consent gating, RBAC) alongside its existing analysis and suggestion behavior.

## Impact

- Backend (`go-b2b-starter`): new migration `agent.conversation_contexts`, SQLC queries (`agent.sql`), domain service for context generation behind the metered LLM client, HTTP handler under the existing agent routes (`auth` + `org_context` + `subscription`, `org:view` for reads).
- Frontend (`next_b2b_starter`): context panel in the inbox thread, contact-intelligence card, "learning" empty state, copy layer additions.
- Dependencies: copy layer from `standardize-spanish-first-copy`.
- Compliance: context generation is an internal LLM analysis; PII masking from `whatsapp-compliance` applies, and consent-gated contacts receive no AI analysis when consent is required.
- Rollback: revert the backend migration/commit in Git (with `down` migration) and the frontend commit; no Stytch tenant policy state changes (RBAC reuses existing `org:view`/`org:manage` roles from Stytch), so no Stytch rollback applies.
- Non-Goals: no autonomous behavior change, no new send path, no changes to the suggestion lifecycle or guardrails, no storage of raw chat text for purposes beyond context caching, no local credential storage, no change to the `ai-usage-metering` ledger contract.

## Assumptions

- Conversation message history is already retrievable in the agent/CRM layer; the context generator reads messages through existing conversation-message queries (verified: `crm.conversations`, `crm.messages`, and `agent` schema tables exist).
- `ai_credits_max`-gated exhaustion already escalates in the agent pipeline; the context endpoint SHALL behave the same way (no unmetered generation).
