# Design: add-inbox-ai-writing-assist

## Context

The inbox composer (`next_b2b_starter/app/dashboard/inbox/components/reply-input.tsx`) is a bare `Input` + Send button. The product already ships AI reply drafts (agent suggestions panel) and conversation summaries (context panel), both backed by a metered, PII-masked, credit-gated LLM pipeline in `internal/modules/agent`. The 2026 AI-first baseline puts writing assistance in the composer itself.

Verified facts (premise validation, 2026-08-11):
- `reply-input.tsx` — controlled/uncontrolled `Input`, `onSend(content)` clears the draft; no AI affordance.
- `agent_service.go:177` `analyze()` — the exact pattern to reuse: `billing.GetAiUsageStatus` credit gate (fail-open on ledger error, `ai_credits_exhausted` skip), `llmdomain.WithOrgID(ctx, orgID)` + `WithPiiFacts`, metered `s.llm.Complete`, tolerant response parse (`extractSuggestedReply`).
- `agent/routes.go` — `/agent` group behind `auth` + `org_context` + `subscription`, per-route `org:view`/`org:manage`.
- FE repository pattern — `agent-repository.ts` uses `apiClient` with the `{"success": bool, "data": T}` envelope; mutations via TanStack Query hooks.
- `lib/copy/ui.ts` has `ui.agent` (Spanish-first) + `en` mirror.
- Existing credit-gate semantics elsewhere: `credit_guard.go:25` returns HTTP 402 when `CreditsMax > 0 && remaining <= 0`, fail-open on ledger errors.

## Goals / Non-Goals

**Goals:**
- One backend endpoint transforming user-authored draft text (4 modes), following the established `analyze()` pattern.
- Composer UX: action dropdown → result replaces draft for review; failure never destroys the draft.
- Full parity with existing governance: metered, org-scoped, credit-gated, Spanish errors.

**Non-Goals:**
- No auto-send, no agentic behavior from the composer.
- No new LLM provider/model/metering, no schema/persistence changes, no consent-machine changes.
- No changes to the passive suggestion pipeline.

## Decisions

### D1: Single endpoint with a mode switch, not four endpoints

`POST /api/agent/rephrase` with `{text, mode}` where `mode ∈ {rephrase, formal, casual, summarize}`. One handler, one service method (`RephraseText`), one system prompt with the mode as instruction. Rationale: all four modes share identical plumbing (credit gate, metering, PII context, error mapping); four endpoints would duplicate it. Alternative considered: mode as part of the prompt only — rejected (mode must be validated up front for a 400 and to keep prompts deterministic).

### D2: Follow the `analyze()` pattern exactly

`RephraseText(ctx, orgID, text, mode)`:
1. Credit gate: `billing.GetAiUsageStatus(ctx, orgID)`; ledger error → warn + fail-open; `CreditsMax > 0 && CreditsRemaining <= 0` → return `ErrAICreditsExhausted` (handler maps to 402 `ai_credits_exhausted`).
2. `ctx = llmdomain.WithOrgID(ctx, orgID)` — metering attribution.
3. Metered `s.llm.Complete` with a small `rephraseSystemPrompt(mode)` (temperature/max-tokens from existing `llm` config defaults).
4. Return trimmed text.

No `WithPiiFacts` here: the input is a draft the user typed themselves (user-authored), not third-party contact data — PII masking applies to inbound analysis, not to transforming the user's own text. This is documented so it doesn't get "fixed" later without thought. (If a future mode ever consumes conversation history, masking must be added then.)

### D3: Handler contract + routing

- Route: `group.POST("/rephrase", auth.RequirePermissionFunc("org", "view"), r.handler.HandleRephrase)` — drafting is view-level; sending remains `org:manage` on the send path.
- Bind `{text, mode}`; validate: non-empty text (400 `text_required`), known mode (400 `invalid_mode` — no LLM call).
- Success: 200 `{"data":{"text": ...},"success":true}` (envelope convention).
- 402 mapping for exhausted credits; 500 wrapped on LLM failure (client shows generic Spanish toast; draft preserved).

### D4: Frontend — dropdown in the composer, replace-draft semantics

- `reply-input.tsx`: Sparkles `DropdownMenu` beside Send, items Rephrase / Hacer formal / Hacer casual / Resumir (from `ui.agent` copy), disabled when `!text.trim() || !conversationId || isSending || isRephrasing`.
- Mutation hook `useRephraseMutation` (TanStack) calling `agentRepository.rephrase(text, mode)`; on success `setText(result.text)`; on error `toast.error(ui.agent.rephraseError)` and draft untouched; on 402 show the existing credits-exhausted message pattern.
- Loading state: dropdown items show a spinner / button disabled.

### D5: Copy under `ui.agent` (Spanish-first)

Add `rephrase`, `rephraseFormal`, `rephraseCasual`, `rephraseSummarize`, `rephraseError`, `rephraseCreditsExhausted` (+ `en` mirror) — consistent with the `ui.agent` namespace and the `SWEPT_FILES` copy-sweep enforcement.

## Risks / Trade-offs

- **Cost**: each click is a metered LLM call; mitigated by credits gate + user-initiated (no background amplification). Same unit economics as suggestions.
- **Latency**: synchronous transform call adds ~1-3s to the composer; acceptable for an explicit action, spinner shown.
- **Draft semantics**: replace-on-success vs insert-at-cursor — chosen replace (the draft is a short WhatsApp reply; replace is what users expect from "rephrase"). Trade-off documented; cursor-insert is a follow-up if needed.
- **Tone drift**: model may produce slightly different meaning on "formal"/"casual"; user reviews before send, so risk is user-controlled.
- **Prompt-injection**: user text is instructions-adjacent; the system prompt is fixed and output is never auto-sent, so blast radius is a rewritten draft only.
