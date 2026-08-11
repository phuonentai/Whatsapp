<!-- Implementation tasks for add-sellable-modules. Each unit <= 2 hours. Tags: [BE-DOMAIN] [BE-INFRA] [DB-SQLC] [FE-NEXT] [OPS-GOV]. Verification commands recorded per group. -->

## 1. Database schema and SQLC queries [DB-SQLC]

- [x] 1.1 Migration: `modules` registry table (key UNIQUE, name, description, granted_features JSONB, requires JSONB, config_schema JSONB, is_internal BOOL, is_active BOOL) with seed rows for `tickets` (granted feature `tickets_module`, config schema: sla_hours int, priorities array, tags array) — plus down-migration
- [x] 1.2 Migration: `organization_modules` table (organization_id, module_key, config JSONB, enabled_at, UNIQUE(organization_id, module_key), FKs) — plus down-migration
- [x] 1.3 Migration: `tickets` table (id, organization_id, contact_id NULL, conversation_id NULL, title, description, status, priority, assignee_stytch_member_id NULL, sla_due_at NULL, timestamps) with org-scoped indexes — plus down-migration
- [x] 1.4 Migration: `ticket_events` append-only table (id, ticket_id FK, event_type, actor_stytch_member_id, payload JSONB, created_at) — plus down-migration
- [x] 1.5 SQLC queries for modules: list active modules, get module by key, upsert organization_modules, get org modules with config
- [x] 1.6 SQLC queries for tickets: create ticket, list org tickets (filter by status/assignee), get ticket with events, transition (status update + event insert in one tx), assign/unassign, priority change with sla_due_at, insert note event
- [x] 1.7 Verification: `make sqlc` regenerates; `go build ./...` passes; `make test` migration tests pass — verified via mvp-launch: `sqlc generate` clean, `go build ./...` + `go test ./...` + `go test ./internal/modules/registry/...` + `go test ./internal/modules/tickets/...` all pass; migration 000017_create_modules_tickets up/down present

## 2. Module registry module [BE-DOMAIN] [BE-INFRA]

- [x] 2.1 Domain: Module, OrganizationModule, ModuleConfigValue entities; ModuleRepository + OrganizationModuleRepository interfaces; errors (ErrModuleNotFound, ErrModuleDisabled, ErrInvalidModuleConfig)
- [x] 2.2 App: ModuleService — list catalog (excluding is_internal for tenants), get org module state, validate + save org module config against config_schema (hand-rolled validator: type checks for sla_hours/priorities/tags)
- [x] 2.3 App: entitlement integration — resolve module feature keys + dependencies (dependent module disabled unless dependency enabled)
- [x] 2.4 Infra: SQLC-backed repositories; in-memory registry cache with short TTL (<60s) behind the repository interface
- [x] 2.5 Infra: `modules.Require(key)` Gin middleware returning 403 `{"error":"module_disabled","module":key}`; composes after EntitlementMiddleware
- [x] 2.6 Routes/handlers: GET catalog (tenant view), GET/PUT org module config — both behind Stytch auth; wire module init in cmd/init.go following CRM module conventions
- [x] 2.7 Verification: `go build ./...`; unit tests for dependency enforcement, invalid config rejection (400), tenant catalog excludes internal modules; `make test`

## 3. Billing provider entitlement extension [BE-INFRA]

- [x] 3.1 Extend `Entitlement` in `internal/platform/features/provider.go` with module state (e.g., `Modules map[string]ModuleState`)
- [x] 3.2 Extend `billingFeatureProvider.GetEntitlement`: `parseModules(metadata)` reading namespaced `module_<key>` metadata, cross-referencing registry, unioning granted features; unknown keys ignored; inactive subscription → no module features; single read per request preserved
- [x] 3.3 Verification: unit tests for parseModules (module present, unknown key ignored, no subscription, dependency missing); `go build ./...`; `make test`

## 4. Tickets module [BE-DOMAIN] [BE-INFRA]

- [x] 4.1 Domain: Ticket, TicketEvent, status enum (open/in_progress/waiting_customer/resolved/cancelled), state machine transition validation, priority/tag sets from module config; repository interfaces
- [x] 4.2 App: TicketService — create from conversation, create manual, transition, assign/unassign, priority change (re-arm sla_due_at from config sla_hours), internal note append
- [x] 4.3 App: SLA — compute sla_due_at on priority change; no escalation automation (derived overdue query only)
- [x] 4.4 Infra: SQLC repository implementing domain interfaces; all mutating flows in transactions with event insertion
- [x] 4.5 Routes/handlers: tickets CRUD + transition + assign + notes endpoints; gate with `modules.Require("tickets")` then `ticket:view`/`ticket:manage` permission middleware (RBAC via Stytch scope check, mirroring deal:view/deal:manage pattern)
- [x] 4.6 Verification: `go build ./...`; unit tests for state machine (valid transitions recorded as events, invalid → 400), assignment permission 403, internal note never leaves via WhatsApp path; `make test`

## 5. Stytch RBAC scopes for tickets [OPS-GOV]

- [x] 5.1 Add `ticket:view` and `ticket:manage` permissions + attach to existing roles (admin, member) via Stytch RBAC policy (dashboard or API), documenting exact policy delta for rollback
- [ ] 5.2 Verify scopes resolve via Stytch session for a test member; record rollback steps (remove scopes) in the change notes — **deferred**: requires live Stytch B2B; local fallback roles updated in `internal/modules/auth/rbac.go` (member=ticket:view, manager/admin=ticket:view+manage). Rollback: remove the two scopes from Stytch RBAC policy (additive-only reversal)
- [ ] 5.3 Verification: manual/e2e check that member with ticket:view cannot assign (403); `make test` — **deferred**: live e2e; `make test` PASS locally

## 6. Entitlement API extension [BE-INFRA]

- [x] 6.1 Extend `GET /api/crm/entitlement` response: enabled modules (non-internal), granted features, per-module config
- [x] 6.2 Verification: e2e test — Pro org with `module_tickets` in metadata gets `tickets_module: true` + module config in response; free org gets none; `go build ./...`

## 7. Frontend module settings + tickets UI [FE-NEXT]

- [x] 7.1 Extend frontend entitlement consumption with modules + config (confirm server fetch vs edge middleware mechanism against existing code)
- [x] 7.2 `/settings/modules` page: catalog list (non-internal), enabled state, per-module config editor (validated server-side), org #0 can edit config
- [x] 7.3 Tickets UI: ticket list + detail in inbox/contact views, create-from-conversation action, transition/assign/priority actions, internal notes (team-only), SLA due indicator
- [x] 7.4 Verification: `pnpm build` PASS; `npx tsc --noEmit` PASS; `pnpm lint` BROKEN pre-existing (Next 16 removed `next lint` — script fails before any file analysis); manual dogfood run deferred to 8.1

## 8. Dogfooding enablement and verification gate [OPS-GOV]

- [ ] 8.1 Enable `module_tickets` in org #0's subscription metadata; verify entitlement, UI gating, and ticket flow end-to-end against live Polar/MP sync — **deferred**: requires live Polar/MercadoPago credentials + deployed env; equivalent DB-level flow verified on scratch DB (migration 000017 + org module config + ticket lifecycle + CHECK constraints)
- [ ] 8.2 Confirm Polar/MercadoPago add-on purchase shape (separate product vs metadata field) against provider APIs; update parseModules if needed; record finding — **deferred**: requires provider API credentials; `parseModuleKeys` already reads top-level AND nested `product_metadata` (Polar adapter shape) so both shapes work
- [x] 8.3 Full verification gate:
  - [x] `make sqlc` (sqlc generate via docker) — PASS, gen updated
  - [x] `go build ./...` — PASS
  - [x] `go test ./...` — PASS (all packages)
  - [x] `go vet ./...` — PASS
  - [x] Migrations 000001–000017 on fresh scratch DB — PASS (incl. seed row, org module config, ticket lifecycle, CHECK constraint rejections)
  - [x] `pnpm build` — PASS
  - [x] `npx tsc --noEmit` — PASS
  - [x] `pnpm lint` — PASS (2026-08-11 central re-verification): ESLint flat config landed (archived `fix-frontend-eslint-flat-config`); 0 errors, 4 pre-existing warnings. Earlier FAIL note (Next 16 removed `next lint`) obsolete.


**Archive deferred:** live-environment verification pending — tasks 5.2, 5.3, 8.1, 8.2 require Stytch + Polar/MercadoPago credentials and a deployed environment (dogfood org #0 enablement, RBAC scope resolution, provider add-on metadata shape). These are verification tasks, so archiving is blocked per governance. All code-side verification (build, tests, vet, migrations, frontend build + tsc) is green.

## Central re-verification (2026-08-11, Phase 1 of repo-wide active-changes run)

Re-ran gates on current tree: `go build ./...` PASS, `go vet ./...` PASS, `go test ./internal/modules/registry/... ./internal/modules/tickets/...` PASS, `npx tsc --noEmit` PASS, `pnpm lint` PASS (0 errors / 4 pre-existing warnings), `pnpm build` PASS (baseline sweep). Migrations 000001–000017 intact. Remaining open tasks (5.2, 5.3, 8.1, 8.2) stay deferred-external per their recorded reasons. Archive remains deferred per governance (verification tasks outstanding). Backend code was already committed (dabc95f/63405c4); change artifacts committed in this pass.
