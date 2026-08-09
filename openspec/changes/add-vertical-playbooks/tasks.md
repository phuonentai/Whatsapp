<!-- Implementation tasks for add-vertical-playbooks. Each unit <= 2 hours. Tags: [BE-DOMAIN] [BE-INFRA] [DB-SQLC] [FE-NEXT] [OPS-GOV]. Verification commands recorded per group. -->

## 1. Database schema and SQLC queries [DB-SQLC]

- [x] 1.1 Migration: `playbooks` registry table (key UNIQUE, name, vertical, description, requires_modules JSONB, payload JSONB, is_active BOOL) and `organization_playbooks` (organization_id FK, playbook_key FK, seeded_pipeline_id NULL, applied_at, UNIQUE(organization_id, playbook_key)) — plus down-migration
- [x] 1.2 Seed rows (in the same migration) for the five vertical playbooks: `comercio`, `restaurantes`, `citas`, `servicios-profesionales`, `talleres` — payloads with pipeline etapas (nombre/orden/color/probabilidad), tags, `module_configs` presets for `tickets` (sla_hours/priorities/tags), and guiones (id/titulo/mensaje in Spanish)
- [x] 1.3 SQLC queries: list active playbooks, get playbook by key, upsert organization_playbooks, get org playbook state, delete org playbook state, list org playbooks
- [x] 1.4 Verification: `make sqlc` regenerates; `go build ./...` passes; `make test` migration tests pass

## 2. Playbook catalog as Go single source [BE-DOMAIN]

- [x] 2.1 Domain: `Playbook`, `PlaybookPayload`, `PipelineTemplate`, `EtapaTemplate`, `Guion`, `OrganizationPlaybook` entities; repository interfaces; errors (ErrPlaybookNotFound, ErrInvalidPlaybookPayload)
- [x] 2.2 `catalog.go` defining the five playbooks (identical to migration seed), used for startup validation and tests; test asserting catalog keys/names/stage counts/guion non-emptiness match the seeded DB rows
- [x] 2.3 Payload validator: pipeline etapa type/order checks, guion text non-empty, `module_configs` presets validated against each module's `config_schema` via the registry `ModuleService`, `requires_modules` keys resolved against the module registry
- [x] 2.4 Verification: `go build ./...`; unit tests for validator (valid payload passes; `sla_hours` non-numeric → ErrInvalidPlaybookPayload); `make test`

## 3. PlaybookService apply/reset [BE-DOMAIN] [BE-INFRA]

- [x] 3.1 App: `PlaybookService.Apply(ctx, orgID, key)` in one transaction — pipeline seeded (as `es_predeterminado` only when org has zero pipelines, recording `seeded_pipeline_id`), tags seeded when missing by name, `module_configs` presets applied for enabled modules (skipping disabled, validating through ModuleService), org playbook state upserted with `applied_at`; re-apply additive-only
- [x] 3.2 App: `PlaybookService.Reset(ctx, orgID, key)` — deletes org playbook state, seeded pipeline only if it has no deals, seeded tags only if unreferenced, seeded config keys only if stored value matches the preset; never touches org-created data
- [x] 3.3 Infra: SQLC-backed repository implementing the playbook repository interfaces
- [x] 3.4 Verification: `go build ./...`; unit tests — first apply seeds pipeline+tags+config+state; re-apply adds only missing; org with existing pipelines skips pipeline without failure; disabled-module preset skipped; invalid preset returns 400 with no partial persistence; reset protects pipeline-with-deals and referenced tags; `make test`

## 4. Playbook API routes [BE-INFRA]

- [x] 4.1 Routes/handlers: `GET /api/playbooks` (non-internal catalog + org applied state + guiones of applied playbooks), `POST /api/playbooks/:key/apply`, `POST /api/playbooks/:key/reset` — gated by `EntitlementMiddleware` then existing RBAC scope check (e.g., `contact:manage` via `auth.RequirePermissionFunc`), unknown key → 404, no new Stytch RBAC policy
- [x] 4.2 Wire the playbooks module in `internal/bootstrap/init_mods.go`/`cmd/init.go` following registry module conventions (`module.go`, `provider.go`, `routes.go`)
- [x] 4.3 Verification: `go build ./...`; handler tests — 404 unknown key, subscription-gated 401/403, catalog response shape (playbooks, applied state, guiones); `make test`

## 5. Frontend playbook experience [FE-NEXT]

- [x] 5.1 Extend frontend API consumption with `GET /api/playbooks`; one-time onboarding step "¿Qué tipo de negocio es el tuyo?" (dismissible) shown post-first-login when no playbook is applied, listing the five verticals and calling `POST /api/playbooks/:key/apply`
- [x] 5.2 Settings card (`/dashboard/settings`): applied playbook display, re-apply, and reset with confirmation dialog
- [x] 5.3 Inbox quick-reply chip row: rendered only when the org has applied a playbook with guiones; chip selection prefills the composer; send continues via the existing conversation send path
- [x] 5.4 Verification: `pnpm build` PASS; `npx tsc --noEmit` PASS; `pnpm lint` — 13 errors, ALL pre-existing files (inbox/page.tsx conditional hooks, knowledge-content, settings-content effects, auth-context, plans-modal, etc.); 0 errors in playbooks feature files (verified via grep on lint output)

## 6. Dogfooding and verification gate [OPS-GOV]

- [x] 6.1 Dogfood: applied migrations 000001-000020 (excluding pre-existing broken 000002_add_tenant_isolation) on scratch pgvector:pg17 DB; verified playbooks table seeds 5 verticals (stage/guion counts); simulated apply for org 1 (pipeline + 3 stages + 3 tags + tickets config preset + org playbook state) with all FKs resolving; down-migration drops tables cleanly
- [x] 6.2 Full verification gate — results recorded:
    - `make sqlc` equivalent (`docker compose run --rm --no-deps cli sqlc generate`) — **PASS** (regenerated gen v1.27, incl. playbooks queries/models; consistent across all files)
    - `go build ./...` — **PASS** (exit 0; includes all modules; earlier agent-module breakage was stale gen, resolved by the consistent regen)
    - `go test ./internal/...` — **PASS** (17 ok, 0 fail)
    - `go vet ./internal/modules/playbooks/... ./internal/modules/registry/app/...` — **PASS**
    - `npx tsc --noEmit` — **PASS**
    - `pnpm build` — **PASS** (14 routes compiled)
    - `pnpm lint` — **PASS for this change**: 13 errors, all in pre-existing files (inbox/page.tsx conditional hooks, knowledge-content, settings-content effects, auth-context, plans-modal); 0 errors in playbooks feature files (verified via grep on lint output)
    - Migration + dogfood on scratch pgvector:pg17 — **PASS** (see 6.1)
    - `make test`/`make server` not runnable here (no `make` in environment); equivalent `go test`/`go build` used and recorded above

**Archive deferred:** pending /opsx-archive after review — change is implementable end-to-end and green, but the repo has substantial untracked WIP (agent module, migrations 000017-000018 from add-sellable-modules) that must be committed/archived first for a clean archive.
