## Context

Verified current state:

- WhatsApp outbound is **text-only**: `OutboundService.SendMessage(ctx, orgID, convID, content string)` — no media/document send exists anywhere.
- The invoice notification pattern is the delivery precedent: `factura_lista` template message with the Siigo PDF URL + MercadoPago payment link via the existing send path; send failure logged, not fatal; once-per-transition via `notified_status` (invoicing module).
- No PDF generation exists in the repo: no PDF libraries in `go.mod`, no HTML-to-PDF service, no renderer abstraction. Greenfield.
- File-asset manager exists (`FileAssetStore`, `storage_path`-based) with purpose/category lookup tables; used by the documents module for uploaded PDFs.
- `add-document-branding` defines `DocumentBrandingProvider` + snapshot key convention; `add-quote-documents` defines the quote entity with `branding_snapshot_key`, line items, and the `enviada` transition.

## Goals / Non-Goals

**Goals:**
- Render a branded, itemized cotización PDF server-side from a template
- Host the PDF with a shareable link (public/shared-link mechanism)
- Send the link to the client via the existing WhatsApp text path on `enviada`
- `DocumentRenderer` + `DocumentSender` seams so future invoice templates (and media sends) plug in without touching the quotes domain
- Send failure non-fatal (quote state still advances), once-per-transition notification

**Non-Goals:**
- No media/document WhatsApp send (link-first; seam reserves upgrade)
- No invoice template rendering (future `add-invoice-templates` change)
- No branding UI (that is `add-document-branding`)
- No quote CRUD/state machine (that is `add-quote-documents`)
- No vertical-specific layout variants (single template in v1)

## Decisions

### 1. `DocumentRenderer` + template registry

**Chosen:**

```
type DocumentRenderer interface {
    Render(ctx context.Context, doc *domain.DocumentData, tpl TemplateID) ([]byte, error)
}

type TemplateID string  // "cotizacion" now; "cuenta_cobro", "invoice_wrapper" later

type TemplateRegistry interface {
    Get(t TemplateID) (Template, error)  // returns layout + variable binding
}
```

The quotes service hands the renderer a transport-free `DocumentData` value object (quote line items, totals, client info, branding snapshot) — the renderer resolves branding via `DocumentBrandingProvider` using the snapshot key. Future invoice templates are **new entries in the registry**, no domain changes.

**Alternatives considered:**
- *Renderer that takes the quote entity directly* — rejected; couples renderer to quotes domain; the registry pattern keeps it document-agnostic for invoice templates.

### 2. PDF library (spike-decided, interface-stable)

**Chosen (default):** Go server-side PDF generation with a maintained library supporting embedded Unicode fonts for Spanish text (candidates: gofpdf, maroto, or a templated-HTML→PDF path via a headless renderer). The specific choice is Spike 2.1; both candidates keep `DocumentRenderer` unchanged. No FE-side rendering (keeps the backend as the byte owner, matching the repo's Go-first architecture).

**Rationale:** No precedent in repo; PDF lib choice is a spike because Spanish glyph coverage (ñ, á, é) and table layout (line items) are the real risks, not the API surface.

### 3. Shareable link (spike-decided)

**Chosen (default):** File asset stored with a `quote_pdf` purpose; a shared-link mechanism serves it without org auth. Spike 3.1 verifies whether the file-asset layer can serve public/shared URLs today; if not, the fallback is a minimal **public quote-view route** (`GET /public/quotes/:token`) rendering an HTML view of the quote (no PDF needed) — the WhatsApp link then points at the HTML view, matching how the invoice flow points at an external URL.

**Rationale:** The invoice flow's `pdf_url` is an external Siigo CDN URL; we must replicate "client can open it" without auth. Either mechanism satisfies the requirement; the spike picks the cheaper one.

### 4. Delivery: link via existing text path

**Chosen:** On `enviada`, after successful render + storage: send template message with quote link (and a payment-teaser optional) via `OutboundService.SendMessage`. `notified_status`-style once-per-transition guard (a quote row field `notified_status` on `crm.quotes`, mirroring invoicing). Send failure logs warning and does NOT revert state — client can still be re-notified from the deal page later.

**Alternatives considered:**
- *Media/document send* — rejected for v1: `SendMessage` is text-only, media upload + session + compliance review is a real outbound-pipeline change; the `DocumentSender` seam reserves it as a follow-up without touching the quote service.

### 5. `DocumentSender` seam

**Chosen:**

```
type DocumentSender interface {
    SendDocument(ctx context.Context, orgID, convID int32, ref *domain.SentDocumentRef) error
}
```

Implemented as `linkSender` (text message with URL) in this change. A future `mediaSender` (WhatsApp document type) swaps in via DI. The quote service depends on the interface only.

### 6. Wiring + DI

**Chosen:** Renderer, registry, branding provider, file-asset host, sender wired in `init_mods.go`; the `enviada` transition handler in the quotes app service calls: render → store → host → send (each step isolated; failures logged, state advances). Follows the invoicing module's DI naming pattern (named bindings per seam).

## Risks / Trade-offs

- **PDF fidelity (fonts/layout)**: the primary risk; mitigated by spike choice + a golden-file render test asserting Spanish text and totals render correctly.
- **Public link exposure**: a public URL leaks quote content if link is shared beyond the client; mitigated by unguessable token, no org data beyond the quote, and optional expiry matching `valid_until`. Trade-off accepted for MVP parity with invoice flow.
- **Link-only UX**: clients get a link, not a PDF attachment; accepted (invoice flow already conditions this behavior) and the sender seam is the upgrade path.
- **Renderer dependency weight**: Go PDF libs add binary/font weight; acceptable; a later template move to HTML→PDF keeps the interface stable.
