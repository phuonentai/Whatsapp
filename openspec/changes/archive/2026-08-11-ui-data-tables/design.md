# Design: ui-data-tables

## Context

Frontend-first change in `next_b2b_starter/` with a small backend touch only if bulk composition proves insufficient (expected: not needed). Verified current state:

- CRM API queries already pass `limit`/`offset` params in the query layer (`hooks/use-crm-queries.ts`), but views render all rows (`contact-table.tsx:123`, `company-table.tsx:100`).
- Plain `<th>` headers, no sorting, no selection; raw `<table>` instead of `components/ui/table.tsx` (which only CRM avoids).
- Kanban: `@dnd-kit/core` with `PointerSensor` only (`deal-kanban.tsx:166`), full cache invalidate on move (`deal-kanban.tsx:176-181`), per-card "Mover a..." select fallback exists.
- Export logic copy-pasted 4x (`contact-table.tsx:33-43`, `company-table.tsx:30-40`, `deal-kanban.tsx:200-210`, `activity-timeline.tsx:36-46`).
- Loading = "Cargando..." text; empty vs no-results states identical (`contact-table.tsx:178-180`).

Spec contract: specs/crm-data-tables/spec.md (new) + crm-frontend delta. Bulk delete composes existing per-item `DELETE` endpoints — no backend changes.

## Goals / Non-Goals

**Goals**: pagination, column sorting, row selection + bulk actions, virtualization above threshold, keyboard-accessible rows, skeleton loading, distinct empty/no-results states, optimistic kanban moves with keyboard drag, shared table primitives + shared export helper.

**Non-Goals**: no server-side sorting/filtering redesign (client-side sort over fetched page); no new bulk endpoints; no search wiring (app-shell change); no ticket table rebuild beyond shared primitives/keyboard access.

## Decisions

### D1: `@tanstack/react-virtual` for virtualization (new dependency)
Virtualize tbody when rendered rows > 100. Fixed row heights (h-14-ish, enforced via the shared table row classes) keep it simple; `useVirtualizer` with `getScrollElement` from the table wrapper.
- Alternatives: `react-window` — rejected: TanStack Query already in stack; same author, same patterns, one dependency family.
- Rationale: unbounded `.map` currently renders every row (`contact-table.tsx:123`); virtualization caps DOM without changing data flow.

### D2: Pagination client-side over `limit/offset`
Page size 25 (default), controls showing page + total. Query key includes page so TanStack caches per page; `placeholderData: keepPreviousData` for smooth paging.
- Alternatives: infinite scroll — rejected: bulk actions + keyboard nav are simpler with explicit pages; API already `limit/offset`-shaped.
- Alternative: all client-side slicing — rejected: defeats the existing API pagination and will break at scale.

### D3: Selection state per table, composed, not a library
Row ids in a `Set` via `useState` in each table; select-all toggles the current page's ids; bulk bar renders when `size > 0`. No TanStack Table dependency (overkill for two tables + kanban).
- Rationale: keeps diff small, matches existing component style.

### D4: Bulk delete = sequential existing endpoints with aggregate reporting
`Promise.allSettled` over `DELETE /api/crm/contactos/:id` (or empresas), then toast "X eliminados, Y fallaron" + refetch. Confirmed via `ConfirmDialog`.
- Alternatives: new `POST /api/crm/contactos/bulk-delete` — rejected: backend + migration cost for a low-volume admin action; sequential is correct (org-scoped, idempotent per item).

### D5: Optimistic kanban with cache rollback
In `use-crm-mutations` moveStage mutation: `onMutate` → cancel queries, snapshot cache, move card via `setQueryData`; `onError` → restore snapshot + Spanish toast; `onSettled` → invalidate. Add `KeyboardSensor` from `@dnd-kit/core` (exists in package already — no new dep) via `KeyboardSensor(activator: drag handle)`.
- Alternatives: wait-then-refetch (current) — rejected: causes card flicker/return (verified in analysis).
- Note: `@dnd-kit/sortable` NOT added — no reorder-within-list requirement; plain draggable suffices.

### D6: Shared table + export refactor
CRM tables adopt `components/ui/table.tsx` primitives; export flow extracted to `lib/csv-export.ts` shared by the 4 copy-pasted sites.

## Risks / Trade-offs

- [Client-side sort only sorts current page] → documented; pagination default sort (newest first) keeps order stable; server-side sort noted as follow-up.
- [Virtualization + fixed height conflicts with variable content] → enforce row min-height in shared table styles; fall back to non-virtualized under threshold.
- [Optimistic rollback correctness across pagination] → snapshot/restore of the exact query key used by the kanban query; verify with component test.
- [Bulk sequential deletes on large selections] → show per-row progress count; cap selection UX at page scope (select-all = visible page).
- [KeyboardSensor conflicts with pointer drag] → use drag-handle activator for keyboard, keep whole-card pointer behavior.

## Migration Plan

1. Shared primitives + export helper refactor (no behavior change) — commit alone.
2. Pagination + sorting on contacts/companies.
3. Selection + bulk bar + bulk delete/export.
4. Virtualization above threshold.
5. Optimistic kanban + KeyboardSensor + rollback test.
6. Skeletons + distinct empty/no-results.
7. Rollback: git revert per commit; no migrations, no backend.

## Open Questions

- Page size default (25 vs 50) — minor; default 25.
- Whether ticket list joins the shared-table refactor — include only if low diff; spec covers CRM + tickets for skeletons (that's in crm-frontend delta).
