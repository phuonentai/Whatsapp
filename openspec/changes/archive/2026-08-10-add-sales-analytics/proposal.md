## Why

SMBs using the CRM lack visibility into sales performance: "¿Quién compra más? ¿Qué vendí esta semana?" cannot be answered today, and inactive customers go undetected until they churn. All required data already lives in the local PostgreSQL (deals, invoices, contacts, messages), and Siigo invoicing now provides a real invoiced-revenue signal. The reporte is the missing final link in the sales flow `promoción → conversación → pago → factura → reporte`, and it closes the retention loop: detect inactivos → re-engage.

## What Changes

- **New `analytics` backend module** (`go-b2b-starter/internal/modules/analytics/`): read-only aggregation endpoints, org-scoped, following the existing module pattern (routes.go / handler.go / domain / app / infra, DI via dig Provider).
- **Four report widgets** served by four aggregation endpoints:
  - Ventas facturadas por periodo (semana/mes, rango configurable) — source: `invoicing.invoices` with `status = 'valid'`
  - Top clientes por ingresos (facturado) — source: invoices → deals → companies (fallback contact)
  - Funnel pipeline (negocios por etapa, monto sumado; ganado/perdido/abandonado separados) — source: `crm.deals`
  - Contactos inactivos (sin mensajes desde umbral configurable) — source: `crm.contacts.last_message_at`
- **New SQLC queries** in `go-b2b-starter/internal/db/postgres/sqlc/query/analytics.sql` with generated code (no new migrations — read-only over existing tables).
- **Feature gating via module registry**: new `analytics` module key enabled through `registry.Require("analytics")`, consistent with sellable-module direction (`add-sellable-modules`).
- **Reused existing permissions**: `invoice:view`, `deal:view`, `contact:view` — no new Stytch RBAC resources, no changes to Stytch tenant policy state.
- **New frontend page** `/dashboard/reportes` in `next_b2b_starter/app/dashboard/reportes/` with Spanish UI, shadcn cards, recharts (already a dependency).
- **New OpenSpec capability spec** `openspec/specs/analytics/spec.md` via delta.

## Capabilities

### New Capabilities
- `analytics`: read-only sales reporting over local CRM/invoicing data — invoiced revenue by period, top customers by revenue, pipeline funnel, inactive contacts; org-scoped aggregation queries; module-registry gating; permission reuse.

### Modified Capabilities
<!-- none: no existing spec-level requirements change -->

## Impact

- **Backend**: new Go module `internal/modules/analytics` (routes, handler, domain models, app services, SQLC repo adapter); new `query/analytics.sql` + regenerated sqlc code; module registration in bootstrap and registry seed data.
- **Frontend**: new `/dashboard/reportes` page, api client functions, server actions; nav entry; no new npm dependencies (recharts already present).
- **Database**: no schema migrations; existing tables (`invoicing.invoices`, `crm.deals`, `crm.companies`, `crm.contacts`, `crm.messages`) read-only.
- **Auth/RBAC**: no Stytch policy changes — endpoints reuse `invoice:view` / `deal:view` / `contact:view`. Runtime SSOT (Stytch) untouched.
- **Rollback**: Git state reverts cleanly (new module + page removed, sqlc gen regenerated); no Stytch tenant policy state exists for this change, so no runtime rollback needed. Feature can also be disabled per-org by disabling the `analytics` module — no data writes occur.
- **Non-Goals**: no local credential storage anywhere (all auth remains Stytch); no forecast/ML; no multi-currency conversion (COP only); no CSV export in v1; no broadcast/campaign analytics (depends on `add-whatsapp-campaigns`, currently empty); no payment-collection (MercadoPago) figures — invoiced ≠ collected; no nightly rollups/materialized views.
