## Why

The MVP story — "a Colombian SME runs the whole sale inside WhatsApp" — has a hole: pedido → cotización → aprobado → **factura** → pago → entrega. Payment (MercadoPago PSE/Nequi), CRM, and WhatsApp ingress exist, but invoicing is missing. Colombian formal SMEs MUST issue DIAN electronic invoices, and Siigo is their default accounting/e-invoicing tool. Closing the loop means a deal that reaches a won/"facturado" stage automatically produces a Siigo invoice (no re-digitar), and the customer gets the invoice + payment link back inside WhatsApp.

## What Changes

- **New `invoicing` capability**: a per-provider invoicing seam mirroring the billing `ProviderRouter` pattern — `InvoicingProvider` domain interface, an `InvoiceRouter` with a Siigo adapter (future Alegra slot), and an OAuth2 token cache (mirroring the Stytch JWKS cache TTL pattern).
- **Deal-stage trigger**: extend the existing `DealStageListener` — a deal moved to a stage named `facturado` (won) triggers invoice creation. Idempotent: one invoice per deal, enforced by a unique `(organization_id, deal_id)` constraint.
- **Status sync**: `POST /api/v1/webhooks/siigo` with signature verification + idempotent processing (pattern from polar/mercadopago webhooks), plus a polling fallback for DIAN/CUFE status.
- **Invoice notification**: WhatsApp message to the contact with the invoice link (CUFE/PDF) and the MercadoPago payment link via the existing send path.
- **Schema**: new table `invoicing.invoices` (org_id, deal_id UNIQUE, external_id, cufe, status, pdf_url, amount, timestamps) + SQLC queries.
- **Config**: viper env split (sandbox vs production) for Siigo credentials, mirroring the MercadoPago env split.

## Capabilities

### New Capabilities
- `invoicing`: per-org electronic invoice creation routed through a provider router (Siigo first), deal-stage auto-trigger, invoice status sync via signed webhook + polling fallback, WhatsApp notification with invoice + payment links, and OAuth2 token cache behavior.

### Modified Capabilities
- (none — the `DealStageListener` extension is implementation detail within `crm`; no spec-level requirement of `crm-core-data` changes because a stage named `facturado` is a playbook data value, not a new system contract. If the crm event contract needs extending beyond the existing `DealStageChanged` event, this will be revisited in design.)

## Impact

- **Go backend**: new module `internal/modules/invoicing/` (domain + app + infra/siigo adapter + webhook handler), routes registered under `/api/v1/webhooks/siigo`, `DealStageListener` in `internal/modules/crm/app/services/` extended to dispatch to the invoicing service, DI wiring in `init_mods.go`. No auth/permission changes.
- **Database**: new migration creating `invoicing.invoices`; no changes to existing tables.
- **Frontend**: none required for MVP path (notifications flow over WhatsApp). Optional later: invoice link shown on the deal detail page.
- **Auth boundary / Stytch**: no change to Stytch contracts, RBAC, sessions, or the JWKS verification path. Invoicing data remains scoped by `organization_id` derived from the authenticated org context (Stytch org membership), so no new local identity tables are introduced. OAuth2 tokens for the Siigo provider are resolved from **environment configuration only**, never stored in PostgreSQL.
- **Dependencies**: new HTTP client dependency surface for the Siigo REST API (no vendor SDK required; keep it as a plain HTTP adapter to avoid domain/SDK coupling).
- **Rollback strategy**: Git — revert the change commit(s); DB — drop `invoicing.invoices` via a down migration (or reverse the migration); Stytch tenant policy state — unaffected (no RBAC/auth/tenant changes), so no Stytch-side rollback is required.

## Non-Goals

- **No local credential storage**: Siigo client secrets/access tokens are read from environment/config only; tokens may be cached in memory with a TTL but MUST NOT be persisted to PostgreSQL or written to logs.
- No generic integration framework or "marketplace"; the router is a small per-capability seam, not a plugin system.
- No Alegra adapter implementation in this change (interface/router slot only).
- No Puntored, shipping, inventory, or payroll integrations.
- No changes to the WhatsApp compliance/consent flow other than reusing the existing send path for the invoice notification.
- No FE work beyond (optionally) surfacing the invoice link on the deal page.
