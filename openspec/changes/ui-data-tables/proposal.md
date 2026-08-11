# Change Proposal: ui-data-tables

## Why

CRM tables render every row with no pagination (API supports `limit/offset` but the UI never sends it — use-crm-queries.ts:5-9), no sorting, no row selection or bulk actions, and no virtualization. Kanban drag-drop performs full cache invalidation instead of optimistic moves (deal-kanban.tsx:176-181) and has no keyboard sensor. Rows are clickable divs unreachable by keyboard. This is the P2 cluster of the UI/UX gap analysis.

## What Changes

- Add client-side pagination to contact, company, and (if applicable) ticket tables; use the existing `limit/offset` query params of the CRM API.
- Add sortable columns on contacts and companies (client-side sort, with column header affordances and `aria-sort`).
- Add row selection with checkboxes and bulk actions (bulk delete, bulk export) on contacts and companies.
- Add column virtualization via `@tanstack/react-virtual` for large result sets (virtualized tbody when row count exceeds a threshold).
- Add optimistic kanban stage moves with rollback on failure (`onMutate`/`onError` + query cache restore); add `KeyboardSensor` from `@dnd-kit/core` (keyboard drag).
- Make table rows keyboard-reachable (tabIndex, role, onKeyDown Enter), replace raw `<table>` with the shared `components/ui/table.tsx`, and standardize export via a shared helper (currently copy-pasted 4x).
- Distinguish empty-data vs no-search-results states with CTAs.
- Use `Skeleton` rows for CRM/tickets loading instead of "Cargando..." text.

## Capabilities

### New Capabilities
- `crm-data-tables`: table behaviors for the CRM — pagination, sorting, row selection/bulk actions, virtualization, keyboard navigation, and optimistic kanban moves.

### Modified Capabilities
- `crm-frontend`: CRM loading SHALL use skeletons; empty vs no-results states SHALL be distinct.

## Impact

- Frontend only (`next_b2b_starter/`): `components/crm/*`, `components/ui/table.tsx`, query hooks `hooks/*` (optimistic mutations).
- Backend: none required — bulk delete composes the existing per-item `DELETE` endpoints; bulk export reuses the existing export endpoints.
- New dependency: `@tanstack/react-virtual`.
- No Stytch changes.

## Non-Goals

- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority.
- No server-side sorting/search rewrite (client-side sort over the fetched page; `searchContacts` wiring is in app-shell-modernization).
- No ticket table overhaul beyond shared-table adoption and keyboard access.

## Rollback

- Git state: revert the change's commits; additive UI, no migrations, no backend changes.
- Stytch tenant policy state: no Stytch resources are created or altered; nothing to roll back.
