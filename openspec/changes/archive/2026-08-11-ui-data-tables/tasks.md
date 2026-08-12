# Tasks: ui-data-tables

## 1. Shared primitives

- [x] 1.1 Add `@tanstack/react-virtual` to package.json ([FE-NEXT]). Verify: `pnpm install`
- [x] 1.2 Extract export flow to `lib/csv-export.ts` ([FE-NEXT]) and replace the 4 copy-pasted sites (`contact-table.tsx:33-43`, `company-table.tsx:30-40`, `deal-kanban.tsx:200-210`, `activity-timeline.tsx:36-46`). Verify: `rg -n "fetch-and-blob|csv" lib/csv-export.ts` exists; `pnpm lint`
- [x] 1.3 Migrate CRM tables to shared `components/ui/table.tsx` primitives ([FE-NEXT]). Verify: `pnpm lint`

## 2. Pagination

- [x] 2.1 Extend contact + company query hooks to accept page + pageSize and pass `limit`/`offset` ([FE-NEXT]) (use-crm-queries.ts). Verify: `pnpm lint`
- [x] 2.2 Add page size 25 default with `keepPreviousData` placeholder ([FE-NEXT]). Verify: `pnpm lint`
- [x] 2.3 Render pagination controls (current page, total count, prev/next) in contact + company tables ([FE-NEXT]); reset page on search/filter change. Verify: `pnpm lint`
- [x] 2.4 Unit test pagination hook ([FE-NEXT]): page changes query offset; keeps previous data while fetching. Verify: `pnpm test -- use-crm-queries`

## 3. Sorting

- [x] 3.1 Add client-side sort state to contact + company tables ([FE-NEXT]) with `aria-sort` on headers. Verify: `pnpm lint`
- [x] 3.2 Toggle asc/desc on header click with visible indicator ([FE-NEXT]); default sort newest-first. Verify: `pnpm lint`
- [x] 3.3 Unit test sort behavior ([FE-NEXT]). Verify: `pnpm test -- crm`

## 4. Row selection + bulk actions

- [x] 4.1 Add row checkboxes + select-all (current page) to contact + company tables ([FE-NEXT]) with keyboard-accessible checkboxes. Verify: `pnpm lint`
- [x] 4.2 Render bulk-actions bar when selection non-empty ([FE-NEXT]) with "Eliminar" and "Exportar". Verify: `pnpm lint`
- [x] 4.3 Bulk delete: `ConfirmDialog` then `Promise.allSettled` over per-item `DELETE` endpoints ([FE-NEXT]), aggregate toast "X eliminados, Y fallaron", refetch. Verify: `pnpm lint`
- [x] 4.4 Bulk export selected rows via existing export endpoints ([FE-NEXT]). Verify: `pnpm lint`
- [x] 4.5 Unit tests for selection + bulk delete flow ([FE-NEXT]). Verify: `pnpm test -- crm`

## 5. Virtualization

- [x] 5.1 Virtualize contact/company tbody with `@tanstack/react-virtual` when rows > 100 ([FE-NEXT]) with fixed row min-height. Verify: `pnpm lint`
- [x] 5.2 Verify scroll height + selection + keyboard nav preserved under virtualization ([FE-NEXT]). Verify: `pnpm test -- crm`

## 6. Keyboard-accessible rows

- [x] 6.1 Make table rows focusable (tabIndex) with Enter/Space activation to detail view ([FE-NEXT]) in contact + company tables. Verify: `pnpm lint`
- [x] 6.2 Add e2e keyboard test for row navigation ([OPS-GOV]). Verify: `pnpm test:e2e` (crm spec)

## 7. Optimistic kanban

- [x] 7.1 Refactor moveStage mutation ([FE-NEXT]) (use-crm-mutations.ts): `onMutate` snapshot + optimistic `setQueryData`, `onError` rollback + Spanish toast, `onSettled` invalidate. Verify: `pnpm lint`
- [x] 7.2 Add `KeyboardSensor` with drag-handle activator to `deal-kanban.tsx` ([FE-NEXT]). Verify: `pnpm lint`
- [x] 7.3 Unit test optimistic move + rollback on failure ([FE-NEXT]). Verify: `pnpm test -- deal-kanban`

## 8. Loading + empty states

- [x] 8.1 Replace "Cargando..." with `Skeleton` rows in contact/company tables and deal kanban ([FE-NEXT]). Verify: `pnpm lint`
- [x] 8.2 Add distinct no-results state for search/filter with no matches ([FE-NEXT]) (clear-filter action); keep existing empty-data state. Verify: `pnpm lint`
- [x] 8.3 Unit test empty vs no-results distinction ([FE-NEXT]). Verify: `pnpm test -- crm`

## 9. Verification

- [x] 9.1 Run frontend unit tests ([FE-NEXT]). Verify: `pnpm test`
- [x] 9.2 Run typecheck + lint ([FE-NEXT]). Verify: `pnpm lint` and `npx tsc --noEmit`
- [x] 9.3 Run production build ([FE-NEXT]). Verify: `pnpm build`
- [x] 9.4 Run e2e suite ([OPS-GOV]) — CRM specs must pass. Verify: `pnpm test:e2e`

- [ ] **Archive decision (2026-08-11):** **Archive** — all 29 tasks complete (last work was the no-results state + central gates); full unit suite 163/163, lint 0, tsc 0, build ✓, e2e 110/110 incl. CRM specs. Executed in archive sweep.
