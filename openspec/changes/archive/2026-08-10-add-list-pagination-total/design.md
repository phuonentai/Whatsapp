# Design: add-list-pagination-total

## Context

Backend prerequisite for `ui-data-tables` pagination controls. Verified current state:

- All four CRM list endpoints respond with a bare array: `response.Success(c, http.StatusOK, r)` where `r` is `[]*domain.*` (`internal/modules/crm/handler.go` — `ListContactos` :71, `ListEmpresas` :150, `ListNegocios` :229, activity lists :375).
- `response.Success` wraps as `{ success: true, data: ... }` (`pkg/response/response.go:10`). No count field exists anywhere in the envelope.
- SQLC has `List*ByOrganization :many` queries with filters, but no count queries.
- Service `List` methods return `([]*domain.X, error)` — no count.
- Frontend envelope `Envelope<T> = { data?: T; success?: boolean }` (`lib/api/api/repositories/crm-repository.ts:39`).
- `ui-data-tables` spec requires page controls showing "current page, page size, and total count" — the total is the missing piece this change supplies.
- Negocios render as a kanban (no page controls); this change does not paginate it.

## Goals / Non-Goals

**Goals**: add org-scoped, filter-matched `total` to the paginated CRM list responses (contactos, empresas, activities); keep `data` array shape unchanged; add frontend envelope support for `total`; keep the change additive and non-breaking.

**Non-Goals**: no pagination for kanban Negocios; no server-side sort redesign; no new bulk endpoints; no FE pagination UI (that is `ui-data-tables`); no migrations; no Stytch changes.

## Decisions

### D1: `{ data: [...], total: N }` — total beside the array, not nested

Keep `data` as the existing array; add sibling `total`. Alternatives: nest `{ data: { items, total } }` — rejected (breaking for existing consumers, larger FE ripple). Top-level `total` keeps old clients fully compatible.

### D2: SQLC count queries mirroring list filters

Add `Count*ByOrganization` queries per list with the same WHERE clauses as the list query minus `LIMIT/OFFSET` (contacts: source/lead_status/company_id/assigned_to; companies: org; activities: tipo/entity_type/entity_id). Alternatives: `COUNT(*) OVER()` window function in the list query — rejected (forces all pages to compute full count on every request; separate count is cacheable and mirrors the filtered list cleanly). Count runs in the same transaction as the list to avoid drift between page and total.

### D3: Service returns a small result struct `(items, total)`

Change each list service method signature to return a `ListResult[T]` (or `(items, total)`) so the handler can emit both. Alternatives: handler runs two service calls — rejected (two round trips, duplicate filter parsing, drift risk). List + count in one service method.

### D4: `response.Paginated` helper

Add `Paginated(c, status, items, total)` to `pkg/response` emitting `{ success: true, data: items, total }`. Keep `Success` unchanged. Alternatives: handlers build the map inline — rejected (duplicated in 4+ sites, easy to forget `total`).

### D5: Frontend envelope gets optional `total`

Add `total?: number` to `Envelope<T>` in `crm-repository.ts`; list hooks that need it read it via the unwrap path. No FE render changes here. Alternatives: new typed wrapper per list — rejected (churn; `ui-data-tables` consumes it next).

## Risks / Trade-offs

- [Count query and list query drift (page rows vs total)] → same transaction, same WHERE filters, single service call.
- [Performance: count over large org tables] → count queries are indexed on `organization_id`; counts happen once per page render, not per row.
- [Changing service signatures touches callers] → grep callers of `List`/`Search` for contacts/companies/activities; update all in this change.
- [Deal list (`ListNegocios`) not paginated] → documented; kanban stays virtualization-based in `ui-data-tables`.
- [FE envelope widening is invisible to existing code] → optional field; old consumers compile unchanged.

## Migration Plan

1. Add SQLC count queries + regenerate (`make sqlc`).
2. Add `ListResult[T]` + update service `List`/`Search` methods.
3. Add `response.Paginated`; update the four handlers.
4. Update FE `Envelope<T>` with `total?: number`.
5. Run backend tests + frontend typecheck.
6. Rollback: git revert per commit; additive field, no migrations, no backend behavior removal.

## Open Questions

- None blocking. Count in same transaction vs separate — resolved (same transaction, D2).
