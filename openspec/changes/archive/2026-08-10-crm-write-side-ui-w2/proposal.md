## Why

The CRM write-side phases W1 (contacts/companies) and W3 (deals kanban, pipelines editor) explicitly deferred their residual scope to this change: detail pages (`?view=<entity>&id=<id>`), an entity tag picker (Etiquetas), and activity type filters / task fields. The data layer for all three is already built but dead-wired: the frontend has `useContactQuery`/`useCompanyQuery`/`useDealQuery`/`useContactActivitiesQuery`/`useDealActivitiesQuery` with zero call sites, and `tagEntity`/`untagEntity` repo calls with zero call sites. The living specs already require behaviour the UI violates or lacks: the Etiquetas tab must render "Desbloquear con Enterprise" (today it shows a hardcoded "(Pro)" badge), contact detail must show negocios + activity timeline + action buttons, and deal stage transitions must produce system activities on the timeline — none of which exist. This change completes the CRM write-side by wiring the pre-built foundation and closing the backend gaps that block it.

## What Changes

- **Detail view routing**: `?view=<contactos|empresas|negocios>&id=<id>` renders a detail view; rows/cards in `contact-table.tsx`, `company-table.tsx`, and `deal-kanban.tsx` navigate to it; back navigation returns to the list/kanban. The `view` tab bar keeps working unchanged.
- **Contact detail view**: profile section showing all fields (including Tipo Documento, Número Documento), associated negocios, and the activity timeline, with Spanish action buttons "Editar", "Agregar nota", "Crear negocio" (per living `crm-frontend` spec).
- **Company detail view**: profile fields, contact/negocio counts, associated negocios, and the company activity timeline.
- **Deal detail view**: profile fields, stage, contact/company refs, and the deal activity timeline.
- **Tag picker**: a reusable `TagPicker` component on detail views to attach/detach tags via the existing `POST /etiquetas/entity/:entityType/:entityId` and `DELETE /etiquetas/entity/:entityType/:entityId/:tagId` endpoints; entity DTOs gain a `tags` array.
- **Backend tag read path**: wire the existing-but-unrouted `tagService.ListByEntity` to a `GET /api/crm/etiquetas/entity/:entityType/:entityId` route (repository + sqlc query already exist); add the missing `PUT /api/crm/etiquetas/:id` route (service + sqlc `UpdateTag` exist, no HTTP route).
- **Backend deal filter**: `GET /api/crm/negocios` gains an optional `contact_id` filter so a contact detail can list its associated negocios.
- **Backend activity entity filters**: `GET /api/crm/actividades` gains optional `entity_type` + `entity_id` filters alongside the existing `tipo` filter, matching the living `activity-timeline` spec.
- **System stage-change activities**: wire the event bus (today injected as `nil` in `module.go`) with a subscriber that writes a `tipo='sistema'` activity when a deal changes stage, so deal timelines include stage transitions.
- **Activity form task fields**: for `tarea` type, the form collects `fecha_vencimiento` (due date) and `estado` (pendiente/hecha); sqlc `CreateActivity` params extended if missing; removes the `as any` cast in `activity-timeline.tsx`.
- **Activity type filter UI**: the Actividad view gains a Spanish filter control (Todos / Nota / Llamada / Correo / Reunión / Tarea) wired to the existing `useActivitiesQuery({ tipo })`.
- **Etiquetas tab gate fix**: the disabled Etiquetas tab renders "Desbloquear con Enterprise" instead of the hardcoded "(Pro)" badge, per the living `crm-frontend` spec.
- **Spec reconciliation**: the `activity-timeline` living spec documents English API paths (`/api/crm/contacts/:id/activities`) while the real API is Spanish (`/api/crm/actividades/contacto/:id`); delta specs correct the documented paths to the implemented Spanish routes.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- `crm-frontend`: new requirements for detail-view routing and contact/company/deal detail views, the entity tag picker, the activity type filter control, the task fields (due date + estado) in the activity form, and the corrected "Desbloquear con Enterprise" Etiquetas tab gate.
- `contact-management`: new requirement that a contact's detail response/API surface supports listing the contact's negocios and tags.
- `company-management`: no new requirements — company detail already returns counts; the detail view is a frontend requirement covered under `crm-frontend`.
- `tag-management`: new requirement for the read path `GET /etiquetas/entity/:entityType/:entityId` returning an entity's tags, and for the tag update route `PUT /etiquetas/:id`.
- `activity-timeline`: corrected endpoint paths (English → Spanish), new requirement for entity_type/entity_id filters on the global activities list, new requirement that deal stage transitions create system activities, and the task activity fields (due date + estado).

## Impact

- **BE**: `internal/modules/crm/routes.go` (+`GET /etiquetas/entity/:entityType/:entityId`, +`PUT /etiquetas/:id`), `internal/modules/crm/handler.go` (wire `ListByEntity`, `UpdateEtiqueta`, `contact_id` deal filter, activity entity filters), `internal/db/postgres/sqlc/query/crm_extended.sql` + regenerated sqlc (deal filter by contact, activity entity_type/entity_id filter, `CreateActivity` task fields if missing), `internal/modules/crm/module.go` (event bus wiring), new subscriber writing `tipo='sistema'` activities on `DealStageChanged`. No migration — all tables exist (`000013_create_crm_activities_tags`).
- **FE**: `app/dashboard/crm/crm-page.tsx` (id routing + tab gate fix), new `components/crm/contact-detail.tsx`, `company-detail.tsx`, `deal-detail.tsx`, `tag-picker.tsx`; `contact-table.tsx` / `company-table.tsx` / `deal-kanban.tsx` (row/card navigation); `activity-timeline.tsx` (filter control, task fields, typed payload); `lib/api/api/dto/crm.dto.ts` (+`tags`); `lib/hooks/mutations/use-crm-mutations.ts` (+`useUpdateTag`, +tag attach/detach hooks if missing); `lib/hooks/queries/query-keys.ts` (+keys).
- **Tests**: `pnpm build` / `npx tsc --noEmit` green; `make test` unaffected; e2e coverage extended in `contacts.spec.ts`, `companies.spec.ts`, `deals.spec.ts`, `activities.spec.ts`, `cross-entity.spec.ts` (detail navigation, tag attach/detach, activity filters, task fields). e2e execution is BLOCKED in this environment (backend stack down — postgres/redis not running, no Stytch credentials), same blocker as W1/W3.
- **Auth boundary**: no change. This change does not touch the Stytch B2B runtime SSOT — no credentials, sessions, or identity data are added locally; `stytch_member_id`/`stytch_organization_id` linkage and all Stytch B2B API contracts (JWKS verification, webhook signatures, circuit breaker) are unaffected. All CRM routes remain gated server-side by the existing `contact:view/manage`, `deal:view/manage` permissions.
- **Rollback**: Git state — revert the FE commits (pure additive code) and the BE commits (routes, wiring, one sqlc regen). Stytch tenant policy state requires no rollback because this change never mutates Stytch state.

## Non-Goals

- Detail pages for pipelines, activity-type configuration, server-side search wiring, pagination UI, inbox changes, sidebar navigation, Spanish/English language unification — all deferred to follow-on phases.
- Task assignment to team members, task reminders, deal probabilidad in the create form, entity tag picker on list rows (detail views only).
- Any DB migration, SQLC query change beyond the additive filters/fields listed above, or contact identity redesign.
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
