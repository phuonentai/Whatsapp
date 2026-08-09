## Why

The platform ships a powerful but blank-slate experience: a new organization gets one generic "Pipeline de Ventas", no tags, no message scripts, no module configuration — even after purchasing modules like `tickets`. The module system (from `add-sellable-modules`) made the catalog sellable but not *ready to use*. The Colombian SME market is concentrated in a handful of verticals (retail/commerce, restaurants, appointment-based services, professional services, workshops) where operations follow well-known, repeatable procedures. Pre-packaged vertical "playbooks" — pipeline, tags, module config, and WhatsApp message scripts seeded in one click — turn the product into an out-of-the-box experience per business type, with procedures the business can run on day one.

## What Changes

- **Playbook registry**: a database catalog of vertical playbooks (`key`, name, vertical, description, `requires_modules`, JSON payload defining: pipeline template, tag set, module config presets, WhatsApp message scripts/guiones) — seeded from Go constants, mirroring the `modules` seeding pattern.
- **Per-org playbook state**: which playbook(s) an organization has applied and when (`organization_playbooks`), enabling re-apply and reset.
- **Apply/reset service**: `PlaybookService.Apply` applies a playbook **one-way, idempotently** — creates the vertical pipeline (as predeterminado only when the org has no pipelines), seeds tags, upserts module config presets (validated against each module's `config_schema`), and records playbook state. Re-apply only adds missing seed data; it never deletes or overwrites org-created data. `PlaybookService.Reset` is an explicit separate call that removes playbook-seeded data.
- **Playbook API**: `GET /api/playbooks` (tenant catalog), `POST /api/playbooks/{key}/apply`, `POST /api/playbooks/{key}/reset` — gated by the existing entitlement/module middleware and RBAC; no new Stytch permissions.
- **Five seeded vertical playbooks** with their procedures: `comercio` (retail/e-commerce: pedido, pago, entrega, reactivación, devolución), `restaurantes` (pedido/domicilio, reserva, promo, reclamo), `citas` (salud/estética/bienestar: agendamiento, recordatorio, bonos, follow-up), `servicios-profesionales` (leads, cotización, hitos, soporte), `talleres` (cotización con fotos, OT, update al cliente, garantía).
- **Guiones (message scripts) as inbox quick replies**: playbook scripts surface as one-tap reply suggestions in the WhatsApp inbox, executed by a human agent via the existing outbound `SendMessage` path — no new WhatsApp template/broadcast capability in this change.
- **Frontend**: post-first-login onboarding step "¿Qué tipo de negocio es el tuyo?" → applies playbook; `/settings` shows the applied playbook with re-apply/reset; inbox renders playbook guiones as quick replies.

## Capabilities

### New Capabilities

- `vertical-playbooks`: playbook catalog, per-org playbook state, one-way idempotent apply/reset, seed payloads (pipeline, tags, module configs, guiones), playbook-gating and API surface, and frontend onboarding/settings/inbox integration.

### Modified Capabilities

- `module-registry`: requirement changes — playbooks reference registry modules and write validated module configs into `organization_modules`; module config presets from a playbook follow the same validation path as manual config edits.

## Impact

- **Backend (Go)**:
  - New tables + SQLC queries: `playbooks`, `organization_playbooks` (migration `000020_create_playbooks`), plus SQLC queries for catalog, state, and seed-data application. Down-migrations included.
  - New module `internal/modules/playbooks/` (domain: Playbook, OrganizationPlaybook; app: PlaybookService with one-way apply; infra: SQLC repository; routes/handler/module/cmd wiring) following the CRM/registry module layout. Seed data lives in a Go constant catalog (single source), validated at startup against the module registry.
  - Pipeline seeding reuse: `pipeline_service.GetOrCreateDefault` pattern extended so playbook apply creates a vertical pipeline; only marked `es_predeterminado` when the org has no pipelines.
  - Route registration: `GET /api/playbooks`, `POST /api/playbooks/:key/apply`, `POST /api/playbooks/:key/reset` behind the existing `EntitlementMiddleware` + registry module middleware + existing RBAC scope checks (`contact:manage`-level scopes; no new Stytch RBAC policy changes).
- **Frontend (Next.js)**: onboarding step after first login/WhatsApp config; `/settings` playbook card with re-apply/reset; inbox quick-reply chip row fed by the applied playbook's guiones (sends via existing `POST /api/crm/conversaciones/:id/mensajes`).
- **Specs**: new `vertical-playbooks` delta spec; `module-registry` delta spec for playbook-driven config validation.

## Non-Goals

- **No local credential storage.** All identity, session, and RBAC concerns remain with Stytch B2B; local PostgreSQL stores only `stytch_member_id`/`stytch_organization_id` foreign keys. Playbooks add no auth surface; playbook apply rides existing RBAC scopes — no new Stytch policy changes, no local auth tables.
- No WhatsApp template API, broadcast/campaign capability, or out-of-24h-window sending — guiones are quick replies executed by humans via the existing free-text send path. Template management is a future change.
- No sellable vertical modules themselves (catalog/pedidos, citas, payment links, IA execution) — playbooks compose **existing** capabilities (pipelines, tags, module config, message scripts); `requires_modules` declares future dependencies but only shipped modules (`tickets`) are referenced today.
- No changes to the billing engine or entitlement derivation semantics; playbooks do not grant features.
- No market/statistical claims enforced as requirements — the vertical selection rationale is documented in Assumptions.

## Rollback Strategy

- **Git state**: revert the change commit(s); DB migrations are forward-only with paired down-migrations (`000020_create_playbooks.down.sql`); SQLC regenerates from schema.
- **Stytch tenant policy state**: no Stytch RBAC policy changes are made by this change (no new permissions/roles), so no Stytch-side rollback is required; a revert leaves RBAC untouched.
- **Org data safety**: playbook apply is additive-only (one-way, idempotent) and reset is explicit and scoped to playbook-seeded rows (pipeline/tags/state flagged as `seeded_by_playbook`), so a revert or reset never touches org-created data. `organization_playbooks` is new and read-only-if-empty, so a reverted build is inert.

## Assumptions

- **Colombian market vertical selection (UNVERIFIED — web search unavailable at proposal time)**: the five verticals were chosen from general knowledge of the Colombian SME landscape — (a) retail/commerce is the largest SME cluster and sells heavily via WhatsApp with Nequi/transfer payments; (b) restaurants push direct WhatsApp ordering to avoid marketplace commissions; (c) appointment-based services (salud/estética) suffer chronic no-shows and book via chat; (d) professional services are the most formalized, first to pay SaaS, and already e-invoicing via DIAN; (e) workshops/reparación are quote-and-approve-by-photo operations. Market-size statistics (e.g., share of informal economy, Nequi user counts, e-commerce growth) were NOT verified against sources and must not be asserted as facts in specs or marketing.
- **Guiones as free text**: WhatsApp Cloud API message templates are not required for the quick-reply flow because human agents send within the platform using free-form text via the existing `SendMessage`; the 24-hour customer-service window applies as it does to all outbound sends today.
- **Pipeline seed trigger**: playbook apply creates a vertical pipeline only when the organization has no pipelines; orgs that already customized a pipeline keep it untouched (re-apply adds nothing pipeline-wise).
- **Playbook availability**: playbooks are available on all plans (they seed configuration, not features); module config presets for a module the org hasn't purchased are skipped, not errored.
