# Tasks: add-ticket-ai-triage

## 1. Backend: triage service + endpoint [BE-DOMAIN]

- [x] 1.1 Create `internal/modules/tickets/app/services/ai_triage_service.go`: `AITriageService` with `NewAITriageService(llm llmdomain.LLMClient, billing billingServices.BillingService, repo domain.TicketRepository)` and `Triage(ctx, orgID, ticketID int32) (*TriageResult, error)` following the `ai_audience_builder.go` pattern (design D2/D3/D4): credit gate via `billing.GetAiUsageStatus` (ledger error → warn + fail-open; `CreditsMax > 0 && remaining <= 0` → `ErrAiCreditsExhausted`), org-scoped ticket load (missing/foreign org → `ErrTicketNotFound`), `llmdomain.WithOrgID(ctx, orgID)`, metered `llm.Complete` with a fixed Spanish triage system prompt (title + description), fence-tolerant JSON parse `{"note","priority"}`, priority validated through `domain.TicketPriority(...).IsValid()` (Spanish aliases alta/media/baja normalized first; invalid/missing → `priority: null`, note still returned). `TriageResult`, `ErrAiCreditsExhausted`, and `triageSystemPrompt` defined. Verify: `go build ./...` — PASS.
- [x] 1.2 Add handler `AiTriage` (bind `:id`, invalid id → 400 `invalid_id`; 200 envelope `{"data":{"note":...,"priority":...},"success":true}`; 402 `ai_credits_exhausted`; 404 `ticket_not_found`; 500 `ai_triage_failed`) and register `POST /:id/ai-triage` in `internal/modules/tickets/routes.go` inside the existing group with `auth.RequirePermissionFunc("ticket", "view")`. Verify: `go build ./...` — PASS; `grep -n "ai-triage" internal/modules/tickets/routes.go` → line 41 `ticketsGroup.POST("/:id/ai-triage", auth.RequirePermissionFunc("ticket", "view"), r.handler.AiTriage)`.
- [x] 1.3 Wire DI in `internal/modules/tickets/provider.go`: provide `*AITriageService` (llm + billing deps from the container) and extend the handler construction (`NewHandler(ticketService, aiTriageService)`) without changing `TicketService`. Verify: `go build ./...` — PASS; `go vet ./...` — PASS.
- [x] 1.4 Unit tests (`ai_triage_service_test.go` + `handler_ai_triage_test.go`): happy path returns note + valid priority (recording mock LLM, metered org-id context asserted); fenced JSON; invalid model priority → `priority: null` + note; missing priority → null; exhausted credits → `ErrAiCreditsExhausted` and NO LLM call; ledger failure → fail-open proceeds; unlimited credits proceed; missing/foreign-org ticket → not-found and NO LLM call; LLM failure wrapped. Handler tests: 401 (unauthenticated), 404 (missing + foreign org), 400 (invalid id), 402 (ai_credits_exhausted code in body), 200 envelope (note + `"high"`), 200 with `priority: null`, 500 (LLM failure). Verify: `go test ./internal/modules/tickets/...` — all pass (7 handler + 10 service + 7 pre-existing service tests, 24 total).

## 2. Frontend: triage action in ticket detail [FE-NEXT]

- [x] 2.1 Add `aiTriage(ticketId)` to `lib/api/api/repositories/ticket-repository.ts` (`AiTriageDto` + `AiTriage` model via `Wrapped<T>` envelope → `{note, priority}`) and a `useAiTriageMutation` hook in `lib/hooks/mutations/use-tickets-mutations.ts` (402 `ai_credits_exhausted` → credits toast, other failures → generic toast; no invalidation since triage never mutates). Verify: `npx tsc --noEmit` — PASS.
- [x] 2.2 Add the "✨ Redactar nota" action to `components/tickets/ticket-detail.tsx` beside the note input (disabled while in-flight, "Generando…" loading label): success fills the note draft and renders a dismissible priority-suggestion chip ("Sugerencia de prioridad: alta/media/baja" + "Aplicar" → existing `useSetTicketPriority` mutation); failure keeps the form untouched (toast handled by the hook). Verify: `pnpm lint` — 0 errors (4 pre-existing warnings); `npx tsc --noEmit` — PASS.
- [x] 2.3 Add `ui.tickets` copy keys in `lib/copy/ui.ts` (+ `en` mirror): `triageDraft`, `triagePrioritySuggestion` (template `{priority}`), `triageApply`, `triageError`, `triageCreditsExhausted`; namespace inserted before `palette` in both `ui` and `en`. Verify: `pnpm lint` — PASS; keys referenced from the component (`ui.tickets.*`) and hook (`copy("tickets", ...)`).
- [x] 2.4 Component tests (`components/tickets/ticket-detail.test.tsx`, new, 7 tests): clicking Redactar nota calls the mutation (`aiTriage(1)`) and fills the note input; priority chip renders and Apply calls `setPriority(1, "high")` then dismisses; dismiss button does not apply; failure (402 + generic) keeps values and shows the correct toast; nothing is saved automatically (no addInternalNote/setPriority/transition); in-flight loading state disables the button. Verify: `pnpm exec vitest run components/tickets/ticket-detail.test.tsx` — 7/7 pass.

## 3. Verification gate [OPS-GOV]

- [x] 3.1 Run backend gate: `go build ./... && go vet ./... && go test ./internal/modules/tickets/...`. Verify: all exit 0; record results here.
  - `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./internal/modules/tickets/...` exit 0 — packages `tickets` (7 handler tests) and `tickets/app/services` (17 tests) pass; `domain`/`infra/repositories` have no test files. All 24 tests PASS. Note: during this change, the full-repo build briefly failed on `internal/modules/organizations` (SessionHardeningFixer's in-flight session-revocation change); after they landed it, the full build is green — no tickets code was implicated.
- [x] 3.2 Run frontend gate: `pnpm lint` (0 errors, pre-existing warnings acceptable) and `npx tsc --noEmit`. Verify: both pass; record results here.
  - `pnpm lint` exit 0 — 0 errors, 4 warnings (baseline: contact-table.tsx, deal-kanban.tsx, prose.tsx; all pre-existing, none in changed files). `npx tsc --noEmit` exit 0.
- [x] 3.3 Run affected component tests: `pnpm exec vitest run components/tickets/ticket-detail.test.tsx`. Verify: passes; record results.
  - 1 file, 7 tests, all PASS (duration ~3.1s).
- [x] 3.4 Record results and archive decision (`/opsx-archive` or `**Archive deferred:** <reason>`) in this file. Verify: entry present.

**Archive deferred:** sibling Wave A/B changes (add-sellable-modules still in flight; add-inbox-ai-writing-assist / add-campaign-ai-message / session-lifetime-hardening are being implemented in parallel right now) touch adjacent surfaces (tickets module ownership, agent rephrase, copy/ui.ts namespaces). Archiving now would freeze delta specs while siblings are mid-flight. Defer archive until the orchestrator's centralized Phase 5 gate (single `pnpm build`) passes and Wave A/B changes settle; the delta spec is fully implemented and all local gates are green.

**pnpm build deferred to centralized Phase 5 gate:** shared `.next` build lock contended twice (initial attempt + retry after 90s wait) — another Wave B agent holds `pnpm build`. Recorded per orchestration rules; `npx tsc --noEmit` + `pnpm lint` + targeted vitest (the primary correctness gates) all pass. Build verification left to the centralized gate.

## Changed files

- go-b2b-starter/internal/modules/tickets/app/services/ai_triage_service.go (new)
- go-b2b-starter/internal/modules/tickets/app/services/ai_triage_service_test.go (new)
- go-b2b-starter/internal/modules/tickets/handler.go (AiTriage handler + service field)
- go-b2b-starter/internal/modules/tickets/handler_ai_triage_test.go (new)
- go-b2b-starter/internal/modules/tickets/routes.go (POST /:id/ai-triage)
- go-b2b-starter/internal/modules/tickets/provider.go (AITriageService DI)
- next_b2b_starter/lib/api/api/repositories/ticket-repository.ts (aiTriage + DTO/model)
- next_b2b_starter/lib/hooks/mutations/use-tickets-mutations.ts (useAiTriageMutation)
- next_b2b_starter/lib/copy/ui.ts (ui.tickets + en mirror)
- next_b2b_starter/components/tickets/ticket-detail.tsx (Redactar nota + suggestion chip)
- next_b2b_starter/components/tickets/ticket-detail.test.tsx (new)
