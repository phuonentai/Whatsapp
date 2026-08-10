## Context

The CRM stores all sales data org-scoped in local PostgreSQL: `crm.deals` (monto, estado `abierto|ganado|perdido|abandonado`, pipeline/stage, assigned_to), `invoicing.invoices` (amount, status `pending|valid|invalid|errored`, linked to deals), `crm.companies` (sector, ciudad), `crm.contacts` (last_message_at), `crm.messages`, `crm.tickets`. Siigo invoicing (`add-siigo-invoicing`) made real invoiced revenue available. No analytics or reporting code exists today (verified: zero matches for analytics/reportes in backend and frontend).

Backend follows a strict module pattern (`internal/modules/<cap>/` with routes.go, handler.go, domain/, app/, infra/, dig Provider), routes gated by feature flags (`features.Require`) or module registry (`registry.Require`, `internal/modules/registry/middleware.go`), and permissions from Stytch RBAC via `auth.RequirePermissionFunc("resource", "action")`. Frontend is Next.js App Router with Spanish UI, shadcn/ui, recharts ^3.3.0 already a dependency, api client under `lib/api/`.

## Goals / Non-Goals

**Goals:**
- Read-only analytics capability with four reports: invoiced revenue by period, top customers by revenue, pipeline funnel, inactive contacts.
- Org-scoped aggregation over existing tables — zero new migrations.
- Gated as a sellable module (`analytics`) in the module registry, enabled by default.
- Reuse existing permissions — no Stytch RBAC policy changes.
- Spanish UI page `/dashboard/reportes` with recharts.

**Non-Goals:**
- No new database tables or schema changes.
- No nightly rollups / materialized views / event sourcing.
- No forecast, ML, or predictive scoring.
- No CSV export (v1).
- No multi-currency — COP only.
- No campaign/broadcast analytics (depends on `add-whatsapp-campaigns`, not implemented).
- No payment-collection figures — revenue is *invoiced* (`status='valid'`), not *collected*.
- No admin/org-wide (cross-tenant) reporting.

## Decisions

### D1: Revenue source is `invoicing.invoices` with `status = 'valid'`
Invoiced revenue is the "sold" signal; `deals.monto` is an estimate and appears in the funnel instead. `deals.estado='ganado'` without a valid invoice counts as pipeline, not revenue — avoiding double counting where an invoice exists for a won deal.
*Alternative considered:* revenue = `deals.monto` where `estado='ganado'` — simpler but ignores Siigo/DIAN reality and diverges from the invoice system of record.

### D2: Direct SQL aggregation on request — no rollups
SMB-scale volumes (thousands of rows per org) make `SUM/GROUP BY` queries cheap and always fresh. Indexes already exist: `idx_invoices_org`, `idx_deals_org`, `idx_deals_estado`, `idx_contacts_last_message`, `idx_messages_created`.
*Alternative considered:* nightly rollup tables — more infrastructure (scheduler, backfill, staleness contract) with no demonstrated need at this scale. Revisit when query latency or volume demands it.

### D3: New `analytics` Go module + registry module key
A separate module keeps the capability independently shippable/monetizable, consistent with the sellable-modules direction. Entitlement gating via existing `registry.Require("analytics")` middleware; seed data enables the module by default for all orgs.
*Alternative considered:* feature flag inside the `crm` module — less wiring but couples analytics to CRM feature gating and blocks future standalone selling.

### D4: Reuse existing permissions — no new Stytch RBAC resources
Endpoints are gated per report: `invoice:view` (revenue, top customers), `deal:view` (funnel), `contact:view` (inactive contacts). Stytch is the runtime SSOT for RBAC; adding `report:view` would require tenant policy changes with no functional gain for v1.
*Alternative considered:* new `report:view` permission — cleaner abstraction, but violates the "no Stytch tenant policy churn for read-only features" bias and requires mock-permission updates in `internal/modules/auth/middleware.go`.

### D5: Period bucketing via `date_trunc` with validated ranges
`GET revenue?period=week|month` + optional `from`/`to` (ISO dates). Validation: `from <= to`, max span 13 months, default window = last 30 days. Buckets via `date_trunc('week'|'month', created_at)`.

### D6: Inactive threshold is a query parameter, default 30 days
`GET inactive-contacts?days=30` — `last_message_at < now() - days`. Strictly < (not <=). Includes contacts with no messages ever (NULL `last_message_at`) — those are new-but-silent, reported in a separate bucket from churned.

### D7: Top customers joins invoices → deals → companies, fallback to contact
`invoicing.invoices.deal_id → crm.deals.company_id → crm.companies.name`; when a deal has no company, fall back to the linked contact's display name/phone. NULL-safe LEFT JOINs, `COALESCE` in SQL.

## Architecture

```
FE  /dashboard/reportes (recharts, shadcn Card, Spanish)
     │  server actions → lib/api client
     ▼
GET /api/analytics/revenue?period=week|month&from&to
GET /api/analytics/top-customers?limit=10
GET /api/analytics/funnel
GET /api/analytics/inactive-contacts?days=30
     │  auth + org_context + registry.Require("analytics")
     │  + invoice:view / deal:view / contact:view
     ▼
internal/modules/analytics/
  routes.go      — route groups + middleware wiring
  handler.go     — parse/validate params, call service, JSON responses
  domain/        — pure types: RevenuePoint, TopCustomer, FunnelStage, InactiveContact
  app/services/  — SalesReportService: validation, org-scoped orchestration
  infra/         — SQLC adapter implementing domain repository interfaces
     │
     ▼
query/analytics.sql (new, SQLC) → make sqlc → gen
  RevenueByPeriod(org, from, to)        — invoices valid, SUM(amount) by date_trunc
  TopCustomersByRevenue(org, limit)     — LEFT JOIN deals→companies/contacts, SUM, ORDER BY DESC
  FunnelByStage(org)                    — deals grouped by stage (abierto) + SUM(monto); estado counts
  InactiveContacts(org, since)          — last_message_at < since OR NULL, contact + last activity
```

All queries take `organization_id` as the first argument (existing `000022_add_tenant_isolation` invariant); results never cross tenant boundaries. Clean Architecture: domain imports no transport/SDK packages; DI registration via `Provider` in the new module, wired in bootstrap alongside `invoicing`/`registry`.

Frontend: `app/dashboard/reportes/page.tsx` + component widgets, nav entry in dashboard layout, api functions + server actions in `lib/api/analytics.ts`, typed models in `lib/types`.

## Risks / Trade-offs

- [Invoiced ≠ collected] → Widget labels say "Facturado"; MercadoPago collection data, when it lands, is a follow-up dimension, not a v1 correction.
- [`contacts.last_message_at` only tracks message recency, not other activity] → Widget is explicitly "contactos inactivos por WhatsApp"; labels clarify scope.
- [Query cost grows with message volume] → Aggregates are org-scoped and indexed; revisit rollups if latency exceeds ~500ms in practice.
- [Date bucketing timezone ambiguity] → Use UTC consistently; document in API; FE renders local.
- [Module registry seeds new module as enabled] → Confirmed acceptable for v1 (feature is read-only); disabling via entitlement API works immediately.
- [recharts bundle weight] → Import chart components directly (tree-shaken); no new dep otherwise.

## Migration Plan

1. Add `query/analytics.sql`, run `make sqlc`, commit generated code.
2. Add `analytics` module seed + registry wiring; endpoints return data behind entitlement.
3. Ship FE page.
4. Rollback: revert Git commits (new module/page removed, sqlc regen); no schema or Stytch state to roll back; per-org kill switch = disable `analytics` module via entitlement API.

## Open Questions

- None blocking. Follow-ups tracked in proposal Non-Goals: collected-vs-invoiced revenue, campaign analytics, CSV export, rollups.
