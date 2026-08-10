## Why

Playbook guiones are single-shot quick replies today: a click fills the composer and the human sends (spec `inbox-ui`). Routine walkthroughs — capture lead → send quote → confirm payment → follow up — force the rep to re-pick pills per step, wasting time and dropping steps. This change lets a guion carry an ordered, predetermined scripted sequence of messages that auto-advances in the composer as each step is sent. No server-side workflow engine: the sequence is a static payload addition driven by the human at send time.

## What Changes

- **Guion payload extension**: a guion in a playbook payload MAY be a sequence — carrying `pasos` (2+ ordered steps, each with `id`/`titulo`/`mensaje` in Spanish) — instead of a single `mensaje`. Single-shot guiones (`mensaje` only) keep current behavior unchanged.
- **Seed data**: the catalog (`catalog.go`) and the mirrored migration seed get sequence guiones for the five verticals (e.g. `comercio` confirmar-pedido, `citas` confirmar-cita, `talleres` cotizacion). Applied playbooks expose them through the existing `GET /api/playbooks` catalog response.
- **Payload validation**: a guion is valid when it has either a non-empty `mensaje` or 2+ complete `pasos`; incomplete sequences are rejected with the existing `ErrInvalidPlaybookPayload` path.
- **Inbox sequence mode**: sequence guiones render as distinct pills (with step count). Clicking starts sequence mode: step `i` fills the composer; after the message is sent, step `i+1` auto-fills; the last step exits sequence mode. A progress indicator shows "Paso k de n".
- **No new tables**: sequences live inside the existing `playbooks.payload` JSONB; `pasos` flows through `PlaybookPayload` → handler → frontend DTO.

## Capabilities

### New Capabilities

- `vertical-playbooks`: playbook registry payload contract — guiones may be single-shot or scripted sequences (`pasos`), validation rules for both shapes. (No living spec exists yet; the delta in the unarchived `add-vertical-playbooks` change is the current reference and folds here on archive.)

### Modified Capabilities

- `inbox-ui`: quick-reply requirement extends from single-shot pills to scripted sequence pills with auto-advance on send.

## Impact

- **Go backend**: `internal/modules/playbooks/domain/entities.go` (`GuionPaso` type + `Guion.Pasos`), `app/services/playbook_service.go` (validator dual-branch), `catalog.go` (sequence seeds), `handler.go` (`pasos` passthrough); handler tests. A new startup `CatalogValidated` check (built in this change — it does not exist today) compares the Go catalog payloads against the seeded `modules.playbooks` rows, including `pasos`, and fails fast on drift.
- **Database**: new forward-only migration `000025_update_playbook_sequence_seeds.{up,down}.sql` updating `playbooks.payload` for the five vertical rows (JSONB `jsonb_set` on `guiones`); no schema change, no SQLC regeneration.
- **Frontend**: `next_b2b_starter/lib/api/api/dto/playbook.dto.ts` (`pasos`), `app/dashboard/inbox/components/quick-replies.tsx` (sequence pills + progress), inbox page/`reply-input.tsx` (send-success advance hook), e2e `inbox-ui.spec.ts`.
- **Dependencies**: none new. **Auth**: no Stytch changes; catalog route keeps existing auth + entitlement + RBAC chain. **Config**: none.
- **Rollback**: Git — revert this change (migration, Go, FE). DB — `000025.down.sql` restores the five payloads to single-shot guiones. Stytch tenant policy state unaffected (no auth/RBAC changes); no credentials involved.
- **Non-Goals**: no auto-send or unattended sequence execution (a human still sends every step); no per-conversation sequence state stored server-side; no branching/conditionals or collected-answer persistence beyond the normal `crm.messages` transcript; no agent/LLM integration (agent prompt remains guion-blind); no interactive WhatsApp buttons/templates; no workflow/DAG engine; rejects any local credential storage — Stytch remains the sole identity authority.

## Assumptions

- **`vertical-playbooks` has no living spec**: its requirements currently exist only as delta specs under the unarchived `add-vertical-playbooks` change (tasks complete; archive decision pending). This change writes its own `vertical-playbooks` delta; if both changes are archived, the two deltas fold into one living spec — content is compatible (this delta extends, never contradicts).
- **`pasos` passthrough requires a handler change**: `playbooks/handler.go` currently maps only `id`/`titulo`/`mensaje`; `pasos` must be added to the catalog response and to `PlaybookGuionDto`.
- **`ListCatalog` returns `Guiones` from the org playbook state payload**; with sequences seeded at the payload level, applied orgs receive them without any apply-path change (re-apply not required for existing orgs).
