## Context

The Negocios kanban and pipeline management are the last read-only remnants of the CRM: `deal-kanban.tsx` hardcodes `pipelines[0]`, renders cards with only nombre/monto/company_name, moves stages only through a per-card "Mover a..." select, and has no create/edit/delete. `pipeline-editor.tsx` is dead code (never rendered, no CRUD). The backend is complete and needs zero changes: deal CRUD routes (`POST/PUT/DELETE /api/crm/negocios`), `PUT /api/crm/negocios/:id/etapa` (validates same-pipeline stage, publishes `crm.negocio.etapa_cambiada`, creates the `sistema` activity), pipeline/stage CRUD (`POST/PUT /api/crm/pipelines`, `POST /api/crm/pipelines/:id/etapas`, `PUT /api/crm/pipelines/:id/etapas/:stageId`), all gated by `deal:view/manage` and `pipeline:view/manage`. `CreatePipelineRequest` accepts only `{nombre, orden}` — there is no bulk stage creation endpoint.

The e2e contract is explicit and must pass unchanged: `deals.spec.ts` (create via "Nuevo negocio", delete with `/confirmar|sí|eliminar/`, `card.dragTo(stage-column)` for stage moves), `pipelines.spec.ts` (`?view=pipelines`, `[data-testid="pipeline-list"]`, "Nuevo pipeline" dialog with dynamic stage rows, `[data-testid="stage-item"]` with ≥4 stages on the default pipeline, stage edit via `aria-label="Editar"`), and `cross-entity.spec.ts` step 3 (create a deal from the kanban). Playwright's `dragTo` dispatches pointer (mouse) events — HTML5 `draggable` will NOT react to it, so drag support must be pointer-event-based.

W1 (archived) delivered the foundations this phase reuses: `queryKeys.crm`, `ConfirmDialog`, contact/company dialogs (shadcn Dialog + RHF + zod + sonner), and Spanish error mapping in `lib/crm/errors.ts`. The frontend stack has react-hook-form, zod v4, sonner, and the full shadcn kit; `@dnd-kit/core` is not yet installed.

## Goals / Non-Goals

**Goals:**
- Kanban becomes interactive: pipeline selector (default `es_predeterminado`), deal create/edit/delete dialogs, drag-and-drop stage moves persisting via the existing etapa endpoint, keeping the "Mover a..." select as fallback.
- Pipelines editor view (`?view=pipelines`, gated by `crm_deals`): pipeline list, "Nuevo pipeline" with dynamic stage rows, stage editing (nombre/color/probabilidad, "Salida" for null).
- All W3 e2e specs (`deals`, `pipelines`, `cross-entity`) pass; `pnpm build` / `pnpm lint` green.
- Zero backend changes: no migrations, no SQLC, no handlers, no services.

**Non-Goals:**
- Detail pages, tag picker, activity filters (W2 scope), stage/pipeline deletion UI, stage reordering, deal probabilidad in forms, import/export.
- Any DB migration, SQLC query change, or backend handler change.
- Server-side search wiring, pagination UI, inbox changes, sidebar nav, language unification, Stytch contract changes.

## Decisions

### D1: `@dnd-kit/core` with PointerSensor for kanban drag & drop

Playwright `dragTo` emits pointer events; HTML5 `draggable` does not respond to them, so a pointer-driven DnD library is required. Add `@dnd-kit/core` (client-only, ~9 kB gzipped) using the PointerSensor (default; keyboard accessibility via the `KeyboardSensor` optionally). Layout: `DndContext` wraps the board; each stage column is a `useDroppable({ id: stage.id })`; each deal card is a `useDraggable({ id: deal.id })`.

On drop, map `active.id` → deal, `over.id` → target stage, resolve the stage names from the selected pipeline's etapas, and call `useMoveDealStage` with `{ stage_id, old_stage_name, new_stage_name }` (the backend logs the transition). `onSuccess` invalidates `queryKeys.crm.deals()`. The "Mover a..." select remains as an accessible fallback (deals.spec does not require it to be removed).

**Alternatives considered:** hand-rolled pointer handlers (zero deps but ~1 day of work and bug-prone hover/scroll edge cases — rejected); HTML5 draggable (incompatible with Playwright dragTo — rejected); rewriting the e2e to use the select only (weakens the contract — rejected).

### D2: Pipeline creation sequenced client-side: `POST /pipelines` → `POST /:id/etapas` per stage

`CreatePipelineRequest` has no `stages` array, and the e2e creates a pipeline with stages in one dialog flow. Sequence inside the dialog submit handler: create the pipeline, then create each stage row (`nombre`, `color`, `orden = rowIndex`) via `POST /api/crm/pipelines/:id/etapas`. On completion invalidate `queryKeys.crm.pipelines()`. Partial failure (pipeline created, stage failed) keeps the dialog open with the Spanish error toast; retry reuses the existing pipeline id. This avoids any backend change and keeps the pipeline as the authoritative owner of stages.

**Alternatives considered:** extending `CreatePipelineRequest` with `stages []` (touches Go service + handler + tests — rejected, zero-backend-change is a goal); a dedicated bulk-stages endpoint (same rejection).

### D3: Kanban pipeline selector replaces the `pipelines[0]` hardcode

Introduce `selectedPipelineId` state on the kanban, initialized to the pipeline with `es_predeterminado === true` (fallback: first pipeline, i.e., today's behavior). A `<select>` renders "Pipeline: <nombre>" for each pipeline from `usePipelinesQuery()`. `useDealsQuery({ pipeline_id: selectedPipelineId })` and the board columns derive from `pipelines.find(p => p.id === selectedPipelineId).etapas`. Deal creation dialog defaults `pipeline_id` to the selected pipeline. `es_predeterminado` is guaranteed seeded by the backend (pipeline-management spec).

**Alternatives considered:** keep `pipelines[0]` (rejected — the spec requires a selector and multi-pipeline support); parallel deal queries per pipeline (unnecessary; single active pipeline per view).

### D4: Deal dialog (create/edit) and card menu

New `components/crm/deal-dialog.tsx` modeled on W1's `contact-dialog.tsx`: shadcn Dialog + react-hook-form + zod schema (`nombre` required; `monto` optional number; `moneda` default COP; `empresa`/`contacto` selects from `useCompaniesQuery`/`useContactsQuery`; `etapa` select from the selected pipeline's stages; `pipeline_id` fixed to the selected pipeline on create, omitted on edit — stage moves go through the etapa endpoint). Card menu (dropdown) with "Mover a...", "Editar", "Eliminar" (Eliminar → `ConfirmDialog` → `useDeleteDeal`). New hooks: `useDeleteDeal`, `useCreatePipeline`, `useUpdatePipeline`, `useCreateStage`, `useUpdateStage`; extend `queryKeys.crm` with any missing keys (pipelines already exists; add `pipeline` detail key only if needed).

**Alternatives considered:** inline row forms (rejected — inconsistent with W1 dialog pattern); edit-in-kanban-card (rejected — dialog keeps validation/toasts uniform).

### D5: Pipelines view replaces the dead editor

New `components/crm/pipeline-view.tsx`: list of pipelines with their stages (`[data-testid="pipeline-list"]`, `[data-testid="stage-item"]`), "Nuevo pipeline" button + dialog implementing D2, stage edit inline (Edit button → form with `stage_name`/`stage_color`/`probabilidad`, "Salida" when null → `useUpdateStage`). Register `"pipelines"` in the `Tab` union + `TAB_LABELS` in `crm-page.tsx` (feature `crm_deals`, upgradePlan "Pro", label "Pipelines"). The old `pipeline-editor.tsx` is deleted (it is dead code with no importers — verified).

## Risks / Trade-offs

- [Playwright `dragTo` + @dnd-kit PointerSensor fidelity] → PointerSensor listens to real pointer events, which Playwright's dragTo emits; verified in CI via `deals.spec.ts`. Fallback: `MouseSensor` if pointer events prove flaky in the Playwright driver.
- [Create-pipeline partial failure leaves an orphan pipeline] → retry in the same dialog reuses the created pipeline id; worst case an empty pipeline exists, which is valid and deletable only via API (deletion is a non-goal).
- [`crm-integrity-phase-a` in flight touches stage/pipeline FKs] → orthogonal: this change only writes `stage_id` via the existing endpoint; the integrity change's trigger keeps `pipeline_id` consistent. No migration here means no conflict window.
- [`add-crm-e2e-tests` may adjust specs] → this change implements against the current spec text and the archived W1 patterns; coordination noted in proposal.
- [No drag cursor/ghost polish] → dnd-kit provides transforms; visual polish is minimal and out of scope.

## Migration Plan

1. `pnpm add @dnd-kit/core` (client-only dependency; verify build).
2. Extend `queryKeys.crm` + add the 5 mutation hooks (gate-zero-adjacent commit).
3. Kanban: selector → deal dialog → card menu + delete → dnd-kit wiring.
4. Pipelines: `pipeline-view.tsx`, register tab, delete `pipeline-editor.tsx`.
5. Verify: `pnpm build`, `pnpm lint`, e2e `deals`, `pipelines`, `cross-entity`; `make test` unaffected.
6. Rollback (Git + Stytch): revert the FE commits (pure additive + one client-only dependency) and remove `@dnd-kit/core` from `package.json`. Stytch tenant policy state requires no rollback — this change never mutates Stytch state and stores no credentials/sessions (constitution-compliant).

## Open Questions

- None blocking.
