## 1. Database Schema & SQLC — migration `000019` `agent` schema + consent [DB-SQLC]

- [x] 1.1 Create `000019_create_agent_schema.up.sql`/`.down.sql`: `agent.conversation_flows` (linear pipeline status), `agent.agent_settings` (mode/tone/brand_voice/autopilot_start/autopilot_end TIME/timezone/kill_switch/max_daily_messages/consent_required/consent_template/guardrails JSONB), `agent.agent_suggestions` (type/status/source/approved_by_member_id), `agent.agent_actions` (append-only, decisions allow/deny/skip); `crm.contacts` gains `consent_status` + `consented_at` with CHECK; updated_at triggers; indexes; down drops schema + consent columns. Verify: `make migrateup && make migratedown && make migrateup` plus psql `\d agent.*`
- [x] 1.2 Add SQLC queries to `query/agent.sql`: flow create/get/get-active-by-conversation/update-status; settings get/upsert; suggestions insert/list-by-status/get/approve/reject/get-pending-by-message/supersede-pending; actions insert; sent-today count; consent update; anonymize; conversations-by-contact; messages-by-conversation. Verify: `make sqlc` regenerates code without errors
- [x] 1.3 Verify generated `gen/agent.sql.go` compiles and mappings (pgtype Time/Timestamptz/JSONB) are handled by repository helpers. Verify: `make sqlc && go build ./...`

## 2. Agent Domain — entities, guardrails, repository contract [BE-DOMAIN]

- [x] 2.1 Create `internal/modules/agent/domain/` entities: `Mode` (copilot/autopilot), `Tone`, `FlowStatus`, `SuggestionType/Status/Source`, `ConsentStatus`, `AgentDecision`, `Guardrails` (NeverRules/EscalateRules + defaults), `AgentSettings` (+ `DefaultSettings`), `ConversationFlow`, `Suggestion`, `AgentAction`, `ContactFacts`, `GuardrailInput`, `GuardrailDecision` — no external package imports. Verify: `go build ./internal/modules/agent/...`
- [x] 2.2 Define `domain/guardrail.go` `GuardrailService` (Evaluate with fail-safe deny) and `domain/repository.go` `AgentRepository` (flows/settings/suggestions/audit/usage/contact-conversation resolution/consent/anonymize). Verify: `go vet ./internal/modules/agent/...`
- [x] 2.3 Implement `infra/guardrails/guardrail_service.go`: kill switch, discount cap, forbidden terms, escalation terms, consent, window (overnight + timezone), daily limit; escalate/draft always allowed; error → `guardrail_error` deny. Unit tests covering the rule table. Verify: `go test ./internal/modules/agent/...`
- [x] 2.4 Implement `infra/repositories/agent_repository.go` on `sqlc.Store` with pgtype/JSONB mappings and idempotent contact/conversation resolution. Verify: `go build ./...`

## 3. Agent Pipeline — ingestion, consent, analysis, approve [BE-DOMAIN]

- [x] 3.1 Implement `app/services/agent_service.go` `HandleMessageReceived`: skip non-text, dedupe by pending suggestion on `whatsapp_message_id`, resolve contact/conversation, settings + kill-switch cancel, consent state machine (none → template + requested; requested + affirmative → granted), metered LLM analysis (credit-gated via `BillingService.GetAiUsageStatus`, PII facts masked), then copilot suggestion or autopilot guarded send with `autopilot_fallback`. Verify: `go test ./internal/modules/agent/...`
- [x] 3.2 Implement approval flow: `ApproveSuggestion` (edited_body support, guardrail eval with autonomous=false, audit, deny → reject + `DenialError`, allow → outbound send + approve + flow succeeded), `RejectSuggestion` (audit `human_rejection`). Verify: `go test ./internal/modules/agent/...`
- [x] 3.3 Implement `app/services/compliance_service.go` (`ExportContact` with withdrawn-consent PII masking, `ForgetContact` idempotent anonymization) and service interfaces/types (`AgentService`, `ComplianceService`, `FlowDebug`, export bundle types). Verify: `go build ./...`
- [x] 3.4 Unit tests with fakes for repository/guardrails/llm/outbound covering: copilot pending + supersede, autopilot allow/deny/fallback/escalation-match, consent grant, credit exhaustion escalation, approval deny/allow. Verify: `go test ./internal/modules/agent/...`

## 4. API — handler, routes, DI wiring [BE-DOMAIN] [BE-INFRA]

- [x] 4.1 Create `internal/modules/agent/handler.go` + `routes.go`: `GET /api/agent/suggestions`, `POST /api/agent/suggestions/:id/approve|reject`, `GET/PUT /api/agent/settings`, `GET /api/agent/flows/:conversationId`, `GET /api/agent/compliance/export/:contactId`, `POST /api/agent/compliance/forget/:contactId`; middleware chain `auth` → `org_context` → `subscription` → `RequirePermissionFunc("org","manage")` (writes) / `org:view` (reads); Spanish error messages; 409 on guardrail denial. Verify: `go build ./...`
- [x] 4.2 Wire module/provider/cmd in `internal/modules/agent/`, register the agent repository in `internal/db/inject.go`, register agent routes in `internal/api/provider.go`, and fix the pre-existing missing WhatsApp/CRM module initialization in `internal/bootstrap/init_mods.go` (webhook → eventbus → CRM + agent listeners now reachable). Verify: `go build ./... && go test ./internal/modules/agent/...`

## 5. Frontend [FE-NEXT]

- [ ] 5.1 Add API client + server actions for suggestions (list/approve/reject) and settings (get/put). Verify: `pnpm lint && pnpm build`
- [ ] 5.2 Render pending `reply` suggestions in `app/dashboard/inbox/components/message-thread.tsx` as draft chips with Approve / Edit (prefills reply input) / Reject; optimistic update. Verify: `pnpm lint && pnpm build`
- [ ] 5.3 Add pending-count badge to `conversation-list.tsx` fed by the suggestions list. Verify: `pnpm lint && pnpm build`
- [ ] 5.4 Create `agent-settings-section.tsx` under `settings/components/`: mode toggle (copilot/autopilot), tone, brand voice, autopilot window + timezone, kill switch, daily limit, consent template, guardrails (discount cap, forbidden/escalate terms); mount beside `whatsapp-config-section.tsx`. Verify: `pnpm lint && pnpm build`

## 6. Verification & CI [OPS-GOV]

- [ ] 6.1 Ensure guardrail + pipeline unit tests run in CI (standard `go test ./...`); confirm no OPA/Rego artifacts remain (`opa` removed from go.mod, `internal/modules/agent/policies` and `infra/governance` deleted). Verify: `grep -r "open-policy" go.mod || echo clean`
- [ ] 6.2 Full verification pass: `make sqlc`, `go build ./...`, `go vet ./...`, `go test ./...`, `pnpm lint`, `pnpm build` all green. Verify: run each command, record results in this change's verification log
