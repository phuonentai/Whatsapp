## Why

Every client-facing commercial document — cotización today, invoice templates / cuentas de cobro / remisiones later — needs consistent company branding (logo, colors, letterhead, terms). Today the platform has **zero** document branding infrastructure: no logo upload, no letterhead, no terms footer, no validity defaults. The playbook guiones already promise clients "Te preparamos la cotización y te la enviamos hoy", but the branded-document foundation to back that promise does not exist. The only branded artifact in the system is the Siigo e-invoice PDF, whose branding lives inside Siigo's configuration (opaque, provider-locked, not reusable for platform-owned documents).

This change establishes the **shared, org-scoped branding foundation** that all future document capabilities consume. It is deliberately provider-agnostic and document-agnostic: the same branding configuration powers cotizaciones now and invoice templates later, so we build the seam once instead of per-document.

## What Changes

- **New `document-branding` capability**: an org-scoped branding configuration entity — logo image (stored via the existing file-asset manager), primary/accent colors, letterhead header text, terms footer text, default validity days, default IVA percentage, and document numbering prefix defaults.
- **`DocumentBrandingProvider` domain interface**: a single seam through which any document renderer (quote renderer in `add-quote-delivery`, future invoice-template renderer) resolves the org's branding. Renderers consume branding via the interface; they never touch storage or file assets directly.
- **Settings UI surface**: a "Marca / Documentos" section in dashboard settings where an `org:manage` member uploads the logo (image validation, size/type constraints), sets colors, letterhead, terms, and defaults. Preview of a sample document header.
- **Audit**: branding changes (create/update) SHALL emit audit events (`branding_updated`) with acting `stytch_member_id`, matching the repo's `admin-panel-audit-log` convention.
- **Schema**: new table `document_branding.org_branding` (single row per org, org-scoped, timestamps, JSONB for extensible fields) + SQLC queries.
- **Extensibility seam**: branding config is versioned in time (updated_at) so later "document template" capabilities can snapshot branding at document-issue time (a quote must render with the branding that was active when it was sent, even if the org changes branding later).

## Capabilities

### New Capabilities
- `document-branding`: org-scoped branding configuration (logo file asset, colors, letterhead, terms footer, validity days, default IVA, prefix defaults) exposed through a domain provider interface, with settings UI, audit events, and time-snapshot semantics for document rendering.

### Modified Capabilities
- `settings-ui`: adds the branding configuration surface to dashboard settings. (Delta spec only; no behavioral change to existing settings requirements.)

## Impact

- **Go backend**: new module `internal/modules/branding/` (domain + app + infra); `DocumentBrandingProvider` interface wired in DI; `FileAssetStore` reused for logo storage; routes under `/api/branding` (org-scoped, `org:manage` write / `org:view` read).
- **Database**: new migration creating `document_branding.org_branding`; no changes to existing tables.
- **Frontend**: new branding settings tab in `next_b2b_starter/app/dashboard/settings/`; logo upload component with client-side validation.
- **Auth boundary / Stytch**: no Stytch contract changes; branding is org-scoped data keyed by the authenticated org context. No new identity tables.
- **Dependencies**: no new external providers. Logo bytes go through the existing file-asset manager (storage abstraction already used by the documents module).
- **Rollback strategy**: Git — revert the change commit(s); DB — drop `document_branding.org_branding` via down migration; Stytch tenant policy — unaffected.

## Non-Goals

- No PDF/document generation in this change (that is `add-quote-delivery`; the renderer consumes this branding).
- No quote/entity work (that is `add-quote-documents`).
- No invoice template rendering (future change; the provider interface is the extension point).
- No per-document branding overrides in this change — a document inherits org branding; per-document overrides are deferred.
- No logo processing beyond validation/storage (no automatic resizing/optimization in v1; defer if needed).

## Assumptions

- A single branding configuration per organization is sufficient (no multi-brand orgs in the current product; if multi-brand arrives, the JSONB extensible field + time-snapshot design can accommodate without schema change).
- The file-asset manager's existing storage (`storage_path`-based) is adequate for logo bytes; no new storage layer is introduced.
- Logo file constraints (PNG/JPEG/SVG?, max dimensions/size) will be validated at upload; SVG support is a question to resolve in design (safe rendering considerations).
