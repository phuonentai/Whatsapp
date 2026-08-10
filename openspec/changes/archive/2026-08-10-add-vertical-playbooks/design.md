## Context

The product (Go backend + Next.js frontend) delivers a WhatsApp-connected CRM: contacts/companies/deals in COP, pipeline with Spanish stages, tags, activity timeline, WhatsApp Cloud API bridge with outbound free-text sending (`internal/modules/crm/app/services/outbound_service.go`), and the newly shipped module system (`internal/modules/registry/` + `internal/modules/tickets/`, tables `modules`/`organization_modules`/`tickets`/`ticket_events` from migration 000017) with per-org module config validated against each module's `config_schema`.

New organizations start blank: one generic "Pipeline de Ventas" (`pipeline_service.GetOrCreateDefault`), no tags, no scripts, no module config. `add-vertical-playbooks` makes the platform an out-of-the-box experience: five Colombian vertical playbooks (`comercio`, `restaurantes`, `citas`, `servicios-profesionales`, `talleres`) that seed pipeline, tags, module config presets, and WhatsApp message scripts (guiones) in one idempotent apply, with quick replies in the inbox executed by humans via the existing send path.

Constraints: Clean Architecture (domain → app → infra), SQLC-generated queries, Stytch B2B as runtime SSOT (no local credentials; no new Stytch RBAC policy changes — playbook apply rides existing scopes), feature/module gating before permissions, module layout conventions (`internal/modules/<capability>/` with `domain/`, `app/services/`, `infra/`, `routes.go`, `handler.go`, `module.go`, `cmd/init.go`), forward-only migrations with down-migrations at `internal/db/postgres/sqlc/migrations/` (next: 000020 — 000019 is the agent schema).

## Goals / Non-Goals

**Goals:**
- A playbook catalog (DB-backed, seeded from Go constants) with per-org apply state.
- One-way, idempotent apply: seeds vertical pipeline, tags, module config presets, guiones; never deletes/overwrites org-created data; re-apply only adds missing seed.
- Explicit reset that removes only playbook-seeded rows.
- Five vertical playbooks with procedures, shipped as seed data.
- Guiones surfaced as inbox quick replies, sent by humans via existing `SendMessage`.
- Frontend onboarding step + settings card + inbox quick-reply chips.

**Non-Goals:**
- No WhatsApp template/broadcast capability or out-of-window sending.
- No new sellable modules (catalog/pedidos, citas, payment links); playbooks compose existing capabilities only, and `requires_modules` references only shipped modules (`tickets`).
- No AI execution of procedures (separate `add-agentic-whatsapp-assistant` change).
- No entitlement/feature derivation changes; no billing changes.
- No new Stytch RBAC policy entries.

## Decisions

### D1: Playbooks live in DB tables seeded from Go constants, mirroring `modules`

New tables: `playbooks` (key UNIQUE, name, vertical, description, requires_modules JSONB, payload JSONB, is_active) and `organization_playbooks` (org_id, playbook_key, seeded_pipeline_id NULL, applied_at, UNIQUE(org_id, playbook_key)). Seed rows inserted by migration from a Go constant catalog (single source in the playbooks module). The `payload` JSONB holds `{pipeline: {nombre, etapas:[{nombre,orden,color,probabilidad}]}, tags: [string], module_configs: {<module_key>: <config>}, guiones: [{id, titulo, mensaje}]}`.

Rationale: catalog must be queryable by API/UI and per-org state must persist; identical proven pattern to the `modules` registry (D1 of add-sellable-modules). Alternatives: (a) hardcoded Go maps only — rejected, no per-org state/queryability; (b) code-defined procedures per vertical — rejected, same flexibility problem and no data-driven re-apply.

### D2: Apply is one-way, additive, and idempotent; reset is explicit and scoped

`PlaybookService.Apply(ctx, orgID, key)` runs in one transaction:
1. Load playbook from registry; verify `requires_modules` satisfied by org module state (per D4 semantics).
2. Pipeline: if org has no pipelines, create vertical pipeline marked `es_predeterminado` and record `seeded_pipeline_id`. If org already has pipelines, skip pipeline seeding entirely.
3. Tags: insert tags from payload that don't already exist by name; never deletes tags. (Tags carry no playbook FK — see risk R3.)
4. Module configs: for each `module_configs[module_key]` where the org has that module enabled, validate against the registry `config_schema` and upsert into `organization_modules.config`. Disabled modules are skipped, not errored.
5. Guiones: stored in the org playbook state row (payload merged), serving the inbox quick-reply UI.
6. Upsert `organization_playbooks` with `applied_at = now()`.

Re-apply = same transaction, additive (steps skip what exists). `Reset` = explicit: deletes `organization_playbooks` row, deletes seeded tags only if unreferenced, deletes the seeded pipeline only if it has no deals, deletes seeded module config keys set by the playbook (only if the config value matches the playbook preset). Reset does not touch org-created data.

Rationale: user-confirmed decision — one-way apply keeps org data sovereign; reset exists for misapplies. Alternative (full re-seed reconciliation) rejected as destructive.

### D3: Playbook API rides existing middleware stack, no new Stytch scopes

Routes: `GET /api/playbooks` (catalog, non-internal), `POST /api/playbooks/:key/apply`, `POST /api/playbooks/:key/reset`. Gated by `EntitlementMiddleware` (active subscription required, matching CRM route conventions) then existing RBAC scope checks (e.g., `contact:manage`-level scope via `auth.RequirePermissionFunc`). No new Stytch RBAC policy changes — rollback remains additive-free.

Rationale: proposal Non-Goals and governance rules; playbooks change org configuration, not identity. Alternative (new `playbook:manage` scope) rejected to keep Stytch policy delta zero.

### D4: Playbook module-dependency semantics match registry dependency rules

`requires_modules` uses the same resolution the registry already implements for `modules.requires`: a dependent playbook's module config presets apply only for enabled modules; playbook apply is allowed if all listed modules are enabled **or** the list is empty. Playbooks themselves are not features and never gate entitlements.

Rationale: avoids coupling playbook availability to billing; procedures are seed data on every plan (proposal Assumptions). Alternative (playbooks gated by purchased modules) rejected — would make OOTB onboarding conditional on purchase, defeating the "ready to use" goal.

### D5: Guiones are quick replies, sent via existing outbound path

Playbook guiones render as a chip row in the inbox UI (per active conversation). Tapping a chip fills the composer with the scripted message; the agent edits/sends via the existing `POST /api/crm/conversaciones/:id/mensajes` (free text). No new backend send endpoint; guiones are fetched with the org playbook state in `GET /api/playbooks` responses.

Rationale: user-confirmed decision (quick replies only); zero new send paths, zero WhatsApp template management; the 24h customer-service window applies as with all outbound sends today.

### D6: Frontend surfaces playbook via onboarding step, settings card, and inbox chips

- After first login / WhatsApp config, `/dashboard` shows a "¿Qué tipo de negocio es el tuyo?" step (one-time, driven by absence of `organization_playbooks` in the entitlement/playbook fetch) listing the 5 verticals → `POST /api/playbooks/{key}/apply` → continue.
- `/dashboard/settings` gains a playbook card: applied playbook, re-apply, and reset (with confirmation).
- Inbox conversation view gains a quick-reply chip row fed by guiones.

Rationale: single backend contract (`GET /api/playbooks` + apply/reset); UI driven by the same data as the API.

### D7: Seed data is the deliverable, validated by schema + tests

The five playbooks' procedures are authored in the Go catalog and covered by unit tests asserting: pipeline stage counts/orders, tag sets, config schema validity (e.g., `sla_hours` numeric), guione text non-empty. This makes the "top 5 verticals with procedures" product content machine-checked rather than doc-only.

Rationale: procedures must be testable seed data; a hand-rolled validator (type checks, matching registry approach) keeps it lightweight.

## Risks / Trade-offs

- [Apply/reset edge: org customized seeds after apply] → Mitigation: re-apply is additive and never overwrites; reset only removes rows matching the playbook preset and never touches rows with user edits (pipeline with deals, referenced tags, config values differing from preset). Documented in D2.
- [Tag deletion cascade: reset deleting tags still referenced by entities] → Mitigation: reset deletes seeded tags only when unreferenced (tag-references check via existing tag attachment queries); otherwise leaves them and notes it in the response.
- [Pipeline seed conflict: vertical pipeline vs. existing generic pipeline] → Mitigation: pipeline seeding only when org has zero pipelines; orgs with the default pipeline keep it (D2), and the playbook's stages are available as guiones/documentation until the user creates a pipeline.
- [Config preset collides with org-edited module config] → Mitigation: playbook config presets are applied only on apply; configs the org later edits are never overwritten by re-apply (upsert only sets missing keys).
- [Payload schema drift (playbook payload vs module config_schema)] → Mitigation: single Go catalog validated at startup and in unit tests against the registry; config presets validated through the same `ModuleService` path as manual edits.
- [Quick replies flood the inbox for orgs with no playbook] → Mitigation: chip row renders only when the org has applied a playbook with guiones; empty state otherwise.

## Migration Plan

1. Deploy migration 000020 (`playbooks`, `organization_playbooks` + seed rows for the 5 playbooks) — additive, safe with current code.
2. Ship playbooks module (domain/app/infra/routes) behind existing middleware — inert without UI.
3. Ship frontend onboarding step, settings card, inbox chips.
4. Dogfood: apply `comercio` playbook on org #0 and verify seeds + quick replies end-to-end.
5. Rollback: revert commits; `000020_create_playbooks.down.sql` drops tables; no Stytch policy state to reverse (no changes made); org data untouched because apply is additive.

## Open Questions

- Whether the onboarding step should be skippable forever or re-prompted until a playbook is applied (default: dismissible with a settings reminder badge).
- Whether reset should be soft (archive state) or hard delete (default: hard delete of seeded rows as scoped in D2, since re-apply restores everything).
- Whether guiones should later support variables (e.g., `{nombre_cliente}`, `{monto}`) — deferred; free text today, template variables belong to a future template capability.
