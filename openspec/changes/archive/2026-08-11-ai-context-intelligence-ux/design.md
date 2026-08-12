## Context

`agent` schema already holds `conversation_flows`, `agent_settings`, `agent_suggestions`, `agent_actions`; `crm.conversations`/`crm.messages` hold chat history. The analysis LLM path is metered by `ai-usage-metering` and PII-masked per `whatsapp-compliance`. Agent routes sit behind `auth` + `org_context` + `subscription` middleware with Stytch RBAC (`org:manage` / `org:view`). The inbox thread renders raw history only.

## Goals / Non-Goals

**Goals:**
- Per-conversation AI context endpoint (summary, intent, key facts).
- Metered generation, PII-masked, consent-gated.
- Cache with new-message invalidation.
- FE context panel + contact intelligence + "learning" state.

**Non-Goals:**
- No autonomous behavior change; no send-path changes.
- No guardrail/suggestion-lifecycle changes.
- No full chat-text storage beyond the context cache.
- No ledger-contract change.

## Decisions

1. **Endpoint `GET /api/agent/conversations/:id/context`** under existing agent routes, `org:view` for reads. Response: `{ conversation_id, summary, detected_intent, key_facts[], source_cursor, generated_at, consent_gated, status }`.
2. **Cache table `agent.conversation_contexts`**: `conversation_id` PK/FK → `crm.conversations(id)` ON DELETE CASCADE; `summary TEXT`, `key_facts JSONB`, `detected_intent VARCHAR`, `source_cursor INTEGER`, `consent_gated BOOLEAN`, `generated_at`, `updated_at` (trigger maintained). One row per conversation.
3. **Regeneration rule:** serve cached context if `source_cursor` equals the latest message id for the conversation and `generated_at` is fresh (TTL 5 min); otherwise regenerate. Idempotent upsert keyed by `conversation_id`.
4. **Metered generation:** context LLM call goes through the same metered decorator as analysis, recording tokens to the ledger; `ai_credits_exhausted` escalates (returns `status: unavailable` with no unmetered fallback).
5. **Consent gating:** when org `consent_required` and contact consent is not `granted`, generate structural-only context (message count, first/last dates, channel) with `consent_gated: true` and NO LLM analysis. This mirrors `whatsapp-agent` consent semantics.
6. **PII masking:** message bodies fed to the LLM pass through the existing masking path before the call.
7. **FE:** `components/agent/conversation-context-panel.tsx` in the thread; contact-intelligence line in the thread header; "El asistente está aprendiendo…" empty state when context status is unavailable/absent. Copy via `lib/copy` namespace `agent`.
8. **Clean Architecture:** domain interface `ConversationContextService` in `domain`; LLM/metered/DB adapters in `infrastructure`; no Stytch SDK imports in domain.

## Risks / Trade-offs

- **LLM cost/latency per new message:** mitigated by cache + TTL + cursor check; context is cheap (short history windowed to last N messages).
- **Consent edge cases:** contacts who withdraw consent after context was generated — context row may contain previously generated facts; mitigation: on read, if consent is `withdrawn`, return masked/structural-only (regenerate or suppress facts).
- **DB migration churn:** new table needs `up`/`down`; regenerate `make sqlc`.
- **Dependency risk:** copy layer from `standardize-spanish-first-copy` must land before FE; BE is independent.
