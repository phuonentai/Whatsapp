## Why

The Negocios kanban and pipeline management are read-only: the kanban hardcodes `pipelines[0]`, shows only nombre/monto/company on cards, offers no create/edit/delete, and moves deals only via a per-card "Mover a..." select; the `PipelineEditor` component is dead code with no CRUD. The e2e suite (`deals.spec.ts`, `pipelines.spec.ts`, `cross-entity.spec.ts`) defines the interactive contract the UI does not meet: "Nuevo negocio" buttons, drag-and-drop between stage columns, and a pipelines editor with stage creation/editing. The backend fully supports all of this (deal CRUD, `PUT /negocios/:id/etapa`, pipeline/stage CRUD, `deal:view/manage` and `pipeline:view/manage`), so this is a pure frontend change building on the W1 foundations (`queryKeys.crm`, `ConfirmDialog`, Spanish toasts) already archived and live in code.

## What Changes

- **Kanban pipeline selector**: replace the `pipelines[0]` hardcode with a selector listing pipelines; default to the `es_predeterminado` pipeline ("Pipeline de Ventas") and drive `useDealsQuery({ pipeline_id })`.
- **Drag-and-drop stage moves**: add `@dnd-kit/core` (PointerSensor) so cards drag between `stage-column`s; on drop, call the existing `PUT /api/crm/negocios/:id/etapa` with `{ stage_id, old_stage_name, new_stage_name }` captured from the source/target column headers (the backend logs the transition activity). The "Mover a..." select stays as an accessible fallback.
- **Deal CRUD on the kanban**: "Nuevo negocio" toolbar button with create dialog (nombre, monto, moneda, empresa/contacto selects, etapa), per-card Editar/Eliminar with edit dialog, delete via the shared `ConfirmDialog`. New mutation hooks `useDeleteDeal` (create/update/move already exist).
- **Pipelines editor view**: add a "Pipelines" tab (`?view=pipelines`, gated by the `crm_deals` feature like Negocios) rendering a pipeline list (`[data-testid="pipeline-list"]`), "Nuevo pipeline" dialog with dynamic stage rows ("Agregar etapa", `input[name="stage_name"]:last`, `input[name="stage_color"]:last`), and stage editing (`[data-testid="stage-item"]`, `aria-label="Editar"`, nombre/color/probabilidad, "Salida" when probabilidad is null). Pipeline creation sequences `POST /api/crm/pipelines` then `POST /api/crm/pipelines/:id/etapas` per stage — no backend change (`CreatePipelineRequest` has no stages array).
- **Hooks & keys**: add `useDeleteDeal`, `useCreatePipeline`, `useUpdatePipeline`, `useCreateStage`, `useUpdateStage`, `useTagEntity`/`useUntagEntity` where missing; extend `queryKeys.crm` as needed; wire Spanish toasts via the W1 error-mapping helper.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- `crm-frontend`: new requirements for the kanban pipeline selector, drag-and-drop stage moves, deal create/edit/delete UI, and the pipelines editor view — all in Colombian Spanish.
- `pipeline-management`: new requirement that the pipelines editor creates pipelines with stages in one dialog flow (frontend sequences the two existing endpoints); backend behavior unchanged.

## Impact

- **FE**: `components/crm/deal-kanban.tsx` (selector, DnD, dialogs, card menu), new `components/crm/deal-dialog.tsx`, `components/crm/pipeline-view.tsx` (replaces dead `pipeline-editor.tsx`), `app/dashboard/crm/crm-page.tsx` (+"Pipelines" tab), `lib/hooks/mutations/use-crm-mutations.ts` (+5 hooks), `lib/hooks/queries/query-keys.ts` (+keys), `package.json` (+`@dnd-kit/core`).
- **BE**: none — all endpoints and permissions already exist. No migration, no SQLC change.
- **Tests**: `deals.spec.ts`, `pipelines.spec.ts`, `cross-entity.spec.ts` pass; `pnpm build` / `pnpm lint` green; `make test` unaffected. Coordination with the in-flight `add-crm-e2e-tests` change.
- **Auth boundary**: no change. This change does not touch the Stytch B2B runtime SSOT — no credentials, sessions, or identity data are added locally; `stytch_member_id`/`stytch_organization_id` linkage and all Stytch B2B API contracts (JWKS verification, webhook signatures, circuit breaker) are unaffected. CRUD routes remain gated server-side by `deal:view/manage` and `pipeline:view/manage`.
- **Rollback**: Git state — revert the FE commits (pure additive code + one client-only dependency). Stytch tenant policy state requires no rollback because this change never mutates Stytch state.

## Non-Goals

- Detail pages (`?view=<entity>&id=<id>`), entity tag picker, activity type filters — deferred to `crm-write-side-ui-w2`.
- Pipeline/stage deletion UI, stage reordering within a pipeline, deal probabilidad in the create form, import/export.
- Any DB migration, SQLC query change, or backend handler change.
- Server-side search wiring, pagination UI, inbox changes, sidebar navigation, Spanish/English language unification.
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
