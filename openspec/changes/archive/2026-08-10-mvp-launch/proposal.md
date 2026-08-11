## Why

The product story — a Colombian SME connects WhatsApp, gets an out-of-box CRM for its vertical, an AI copilot that drafts replies (a human approves), Ley 1581 compliance, and PSE/Nequi billing — is almost entirely built across six in-flight OpenSpec changes, but none are verified, and the AI experience is single-turn and non-streaming (`handler.go` Chat returns one JSON body; `CompleteStream` exists in `internal/platform/llm` but no handler calls it). The platform cannot launch: verification gates are stalled by broken frontend linting, the largest feature changes sit at 0% task completion despite working code, and the build must be proven green end-to-end. This change completes and verifies the in-flight work, adds end-to-end SSE streaming (the one missing MVP feature), fixes a stale migration reference in the playbooks change, defines the launch gate, and captures the post-MVP roadmap.

## What Changes

- **Completion of six in-flight changes**: `fix-frontend-eslint-flat-config`, `add-sellable-modules`, `add-agentic-whatsapp-assistant` (copilot path only), `add-vertical-playbooks`, `add-mercadopago-billing`, `wire-mercadopago-billing`, and `add-crm-e2e-tests` — driven across their verification gates (their `tasks.md` checkboxes are corrected where they do not match reality, e.g. the playbooks migration number).
- **Reconcile playbooks migration reference**: `add-vertical-playbooks` proposal/design claim migration `000019_create_playbooks`, but the actual file is `000020_create_playbooks` (000019 is the agent schema). The stale references are corrected and a validation task ensures every change's migration claim matches the on-disk migration.
- **New capability `cognitive-streaming`**: end-to-end SSE streaming over the existing `POST /example_cognitive/chat` route — backend streams tokens via the existing `CompleteStream` seam (through the metered client so usage recording still applies), frontend consumes via the Vercel AI SDK `useChat` pattern replacing the current mutation-based `use-chat.ts`.
- **Launch gate**: define and execute a single verification gate across backend (`make sqlc`, `go build ./...`, `go vet ./...`, `make test`), frontend (`pnpm lint`, `pnpm build`, `tsc --noEmit`), and E2E (`pnpm exec playwright test`), plus a CI matrix task (backend CI exists but is stale at `golang:1.22.4`; frontend/E2E CI does not exist).
- **Roadmap artifact**: write `ROADMAP.md` at the repo root capturing the post-MVP roadmap (Q2 autopilot + LLM provider router + AI cost dashboards; Q3 agent tool-use runtime + CRM AI; Q4 AI-native UX + scale hygiene), each future phase to become its own OpenSpec change when started.
- **Deferred (explicitly out of MVP)**: autopilot autonomous sending, multi-model routing, prompt registry, semantic cache, evals, CRM business AI, copilot panel / command palette, OTel tracing, agent tool-use runtime. These appear in `ROADMAP.md` only.

## Capabilities

### New Capabilities
- `cognitive-streaming`: end-to-end SSE streaming for the AI chat endpoint — streaming transport contract, metered-token integrity during streaming, credit-guard behavior on streamed responses, and frontend streaming consumption.

### Modified Capabilities
- `governance-workflow`: requirement changes — a launch change (this one) SHALL reconcile in-flight changes' task state and migration-number claims against the codebase before declaring completion; the launch verification gate SHALL run backend + frontend + E2E commands in a defined order and record results in `tasks.md`; frontend CI SHALL be present before launch.

## Impact

- **Go backend**: `internal/modules/cognitive/handler.go` (SSE handler), `internal/modules/cognitive/app/services/rag_service.go` (streaming path via `CompleteStream`), DI wiring; no schema changes (chat persistence tables already exist from `000009`). Existing routes, middleware chain (`auth`, `org_context`, `subscription`, `ai_assistant` flag, credit guard) unchanged in ordering.
- **Frontend**: `lib/hooks/mutations/use-chat.ts` → streaming consumer (Vercel AI SDK or native `EventSource`/fetch-stream), `app/dashboard/knowledge/components/chat-interface.tsx` + `chat-message.tsx` token rendering.
- **Database**: no new migrations from this change. Migration `000019` (agent) and `000020` (playbooks) already exist; the playbooks *proposal/design* text is corrected to `000020`, not the migration file itself.
- **CI**: `go-b2b-starter/.gitlab-ci.yml` updated (Go image aligned to go.mod); new frontend + E2E CI job.
- **Tooling/docs**: `ROADMAP.md` added at repo root.
- **Auth boundary**: no Stytch contract changes; this change introduces no new permissions, no session handling changes, and no local credential storage. Existing RBAC scopes for agent/compliance endpoints (from `add-agentic-whatsapp-assistant`) are unchanged.
- **Rollback strategy**: Git — revert the change commit(s) and the reconciled `tasks.md`/proposal text edits; DB — none needed (no migrations added); Stytch tenant policy state — unaffected (no RBAC/auth changes), so no Stytch-side rollback is required.
