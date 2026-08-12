# Proposal: add-inbox-ai-writing-assist

## Why

A 2026 AI-first inbox ships writing assistance **inside the composer** (Gmail "Help Me Write", Slack, TinyMCE AI) — rewrite, make formal/casual, summarize on demand. This product already generates AI reply drafts passively (agent suggestions panel) and summarizes conversations (context panel), but the composer itself (`reply-input.tsx`) is a bare input: the primary daily action (replying) has zero AI affordance. This change closes that gap by reusing the existing metered, PII-masked, credit-gated LLM pipeline.

## What Changes

- **Backend**: new `POST /api/agent/rephrase` endpoint in the agent module (`internal/modules/agent/routes.go`) with modes `rephrase | formal | casual | summarize`. Request `{text, mode}`; response `{"data": {"text": "<rewritten>"}, "success": true}`. Behavior mirrors the existing `analyze()` pattern (`agent_service.go:177`): credit gate via `billing.GetAiUsageStatus` (fail-open on ledger errors, 402 on exhausted), org-scoped context (`WithOrgID`), metered LLM call (`TokenLedger`), no auto-send.
- **Frontend**: Sparkles action dropdown in `reply-input.tsx` (Rephrase / Hacer formal / Hacer casual / Resumir) beside the Send button. Result **replaces the draft text** for the user to review — never sent automatically. Loading state disables the dropdown; failure leaves the draft untouched and shows a Spanish toast.
- **Copy**: labels + errors under the existing `ui.agent` namespace (Spanish-first).
- **No persistence, no auth-flow, no Stytch policy changes.**

## Capabilities

### New Capabilities

- `agent-writing-assist`: composer-level AI text transformation (rephrase, tone change, summarize) for outbound drafts — authenticated, org-scoped, credit-gated, metered, never auto-sent.

### Modified Capabilities

None.

## Impact

- **Code**: `go-b2b-starter/internal/modules/agent/` — `app/services/agent_service.go` (+1 method following `analyze()`), `handler.go` (+1 handler), `routes.go` (+1 route, `org:view`), new unit tests. `next_b2b_starter/` — `app/dashboard/inbox/components/reply-input.tsx` (dropdown), `lib/api/api/repositories/agent-repository.ts` (+`rephrase`), DTO/model, hook, `lib/copy/ui.ts` (`ui.agent` keys), component test.
- **Dependencies**: none new — reuses `internal/platform/llm` (metered client, PII masking, ledger) and billing credits.
- **Systems**: AI credit metering consumed per call, identical to suggestions/audience builder.

## Non-Goals

- No auto-send / agentic sending from the composer — output is always user-reviewed before send.
- No new LLM provider, model, or metering changes.
- No change to the suggestion pipeline, consent state machine, or compliance flows.
- No local credential storage; no credentials involved (LLM keys already configured via existing platform config).
- No new persistence or DB schema.

## Rollback

- **Git state**: revert the touched files (`agent_service.go`, `handler.go`, `routes.go`, `routes_test.go`/service test, `reply-input.tsx`, `agent-repository.ts`, DTO/model/hook, `lib/copy/ui.ts`, component test, this change's artifacts). All additions are additive; no migration, no data.
- **Stytch tenant policy state**: no policy changes, so no policy rollback required.
