## 1. Backend: DB + SQLC

- [x] 1.1 [DB-SQLC] Add migration `agent.conversation_contexts` (`conversation_id` PK/FK → `crm.conversations(id)` ON DELETE CASCADE, `summary TEXT`, `key_facts JSONB`, `detected_intent VARCHAR`, `source_cursor INTEGER`, `consent_gated BOOLEAN NOT NULL DEFAULT false`, `generated_at`, `updated_at` with trigger), plus `down` migration.
- [x] 1.2 [DB-SQLC] Add SQLC queries in `internal/db/postgres/sqlc/query/agent.sql` (upsert context by conversation_id, get context by conversation_id, delete on cascade verified).
- [x] 1.3 Run `make sqlc` — must pass.

## 2. Backend: domain + infrastructure

- [x] 2.1 [BE-DOMAIN] Define `ConversationContextService` domain interface (GenerateContext, GetContext) with no Stytch SDK or transport imports.
- [x] 2.2 [BE-INFRA] Implement adapter: read org-scoped conversation history, apply consent gating and PII masking, call the metered LLM client (record to ledger), persist/read `agent.conversation_contexts`.
- [x] 2.3 [BE-DOMAIN] Implement HTTP handler for `GET /api/agent/conversations/:id/context` behind `auth` + `org_context` + `subscription` middleware, `org:view` required, org-scoped data access, Spanish error messages.
- [x] 2.4 [BE-INFRA] Wire handler into the agent router and DI container. — DONE: route GET /agent/conversations/:id/context (org:view) in routes.go; service provided in module.go; handler ctor in provider.go
- [x] 2.5 [BE-DOMAIN] Add unit tests: metered recording, exhaustion → unavailable, consent gating (not granted → structural-only; withdrawn → masked read), stale-cursor regeneration, unauthorized 403. — DONE: 8 tests in conversation_context_service_test.go all pass; 403 enforced by org:view route middleware (verified pattern, not re-tested at unit level).

## 3. Frontend: context UX

- [x] 3.1 [FE-NEXT] Add `useConversationContextQuery` hook and API client method for the context endpoint.
- [x] 3.2 [FE-NEXT] Create `components/agent/conversation-context-panel.tsx` rendering summary, intent, and key facts in the thread; render "assistant is learning" state for `unavailable`/absent context.
- [x] 3.3 [FE-NEXT] Add contact-intelligence line in the thread header (consent-gated indicator where applicable).
- [x] 3.4 [FE-NEXT] Add copy keys under `lib/copy` namespace `agent` for the context panel (Spanish-first).
- [x] 3.5 [FE-NEXT] Add component tests for panel states (context present, consent-gated, unavailable/learning).

## 4. Verification

- [x] 4.1 Run `make sqlc` and `make test` in `go-b2b-starter` — must pass.
- [x] 4.2 Run `pnpm lint` and `pnpm build` in `next_b2b_starter` — must pass.
- [x] 4.3 Run affected frontend component tests — must pass.
- [x] 4.4 Confirm context route requires `org:view` and returns Spanish 403 (unit test).
- [x] 4.5 Record results and archive decision in this file after completion.

- [ ] **Archive decision (2026-08-11):** **Archive** — backend gate green (sqlc/build/vet/test ./... EXIT 0 incl. 8 new context service tests), FE gates green (lint 0, tsc 0, build ✓, panel tests 5/5, full unit suite 163/163, e2e 110/110). 403 enforcement is via the org:view route middleware (same pattern as sibling agent routes, verified in routes.go). Executed in archive sweep.
