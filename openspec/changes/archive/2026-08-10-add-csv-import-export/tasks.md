## 1. RBAC export action [BE-INFRA] [OPS-GOV]

- [x] 1.1 Add typed permission constants `PermContactExport`, `PermDealExport`, `PermActivityExport` to the auth package (`NewPermission("contact","export")` etc.). Verify: `go build ./...`
- [x] 1.2 Grant the three export permissions to `RoleManagerInfo` and `RoleAdminInfo` in `internal/modules/auth/rbac.go` (Go fallback/mock parity; Stytch policy remains SSOT). Verify: `go test ./internal/modules/auth/...`
- [x] 1.3 Version the RBAC policy cache key (`rbacPolicyCacheKey` in `internal/modules/auth/adapters/stytch/rbac_policy.go`) so the new `export` action applies without waiting for the cache TTL. Verify: cache key constant changed; unit test references updated
- [x] 1.4 Document the Stytch RBAC policy update (add `export` action to `contact`/`deal`/`activity` resources, grant to `admin`/`manager`, ensure `export` listed in each resource's actions array for wildcard expansion) as a deployment step in tasks notes. Verify: notes present; policy change executed during deployment

> **Deployment step (Stytch RBAC policy — Runtime SSOT):** Before enabling the export endpoints in production, update the Stytch RBAC policy via the RBAC Policy API (`GET /v1/b2b/rbac/policy`, `PUT /v1/b2b/rbac/policy`):
> 1. Add the `export` action to the `contact`, `deal`, and `activity` resource definitions — `export` MUST appear in each resource's `actions` array so `expandWildcardActions` expands `contact:*`/`deal:*`/`activity:*` grants to it.
> 2. Grant `contact:export`, `deal:export`, and `activity:export` to the `admin` and `manager` roles.
> 3. No rollback of this change is required to ship the backend (cache key already versioned to `auth:stytch:rbac:policy:v2`); if the policy update is not executed, export endpoints return 403 for all users until it is.

## 2. CSV export backend [BE-INFRA]

- [x] 2.1 Create `internal/modules/crm/app/services/csv_service.go` with a shared `ExportService` that writes a UTF-8 BOM, Spanish headers, and streams rows via a pagination callback through `encoding/csv`. Verify: `go build ./...`
- [x] 2.2 Implement `csvSanitizeCell` in the CSV service: prefix a single quote when a cell starts with `=`, `+`, `-`, or `@`. Verify: unit test asserts sanitized output for each trigger prefix
- [x] 2.3 Implement entity mappers for contacts, companies, deals, activities with Spanish column headers matching the CRM list views, applying withdrawn-consent PII masking (`[TELEFONO]`/`[NOMBRE]`/`[EMAIL]`/`[DOCUMENTO]`) via the contact consent state. Verify: unit tests assert header row and masked values
- [x] 2.4 Add four export handlers on `CRMHandler` (`ExportContactos`, `ExportEmpresas`, `ExportNegocios`, `ExportActividades`) that set `Content-Type: text/csv` + `Content-Disposition: attachment`, resolve org scope from `auth.GetRequestContext(c)`, paginate via the existing `List` repository methods, and stream through the CSV service. Verify: `go build ./...`; handler tests assert BOM prefix, content-type, and org scoping
- [x] 2.5 Record an export audit event as a `sistema` activity (`crm.activities`, `ActivityTypeSistema`) with entity, row count, member, and timestamp. Verify: unit test asserts a `sistema` activity row is created after export

## 3. CSV import backend [BE-INFRA]

- [x] 3.1 Add `GET /crm/import/contactos/template.csv` handler serving the exact import template (required `teléfono`, `nombre`; optional `email`, `tipo_documento`, `numero_documento`, `empresa`, `origen`, `estado`) with two example rows, gated `contact:view`. Verify: handler test returns the template bytes
- [x] 3.2 Implement `POST /crm/import/contactos` handler: parse CSV, enforce the 5000-row cap, upload size cap, and content sniff (reject non-CSV with 400) before any write; validate rows, dedupe by existing phone (skip, never overwrite), create via `contactService.Create`; return `{importados, omitidos, errores:[{fila, razon}]}`. Verify: `go build ./...`; tests cover valid rows, invalid rows, duplicate skip, and the summary shape
- [x] 3.3 Register all six export/import routes in `internal/modules/crm/routes.go` under the existing `/crm` group with `RequirePermissionFunc` gates (`contact:export` ×2 contact-group export endpoints + `deal:export` + `activity:export`, `contact:view` template, `contact:manage` import) and matching feature guards. Verify: `go build ./...`; route table inspected

## 4. Frontend [FE-NEXT]

- [x] 4.1 Add export actions to the Contactos, Empresas, Negocios, and Actividades list views that fetch the CSV with the session token and trigger a blob download (fetch+blob; no `window.location`); hide the action when the user lacks the `export` permission. Verify: `npx tsc --noEmit`; `pnpm lint` at documented baseline
- [x] 4.2 Add a Contactos import modal (visible with `contact:manage`): downloadable template link, CSV file picker, submit to `POST /crm/import/contactos`, and a result summary showing imported/omitted/error counts with row numbers, in Colombian Spanish. Verify: `npx tsc --noEmit`; manual check on `pnpm dev`

## 5. Verification gate [OPS-GOV]

- [x] 5.1 Run backend gate: `go build ./...`, `go vet ./...`, `go test ./...`. Verify: all exit 0; record results here
- [x] 5.2 Run frontend gate: `npx tsc --noEmit`, `pnpm lint`, `pnpm build`. Verify: tsc and build exit 0; lint at documented baseline (no new violations); record results here
- [x] 5.3 Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>`. Verify: entry present

### Verification results (gate)

- **Backend (5.1):** `go build ./...` exit 0; `go vet ./...` exit 0 (no findings); `go test ./...` exit 0 — all packages pass (auth, crm, crm/app/services and 25 others). Recorded 2026-08-09.
- **Frontend (5.2):** `pnpm exec tsc --noEmit` exit 0; `pnpm lint` exit 0 (0 errors; 1 pre-existing warning in `components/crm/deal-kanban.tsx` `stages` useMemo — untouched code, present before this change); `pnpm build` exit 0. Recorded 2026-08-09.
- **Archive decision (5.3):** **Archive deferred:** the Stytch RBAC policy update (Runtime SSOT — add `export` action to `contact`/`deal`/`activity`, grant to `admin`/`manager`, per task 1.4 deployment note) is a deployment step executed during rollout, not in this offline environment. Archive once the policy is applied in production so export endpoints become live.
