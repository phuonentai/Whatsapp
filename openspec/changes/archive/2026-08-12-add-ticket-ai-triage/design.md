# Design: add-ticket-ai-triage

## Context

The tickets module (shipped by the in-flight `add-sellable-modules` change) has a full CRUD surface (`routes.go`: list/get/events/create/transition/assign/priority/tags/internal-note, `ticket:view`/`ticket:manage`), a state machine with valid transitions, and a priority model (`domain/ticket.go`: `TicketPriority` with `DefaultPriorities = [low, normal, high]`, `IsValid()`, `DefaultSLASeconds`). The UI (`components/tickets/ticket-detail.tsx`) renders the events timeline and a bare note input + "Agregar" button.

The metered-LLM pattern is established twice in the repo: `agent_service.go:177` `analyze()` (credit gate fail-open on ledger error, skip on exhausted, `WithOrgID`, metered `Complete`) and `ai_audience_builder.go:53` `Build()` (credit check, `WithOrgID`, metered `Complete`, JSON parse + validation). Tickets currently has neither `llmdomain.LLMClient` nor `billingServices.BillingService` in its DI (`provider.go` wires only `repo + moduleService → TicketService → Handler → Routes`).

Verified facts (premise validation, 2026-08-11):
- `tickets/routes.go` — `/tickets` group behind `auth`, `org_context`, `EntitlementMiddleware`, `registry.Require("tickets")`; per-route `ticket:view`/`ticket:manage`.
- `tickets/domain/ticket.go` — `TicketPriority` + `IsValid()`, `DefaultPriorities` fallback, SLA map.
- `tickets/provider.go` — DI does not yet provide LLM/billing deps.
- `ai_audience_builder.go` — constructor-injected `LLMClient` + `BillingService`, credit gate, `WithOrgID`, `parseFilterSpecJSON` pattern.
- `ticket-detail.tsx` — note input + Agregar button confirmed; existing `SetPriority` mutation available for the Apply action.
- FE repository envelope convention `{"success": bool, "data": T}`.

## Goals / Non-Goals

**Goals:**
- One triage endpoint producing a note draft + validated priority suggestion from the stored ticket.
- Composer UX: one click fills the note draft and shows a priority chip with explicit Apply.
- Parity with existing AI governance: metered, org-scoped, credit-gated, Spanish errors, no mutation.

**Non-Goals:**
- No auto-save of notes or auto-apply of priority.
- No state-machine / SLA / module-config changes.
- No new LLM/model/metering, no schema changes.

## Decisions

### D1: Server-side ticket fetch — `POST /api/tickets/:id/ai-triage`, not a client-passed description

The endpoint loads the ticket by id + org from the repository and uses its stored title + description. Rationale: the server controls the input (no client-supplied text beyond the id, narrower injection surface), works even if the UI copy differs, and keeps the triage consistent with what the agent actually sees. Alternative (client passes `description`) rejected: duplicates data, widens the prompt surface, and the FE may not hold the full description.

### D2: New `AITriageService`, `TicketService` untouched

New file `app/services/ai_triage_service.go` with `NewAITriageService(llm, billing, repo)` — mirrors `aiAudienceBuilder`'s constructor injection. `TicketService` keeps its transactional responsibilities. DI (`provider.go`) gains two provides: the service and the handler method wiring (or handler param). No signature churn on existing types.

### D3: Credit semantics — follow `analyze()` (fail-open on ledger error, 402 on exhausted)

`GetAiUsageStatus` error → warn + proceed (fail-open); `CreditsMax > 0 && remaining <= 0` → `ErrAiCreditsExhausted` → 402 `ai_credits_exhausted`. Note: `ai_audience_builder.go` is fail-closed on ledger error — this change deliberately follows the *agent* semantics (more recent, and triage is an assist, not a send); the discrepancy is documented here, not silently mixed.

### D4: Prompt + JSON contract, resilient priority

Prompt: fixed Spanish system prompt (role: asistente de triage de tickets) + ticket title/description, `WithOrgID` for metering, `MaxTokens` from platform defaults. Response parsed as JSON `{"note": string, "priority": string}` (fence-tolerant parse copied from the audience builder's pattern); `priority` mapped through `domain.TicketPriority(...).IsValid()` — invalid or missing → `priority: null`, note still returned. LLM failure → 500 wrapped (FE shows generic toast, form untouched). This keeps a bad model answer from blocking the draft.

### D5: Routing + permissions

Route in the existing `/tickets` group (already has auth/org/entitlement/module middleware): `ticketsGroup.POST("/:id/ai-triage", auth.RequirePermissionFunc("ticket", "view"), r.handler.AiTriage)`. Drafting is view-level; note creation / priority changes remain `ticket:manage`. 404 for missing/foreign-org ticket (org-scoped repository lookup).

### D6: Frontend — one-click draft + explicit Apply

`ticket-detail.tsx`: "✨ Redactar nota" button beside Agregar (disabled while in-flight or without a ticket). Success → `setNote(result.note)` + priority chip ("Sugerencia de prioridad: alta" + "Aplicar" → existing `SetPriority` mutation; dismissible). Failure/402 → toast (`ui.tickets.triageError` / credits message), form untouched. Hook `useAiTriageMutation` + `ticketRepository.aiTriage(id)` (envelope DTO).

### D7: Copy under new `ui.tickets` namespace

Keys: `triageDraft`, `triagePrioritySuggestion`, `triageApply`, `triageError`, `triageCreditsExhausted` (+ `en` mirror) — Spanish-first, consistent with the copy-sweep enforcement (`SWEPT_FILES`).

## Risks / Trade-offs

- **Cost**: each triage is a metered LLM call; user-initiated + credit-gated (same economics as suggestions).
- **Fail-open vs fail-closed divergence**: documented in D3; choosing agent-style fail-open deliberately.
- **Model reliability on priority**: mitigated by D4 validation + drop (never fails the call, never applies).
- **Prompt injection**: ticket description is user-authored content; the system prompt is fixed and output is never auto-applied — blast radius is a draft + suggestion only.
- **Module interplay**: tickets module is owned by the in-flight `add-sellable-modules`; this change is additive to its shipped surface and touches no shared files except via DI additions in `provider.go` (isolated provides).
