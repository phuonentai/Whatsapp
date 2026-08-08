## 1. Setup

- [x] 1.1 Add `@dnd-kit/core` to `next_b2b_starter/package.json` and install it (`pnpm add @dnd-kit/core`) [FE-NEXT]
- [x] 1.2 Verify `pnpm build` still passes after the dependency addition (client-only, no server code impact) — build GREEN; `pnpm lint` has the pre-existing Next 16 failure documented in W1 (`next lint` removed, "Couldn't find any pages or app directory"), unrelated to this change [FE-NEXT]

## 2. Mutation Hooks and Query Keys

- [x] 2.1 Add `useDeleteDeal` to `lib/hooks/mutations/use-crm-mutations.ts` (pattern: invalidate `queryKeys.crm.deals()`, reuse W1 Spanish error toast) [FE-NEXT]
- [x] 2.2 Add `useCreatePipeline`, `useUpdatePipeline`, `useCreateStage`, `useUpdateStage` mutation hooks (invalidate `queryKeys.crm.pipelines()`) [FE-NEXT]
- [x] 2.3 Extend `queryKeys.crm` in `lib/hooks/queries/query-keys.ts` with any missing keys required by the new hooks (e.g. `pipeline` detail key if needed) — no new keys needed; `deals`/`deal`/`pipelines` cover all new hooks, which invalidate `queryKeys.crm.all` per the W1 pattern [FE-NEXT]

## 3. Kanban Pipeline Selector

- [x] 3.1 In `components/crm/deal-kanban.tsx`, replace the `pipelines?.[0]` hardcode with `selectedPipelineId` state initialized to the `es_predeterminado` pipeline (fallback: first pipeline) [FE-NEXT]
- [x] 3.2 Render a `<select>` of pipelines from `usePipelinesQuery()` labeled "Pipeline:" with Spanish option labels; changing selection sets `selectedPipelineId` [FE-NEXT]
- [x] 3.3 Drive `useDealsQuery({ pipeline_id: selectedPipelineId })` and render stage columns from the selected pipeline's `etapas` (ordered by `orden`) [FE-NEXT]

## 4. Deal Dialog and Card Actions

- [x] 4.1 Create `components/crm/deal-dialog.tsx` (shadcn Dialog + react-hook-form + zod, Spanish labels): fields `nombre` (required), `monto`, `moneda` (default COP), `empresa` select, `contacto` select, `etapa` select; create mode sends `pipeline_id` of the selected kanban pipeline [FE-NEXT]
- [x] 4.2 Add "Nuevo negocio" toolbar button to the kanban opening the create dialog; wire `useCreateDeal` with invalidation of `queryKeys.crm.deals()` [FE-NEXT]
- [x] 4.3 Add a per-card dropdown menu with "Mover a...", "Editar" (opens dialog prefilled, `useUpdateDeal`), and "Eliminar" (opens shared `ConfirmDialog`, then `useDeleteDeal`) [FE-NEXT]

## 5. Drag-and-Drop Stage Moves

- [x] 5.1 Wrap the kanban board in `DndContext`; make each `stage-column` a `useDroppable` (id = stage id) and each `deal-card` a `useDraggable` (id = deal id) [FE-NEXT]
- [x] 5.2 Implement `onDragEnd`: if the card's stage changed, call `useMoveDealStage` with `{ stage_id, old_stage_name, new_stage_name }` resolved from the selected pipeline's etapas; invalidate deals [FE-NEXT]
- [ ] 5.3 Verify `deals.spec.ts` "moves deal between stages" passes (Playwright `dragTo` emits pointer events consumed by PointerSensor) — BLOCKED: backend stack down (saas_postgres/redis not running, no Stytch credentials to start it); e2e cannot run in this environment, same as W1 tasks 5.4/6.3 [FE-NEXT]

## 6. Pipelines View

- [x] 6.1 Create `components/crm/pipeline-view.tsx`: pipeline list with `[data-testid="pipeline-list"]`, per-pipeline stages as `[data-testid="stage-item"]`, stage edit button (`aria-label="Editar"`) opening an inline edit form (stage_name/color/probabilidad, "Salida" when null) wired to `useUpdateStage` [FE-NEXT]
- [x] 6.2 Implement the "Nuevo pipeline" dialog: pipeline name + dynamic stage rows ("Agregar Etapa", `input[name="stage_name"]`, `input[name="stage_color"]`), sequenced submission: `useCreatePipeline` then `useCreateStage` per row; on partial failure keep dialog open with Spanish error and reuse the created pipeline id [FE-NEXT]
- [x] 6.3 Register the "Pipelines" tab in `crm-page.tsx` (`Tab` union + `TAB_LABELS`: feature `crm_deals`, upgradePlan "Pro", label "Pipelines") and render `PipelineView` for `?view=pipelines` [FE-NEXT]
- [x] 6.4 Delete the dead `components/crm/pipeline-editor.tsx` (no importers — verified with grep before deleting) [FE-NEXT]

## 7. Verification

- [x] 7.1 Run `pnpm build` — GREEN (exit 0, verified 3x incl. `/dashboard/crm` route); `npx tsc --noEmit` GREEN. `pnpm lint` — pre-existing failure (same as W1: `next lint` removed in Next 16, ESLint 9 flat-config migration pending) [FE-NEXT]
- [ ] 7.2 Run e2e specs `deals.spec.ts`, `pipelines.spec.ts`, and `cross-entity.spec.ts` — BLOCKED: backend stack down (postgres/redis not running, Stytch credentials absent); e2e cannot run in this environment [FE-NEXT]
- [x] 7.3 Re-read the change artifacts; confirm proposal/design/specs/tasks stay consistent with the implementation; record deviations back into `design.md` — implementation matches D1–D6; noted: deal-edit dialog gates on `crm_deals` (no `crm_deals_manage` feature exists), kanban board added `data-testid="kanban-board"` and stage inputs carry `name="stage_name"/"stage_color"` per the e2e contract; the `PipelineView` stage-edit form uses `stage_name` field names to match the page object [OPS-GOV]

## 8. Archive Decision

**Archive deferred:** Implementation is code-complete and build-verified (19/21 tasks `[x]`, `pnpm build` GREEN ×3, `tsc --noEmit` clean). The two open tasks (5.3, 7.2) are e2e verification tasks blocked by the environment — backend stack down (postgres/redis not running), Stytch credentials absent, so e2e cannot run here (same blocker recorded on W1's 5.4/6.3). Per AGENTS.md, archive BLOCKS on incomplete verification tasks; deferred until the e2e specs (`deals`, `pipelines`, `cross-entity`) pass against a live backend. No behavioral gaps remain. [OPS-GOV]
