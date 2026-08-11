# Change Proposal: add-list-pagination-total

## Why

The `ui-data-tables` change requires CRM list views to show total counts in pagination controls, but the backend CRM list endpoints return bare arrays with no total. The UI cannot render accurate page controls ("Página 2 de 12") without a server-provided count. Verified in `go-b2b-starter/internal/modules/crm/handler.go`: `ListContactos` (:71), `ListEmpresas` (:150), `ListNegocios` (:229), and activity lists (:375) all respond `response.Success(c, http.StatusOK, r)` where `r` is a bare `[]*domain.*` slice — no count. This change supplies the missing `total` so `ui-data-tables` pagination controls can be honest.

## What Changes

- Add `Count*ByOrganization` SQLC queries mirroring each list's filters (contacts, companies, deals, activities).
- Extend the four CRM list service methods to return total alongside items (counted with the same org scope + filters as the page query).
- Wrap paginated list responses as `{ data: [...], total: N }` (via a `PaginatedResponse` helper), keeping `data` as the existing array shape so non-paginated consumers are unaffected.
- Add optional `total?: number` to the frontend `Envelope<T>` type (`lib/api/api/repositories/crm-repository.ts:39`) so list hooks can surface it. **No frontend behavior changes in this change** — consumption lands in `ui-data-tables`.
- Scope the pagination requirement: the `ui-data-tables` spec names "Negocios list views" for pagination, but Negocios render as a kanban without page controls. This change does not paginate Negocios; kanban keeps virtualization (per `ui-data-tables` design). The pagination-total requirement applies to Contactos and Empresas table endpoints, plus activities.

## Capabilities

### New Capabilities
- `crm-list-pagination`: the four paginated CRM list endpoints SHALL include a total count in responses so clients can render page controls.

### Modified Capabilities
- `contact-management`: the contact list pagination requirement SHALL gain a total count in the response envelope.
- `crm-frontend`: the frontend API envelope SHALL carry an optional `total` on paginated list responses.

## Impact

- Backend (`go-b2b-starter/`): new SQLC count queries, list service methods return `(items, total)`, handler response shape gains `total`. Additive — existing consumers keep working.
- Frontend (`next_b2b_starter/`): `Envelope<T>` gains optional `total`; no render changes.
- No migrations, no Stytch changes, no new dependencies.
- Dependent on `ui-data-tables` (FE consumption of `total`); this change is the backend prerequisite.

## Non-Goals

- No server-side sorting/filtering redesign — only the count is added.
- No pagination for the kanban (Negocios); kanban stays virtualization-based.
- No new bulk endpoints.
- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority.
- No changes to non-paginated CRM endpoints.

## Rollback

- Git state: revert the change's commits. Additive response field; old clients ignore `total`.
- Stytch tenant policy state: no Stytch resources created or altered; nothing to roll back.
