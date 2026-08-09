## Why

The product today ships features as monolith plan tiers: subscription metadata carries a CSV of CRM feature keys (`crm_features`) that `billingFeatureProvider.GetEntitlement` parses into booleans. New capabilities that clients are willing to pay for separately — first among them a helpdesk **tickets** module — cannot be sold, enabled, or configured independently of plans. Mix-and-match module purchases are the most economical pricing model for this B2B SaaS: a low entry point on the base plan, multiple independent upsell paths, and clients pay only for what they use. The business also wants to run its own product (org #0 dogfooding), which a module system makes natural: an internal vendor-only module for ops tooling, invisible to tenants.

## What Changes

- **Module registry**: a catalog of sellable modules (`key`, name, description, feature keys it gates, module dependencies, config schema, `is_internal` flag for vendor-only modules like ops tooling).
- **Per-org module state**: which modules an organization has purchased/enabled, plus per-org module configuration (JSONB) so clients can adjust each module to their needs (e.g., ticket SLA hours, priorities, tags).
- **Entitlement derivation change**: `FeatureProvider.GetEntitlement` SHALL union base-plan features with features granted by purchased modules. Feature flags remain independent of RBAC permissions (unchanged invariant).
- **Purchase-to-enable loop**: module purchases flow through the existing billing pipeline (Polar/MercadoPago webhook → subscription sync) and surface in entitlement without code changes per sale.
- **First sellable module — Tickets (helpdesk)**: ticket entity with state machine, assignment, priorities, tags, SLA timers, and internal (team-only) notes. Gated by the `tickets` module feature flag; wired to existing WhatsApp inbox conversations and CRM contacts.
- **Entitlement API**: extend the existing entitlement endpoint to include module state and per-module config for the frontend.

## Capabilities

### New Capabilities

- `module-registry`: sellable module catalog, per-org module enablement and configuration, module-gating middleware, and integration of module entitlements into the platform `FeatureProvider`.
- `tickets`: helpdesk ticket capability — ticket lifecycle/state machine, assignment, priorities, tags, SLA timers, internal notes, and module-config-driven behavior.

### Modified Capabilities

- `feature-gating`: requirement changes — feature derivation now unions plan features with purchased module features, and the entitlement contract is described against the actual codebase contract (`FeatureProvider.GetEntitlement` returning `Entitlement{Features, Quotas, Usage, ...}`), replacing the stale `FeatureService.IsEnabled`/`plans.go` description. Quota/usage semantics are unchanged.

## Impact

- **Backend (Go)**:
  - `internal/modules/billing/infra/features/billing_provider.go` — extend entitlement derivation to include purchased modules.
  - `internal/platform/features/provider.go` — `Entitlement` extended with module state/config.
  - New module (e.g., `internal/modules/module-registry`) for registry + org module state + config.
  - New module (e.g., `internal/modules/tickets`) for the tickets capability.
  - New DB tables + SQLC queries: module registry, org module state/config, tickets + related (assignees, tags, notes, SLA events).
  - New API routes for module state/config and tickets; RBAC scopes for ticket operations (via Stytch RBAC policy, e.g., `ticket:view`, `ticket:manage`).
- **Billing**: Polar/MercadoPago — add-on purchase surface must flow into subscription metadata; exact product shape to be confirmed against provider APIs during implementation (see Assumptions).
- **Frontend (Next.js)**: entitlement consumption extended; module settings UI; tickets UI integrated with the WhatsApp inbox and CRM contact views.
- **Specs**: `feature-gating` delta; new `module-registry` and `tickets` specs.

## Non-Goals

- **No local credential storage.** All identity, RBAC, and session concerns remain with Stytch B2B; local PostgreSQL stores only `stytch_member_id`/`stytch_organization_id` foreign keys. Module gating rides existing Stytch RBAC; new permissions are added via Stytch RBAC policy, never a local auth table.
- No agentic ops tooling (log watchers, uptime agents, Telegram) in this change — the `is_internal` module concept provisions for it, but the tools themselves are a future change.
- No customer self-service portal (clients' customers opening tickets directly).
- No changes to the billing engine itself (invoice quotas, payment verification); only the entitlement derivation is touched.

## Rollback Strategy

- **Git state**: revert the change commit(s); DB migrations are forward-only with paired down-migrations; SQLC generated code regenerates from schema.
- **Stytch tenant policy state**: the only Stytch touchpoints are new RBAC permission scopes (`ticket:view`, `ticket:manage`) attached to existing roles. Rollback = remove the new scopes via Stytch dashboard/API. No role semantics are modified, so rollback is additive-only reversal.
- **Billing metadata**: if module metadata parsing misbehaves, `parseCRMFeatures` keeps existing keys; module keys are namespaced (`module_tickets`) so they can be ignored by a reverted build. Org module state tables are new and read-only-if-empty, so a revert leaves plans unchanged.

## Assumptions

- **Module purchase surface**: how a module purchase appears in Polar/MercadoPago subscription metadata (separate add-on product vs plan metadata field) is NOT yet verified against the provider APIs. Implementation will confirm the mechanism; until then the design treats "purchased modules arrive in subscription metadata" as the contract, matching the existing `crm_features` pattern.
- **Frontend gating mechanism**: how enabled modules surface in the Next.js app (server-side entitlement fetch vs edge middleware) is not yet verified in detail; the entitlement API shape is designed to serve either.
- **SLA semantics**: no SLA engine exists; this change ships a minimal SLA timer (due-at timestamp per priority configured in module config) without escalation automation.
