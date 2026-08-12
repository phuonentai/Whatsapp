# Design: add-campaign-ai-message

## Context

The campaigns module (v1) creates drafts as `nombre + segment_id` (`domain/entities.go` — `Campaign` has no message field; launch snapshots recipients and "the scheduler consumes recipients later"). The AI builder (`ai_audience_builder.go`) converts a description into a validated `filter_spec` + preview via one metered LLM call; `whatsapp-campaigns` spec requires metering, no-PII prompts, whitelist validation, 402 on exhausted credits, and "nothing persisted until the user explicitly saves".

Verified facts (premise validation, 2026-08-11):
- `routes.go` — `POST /segments/ai-build` (`org:manage`), `POST ""` CreateCampaign (`org:manage`, body `{nombre, segment_id}`), `POST /:id/launch`.
- `handler.go:151-173` — AiBuild binds `{descripcion}`, returns the result envelope, 402 on `ErrAiCreditsExhausted`.
- `domain/entities.go` — `AudienceBuildResult{FilterSpec, Preview}`; `Campaign` has no message field.
- `campaign_repository.go:21` — `Create` maps `nombre`, `segment_id`, `created_by` into `sqlc.CreateCampaignParams`.
- FE `campaign-manager.tsx` — create form is nombre input + segment select only; `createCampaign.mutate({nombre, segment_id})`.
- Migrations head: `000034` — next free is `000035`.
- `parseFilterSpecJSON` (fence-tolerant) exists; the ticket-triage change introduced the same resilient-drop pattern for a secondary output.

## Goals / Non-Goals

**Goals:**
- One AI build call produces segment + message draft (no extra metered cost).
- Campaign drafts can carry an optional `mensaje`, persisted, never sent by this change.
- FE: AI result pre-fills an editable message textarea; create payload includes `mensaje`.

**Non-Goals:**
- No scheduler/send changes, no template approval, no backfill of existing campaigns.
- No new LLM/model/metering; no permission changes.

## Decisions

### D1: One LLM call returns both outputs — `{"filter_spec": [...], "message_draft": "..."}`

Extend the existing single call's JSON contract; `Build()` parses both, validates `filter_spec` as today, and treats `message_draft` as optional (drop on parse failure, never fail the call). Rationale: two calls would double metered cost per build and double latency; one call keeps the credit economics unchanged. Alternative (second endpoint) rejected: redundant credit gate + LLM round trip for the same description.

### D2: Prompt gains message-drafting rules, still zero PII

Append to `audienceSystemPrompt`: role continues as Colombian-PYME WhatsApp marketing assistant; draft rules — Spanish, 1-3 sentences, tone derived from the description (formal/salesy/urgent as requested), clear CTA, Ley 1581-compliant (no misleading claims, optional consent reminder), no placeholders that require contact PII (no names/phones in the copy). The prompt inputs remain the user's description + tag dictionary only.

### D3: `AudienceBuildResult.MessageDraft string` — additive contract

`Build` returns `AudienceBuildResult{FilterSpec, Preview, MessageDraft}`; handler response unchanged shape + new field; FE reads it when present. `message_draft` omitted when unparsable or empty.

### D4: Schema — nullable `mensaje` on campaigns

Migration `000035_add_campaign_message` (up: `ALTER TABLE campaigns ADD COLUMN mensaje TEXT NULL`; down: drop column). SQLC: `CreateCampaign` gains `Mensaje sql.NullString` param (optional); repository/service pass it through; `Campaign` domain gains `Mensaje *string` (nil-safe, absent for existing rows). Backward compatible: old clients (no `mensaje`) create null-message drafts.

### D5: FE — textarea + prefill, no auto-save

`campaign-manager.tsx`: optional message textarea in the create form (copy under `ui.campaigns`); after a successful `aiBuild`, `message_draft` pre-fills the textarea (editable); `createCampaign` payload includes `mensaje` when non-empty; cleared on successful create. Nothing is sent by creating a draft; no auto-launch.

### D6: Permissions and error mapping unchanged

`ai-build` and create stay `org:manage`; 402/exhausted, 400 validation, and the envelope conventions are untouched.

## Risks / Trade-offs

- **Single-call contract coupling**: one JSON response now carries two outputs; mitigated by per-field tolerant parsing (spec failure still 400s; message failure degrades to today's behavior).
- **Copy quality**: message drafts are heuristic; the user edits before create, and nothing is sent by this change — user-controlled blast radius.
- **Schema addition**: nullable column, additive; rollback is a drop migration; no backfill needed.
- **Scheduler ambiguity**: `mensaje` has no consumer yet (scheduler is future work) — the field is stored for the eventual send path; documented so it isn't assumed dead.
