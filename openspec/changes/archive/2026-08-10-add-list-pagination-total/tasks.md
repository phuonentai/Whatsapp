# Tasks: add-list-pagination-total

## 1. Database: count queries

- [x] 1.1 Add `CountContactsByOrganization` SQLC query to `internal/db/postgres/sqlc/query/crm.sql` matching `ListContactsByOrganization` filters (source, lead_status, company_id, assigned_to) without `LIMIT/OFFSET` ([DB-SQLC]). Verify: `make sqlc` regenerates without error
- [x] 1.2 Add `CountCompaniesByOrganization` SQLC query matching company list filters ([DB-SQLC]). Verify: `make sqlc`
- [x] 1.3 Add `CountActivitiesByOrganization` SQLC query matching activity list filters (tipo, entity_type, entity_id) ([DB-SQLC]). Verify: `make sqlc`

## 2. Backend: service layer returns totals

- [x] 2.1 Add `ListResult[T]` struct (or equivalent) in the CRM app services package ([BE-DOMAIN]). Verify: `make build`
- [x] 2.2 Update `contactService.List`/`Search` to return `(items, total)` by issuing the count in the same transaction as the list ([BE-DOMAIN]). Verify: `make build`
- [x] 2.3 Update `companyService.List`/`Search` to return `(items, total)` ([BE-DOMAIN]). Verify: `make build`
- [x] 2.4 Update activity list methods to return `(items, total)` ([BE-DOMAIN]). Verify: `make build`
- [x] 2.5 Update all callers of the changed service signatures (handlers, other services, tests) ([BE-DOMAIN]). Verify: `make build && make test`

## 3. Backend: response envelope

- [x] 3.1 Add `Paginated(c, status, items, total)` helper to `pkg/response/response.go` emitting `{ success, data: items, total }` ([BE-INFRA]). Verify: `make build`
- [x] 3.2 Update `ListContactos` handler to emit `Paginated` with total from service ([BE-INFRA]). Verify: `make test` (contact handler tests)
- [x] 3.3 Update `ListEmpresas` handler to emit `Paginated` ([BE-INFRA]). Verify: `make test`
- [x] 3.4 Update activity list handlers to emit `Paginated` ([BE-INFRA]). Verify: `make test`
- [x] 3.5 Add/extend handler tests asserting `total` equals filter-matched count and `data` shape is unchanged ([BE-INFRA]). Verify: `make test`

## 4. Frontend: envelope type

- [x] 4.1 Add `total?: number` to `Envelope<T>` in `lib/api/api/repositories/crm-repository.ts` ([FE-NEXT]). Verify: `pnpm lint && npx tsc --noEmit`
- [x] 4.2 Add unit test for list repository unwrapping exposing `total` alongside `data` ([FE-NEXT]). Verify: `pnpm test -- crm`

## 5. Verification

- [x] 5.1 Run backend full test suite ([BE-INFRA]). Verify: `make test`
- [x] 5.2 Run frontend typecheck + lint ([FE-NEXT]). Verify: `pnpm lint && npx tsc --noEmit`
- [x] 5.3 Run frontend production build ([FE-NEXT]). Verify: `pnpm build`
- [x] 5.4 Run e2e suite to confirm no regression in CRM specs ([OPS-GOV]). Verify: `pnpm test:e2e`
