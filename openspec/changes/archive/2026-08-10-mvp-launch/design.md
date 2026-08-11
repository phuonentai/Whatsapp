## Context

The platform's launch story is built across six in-flight OpenSpec changes, none verified: `fix-frontend-eslint-flat-config` (7/9), `add-sellable-modules` (30/35), `add-agentic-whatsapp-assistant` (0/30, code exists in the tree), `add-vertical-playbooks` (0/21, code exists), `add-mercadopago-billing` (20/48) + `wire-mercadopago-billing` (46/59), and `add-crm-e2e-tests` (35/54). Several in-flight changes contain code but 0 verified tasks, and one has a stale factual premise: `add-vertical-playbooks` claims migration `000019_create_playbooks` while the actual migration file on disk is `000020_create_playbooks` (`000019` is the agent schema). The AI chat is single-turn: `Handler.Chat` (`internal/modules/cognitive/handler.go:48`) calls `ragService.Chat` and returns one JSON body. The streaming seam already exists end-to-end at the transport layer — `domain.LLMService.CompleteStream` (`internal/platform/llm/domain/service.go:37`) and its implementations in `openai_client.go:427` and `metered_llm_client.go` — but no handler, service, or route calls it, and the frontend `use-chat.ts` is a TanStack `useMutation` with no token rendering. `ai-usage-metering` is complete (36/36) and exposes `GET /api/crm/usage/ai` (`internal/modules/billing/routes.go:38`); its metered client records tokens after a successful call. CI is backend-only (`go-b2b-starter/.gitlab-ci.yml`, image `golang:1.22.4` — stale vs go.mod), no frontend or E2E job.

## Goals / Non-Goals

**Goals:**
- Drive the six in-flight changes across their verification gates so the MVP is launchable, correcting their task checkboxes and stale premises (notably the playbooks migration number) where they don't match reality
- Deliver end-to-end SSE streaming over the existing `POST /example_cognitive/chat` route: backend streams tokens via `CompleteStream` through the metered client (usage recording and credit guard preserved), frontend consumes and renders tokens incrementally
- Define and execute a single launch verification gate (backend + frontend + E2E) and a CI matrix including a new frontend/E2E job
- Produce `ROADMAP.md` at the repo root capturing the post-MVP phases
- Never change the webhook ingress contract, the feature-flag derivation model, or Stytch identity flows

**Non-Goals:**
- No autopilot autonomous sending in the MVP — only the copilot (draft → human approve/reject → send) path of the agent change is verified; autopilot is a Q2 roadmap item
- No multi-model routing, prompt registry, semantic cache, evals, or OTel tracing (roadmap items)
- No new migrations from this change — `000019` (agent) and `000020` (playbooks) already exist; only the playbooks *proposal/design text* is corrected
- No re-architecture of the agent pipeline (linear DAG stays); no new policy engine
- No local credential storage — Stytch remains the sole identity authority; this change introduces no auth/RBAC changes and no credentials

## Decisions

### D1: Streaming is added to the existing route, not a new endpoint

`POST /example_cognitive/chat` stays the contract. The handler detects an `Accept: text/event-stream` (or `stream: true` request body field) and switches to an SSE response; otherwise it preserves the existing JSON behavior. Rationale: no frontend routing changes, no duplicated middleware, the existing `ai_assistant` flag + credit guard + RBAC chain applies unchanged to both modes.

Alternative considered: a separate `/example_cognitive/chat/stream` route. Rejected — it duplicates the middleware chain and the frontend's URL selection logic for no benefit.

### D2: Streaming path mirrors the non-streaming path in `ragService`

`RAGService` gains `ChatStream(ctx, orgID, accountID, req, emit func(domain.StreamEvent) error) (*domain.ChatResponse, error)`:
1. Get-or-create session and save the user message (identical to `Chat`)
2. Build the RAG/plain prompt and history (identical to `Chat`)
3. Call `assistantProvider.GenerateResponseStream(ctx, prompt, emit)` — a new `AssistantProvider` method that wraps `llmClient.CompleteStream` through the same metered client, forwarding `domain.StreamChunk{Content, Done}` and recording `TokensUsed` into the ai-usage ledger exactly as the non-streaming path does
4. On completion, persist the assistant message with final `TokensUsed` and return the same `ChatResponse` shape used for the JSON path (so history is consistent)

Rationale: the metered client already decorates `CompleteStream` (verified in `metered_llm_client.go`), so the credit guard and ledger recording need zero new work — the streaming handler calls the same decorated client. The `emit` callback keeps the service transport-free (no gin/SSE types in the domain layer).

### D3: SSE framing is thin

The handler writes `Content-Type: text/event-stream`, sends `data:` lines per chunk (`{"token": "<text>"}`), a final `data: {"done": true, "session_id": N, "message_id": N}` event, then `event: error` frames with the existing HTTP error shape on failure. A flush after each frame (`c.Writer.Flush()`). Rationale: minimal, debuggable, and the frontend can parse with `EventSource` or the Vercel AI SDK's streamable protocol; no new dependency on the backend.

### D4: Frontend streaming via Vercel AI SDK `useChat` (or native fetch-stream fallback)

Replace the mutation in `lib/hooks/mutations/use-chat.ts` with a streaming consumer. Preferred: `@ai-sdk/react` `useChat` posting to `/example_cognitive/chat` with `Accept: text/event-stream`; fallback: a thin `fetch` + `ReadableStream` reader wrapping the same SSE framing (no SDK dependency). `chat-message.tsx` renders incrementally. Rationale: the Vercel AI SDK is the industry-standard streaming consumer (already identified in `AI-architecture.md`); the fallback keeps the change shippable if the SDK's stream protocol doesn't match our thin framing.

### D5: In-flight changes are reconciled, not rewritten

For each in-flight change, tasks verify the on-disk state, run the change's own verification commands, and correct stale claims. Concretely: the playbooks proposal/design text is updated `000019` → `000020`; `add-mercadopago-billing` and `wire-mercadopago-billing` task checkboxes are reconciled with reality (13.2–13.6 marked as needing live sandbox credentials — deferred to deployment, not blocked). The agent change's copilot path is completed and verified; autopilot tasks are explicitly marked deferred to Q2 in that change's `tasks.md` (not silently dropped).

Rationale: these are governance corrections that unblock the apply workflow; rewriting the feature code is out of scope and would duplicate the owning changes.

### D6: Launch gate runs a fixed command sequence

The single gate, in order: `make sqlc` → `go build ./...` → `go vet ./...` → `make test` (backend); `pnpm lint` → `tsc --noEmit` → `pnpm build` (frontend); `pnpm exec playwright test` (E2E, against test infra). Results recorded in `tasks.md`. The stale backend CI image is bumped to the go.mod version and a frontend + E2E CI job is added.

## Risks / Trade-offs

- **Playbooks proposal text vs migration file drift** → A validation task in this change greps every active change's proposal/design for migration claims and compares against on-disk migrations; the playbooks reference is corrected and the check is recorded.
- **Vercel AI SDK protocol mismatch with thin SSE framing** → The D4 fallback (native fetch-stream) is designed in from the start; the tasks land the fallback first, then the SDK consumer, so either can ship.
- **Streamed response + credit guard ordering** → The metered client records tokens only after a successful call (verified behavior); a mid-stream failure records nothing and returns an SSE error event. Accepted: an exhausted guard rejects before the stream starts (402), never mid-stream.
- **In-flight changes still in progress (agent 0/30, playbooks 0/21)** → Their code exists in the tree; this change's completion tasks are scoped to verify-and-correct, and any task that surfaces a genuine gap becomes a blocking item recorded in this change's `tasks.md`.
- **Live billing/E2E credential dependencies** → MercadoPago/Polar sandbox verification and live-webhook re-pointing are explicitly deferred to deployment (recorded as such in the owning changes), not made blockers for the code-level gate.
- **No Stytch state changes** → no rollback surface on the Stytch side; rollback is Git revert plus (none needed) DB down-migrations since no migrations are added by this change.

## Migration Plan

1. Land in-flight change completions in dependency order: eslint → sellable-modules → agent (copilot path) → playbooks → mercadopago/wire → e2e.
2. Add streaming backend handler + `RAGService.ChatStream` + `AssistantProvider.GenerateResponseStream`.
3. Add frontend streaming consumer; wire into `chat-interface.tsx`.
4. Add CI job(s) and bump the backend CI image.
5. Run the D6 launch gate; record results.
6. Write `ROADMAP.md`.
7. Rollback: Git revert; no DB down-migrations required (no schema changes).

## Open Questions

- Vercel AI SDK version pinning: confirmed available in the pnpm store? (The eslint change already proved `eslint-config-next@16` + `typescript-eslint@8.49` resolve; `@ai-sdk/react` availability to be confirmed in task 5.1 — fallback covers it.)
- Whether autopilot's deferred tasks stay in `add-agentic-whatsapp-assistant/tasks.md` as deferred or move to a new Q2 change — decision recorded in `ROADMAP.md` and the agent change's tasks notes.
