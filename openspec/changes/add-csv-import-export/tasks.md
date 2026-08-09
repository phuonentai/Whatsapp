## 1. RBAC export action [BE-INFRA] [OPS-GOV]

- [ ] 1.1 Add typed permission constants `PermContactExport`, `PermDealExport`, `PermActivityExport` to the auth package (`NewPermission("contact","export")` etc.). Verify: `go build ./...`
- [ ] 1.2 Grant the three export permissions to `RoleManagerInfo` and `RoleAdminInfo` in `internal/modules/auth/rbac.go` (Go fallback/mock parity; Stytch policy remains SSOT). Verify: `go test ./internal/modules/auth/...`
- [ ] 1.3 Version the RBAC policy cache key (`rbacPolicyCacheKey` in `internal/modules/auth/adapters/stytch/rbac_policy.go`) so the new `export` action applies without waiting for the cache TTL. Verify: cache key constant changed; unit test references updated
- [ ] 1.4 Document the Stytch RBAC policy update (add `export` action to `contact`/`deal`/`activity` resources, grant to `admin`/`manager`, ensure `export` listed in each resource's actions array for wildcard expansion) as a deployment step in tasks notes. Verify: notes present; policy change executed during deployment

## 2. CSV export backend [BE-INFRA]

- [ ] 2.1 Create `internal/modules/crm/app/services/csv_service.go` with a shared `ExportService` that writes a UTF-8 BOM, Spanish headers, and streams rows via a pagination callback through `encoding/csv`. Verify: `go build ./...`
- [ ] 2.2 Implement `csvSanitizeCell` in the CSV service: prefix a single quote when a cell starts with `=`, `+`, `-`, or `@`. Verify: unit test asserts sanitized output for each trigger prefix
- [ ] 2.3 Implement entity mappers for contacts, companies, deals, activities with Spanish column headers matching the CRM list views, applying withdrawn-consent PII masking (`[TELEFONO]`/`[NOMBRE]`/`[EMAIL]`/`[DOCUMENTO]`) via the contact consent state. Verify: unit tests assert header row and masked values
- [ ] 2.4 Add four export handlers on `CRMHandler` (`ExportContactos`, `ExportEmpresas`, `ExportNegocios`, `ExportActividades`) that set `Content-Type: text/csv` + `Content-Disposition: attachment`, resolve org scope from `auth.GetRequestContext(c)`, paginate via the existing `List` repository methods, and stream through the CSV service. Verify: `go build ./...`; handler tests assert BOM prefix, content-type, and org scoping
- [ ] 2.5 Record an export audit event as a `sistema` activity (`crm.activities`, `ActivityTypeSistema`) with entity, row count, member, and timestamp. Verify: unit test asserts a `sistema` activity row is created after export

## 3. CSV import backend [BE-INFRA]

- [ ] 3.1 Add `GET /crm/import/contactos/template.csv` handler serving the exact import template (required `teléfono`, `nombre`; optional `email`, `tipo_documento`, `numero_documento`, `empresa`, `origen`, `estado`) with two example rows, gated `contact:view`. Verify: handler test returns the template bytes
- [ ] 3.2 Implement `POST /crm/import/contactos` handler: parse CSV, enforce the 5000-row cap, upload size cap, and content sniff (reject non-CSV with 400) before any write; validate rows, dedupe by existing phone (skip, never overwrite), create via `contactService.Create`; return `{importados, omitidos, errores:[{fila, razon}]}`. Verify: `go build ./...`; tests cover valid rows, invalid rows, duplicate skip, and the summary shape
- [ ] 3.3 Register all six export/import routes in `internal/modules/crm/routes.go` under the existing `/crm` group with `RequirePermissionFunc` gates (`contact:export` ×3 contact-group endpoints, `deal:export`, `activity:export`, `contact:view` template, `contact:manage` import) and matching feature guards. Verify: `go build ./...`; route table inspected

## 4. Frontend [FE-NEXT]

- [ ] 4.1 Add export actions to the Contactos, Empresas, Negocios, and Actividades list views that fetch the CSV with the session token and trigger a blob download (fetch+blob; no `window.location`); hide the action when the user lacks the `export` permission. Verify: `npx tsc --noEmit`; `pnpm lint` at documented baseline
- [ ] 4.2 Add a Contactos import modal (visible with `contact:manage`): downloadable template link, CSV file picker, submit to `POST /crm/import/contactos`, and a result summary showing imported/omitted/error counts with row numbers, in Colombian Spanish. Verify: `npx tsc --noEmit`; manual check on `pnpm dev`

## 5. Verification gate [OPS-GOV]

- [ ] 5.1 Run backend gate: `go build ./...`, `go vet ./...`, `go test ./...`. Verify: all exit 0; record results here
- [ ] 5.2 Run frontend gate: `npx tsc --noEmit`, `pnpm lint`, `pnpm build`. Verify: tsc and build exit 0; lint at documented baseline (no new violations); record results here
- [ ] 5.3 Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>`. Verify: entry present

### Verification results (gate)

- To be filled during apply.
