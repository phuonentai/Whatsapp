## Context

The platform is a WhatsApp-first B2B SaaS for Colombian SMEs (MVP scope). Today the CRM has zero file interoperability: the only export is the Habeas Data JSON bundle in the agent module, the `files` module deliberately bans `.csv`/`.xlsx` uploads, and no CSV/XLSX dependency exists in either `go.mod` or `package.json`. SME owners keep contacts in Excel and must send month-end reports to their accountant; the MVP cannot move data in or out except through the UI.

Enforcement stack (verified): `auth.RequirePermissionFunc(resource, action)` middleware (internal/modules/auth/middleware.go:505); permissions resolve from Stytch RBAC policy via `RBACPolicyService` (adapters/stytch/rbac_policy.go:45), Redis-cached under `rbacPolicyCacheKey`, with a Go fallback map in `rbac.go`/`roles.go` for standard roles/dev. CRM routes (internal/modules/crm/routes.go) use `contact`, `deal`, `pipeline` resources; companies and activities ride the `contact` resource. Handlers read org scope from `auth.GetRequestContext(c).OrganizationID`, never from input.

Constraint: this environment has no network, so the design MUST NOT introduce new dependencies — Go stdlib `encoding/csv` only.

## Goals / Non-Goals

**Goals:**
- Streamed CSV export of contacts, companies, deals, activities, gated by a new RBAC `export` action.
- Scoped CSV import of contacts with template, per-row validation, dedupe-skip, and a summary.
- Hardening: formula-injection sanitization, withdrawn-consent PII masking, org scoping, audit logging.
- Zero migrations, zero new dependencies, offline-testable.

**Non-Goals:**
- No `.xlsx` generation (deferred; drop-in on the same writer seam once `excelize`/SheetJS installable).
- No Google Sheets sync (needs OAuth2 token storage that conflicts with the no-credentials-in-PostgreSQL invariant).
- No import of companies/deals/activities.
- No change to the general `files` upload surface.
- No new audit table (see decision D4 — reuse `sistema` activities).

## Decisions

### D1: RBAC — new `export` action in Stytch policy, not `view`
Bulk download of all org PII is materially different from paging through the UI; `view` is granted to `member`, which would let any staff member exfiltrate the whole CRM as CSV. So export rides a new `export` action on the `contact`, `deal`, and `activity` resources, granted to `admin`/`manager`.
- Add the action + grants via the Stytch RBAC Policy API; the `Resources` entries must list `export` in their actions array so `expandWildcardActions` (rbac_policy.go:171) expands `contact:*` to it.
- Mirror in Go fallback (`rbac.go` RoleManagerInfo/RoleAdminInfo) for dev/mock-auth parity.
- Version `rbacPolicyCacheKey` at rollout so the action applies without waiting for the cache TTL.
- Import uses existing `contact:manage` (same as `CreateContacto`), template uses `contact:view`.
- **Alternatives rejected**: `contact:view` (member exfiltration), `contact:manage` for export (asymmetric, and members may hold manage).

### D2: Shared streaming CSV writer service
New `internal/modules/crm/app/services/csv_service.go`: one `ExportService` that streams any entity through a pagination loop (`repo.List(ctx, orgID, limit, offset)`), writing through `encoding/csv` into `c.Writer`, preceded by the UTF-8 BOM (`\xEF\xBB\xBF`). Handler sets `Content-Type: text/csv` and `Content-Disposition: attachment; filename=*.csv` before the service streams. BOM is mandatory — Excel-on-Windows renders Spanish accents incorrectly without it.

### D3: Formula-injection sanitization at the cell writer
Every cell passes `csvSanitizeCell`: if the trimmed value starts with `=`, `+`, `-`, or `@`, prefix a single quote. Applied centrally in the CSV writer so all four exporters inherit it. Rationale: Colombia SMEs open these files in Excel; unsanitized cells are a real exfiltration path. Mitigation is one function, unit-tested.

### D4: Audit via existing `sistema` activity type
Spec requires persisting an audit event per export. `crm.activities` already has `ActivityTypeSistema` (domain/activity.go:14) — exports insert a `sistema` activity (entity, row count, member, timestamp) with no migration. **Alternative rejected**: a new `audit_export` table (violates the zero-migration constraint); logger-only (does not satisfy "persist").

### D5: Withdrawn-consent masking at the mapper layer
The compliance invariant (mask PII when `consent_status = 'withdrawn'`) is enforced in the CSV mapper, not the writer: each entity mapper takes the contact's consent state and swaps PII columns for `[TELEFONO]`/`[NOMBRE]`/`[EMAIL]`/`[DOCUMENTO]`. Same placeholder set as the Habeas Data export for consistency.

### D6: Import — validate via existing contact service, dedupe via phone pre-check
`POST /api/crm/import/contactos` parses the CSV in one pass, validates rows, and calls the existing `contactService.Create` (reuses Spanish validation + `tipo_documento` rules). Dedupe: pre-query existing phones for the org (`List` by phone) and skip matches — never overwrite, per the "SME files are stale, CRM edits are sovereign" decision. Response is `{importados, omitidos, errores:[{fila, razon}]}`; errors are also downloadable so the SME can fix and re-import. Limits: 5000 rows, upload size cap, content sniff (reject non-CSV via first-line/extension check) before any write. Import is sync — SME volumes are small.

### D7: Frontend — fetch+blob, no window.location
Export buttons on the four list views use `fetch` with the Stytch session token then `URL.createObjectURL` download — the same pattern as the compliance section (settings/components/compliance-section.tsx). `window.location` is rejected because it cannot attach the session token. Import modal on Contactos: template link → file picker → summary. No new npm deps (client does not parse CSV).

### D8: Route + permission wiring
New routes in `internal/modules/crm/routes.go`, all under the existing `/crm` group (auth + org_context + entitlement middleware already applied):
```
GET  /crm/export/contactos.csv     contact:export
GET  /crm/export/empresas.csv      contact:export
GET  /crm/export/negocios.csv      deal:export
GET  /crm/export/actividades.csv   activity:export
GET  /crm/import/contactos/template.csv  contact:view
POST /crm/import/contactos         contact:manage
```
Handlers use `auth.GetRequestContext(c)` for org scope, mirroring `ListContactos` (handler.go:61). New typed permission constants (`PermContactExport`, `PermDealExport`, `PermActivityExport`) in the auth package for the Go fallback maps.

## Risks / Trade-offs

- **[Policy cache staleness at rollout]** → Version `rbacPolicyCacheKey`; stale grants resolve only to endpoints that 404 once the cache turns over — no exposure window.
- **[Wildcard roles silently miss export]** → `expandWildcardActions` only expands `*` to actions declared on the resource; tasks assert `export` is in each resource's actions array in the Stytch policy, and tests cover the expansion path.
- **[CSV injection still possible via crafted multi-cell payloads]** → Sanitize is applied to every exported cell; remaining edge (formulas spanning cells) is out of scope, documented.
- **[Import overwrites CRM edits]** → Dedupe is skip-only (D6); unit tests assert existing contacts are untouched.
- **[Large exports hold memory]** → Streaming + pagination (D2); row cap on import prevents unbounded parse.
- **[Two sources of truth for perms (Stytch + Go fallback)]** → Accepted, pre-existing pattern; Stytch policy is authoritative in production, Go maps are dev/mock fallback per governance. No new hardcoded source added beyond the fallback maps.

## Migration Plan

1. **RBAC first**: Stytch RBAC policy (add `export` action to `contact`/`deal`/`activity`, grant `admin`/`manager`), Go fallback constants + maps, version `rbacPolicyCacheKey`.
2. **Backend**: CSV export service + 4 handlers; import service + template + handler; audit wiring; routes.
3. **Tests**: permission matrix (mock identity: member denied, admin/manager granted), org-scope IDOR attempt, masking, sanitize, BOM bytes, import valid/invalid/dup, 401/403.
4. **Frontend**: export buttons + import modal.
5. **Gate**: `go build ./...`, `go vet ./...`, `go test ./...`, `npx tsc --noEmit`, `pnpm lint` (documented baseline). E2E deferred per repo pattern.
6. **Rollback**: Git revert removes endpoints + fallback maps. Stytch: revert policy via RBAC Policy API + restore cache key version. No migrations to reverse.

## Open Questions

- Whether `manager` should keep `export` permanently or only `admin` — default both; narrow later via Stytch policy without code change.
- Whether export should respect entity feature gating (e.g., hide Negocios export when deals feature disabled) — defaults to same `features.Require` guards as the list views.
