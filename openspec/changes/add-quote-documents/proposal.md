## Why

The playbook pipeline defines a **"Cotización"** stage and its guiones promise to send the client a cotización ("Te preparamos la cotización y te la enviamos hoy"), but **no quote entity exists**. Deals carry only a lump-sum `monto`; there is no itemized quote, no versioning, no approval state, no revision loop. Colombian B2B reality is that quotes get revised (descuentos, mejor precio, quita), and each revision is commercially and legally distinct. An agent cannot track which version the client approved, and there is no audit trail of what was offered when.

This change introduces the **first-class, versioned quote entity** (Option B): quotes are 1:N per deal, each quote is versioned, has line items, and moves through a state machine (borrador → enviada → aprobada / rechazada / vencida). It is the core domain foundation; delivery (PDF/WhatsApp) comes in `add-quote-delivery`, branding in `add-document-branding`.

## What Changes

- **New `quotes` capability**: org-scoped quote entity — `crm.quotes` (org_id, deal_id FK, version, status, valid_until, currency COP, branding snapshot ref, timestamps) with `UNIQUE(organization_id, deal_id, version)`; line items in `crm.quote_items` (quote_id, description, sku ref optional, quantity, unit_price, discount %, IVA %, line totals). No PDF, no WhatsApp — pure domain + API + persistence.
- **State machine** with guarded transitions: `borrador → enviada → aprobada | rechazada`, plus `vencida` (validity expiry) and a revision loop (`rechazada/vencida → nueva versión`). Unknown transitions rejected, mirroring the invoicing connection state-machine pattern.
- **Deal integration**: deal `monto` syncs from the active quote's total when a quote is aprobada; deal activity recorded on quote create/send/aprobada/rechazada (matching the `DealStageListener` activity convention); a deal can move to `facturado` only from an aprobada quote state (guard, advisory).
- **Audit**: quote lifecycle events (created/sent/approved/rejected/expired) with acting `stytch_member_id`.
- **Extensibility**: quote stores a branding snapshot key (org branding at issue time) and a `payload` JSONB field so later capabilities (line-item tax rules, payment terms, delivery terms, AI-assisted drafting) can extend without schema churn. The `QuoteRepository` is an interface seam so `add-quote-delivery` can extend behavior without touching the domain.
- **Schema**: `crm.quotes`, `crm.quote_items` + SQLC queries; migration `000022+`.

## Capabilities

### New Capabilities
- `quotes`: first-class, org-scoped, versioned quote documents linked to deals — line items, guarded state machine (borrador/enviada/aprobada/rechazada/vencida), deal monto sync, audit events, and repository/domain seams for later delivery and template capabilities.

### Modified Capabilities
- `crm-core-data`: deal behavior gains quote-driven `monto` sync and the aprobada-guard on moving to `facturado`. (Delta spec only.)
- `deal-management`: deals expose an active quote relationship and sync monto from it. (Delta spec only.)

## Impact

- **Go backend**: new module `internal/modules/quotes/` (domain + app + infra), DI wiring, routes under `/api/quotes` (org-scoped, `org:manage` write / `org:view` read); `DealStageListener` unchanged (quotes are agent-managed, not stage-driven) but deal service gains quote-aware monto sync.
- **Database**: new migrations creating `crm.quotes` + `crm.quote_items`; `crm.deals` unchanged (sync is service-level).
- **Frontend**: deal detail page gains a "Cotizaciones" section (list, create, edit items, change state). No delivery UI (that is `add-quote-delivery`).
- **Auth boundary / Stytch**: no Stytch contract changes; quotes org-scoped by authenticated org context. No new identity tables.
- **Dependencies**: none new in this change (PDF/WhatsApp are `add-quote-delivery`).
- **Rollback strategy**: Git — revert; DB — drop `crm.quotes` / `crm.quote_items` via down migrations; Stytch tenant policy — unaffected.

## Non-Goals

- No PDF rendering, file assets for documents, or WhatsApp sending (delivered by `add-quote-delivery`).
- No document branding (delivered by `add-document-branding`; quotes reference branding by snapshot key only).
- No AI-driven quote drafting or approval extraction in this change (deferred; the payload/repository seams exist).
- No invoice/template rendering (future capability; quote template engine arrives with delivery).
- No DIAN/compliance obligations — a cotización is a commercial offer, not an electronic invoice; the e-invoice rail (Siigo) is untouched.

## Assumptions

- A quote's line items may be entered manually in v1; linkage to the procurement SKU catalog is a future extension (the `sku_ref` nullable column reserves it).
- Quote numbering is a simple consecutive per-org sequence in v1 (e.g., COT-0001); advanced numbering (prefix per pipeline/vertical) is deferred and the prefix is a branding-config value.
- "Aprobada" requires an explicit agent action (manual); AI-extracted approval from client WhatsApp replies is deferred.
- The aprobada-guard on `facturado` is advisory (warning + activity) in v1, not a hard block, to avoid regressions in existing invoice automation; hard-block is a follow-up decision.
