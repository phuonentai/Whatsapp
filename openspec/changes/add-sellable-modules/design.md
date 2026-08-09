## Context

The product (Go backend `go-b2b-starter` + Next.js frontend) currently derives tenant entitlements from subscription metadata: `internal/modules/billing/infra/features/billing_provider.go` implements `features.FeatureProvider.GetEntitlement`, parsing a `crm_features` CSV from subscription `Metadata` into a boolean map, plus quotas (`max_contactos`, `max_negocios`) and usage (DB counts). Middleware in `internal/platform/features/middleware.go` (`EntitlementMiddleware`, `Require`, `RequireActiveSubscription`) gates routes; handlers read the entitlement via `features.GetEntitlement(c)`; `GET /api/crm/entitlement` exposes it. Billing flows through Polar (subscription sync) and MercadoPago (in-flight changes).

This change makes the product a **modular platform**: sellable modules (first: `tickets` helpdesk) that clients enable and configure per-org, purchased mix-and-match via the billing pipeline. The same module system supports a future vendor-only (`is_internal`) module for the business's own ops tooling (org #0 dogfooding), but no ops tools ship in this change.

Constraints from governance: Clean Architecture (domain → app → infra), SQLC-generated queries, Stytch B2B as runtime SSOT for identity/RBAC (new permissions via Stytch RBAC policy), feature flags independent of permissions, and existing module layout conventions (`internal/modules/<capability>/` with `domain/`, `app/services/`, `infra/`, `routes.go`, `handler.go`, `module.go`, `cmd/init.go`).

## Goals / Non-Goals

**Goals:**
- A registry of sellable modules, data-driven and extensible without code changes per sale.
- Per-org module enablement and per-org module configuration (validated JSONB).
- Entitlement derivation = plan features ∪ purchased-module features, extending the existing `billingFeatureProvider`.
- `tickets` as the first shipped module: lifecycle, assignment, priorities/tags, SLA due-at, internal notes, append-only events, gated by module + RBAC scopes.
- Purchase-to-enable loop riding the existing Polar/MercadoPago webhook → subscription sync pipeline.

**Non-Goals:**
- No agentic/ops tooling (log watchers, uptime agents, Telegram) — only the `is_internal` registry concept.
- No SLA escalation automation (minimal due-at timestamps only).
- No customer self-service portal.
- No billing engine changes (invoice quotas, payment verification) — entitlement derivation only.
- No local credential/identity storage; Stytch remains sole identity/RBAC authority.

## Decisions

### D1: Modules live in a database registry + per-org state table, not code flags

New tables: `modules` (registry: key, name, description, granted_features JSONB, requires JSONB, config_schema JSONB, is_internal, is_active) and `organization_modules` (org_id, module_key, config JSONB, enabled_at, UNIQUE(org_id, module_key)). Seeded via migration from a Go constant list (single source in the registry module).

Rationale: sellable catalog must be queryable by billing/entitlement and API, and config needs persistence per org. Alternatives considered: (a) hardcoded Go map — rejected, no per-org config persistence; (b) feature flags only — rejected, no catalog metadata or config for UI/pricing.

### D2: Module enablement derives from subscription metadata, mirroring `crm_features`

`billingFeatureProvider.GetEntitlement` gains a `parseModules(metadata)` step reading namespaced keys (e.g., `module_tickets`), cross-referencing the registry, and materializing `organization_modules` state as a view of metadata (derived, not independently mutable by API). Module feature keys (e.g., `tickets_module`) are namespaced to avoid plan-key collisions.

Rationale: keeps the purchase loop free — Polar/MP webhook → subscription sync → entitlement, no code per sale; identical mechanics to the proven `crm_features` path. Alternative: separate `module_purchases` table written by a billing webhook handler — deferred until the provider add-on shape is confirmed (see Open Questions); the derived-from-metadata approach degrades safely to it.

### D3: `Entitlement` is extended, not forked

`internal/platform/features/provider.go` `Entitlement` gains `Modules map[string]ModuleState` (key → {enabled, config}) plus `ModuleConfigs`. The existing per-request caching in `EntitlementMiddleware` is preserved; module state rides the same single read per request.

Rationale: one entitlement contract for middleware, handlers, and the frontend; avoids a second lookup path that can drift from the feature cache.

### D4: Module-gating middleware composes with existing gates

New `modules.Require(key)` in the registry module's infra, wired as `EntitlementMiddleware` → `modules.Require("tickets")` → permission middleware, returning `{"error":"module_disabled","module":"tickets"}` 403. Dependency enforcement (`requires`) evaluated inside entitlement derivation: dependent module features disabled unless dependency enabled.

Rationale: matches existing middleware ordering (feature gate before RBAC), keeps a single choke point.

### D5: Tickets is a conventional Clean Architecture module

`internal/modules/tickets/` with `domain/` (Ticket, TicketEvent, state machine), `app/services/` (create/assign/transition/notes/SLA), `infra/` (SQLC repository), `routes.go`/`handler.go`/`module.go`/`cmd/init.go`, following the CRM module layout. Tables: `tickets`, `ticket_events`, `ticket_notes` (or events-of-type-note), all org-scoped (existing `organization_id` convention) and referencing contacts/conversations by FK where applicable. Assignees stored as `stytch_member_id` (no local member table).

Rationale: consistency with the codebase's established module pattern; RBAC scopes (`ticket:view`, `ticket:manage`) added via Stytch RBAC policy, mirroring `deal:view`/`deal:manage` precedent.

### D6: SLA is a computed timestamp, not a scheduler

`sla_due_at` computed on priority change from module config `sla_hours[priority]`; no background escalation job in this change. SLA-overdue is a derived query (due_at < now and not resolved).

Rationale: minimal scope per proposal Assumptions; avoids a cron subsystem until demand justifies it.

### D7: Internal notes are a ticket-event variant with team-only visibility

Note entries recorded in `ticket_events` (event_type `note_internal`, actor, body), excluded from any WhatsApp outbound path by construction — the tickets module has no WhatsApp-send tool.

Rationale: append-only event history already required by spec; no separate note table/API surface.

### D8: Frontend consumes entitlement modules, renders module settings

Frontend extends the existing entitlement consumption (`useEntitlement`/equivalent) with `modules`; a `/settings/modules` page lists the catalog (non-internal), shows enabled state, and edits per-module config (validated by backend schema). Tickets UI adds a ticket panel in the inbox/contact views with list, detail, transition, assign, and internal-note actions.

Rationale: one backend contract; UI gating driven by the same flags as API gating. Frontend mechanism detail (server-side fetch vs edge middleware) to be confirmed against the actual entitlement consumption during implementation (see proposal Assumptions).

## Risks / Trade-offs

- [Module metadata parsing mismatch (Polar/MP add-on shape unknown)] → Mitigation: keep `module_*` keys namespaced and additive; a revert ignores unknown keys (existing `parseCRMFeatures` behavior); isolate parsing in one function for a one-line swap when provider shape is confirmed.
- [Spec/code drift: `feature-gating` spec described a `FeatureService`/`plans.go` that no longer exists] → Mitigation: this change's MODIFIED requirement documents the real `FeatureProvider.GetEntitlement` contract; archive will reconcile the living spec.
- [Per-request entitlement read grows heavier (registry + modules + usage)] → Mitigation: registry is tiny and can be cached in-memory with a short TTL; module state rides the existing single read per request; usage counts unchanged.
- [Config schema validation complexity (JSONB schema in registry)] → Mitigation: start with a minimal hand-rolled validator keyed by module (e.g., type checks for `sla_hours`, `priorities`, `tags`), not a full JSON-Schema engine.
- [Tenant visibility leak of `is_internal` modules] → Mitigation: internal modules filtered in both catalog and entitlement API paths; add a regression test asserting absence.
- [Tickets + WhatsApp: customers replying inside a linked conversation may bypass ticket lifecycle] → Mitigation: tickets are a view over conversations, not a replacement; inbound messages still create Conversations/Messages; ticket state is a separate concern — no coupling change to the WhatsApp bridge.

## Migration Plan

1. Deploy registry seed migration (new tables) — additive, safe with current code.
2. Ship `billingFeatureProvider` extension (module parsing) — unknown metadata keys ignored, so existing subscriptions unaffected.
3. Ship registry + tickets module routes/UI behind `modules.Require("tickets")` — inert until a subscription carries `module_tickets`.
4. Enable org #0 (dogfood) by setting `module_tickets` in its subscription metadata.
5. Rollback: revert commits; down-migrations drop new tables; Stytch RBAC changes are additive scope removals; module keys ignored by a reverted provider build (per proposal Rollback Strategy).

## Open Questions

- Exact Polar/MercadoPago add-on purchase shape: separate product vs plan metadata field, and how webhook sync surfaces it in `Metadata` (needs provider API verification during implementation).
- Whether module config should be mutable by API for org #0 only (dogfooding) or for all tenants immediately (pricing/billing of config features like SLA hours is TBD).
- Frontend entitlement consumption mechanism to be confirmed (server fetch vs edge middleware) — affects where module state is injected.
