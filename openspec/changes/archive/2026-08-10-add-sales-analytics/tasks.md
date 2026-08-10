## 1. SQLC aggregation queries [DB-SQLC]

- [x] 1.1 Create `internal/db/postgres/sqlc/query/analytics.sql` with `RevenueByPeriod(org_id, from, to)` — valid invoices, `SUM(amount)` grouped by `date_trunc('week'|'month')` over `invoicing.invoices`, org-filtered. Verify: `make sqlc` passes and generated code compiles.
- [x] 1.2 Add `TopCustomersByRevenue(org_id, limit)` — LEFT JOIN `crm.deals` → `crm.companies` with `COALESCE` fallback to contact display name/phone, `SUM(amount)` of valid invoices, `ORDER BY total DESC`. Verify: `make sqlc` passes and generated code compiles.
- [x] 1.3 Add `FunnelByStage(org_id)` — open deals grouped by stage of default pipeline (`crm.pipelines.es_predeterminado`) with count + `SUM(monto)`, plus `ganado`/`perdido`/`abandonado` aggregates and `otras_pipelines` for non-default pipelines. Verify: `make sqlc` passes and generated code compiles.
- [x] 1.4 Add `InactiveContacts(org_id, since)` — contacts with `last_message_at < since` (bucket `inactivo`) and NULL `last_message_at` (bucket `sin_actividad`). Verify: `make sqlc` passes and generated code compiles.

## 2. Analytics domain models [BE-DOMAIN]

- [x] 2.1 Create `internal/modules/analytics/domain/` pure types: `RevenuePoint` (periodo, monto_total), `TopCustomer` (nombre, monto_total), `FunnelEntry` (etapa, cantidad, monto_total), `InactiveContact` (telefono, nombre, ultimo_mensaje_at, clasificacion), and repository interfaces (no Stytch SDK or transport imports). Verify: `go build ./...` passes.
- [x] 2.2 Create `internal/modules/analytics/app/services/sales_report_service.go` — date-range validation (`from <= to`, max 13 months, default 30d window), `days` validation (1–365, default 30), `limit` clamp (default 10, max 50), delegates to domain repository interfaces. Verify: `go build ./...` passes.

## 3. Infrastructure adapter [BE-INFRA]

- [x] 3.1 Create `internal/modules/analytics/infra/` SQLC adapter implementing the domain repository interfaces (org-scoped queries, `organization_id` first argument on every call). Verify: `go test ./internal/modules/analytics/...` passes with repository tests using the mock/queries pattern from `invoicing`.
- [x] 3.2 Wire dependencies in `internal/modules/analytics/provider.go` (dig `Provide` for repo, service, handler) and register the provider in the server bootstrap alongside `invoicing`/`registry`. Verify: server starts with `make server` (or `go build ./...` + existing boot test).

## 4. HTTP handlers, routes, and gating [BE-INFRA]

- [x] 4.1 Implement `handler.go` with `GET /api/v1/org/analytics/revenue`, `/top-customers`, `/funnel`, `/inactive-contacts`; Spanish error messages; JSON responses matching domain types. Verify: `go test ./internal/modules/analytics/...` handler tests pass.
- [x] 4.2 Implement `routes.go` — group under `/api/v1/org/analytics` with `auth` + `org_context` + `registry.Require("analytics")` middleware; per-endpoint permissions `invoice:view` (revenue, top-customers), `deal:view` (funnel), `contact:view` (inactive-contacts). Verify: handler tests assert HTTP 403 `module_disabled` when module disabled and HTTP 403 Spanish message when permission missing (mock permission context path).
- [x] 4.3 Seed the `analytics` module in the module registry (enabled by default for organizations, consistent with the `tickets` module seeding pattern). Verify: `GET /api/v1/org/modules` (entitlement API) includes `analytics` enabled; integration test for module seeding passes.

## 5. Frontend reports page [FE-NEXT]

- [x] 5.1 Add typed api client + server actions in `lib/api/analytics.ts` (and typed models in `lib/types`) calling the four endpoints. Verify: `pnpm build` passes.
- [x] 5.2 Create `app/dashboard/reportes/page.tsx` with Spanish UI: cards for Ventas facturadas (period selector week/month), Top clientes, Funnel pipeline, Contactos inactivos (days selector); recharts for revenue chart and funnel bar; nav entry in dashboard layout. Verify: `pnpm build` and `pnpm lint` pass.
- [x] 5.3 Handle empty states and loading in Spanish ("Sin datos para el periodo seleccionado"); client components wrapped with TanStack Query following existing patterns. Verify: `pnpm build` and `pnpm lint` pass.

## 6. Governance and validation [OPS-GOV]

- [x] 6.1 Run `openspec validate` on the change; fix any spec-format issues (scenarios MUST use 4-hashtag `####` headers). Verify: `openspec validate --change add-sales-analytics` passes.
- [x] 6.2 Run full backend and frontend verification: `make test` (or `go test ./...`) and `pnpm build && pnpm lint`. Verify: all commands pass.

## Verification gate (2026-08-10)

- [x] `sqlc generate` (docker cli container) — passed; `gen/analytics.sql.go` + `querier.go` updated
- [x] `go build ./...` — passed
- [x] `go test ./...` — passed (37 packages ok, 0 failures; incl. analytics handler + service tests, billing default-grant test)
- [x] `go vet ./internal/modules/analytics/...` — passed; gofmt clean on new files
- [x] Migration `000032_seed_analytics_module` applied to local PG — passed; `modules.modules` contains `analytics` (is_active=t)
- [x] `pnpm lint` — passed (0 errors; 1 pre-existing warning in `components/crm/deal-kanban.tsx`)
- [x] `pnpm build` — passed; `/dashboard/reportes` in route manifest
- [x] `openspec validate add-sales-analytics` — passed

Note: pre-existing repo debt fixed along the way (worktree mid-refactor): sqlc gen was stale vs `query/*.sql`; regenerated all gen (v1.27.0, matches committed version). Fixed broken builds in crm/agent/payments/instagram modules (missing `outbox.Repository`/IG wiring, `PaymentLinker` 4-arg interface per `add-client-payments` design, import/alias fixes, `maskConfig` mutation bug, test noop-logger drift). All pre-existing; the repo did not build before this change.
