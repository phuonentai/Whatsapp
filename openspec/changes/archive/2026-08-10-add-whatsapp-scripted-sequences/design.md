## Context

Playbooks seed per-vertical procedure data — pipeline, tags, module config presets, and guiones (WhatsApp message scripts) — stored as JSONB in `playbooks.payload` and mirrored in Go via `catalog.go` (startup `CatalogValidated` keeps both in sync). The inbox renders one pill per applied guion; clicking fills the composer and the human sends via the existing conversation send path (`quick-replies.tsx` → `useSendMessage`). Today every guion is a single-shot message, so multi-step walkthroughs (capture lead → quote → confirm → follow up) require picking pills step by step.

This change makes a guion optionally a **scripted sequence**: an ordered list of messages that auto-advances in the composer as the human sends each step. It is a payload contract + UI behavior change only — no new tables, no agent/LLM integration, no server-side workflow state.

Constraints: Clean Architecture (domain → app → infra), SQLC-generated queries untouched (JSONB payload, no schema change), forward-only migrations with down-migrations, module conventions (`internal/modules/playbooks/`), Stytch B2B untouched (no auth/RBAC change), no local credentials.

## Goals / Non-Goals

**Goals:**
- Guiones optionally carry ordered steps (`pasos`) with id/titulo/mensaje
- Sequence pills in the inbox with auto-advance on successful send, progress indicator, reset on conversation change, never auto-send
- Sequence seeds for all five verticals, kept in sync between `catalog.go` and the migration seed
- Validation rejects incomplete sequences through the existing invalid-payload path
- Backward compatible: single-shot guiones behave exactly as today; existing orgs with applied playbooks pick up sequences without re-applying (payload-level data)

**Non-Goals:**
- No server-side workflow engine, no DAG/nodes/edges, no per-conversation sequence state in the DB
- No branching/conditionals or variables in sequences (linear only)
- No agent/LLM awareness of guiones (agent prompt remains unchanged)
- No auto-send or unattended execution — every step is human-sent
- No interactive WhatsApp buttons/templates
- No new tables, no SQLC query changes, no Stytch policy changes

## Decisions

### D1: Sequence shape — `pasos` array, exclusive with `mensaje`

Payload contract:

```json
{ "id": "confirmar-pedido", "titulo": "Confirmar pedido",
  "pasos": [
    { "id": "p1", "titulo": "Detalle del pedido", "mensaje": "¡Perfecto! ¿Qué producto(s) quieres?" },
    { "id": "p2", "titulo": "Dirección", "mensaje": "¿A qué dirección lo enviamos?" },
    { "id": "p3", "titulo": "Link de pago", "mensaje": "Te enviamos el link de pago. Cuando esté confirmado, lo despachamos." }
  ] }
```

`Guion` gains `Pasos []GuionPaso` (`GuionPaso{ID, Titulo, Mensaje}`); `PlaybookPayload` unchanged structurally. A guion is single-shot (non-empty `mensaje`, no `pasos`) OR sequence (2+ steps, no `mensaje` required). Validation is exclusive: a guion declaring both is invalid.

*Alternative rejected:* `mensaje` kept as the first step's text with `pasos[1:]` as remainder — ambiguous, forces duplicated content between `mensaje` and `pasos[0]`, complicates UI branching.

### D2: No new tables — seed update via forward-only migration `000025`

Sequences live in the existing `playbooks.payload` JSONB. `000025_update_playbook_sequence_seeds.up.sql` runs `UPDATE playbooks SET payload = payload || jsonb_set(...)` per vertical key to inject `pasos` into the chosen guiones (and `.down.sql` restores the single-shot payloads). No SQLC regeneration. `catalog.go` mirrors the same data so the `CatalogValidated` startup check passes; both files change together.
*Alternative rejected:* editing migration `000020` in place — violates forward-only migrations (already applied in environments). *Alternative rejected:* a new `sequence_steps` table + FK — more machinery than a linear human-driven walkthrough needs, adds SQLC + repository + apply-path changes.

### D3: API passthrough — no service changes

`PlaybookService.ListCatalog` already returns `Guiones` decoded from the org playbook state payload; the new `Pasos` field flows through automatically once the type carries it. Only `playbooks/handler.go` (catalog mapping, currently `id`/`titulo`/`mensaje`) and the frontend `PlaybookGuionDto` need to surface `pasos`. Apply path untouched — orgs that already applied a playbook see sequences after the migration without re-applying (payload is read from the org playbook state row).

### D4: Sequence mode is frontend state only

A `use-sequence` hook (inbox page level) holds `{guionId, stepIndex}`. `QuickReplies` renders sequence pills with a step-count badge; clicking starts the sequence and fills the composer with `pasos[0].mensaje` via the existing `onSelect` seam. The reply-input send flow exposes a `onSent` callback invoked only after a successful send and the composer clear; the hook's `advance()` then re-fills the composer with the next step (ordering guarantees the re-fill is never clobbered by the send-clear); on failure no advance. Reaching the end clears the hook. Selecting another conversation resets it (conversation id is the hook reset key). No persistence — reload or navigation simply ends sequence mode.

### D5: Seeds — which guiones become sequences

- `comercio` → `confirmar-pedido` (detalle → dirección → link de pago)
- `restaurantes` → `domicilio` (confirmación → entrega → queja/follow-up)
- `citas` → `confirmar-cita` (fecha → hora → confirmación)
- `servicios-profesionales` → `cotizacion` (necesidad → alcance → envío propuesta)
- `talleres` → `cotizacion` (síntoma → cotización → aprobación)

All step text in Spanish, consistent with existing guion tone. Single-shot guiones (`saludo`, `seguimiento`, etc.) stay as-is.

### D6: Startup `CatalogValidated` check — new work, not pre-existing

Today no such check exists: `catalog.go` only carries a comment claiming it (line 10). This change builds a startup validation that loads the five seeded `modules.playbooks` rows via `PlaybookRepository.ListActive`, decodes each payload with `ParsePayload`, and compares `guiones` (including `pasos`) against `catalog.go`. Mismatch (count, id, titulo, mensaje, pasos, or vertical keys) fails fast at boot. Wired in `internal/bootstrap/init_mods.go` after the playbooks provider registers, alongside existing module init validation patterns. `catalog_test.go` keeps asserting seed completeness at test time; the startup check is the runtime counterpart.

*Alternative rejected:* rely on `catalog_test.go` alone — it validates the Go side but never reads the DB, so migration/catalog drift (the positional `jsonb_set` risk in D2) would only surface in prod after `000025` applies.

## Risks / Trade-offs

- **Dual seed source drift** (`catalog.go` vs migration) → the new `CatalogValidated` startup check (D6) compares catalog guiones incl. `pasos` against seeded DB rows and fails fast at boot; `catalog_test.go` asserts Go-side completeness; the seed task updates both files in the same unit
- **Sequence UX ambiguity** (auto-advance may surprise reps mid-thought) → progress indicator "Paso k de n" + tooltip shows the full sequence on the pill; failure never advances; conversation switch resets
- **Existing applied orgs** see new pills immediately after migration — additive, no data loss; re-apply is additive-only and untouched
- **Unarchived `add-vertical-playbooks` change** still owns the `vertical-playbooks` delta → this change's delta only ADDs requirements; archive of either change folds compatible content
- **Message cleared after send** — the composer clears on send today; the sequence re-fills it on the send-success callback, so a race with user typing must keep `onSent` filling only when sequence mode is active

## Migration Plan

1. Land migration `000025` + `catalog.go` + validation + handler passthrough + `CatalogValidated` startup check + tests (backend)
2. Land frontend DTO + hook + quick-replies UI + e2e update
3. Verify: `make sqlc` (no-op), `go build ./...`, `go test ./...`, `pnpm build`, `pnpm lint`, targeted e2e
4. Rollback: `000025.down.sql` restores single-shot payloads; revert frontend/backend edits (Git). No Stytch tenant state involved; no credentials stored

## Open Questions

- None blocking. Step text wording is copy, tunable at implementation time.
