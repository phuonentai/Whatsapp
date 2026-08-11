## 1. Backend: DB + SQLC

- [ ] 1.1 [DB-SQLC] Add migration `agent.conversation_contexts` (`conversation_id` PK/FK → `crm.conversations(id)` ON DELETE CASCADE, `summary TEXT`, `key_facts JSONB`, `detected_intent VARCHAR`, `source_cursor INTEGER`, `consent_gated BOOLEAN NOT NULL DEFAULT false`, `generated_at`, `updated_at` with trigger), plus `down` migration.
- [ ] 1.2 [DB-SQLC] Add SQLC queries in `internal/db/postgres/sqlc/query/agent.sql` (upsert context by conversation_id, get context by conversation_id, delete on cascade verified).
- [ ] 1.3 Run `make sqlc` — must pass.

## 2. Backend: domain + infrastructure

- [ ] 2.1 [BE-DOMAIN] Define `ConversationContextService` domain interface (GenerateContext, GetContext) with no Stytch SDK or transport imports.
- [ ] 2.2 [BE-INFRA] Implement adapter: read org-scoped conversation history, apply consent gating and PII masking, call the metered LLM client (record to ledger), persist/read `agent.conversation_contexts`.
- [ ] 2.3 [BE-DOMAIN] Implement HTTP handler for `GET /api/agent/conversations/:id/context` behind `auth` + `org_context` + `subscription` middleware, `org:view` required, org-scoped data access, Spanish error messages.
- [ ] 2.4 [BE-INFRA] Wire handler into the agent router and DI container.
- [ ] 2.5 [BE-DOMAIN] Add unit tests: metered recording, exhaustion → `unavailable`, consent gating (not granted → structural-only; withdrawn → masked read), stale-cursor regeneration, unauthorized 403.

## 3. Frontend: context UX

- [ ] 3.1 [FE-NEXT] Add `useConversationContextQuery` hook and API client method for the context endpoint.
- [ ] 3.2 [FE-NEXT] Create `components/agent/conversation-context-panel.tsx` rendering summary, intent, and key facts in the thread; render "assistant is learning" state for `unavailable`/absent context.
- [ ] 3.3 [FE-NEXT] Add contact-intelligence line in the thread header (consent-gated indicator where applicable).
- [ ] 3.4 [FE-NEXT] Add copy keys under `lib/copy` namespace `agent` for the context panel (Spanish-first).
- [ ] 3.5 [FE-NEXT] Add component tests for panel states (context present, consent-gated, unavailable/learning).

## 4. Verification

- [ ] 4.1 Run `make sqlc` and `make test` in `go-b2b-starter` — must pass.
- [ ] 4.2 Run `pnpm lint` and `pnpm build` in `next_b2b_starter` — must pass.
- [ ] 4.3 Run affected frontend component tests — must pass.
- [ ] 4.4 Confirm context route requires `org:view` and returns Spanish 403 (unit test).
- [ ] 4.5 Record results and archive decision in this file after completion.
