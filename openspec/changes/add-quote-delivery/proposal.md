## Why

Once quotes are first-class (`add-quote-documents`) and org branding exists (`add-document-branding`), the promise the playbook guiones make — "te enviamos la cotización" — must be fulfilled: a branded, itemized cotización must reach the client's WhatsApp. The invoice flow already established the delivery pattern: generate/host the document, notify via the existing text send path with a link (`factura_lista` template + payment link). Cotización delivery inherits that proven pattern — link-first, zero changes to the WhatsApp outbound send path (which is text-only today: `SendMessage(ctx, orgID, convID, content string)`).

This change delivers the **document side**: template-driven PDF rendering (extensible to invoice templates later), branded via `DocumentBrandingProvider`, stored as file assets with a shareable link, and sent through the existing WhatsApp path.

## What Changes

- **Template-driven renderer seam**: a `DocumentRenderer` domain interface with a template registry — named document templates (`cotizacion` now; `cuenta_cobro`, invoice wrappers later). Rendering resolves org branding via `DocumentBrandingProvider` and renders to PDF bytes. The template registry is the explicit extension point for future invoice templates — a later `add-invoice-templates` change plugs a new template into the same engine without touching the quotes domain.
- **PDF generation (Go server-side)**: rendering library with embedded Unicode font support for Spanish (ñ, á, é); no precedent exists in the repo (no PDF deps in go.mod today), so the specific library choice is a spike item in tasks.
- **File asset + shareable link**: generated PDFs stored via the file-asset manager; a public/shared-link mechanism for hosting the PDF (the invoice flow points at Siigo's external CDN — our file assets are org-internal today, so a share mechanism is a spike item).
- **WhatsApp delivery**: on `enviada`, send the quote link via the existing text send path (template message), matching the invoice notification pattern (`notified_status`-style once-per-transition guard); a `DocumentSender` seam so a future media/document send implementation can replace the link without touching the quote service.
- **Deal activity + notification**: quote-sent activity recorded on the deal; send failure SHALL NOT fail quote state transitions (log warning, per invoicing precedent).
- **Extensibility**: renderer/template registry + branding snapshot-at-issue semantics mean future invoice templates reuse branding, renderer, file-asset, and sender seams unchanged.

## Capabilities

### New Capabilities
- `quote-delivery`: template-driven PDF rendering of branded quotes, file-asset storage with shareable links, and WhatsApp link delivery through the existing send path — behind `DocumentRenderer`/`DocumentSender` seams designed for future invoice templates.

### Modified Capabilities
- `quotes`: the `enviada` transition now renders + sends the quote document (delivery behavior added on top of the entity core).

## Impact

- **Go backend**: `internal/modules/quotes/infra/renderer/` (renderer + template registry + branding consumer), `internal/modules/quotes/infra/delivery/` (WhatsApp link sender via existing outbound path + file-asset link host), DI wiring; new deps: PDF rendering library (spike decision).
- **Database**: no new tables (PDFs are file assets; quote rows already exist). Possibly a `file_assets` purpose/category for quote PDFs (reuse existing lookup tables).
- **Frontend**: none required for MVP (delivery flows over WhatsApp, mirroring the invoice pattern); optional later: view/regenerate link on the deal page.
- **Auth boundary / Stytch**: no changes; org-scoped as always.
- **Dependencies**: PDF rendering library (new); otherwise existing file-asset + WhatsApp outbound.
- **Rollback strategy**: Git — revert; DB — file-asset rows are non-critical (orphaned assets can be cleaned or left; no state machine impact); Stytch — unaffected.

## Non-Goals

- No media/document WhatsApp send in this change (link-first; `DocumentSender` seam reserves the upgrade path).
- No invoice template rendering (future `add-invoice-templates` change — same engine, new template).
- No public internet hosting infra (shareable link leverages existing file-asset serving; if absent, the spike decides between signed URLs vs a minimal public view route).
- No branding UI (that is `add-document-branding`).
- No quote CRUD/state machine (that is `add-quote-documents`).

## Assumptions

- Link-first delivery is acceptable for MVP (matches the invoice flow's UX; clients already receive invoice links over WhatsApp).
- A minimal public/shared-link mechanism can be built on the existing file-asset layer (spike item 1 verifies; fallback is a dedicated public quote-view route rendering an HTML view of the quote — no PDF needed for the fallback).
- The PDF library choice (Go native vs. HTML-to-PDF) is resolved by spike item 2; both keep the `DocumentRenderer` interface stable.
- Quote templates are single-layout in v1; layout variants (vertical-specific) are a later template-registry extension.
