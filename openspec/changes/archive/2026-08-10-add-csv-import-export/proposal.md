## Why

Colombian SMEs run on WhatsApp but keep their books in Excel and Google Sheets. Today the platform has zero file interoperability: the only export is the Habeas Data JSON bundle (agent module), the `files` module explicitly blocks `.csv`/`.xlsx` uploads, and there is no CSV/XLSX library anywhere in the stack. The loop "mis contactos están en Excel" (onboarding) and "reporte para el contador" (month-end) is broken — data lives behind the CRM UI and can only be moved by retyping. CSV import/export is the cheapest, highest-value interoperability the MVP can ship without new dependencies.

## What Changes

- **CSV export** of the four main CRM entities: contacts, companies, deals, activities — streamed server-side with Spanish headers, UTF-8 BOM (Excel-on-Windows accent compatibility), and `Content-Disposition: attachment`.
- **CSV import** of contacts from a strict template (downloadable), with per-row validation, dedupe-by-phone that skips existing records, row-level error reporting, and hard size/row caps.
- **New RBAC `export` action** (Stytch RBAC policy SSOT) gating bulk download on the `contact`, `deal`, and `activity` resources, granted to `admin`/`manager` roles. Import is gated on the existing `contact:manage`.
- **Export hardening**: CSV formula-injection sanitization (`=`, `+`, `-`, `@` cell prefixes), PII masking for withdrawn-consent contacts (compliance invariant), org scoping from the authenticated request context only, and an audit log entry per bulk export.
- **Frontend**: export buttons on the Contactos/Empresas/Negocios/Actividades views (fetch+blob with the Stytch session token) and an import modal on Contactos with template download and result summary.
- **Security relaxation, scoped**: the global `files` module ban on `.csv` uploads stays; only the dedicated import route accepts CSV, with content sniffing and size limits.

## Capabilities

### New Capabilities
- `data-transfer`: CSV export/import of CRM entities — endpoint contract, streaming CSV writer, Spanish column templates, import validation/dedupe semantics, formula-injection sanitization, withdrawn-consent masking, org scoping, and audit logging.

### Modified Capabilities
- `stytch-authorization`: RBAC policy requirement change — the `contact`, `deal`, and `activity` resources gain an `export` action that must exist in the Stytch RBAC policy (and in the Go fallback maps) before bulk-export endpoints are enabled.
- `crm-frontend`: CRM views gain export/import entry points and result reporting.

## Impact

- **Go backend** (`go-b2b-starter/`):
  - `internal/modules/crm/` — new `export/` + `import/` handlers and a shared CSV writer service; routes registered with `auth.RequirePermissionFunc` (`contact:export`, `deal:export`, `activity:export`, `contact:manage`, `contact:view`).
  - `internal/modules/auth/` — fallback permission maps in `rbac.go`/`roles.go` gain `contact:export`/`deal:export`/`activity:export` for `manager` and `admin` (dev/mock parity only; Stytch policy is SSOT).
  - `internal/modules/agent/` — audit events for export (reuses compliance logging precedent).
- **Stytch RBAC policy (Runtime SSOT)**: add `export` action to the `contact`, `deal`, and `activity` resources and grant it to `admin` and `manager` roles via the Stytch RBAC Policy API (`GET /v1/b2b/rbac/policy`, `PUT /v1/b2b/rbac/policy`). Policy is cached in Redis (`rbacPolicyCacheKey`) — cache key must be versioned at rollout so the new action takes effect without waiting for TTL expiry.
- **Frontend** (`next_b2b_starter/`): `app/dashboard/crm/` list views (export buttons), Contactos import modal, template download — no new npm dependencies (server parses CSV; client uploads/downloads blobs).
- **Database**: no migrations. Existing `crm.contacts` unique constraint `(organization_id, phone_number)` drives import dedupe.
- **Dependencies**: none new. Go stdlib `encoding/csv` only.
- **Rollback strategy**:
  - Git: revert the change commit(s); endpoints and permission fallback maps disappear with the code.
  - Stytch tenant policy: revert the RBAC policy update (remove `export` action grants, restore prior policy via the Stytch RBAC Policy API) and restore the prior `rbacPolicyCacheKey` version. Until the cache key is reverted, any stale grant resolves only to endpoints that no longer exist (404), so no data exposure window.

## Non-Goals

- **No .xlsx support** in this change: CSV only; `.xlsx` (via `excelize`/SheetJS) is deferred until a network-enabled environment can install dependencies, and is a drop-in swap on the same CSV writer seam.
- **No Google Sheets sync**: bidirectional Sheets sync requires Google Cloud OAuth2 + token storage that would conflict with the no-credentials-in-PostgreSQL invariant; deferred to a post-launch change.
- **No import of companies, deals, or activities**: contacts-only import in v1.
- **No local credential storage**: this change introduces no local storage of Stytch session, Google, or Stytch RBAC credentials; authorization continues to resolve exclusively from the Stytch RBAC policy API with the existing Redis cache.
- **No change to the general file-upload surface**: the `files` module's ban on Office/CSV uploads remains in force everywhere except the single scoped import route.
