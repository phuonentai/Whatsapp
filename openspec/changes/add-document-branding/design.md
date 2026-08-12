## Context

Verified current state:

- No document branding exists: no logo upload, no letterhead, no terms footer anywhere in the system. Verified: no `logo`/`brand` references in specs or FE settings beyond marketing pages; the only branded artifact is the Siigo e-invoice PDF whose branding lives inside Siigo's configuration.
- The repo already has an org-scoped file-asset manager (`file_asset_store.go`, `storage_path`-based) used by the documents module — logo bytes can reuse it without a new storage layer.
- Settings UI exists (`settings-ui` capability, `next_b2b_starter/app/dashboard/settings/`) with per-capability sections (Siigo integration section is the precedent for a provider-agnostic "Marca / Documentos" section).
- Audit conventions exist (`admin-panel-audit-log`): events carry `stytch_member_id`, org-scoped.
- The invoicing module already solves the analogous problem of per-org config with lifecycle states; branding is simpler (single row, no state machine).

## Goals / Non-Goals

**Goals:**
- One org-scoped branding config consumed by every future document renderer through a single domain interface (`DocumentBrandingProvider`)
- Logo stored via the existing file-asset manager; validated at upload
- Settings UI for `org:manage` members; audit trail on changes
- Time-snapshot semantics: documents capture branding as of issue time, so later branding changes don't mutate historical documents

**Non-Goals:**
- No PDF/render work (deferred to `add-quote-delivery`)
- No per-document branding overrides (deferred)
- No multi-brand orgs (deferred; JSONB extensibility reserves it)
- No logo image processing/optimization (v1 validates + stores)

## Decisions

### 1. Single row per org, JSONB for extensible fields

**Chosen:** `document_branding.org_branding` — one row per org with typed core columns (logo_file_asset_id, primary_color, accent_color, letterhead_text, terms_footer, validity_days, default_iva_percent, quote_number_prefix, updated_at) plus a `config` JSONB for future fields.

**Rationale:** One row avoids an over-engineered multi-entity design for v1; JSONB gives schema-escape-hatch for fields that don't yet exist (invoice-template-specific options). This matches the repo's use of JSONB (`deal.Metadata`, playbook payloads).

### 2. `DocumentBrandingProvider` domain interface

**Chosen:**

```
type DocumentBrandingProvider interface {
    GetBranding(ctx context.Context, orgID int32) (*domain.OrgBranding, error)
}
```

Implemented in `infra/branding_repository.go`; consumed by renderers (quotes now, invoice templates later). Domain models MUST NOT import file assets or HTTP transport — the provider returns the resolved domain value object (with the logo's public URL already resolved by the infra layer, not a raw asset ID).

**Rationale:** Mirrors the `InvoicingProvider`/`ProviderRouter` seam pattern: renderers depend on the interface, never on storage. A later `add-invoice-templates` change consumes the same provider unchanged.

### 3. Logo as file asset with purpose/category

**Chosen:** logo uploaded via `POST /api/branding/logo` → validated (content type whitelist PNG/JPEG, size limit e.g. ≤ 2MB, dimension sanity) → stored via `FileAssetStore` with a branding category/purpose → `logo_file_asset_id` set on the org row. SVG is **not** accepted in v1 (safe-rendering/XSS concerns in PDF renderers; PNG/JPEG only).

**Alternatives considered:**
- *SVG allowed* — rejected; SVG rendering in PDF libs is inconsistent and SVG upload is a vector concern (scripts in SVG). Defer.
- *Direct base64 in the org row* — rejected; duplicates bytes, loses file-manager features (cleanup, audit, existing serving).

### 4. Snapshot semantics

**Chosen:** `updated_at` is the branding version. Document capabilities capture a `branding_snapshot_key` (org_id + branding updated_at) at issue time and resolve branding by that key when rendering later (renderer re-reads the row but the *quote row* stores the snapshot key; if branding changed since issue, the renderer uses the snapshot values persisted in the quote's payload, or documents the version used).

**Rationale:** A re-sent or reprinted cotización must look like it did when sent. The quote row in `add-quote-documents` stores the snapshot key; the renderer in `add-quote-delivery` renders with the matching branding version. Keeps this change small (just `updated_at` + key convention) while making history correct.

### 5. API + RBAC

**Chosen:** `GET /api/branding` (org:view), `PUT /api/branding` (org:manage), `POST /api/branding/logo` (org:manage, multipart). All org-scoped via authenticated org context. 403 with Spanish error on missing permission (repo convention).

### 6. Audit

**Chosen:** `branding_updated` event on every mutation with acting `stytch_member_id` + changed field list; `logo_updated` for logo replacement (old asset cleanup follows file-manager deletion semantics).

## Risks / Trade-offs

- **Logo display fidelity**: PDF renderers may need rasterization/sizing; mitigated by PNG/JPEG whitelist and dimension sanity checks; exact PDF embedding behavior is a spike in `add-quote-delivery`.
- **Single-row race**: concurrent branding writes — mitigated by transaction + `updated_at` optimistic concurrency (last-write-wins accepted for v1; audit records both).
- **JSONB escape hatch discipline**: without care, `config` JSONB can become a dumping ground; mitigated by documenting known future keys in the spec and adding them as typed columns when they stabilize.
- **Branding snapshot resolution across capabilities**: the snapshot key convention spans three changes; mitigated by defining the key format (org_id:updated_at) in this change's spec so `add-quote-documents` and `add-quote-delivery` implement against it.
