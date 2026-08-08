## Context

The CRM write-side shipped in two archived phases: W1 (`crm-write-side-ui`: contacts/companies CRUD, 409 conflict mapping) and W3 (`crm-write-side-ui-w3`: kanban DnD, deal CRUD, pipelines editor). Both explicitly deferred the residual scope to this change: detail pages, entity tag picker, and activity type filters / task fields.

The current state is a "pre-built but dead-wired" foundation:

- **Frontend**: `useContactQuery` / `useCompanyQuery` / `useDealQuery` / `useContactActivitiesQuery` / `useDealActivitiesQuery` exist with zero call sites (`lib/hooks/queries/use-crm-queries.ts`). `tagEntity` / `untagEntity` repo calls exist, never called. `ActivityDto` already carries `estado`, `fecha_vencimiento`, `metadata` — unused. The activity form uses an `as any` cast. The disabled Etiquetas tab renders a hardcoded "(Pro)" badge instead of the spec'd "Desbloquear con Enterprise".
- **Backend**: tag service (`ListByEntity`, `Update`) and sqlc queries (`ListTagsByEntity`, `UpdateTag`) exist but `ListByEntity` has no HTTP route and there is no `PUT /etiquetas/:id`. The global activities list filters only by `tipo`, missing `entity_type`/`entity_id`. The event bus is injected as `nil` in `module.go`, so `DealStageChanged` events never fire and no `tipo='sistema'` activity is ever written.
- **Spec drift**: the `activity-timeline` living spec documents English endpoints (`/api/crm/contacts/:id/activities`) that do not exist; the real API is Spanish (`/api/crm/actividades/contacto/:id`).

No migrations are needed — all tables (`crm.activities`, `crm.tags`, `crm.entity_tags`) exist since `000013`.

## Goals / Non-Goals

**Goals:**
- Complete the CRM write-side: detail views for contacts, companies, deals, and entity tag pickers on those views.
- Wire the dead data-layer foundation rather than rebuilding it.
- Add the missing backend read paths (entity tags, deal filter by contact, activity entity filters) and the stage-change system activity logging.
- Correct the `activity-timeline` spec endpoints to the implemented Spanish API.
- Keep the change additive: no migrations, no Stytch interaction changes.

**Non-Goals:**
- Detail pages for pipelines; server-side search wiring; pagination UI; inbox changes; sidebar navigation; Spanish/English language unification.
- Task assignment to team members, task reminders, notifications.
- Tag picker on list rows (detail views only); tag deletion UI changes beyond what exists.
- Any change to Stytch B2B contracts, sessions, or credential storage.

## Decisions

### D1: Detail routing via `?view=<entity>&id=<id>`

Extend the existing `view` search param in `crm-page.tsx` with an optional `id` param. When `id` is present for an entity view, render the detail component instead of the table/kanban; otherwise render as today. Back navigation uses `router.back()`; the detail view renders a "Volver" button.

Rationale: the URL-driven view pattern already exists (`?view=negocios`), deep-linkable, and requires no new route files. Alternative considered: nested route `/dashboard/crm/contactos/:id` — rejected because it would fragment the existing tab bar logic and require duplicating feature gating per route.

### D2: Frontend-composed detail data

Detail views compose existing query hooks (`useContactQuery`, `useContactActivitiesQuery`, `useDealsQuery({ contact_id })`, etc.) rather than adding a backend "detail response" that embeds tags/deals/activities.

Rationale: the dead hooks and repo methods already implement exactly this. The only backend additions are the three missing read paths (D3/D5/D6), each independently useful. Alternatives considered: backend enrichment endpoints returning a combined payload — rejected as more backend surface for no client benefit, and it would duplicate the existing per-entity activity endpoints.

### D3: Entity tag read path, symmetric with existing mutations

Add `GET /api/crm/etiquetas/entity/:entityType/:entityId` wired to the existing `tagService.ListByEntity` (repo + sqlc `ListTagsByEntity` already exist), plus the missing `PUT /api/crm/etiquetas/:id` wired to `tagService.Update` (sqlc `UpdateTag` exists).

Rationale: the POST/DELETE attach/detach routes already use the `entity/:entityType/:entityId` shape; a GET on the same path is symmetric and discoverable. The tag picker needs both the read path and update for full CRUD.

### D4: System activity via the event bus, not a direct service call

Wire the event bus in `module.go` (currently `NewDealService(..., nil)`) and add a subscriber that handles `DealStageChangedEventType` by writing a `tipo='sistema'` activity to the deal's timeline. The `DealStageChanged` event and its publisher already exist in `deal_service.go`; the activity repository already supports creating system activities (`ActivityTypeSistema` is defined, never written).

Rationale: the architecture already declares the event; `module.go` injecting `nil` is a wiring bug, not a design decision. A subscriber keeps the deal service decoupled from the activity service (Clean Architecture: domain → app → infrastructure) and matches how `whatsapp_message` activities are created today. Alternative considered: calling the activity service directly inside `UpdateStage` — rejected as it couples the deal service to the activity service and bypasses the declared event contract.

### D5: Deals filtered by contact via existing list endpoint

Add an optional `contact_id` filter to `GET /api/crm/negocios` (sqlc + service + handler).

Rationale: the contact detail view must show "associated negocios" per the living `crm-frontend` spec, and no endpoint returns deals-for-contact today. Reusing the list endpoint with a filter avoids a new route. The deal list already supports filtering (per `deal-management` spec).

### D6: Activity entity filters on the global list

Extend `GET /api/crm/actividades` to accept optional `entity_type` + `entity_id` alongside the existing `tipo` filter, matching the corrected `activity-timeline` spec.

Rationale: the spec already promises these filters; the FE filter control needs only `tipo`, but the spec contract must match reality. Sqlc `ListActivitiesByOrganization` is extended with the two optional params.

### D7: Task fields via typed payload

Extend the activity creation form: when `tarea` is selected, show `fecha_vencimiento` (date input) and `estado` (pendiente/hecha select); send a typed payload (no `as any`). Verify the sqlc `CreateActivity` params accept `estado`/`fecha_vencimiento`; if not, extend the query (columns already exist on `crm.activities`).

Rationale: `ActivityDto` already carries these fields; the `as any` cast hides a shape mismatch. If `CreateActivity` doesn't persist the fields, that is a query gap in the same change.

### D8: Spec reconciliation of activity endpoints

The `activity-timeline` delta specs update the documented endpoints to the implemented Spanish routes (`/api/crm/actividades/contacto/:id`, `/api/crm/actividades/negocio/:id`, `/api/crm/actividades` with `tipo`/`entity_type`/`entity_id`).

Rationale: per governance, code-vs-spec drift is resolved by correcting the spec through a change proposal (this one), not by renaming the API.

## Risks / Trade-offs

- [Detail views depend on backend read paths that are currently unrouted] → Backend tasks land first (groups 1–3) with the FE detail views after; `make test` + curl verification gate each backend group.
- [`ListByEntity` sqlc query or mapping may not return colors/names as the picker needs] → Verify the query/scan mapping in task 1.1 before FE work; adjust mapping in the same group.
- [Event bus wiring may introduce ordering/duplication of system activities on retries] → The subscriber creates activities transactionally with the event handler; system activities are append-only and idempotency is acceptable at the handler boundary (each stage change is a distinct event). Verify with `make test` + manual curl that a stage move writes exactly one `sistema` activity.
- [e2e verification is blocked (backend stack down, no Stytch credentials), same as W1/W3] → e2e specs are written and recorded as verification tasks; execution is recorded as BLOCKED with the environment reason, and the archive decision defers on the same basis as W1/W3.
- [`crm_activities` feature gating may hide the Actividad view from some tiers, making task-field verification harder] → Gate the filter control and task fields behind the same view gating already in place; unit/build verification covers the code path.

## Migration Plan

- **Deploy order**: backend read paths + system logging (groups 1–3) → FE detail views + tag picker (groups 4–6) → FE activity filters + task fields (group 7) → verification (group 8).
- **Rollback**: Git state — revert FE commits (pure additive) and BE commits (routes, wiring, sqlc regen; all reversible, no data migration). Stytch tenant policy state requires no rollback — this change never mutates Stytch state.

## Open Questions

- Whether the `sistema` activity should record `old_stage_name`/`new_stage_name` in `metadata` JSONB — default yes, as the handler already receives both names; confirm at apply time that `metadata` is included in the system activity.
- Whether `GET /etiquetas/entity/:entityType/:entityId` should paginate — default no (entity tag counts are small); paginate only if the FE needs it.
