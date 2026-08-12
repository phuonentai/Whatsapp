# Proposal: add-campaign-ai-message

## Why

"AI campaigns" today means AI audiences only: `POST /crm/campanas/segments/ai-build` turns a natural-language description into a segment filter spec, but the campaign itself is just `nombre + segment_id` — the *message* is left to the user. A 2026 AI-first product drafts the copy too (Salesforce "Draft with AI", HubSpot AI content). This change makes the single AI input produce both the **who** (segment) and the **what** (message draft), and stores the draft on the campaign.

## What Changes

- **Backend, AI builder**: `ai_audience_builder.go` `Build()` returns both `filter_spec` **and** a `message_draft` from the **same single metered LLM call** (one JSON contract `{"filter_spec": [...], "message_draft": "..."}`). The prompt gains message-drafting rules (Spanish WhatsApp promo copy, tone from the description, CTA, Ley 1581-compliant — no misleading claims; still zero contact PII). Tolerant per-field parse: a failed `message_draft` is dropped (filter spec still returned), mirroring the ticket-triage resilience pattern.
- **Backend, schema**: migration `000035_add_campaign_message` — `ALTER TABLE campaigns ADD COLUMN mensaje TEXT NULL` (+ down). `CreateCampaign` SQLC query + params gain optional `mensaje`; repository/service/handler accept it. Existing rows and clients are unaffected (nullable, optional).
- **Backend, endpoint**: `POST /crm/campanas` accepts optional `mensaje` (was `nombre` + `segment_id`); `POST /crm/campanas/segments/ai-build` response gains `message_draft` (additive, backward compatible). Permissions unchanged (`org:manage` both).
- **Frontend**: the campaign form (`campaign-manager.tsx`) gains an optional message textarea; a successful AI build pre-fills it with `message_draft` (editable, nothing auto-saved or sent). Copy under `ui.campaigns` (Spanish-first + `en` mirror).
- **No Stytch policy changes; no send-path changes** (launch still snapshots recipients only; the scheduler consumes `mensaje` later — out of scope).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `whatsapp-campaigns`: two requirement updates — (1) *AI audience builder with human approval* gains the `message_draft` output (one-call contract, resilient parse); (2) *Campaign draft lifecycle* gains an optional `mensaje` field persisted with the draft.

## Impact

- **Code**: `go-b2b-starter/` — migration `000035` (up/down), `sqlc/query/campaigns.sql` + regen, `campaigns/domain/entities.go` (`AudienceBuildResult.MessageDraft`, `Campaign.Mensaje`), `ai_audience_builder.go` (prompt + JSON contract), `campaign_service.go`/`campaign_repository.go` (optional mensaje), `handler.go` (create body + ai-build response), tests. `next_b2b_starter/` — `campaign-manager.tsx` (textarea + prefill), `campaign-repository.ts` (payload), model/DTO, `lib/copy/ui.ts` (`ui.campaigns`), component test.
- **Dependencies**: none new — reuses `internal/platform/llm` + billing credits; SQLC regen required.
- **Systems**: one metered LLM call per AI build (unchanged cost — both outputs come from the same call).

## Non-Goals

- No scheduler/send-path changes — `mensaje` is stored with the draft; sending/launch behavior (`draft → ready`, recipient snapshot) is untouched.
- No WhatsApp template approval, no message delivery.
- No retroactive message backfill for existing campaigns (nullable column, displayed only when set).
- No new LLM provider/model/metering; no local credential storage; no credentials involved.

## Rollback

- **Git state**: revert the touched files (migration `000035` up+down, sqlc query + regen, domain/service/repository/handler edits, builder prompt/contract, FE form/repo/copy/tests, this change's artifacts). Migration down drops the column (nullable, additive — safe).
- **Stytch tenant policy state**: no policy changes, so no policy rollback required.
