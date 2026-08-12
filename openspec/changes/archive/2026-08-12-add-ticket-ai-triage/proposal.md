# Proposal: add-ticket-ai-triage

## Why

Tickets have SLA/priority fields but zero AI: support agents hand-type the first internal note from the ticket description and pick a priority themselves. A 2026 AI-first product drafts the note and suggests the priority automatically (Salesforce Agentforce "summarizes records, drafts emails, logs activities" as a default). This change adds AI first-touch triage to the tickets module, reusing the existing metered, credit-gated LLM pipeline (same pattern as `ai_audience_builder.go`).

## What Changes

- **Backend**: new `POST /api/tickets/:id/ai-triage` endpoint in the tickets module (`internal/modules/tickets/routes.go`). Reads the stored ticket (title + description), runs a metered LLM call (org-scoped, credit-gated), returns `{"data": {"note": "<draft>", "priority": "alta" | null}, "success": true}`. No mutation — the note is a draft, the priority a suggestion.
- **New service**: `AITriageService` in `internal/modules/tickets/app/services/` following the `ai_audience_builder.go` pattern (`llmdomain.LLMClient` + `billingServices.BillingService` injected); `TicketService` unchanged. Priority output is validated against the ticket domain's valid set (`domain.IsValid`); an invalid model priority is dropped (note returned alone) rather than failing the call.
- **Frontend**: `components/tickets/ticket-detail.tsx` — "✨ Redactar nota" action beside the note input; success fills the note draft and shows a priority-suggestion chip ("Sugerencia: alta") with an "Aplicar" action that reuses the existing `SetPriority` mutation. Failure leaves the form untouched with a Spanish toast.
- **Copy**: `ui.tickets` keys (Spanish-first + `en` mirror).
- **DI**: tickets provider gains the two platform deps (`llmdomain.LLMClient`, billing service) for the new service only.
- **No persistence, no auth-flow, no Stytch policy changes.**

## Capabilities

### New Capabilities

- `ticket-ai-triage`: AI first-touch triage for tickets — draft an internal note and suggest a priority from the ticket description; authenticated, org-scoped, credit-gated, metered, never mutates the ticket.

### Modified Capabilities

None.

## Impact

- **Code**: `go-b2b-starter/internal/modules/tickets/` — new `app/services/ai_triage_service.go`, `handler.go` (+1), `routes.go` (+1 route, `ticket:view`), `provider.go` (DI), new tests. `next_b2b_starter/` — `components/tickets/ticket-detail.tsx`, `lib/api/api/repositories/ticket-repository.ts` (+`aiTriage`), DTO/model, hook, `lib/copy/ui.ts` (`ui.tickets`), component test.
- **Dependencies**: none new — reuses `internal/platform/llm` and billing credits.
- **Systems**: AI credit metering per call (same economics as suggestions/audience builder).

## Non-Goals

- No auto-creation of notes or auto-application of priority — output is always user-reviewed (note is a draft, priority is a suggestion with an explicit Apply).
- No changes to the ticket state machine, SLA computation, or module config.
- No new LLM provider/model/metering; no schema or persistence changes.
- No local credential storage; no credentials involved.
- No changes to the in-flight `add-sellable-modules` change (tickets module ownership stays there; this builds on its shipped surface).

## Rollback

- **Git state**: revert the touched files (`ai_triage_service.go`, `handler.go`, `routes.go`, `provider.go`, new tests, `ticket-detail.tsx`, `ticket-repository.ts`, DTO/model/hook, `lib/copy/ui.ts`, component test, this change's artifacts). All additions are additive; no migration, no data.
- **Stytch tenant policy state**: no policy changes, so no policy rollback required.
