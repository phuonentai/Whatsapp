## Context

The existing CRM module (`go-b2b-starter/internal/modules/crm/`) currently handles WhatsApp message ingestion only. It has domain entities for Contact, Conversation, and Message, with repository interfaces backed by SQLC-generated code. There are no HTTP handlers, API routes, or frontend pages — data is only accessible through the database.

The module follows the codebase's Clean Architecture pattern: `domain/` (entities, repository interfaces, errors), `app/services/` (business logic), `infra/repositories/` (SQLC implementations), `cmd/init.go` (DI wiring + event subscriptions). All data is multi-tenant scoped via `organization_id` FK to `organizations.organizations(id)`.

The frontend uses Next.js 16 with React 19, shadcn/ui, TanStack Query v5, and follows a 3-layer data pattern: DTOs → Repositories → Models → Query/Mutation Hooks. The dashboard uses an SPA-within-Next.js view stack pattern.

**Constraints:**
- All CRM data must be scoped per `organization_id` (multi-tenant isolation)
- No new external dependencies (reuse PostgreSQL, SQLC, gin, dig, shadcn/ui, TanStack Query)
- WhatsApp message flow must remain unchanged (backward compatibility)
- Follow existing module patterns (handler, route, provider, DI wiring)
- RBAC permissions use `resource:action` format with middleware checks
- **Platform layer must not import module code** (Clean Architecture: dependencies point inward)

## Goals / Non-Goals

**Goals:**
- Evolve the WhatsApp-only CRM into a general-purpose CRM with contacts, companies, deals, pipelines, activities, and tags
- Expose all CRM data via a RESTful HTTP API with RBAC-protected endpoints
- Build a dashboard SPA with contact list, company list, deal kanban, activity timeline, and pipeline editor
- Create Activity records for inbound WhatsApp messages (bridge existing messaging data into the new timeline)
- Auto-seed a default sales pipeline per tenant on first access
- Gate CRM features by subscription tier so tenants only access what their plan includes
- Support degraded subscription states (grace period, read-only, disabled)
- Keep the entitlement architecture dependency-inverted: platform defines the contract, billing implements it, CRM owns its feature names

**Non-Goals:**
- Email integration or multi-channel messaging beyond WhatsApp
- Full-text search (ILIKE queries for v1; PostgreSQL FTS can be added later)
- CSV import/export (source field is ready but import endpoints are out of scope)
- Real-time updates (polling via TanStack Query refetch; no WebSocket)
- Custom fields beyond the `metadata JSONB` column already present on all entities
- Drag-and-drop kanban reordering (v1 uses stage-change API calls, not real drag)
- Quota enforcement at database level (quotas are informational for v1; hard enforcement later)
- Plan mapping UI (plan names and feature sets are configured in Polar.sh product metadata)

## Decisions

### D1: Contact identity stays phone-centric

**Choice:** Keep `UNIQUE(org_id, phone_number)` and add email as nullable with a partial unique index (`WHERE email IS NOT NULL`).

**Alternatives considered:**
- Switch to email-as-identity: would break WhatsApp flow, require migration rewriting data
- Remove unique constraint entirely: duplicate detection becomes application-level, more error-prone

**Rationale:** Zero risk to existing WhatsApp flow. The partial unique index on email prevents duplicate email contacts without blocking nulls.

### D2: Activity and Message are separate layers

**Choice:** Keep `Message` as a WhatsApp-specific payload table and create `Activity` as a higher-level CRM timeline. When a WhatsApp message arrives, both a Message AND an Activity are created (if the feature is enabled).

**Alternatives considered:**
- Repurpose Message as Activity: WhatsApp-specific fields (whatsapp_message_id, direction, message_type) don't fit general activities; schema becomes confusing
- Unify into one polymorphic table: loses type safety of individual tables; complicates queries

**Rationale:** Two-layer model keeps messaging infrastructure clean while giving the CRM layer a unified timeline. Activity has an optional FK to `conversation_id` and `metadata {message_id: ...}` for backlinks.

### D3: Company, not Organization, for CRM companies

**Choice:** Name the CRM company entity `Company` (table: `crm.companies`).

**Alternatives considered:**
- Call it Organization: catastrophic confusion with `organizations.organizations` (the tenant table)
- Call it Account: already used for `organizations.accounts` (user accounts)

**Rationale:** `Company` is the industry-standard term for a CRM company record. `Company` is unambiguous in this codebase.

### D4: Pipeline seeding in service layer

**Choice:** Auto-create a default pipeline with 6 stages on first tenant access via `PipelineService.GetDefaultForOrg()`.

**Alternatives considered:**
- Migration-based seeding: requires per-tenant rows at migration time, doesn't scale to new tenants
- Always create on module init: wasteful for tenants that never use CRM

**Rationale:** Lazy creation on first access is the standard pattern (same as how the system handles Stytch organization bootstrapping).

### D5: Stage transitions via explicit API endpoint

**Choice:** `PUT /api/crm/deals/:id/stage` with stage_id in the body, validated by the service layer.

**Alternatives considered:**
- Update stage via general `PUT /deals/:id`: conflates stage change with general deal updates, harder to emit proper events
- Kanban drag-and-drop with optimistic UI: requires complex state management; v1 uses simple API call

**Rationale:** Dedicated endpoint makes stage transitions explicit, emits `crm.deal.stage_changed` events, and validates that the stage belongs to the deal's pipeline.

### D6: SQLC queries in separate file

**Choice:** New SQLC queries go in `crm_extended.sql`, not appended to existing `crm.sql`.

**Alternatives considered:**
- Append to existing crm.sql: regenerating would touch existing generated code, increasing risk

**Rationale:** Separate file means existing generated code (`gen/crm.sql.go`) is untouched. Only new generated code is added.

### D7: Activity creation failure is non-blocking

**Choice:** If `ActivityRepo.Create` fails during WhatsApp message processing, log and continue. The Message is still saved.

**Alternatives considered:**
- Fail the entire message processing: a non-critical Activity failure would prevent message persistence
- Retry Activity creation: adds complexity for an auxiliary record

**Rationale:** The core WhatsApp flow (contact/message persistence) must not be degraded by the new Activity feature. Activity is a value-add, not a critical path.

### D8: Dependency inversion — platform defines the contract, billing implements it

**Choice:** The `platform/features/` package defines `FeatureProvider` interface and `Entitlement` struct. The billing module implements `FeatureProvider` in `modules/billing/infra/features/billing_provider.go`. CRM module defines its own feature name constants. **Platform imports nothing from modules.**

```
┌──────────────────────────────────────────────────────────────────────┐
│                    DEPENDENCY DIRECTION                              │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  platform/features/          (defines contract only)                 │
│  ├── provider.go             FeatureProvider interface               │
│  ├── entitlement.go          Entitlement struct                      │
│  ├── middleware.go           Require(feature) gin handler            │
│  └── context.go              SetEntitlement / GetEntitlement         │
│       ▲                                                              │
│       │  implements                                                  │
│       │                                                              │
│  modules/billing/infra/features/                                     │
│  └── billing_provider.go     Implements FeatureProvider              │
│       │  reads subscription metadata, parses feature lists,          │
│       │  computes usage from CRM tables, returns Entitlement         │
│                                                                      │
│  modules/crm/domain/features.go   CRM owns its feature names         │
│  FeatureDeals = "crm_deals"                                         │
│                                                                      │
│  Route usage:                                                        │
│  features.Require(crmDomain.FeatureDeals)  ← CRM provides the key    │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

**Alternatives considered:**
- Platform imports billing: violates Clean Architecture at the foundation layer. Every future module that needs features would drag billing into platform.
- FeatureService lives in CRM module: other modules (documents, cognitive) cannot reuse it. Duplicated entitlement logic.
- Feature names in platform: platform accumulates module-specific knowledge over time.

**Rationale:** This is the standard Dependency Inversion Principle pattern from Clean Architecture. Platform provides the generic machinery (interface, struct, middleware). Modules bring their own domain-specific feature keys. The billing module bridges the two by implementing the platform interface.

### D9: Polar.sh product metadata is the source of truth for entitlements

**Choice:** Feature lists and quota values live in Polar.sh product metadata, synced to `subscriptions.metadata` via webhook. The billing provider parses this metadata to build the `Entitlement`. No hardcoded plan-to-feature maps in Go code.

**Polar.sh product metadata example:**
```json
{
  "crm_features": "contacts_manage,companies,deals,activities",
  "max_contacts": 1000,
  "max_deals": 100
}
```

Sync flow:
```
Polar.sh "Pro Plan" product config
  └── metadata: { crm_features: "contacts_manage,companies,deals,activities", max_contacts: 1000 }
      │
      │  Polar webhook or subscription sync
      ▼
subscriptions.metadata (JSONB column)
      │
      │  FeatureProvider reads on every request
      ▼
Entitlement{ Features: {"crm_deals": true}, Quotas: {"contacts": 1000} }
```

**Alternatives considered:**
- Go map of plan names to features: every plan change requires a code deploy; PM can't self-serve
- Database feature_flags table: source of truth drifts from billing system; reconciliation nightmare

**Rationale:** The billing system IS the authority on what a customer paid for. Duplicating that in code or a separate DB table creates drift. Polar.sh already supports arbitrary product metadata — use it.

### D10: Entitlement is richer than binary features

**Choice:** The `Entitlement` struct carries more than a feature on/off map:

```go
type Entitlement struct {
    Features   map[string]bool  // "crm_deals": true
    Quotas     map[string]int32 // "contacts": 1000, "deals": 100  (0 = unlimited)
    Usage      map[string]int32 // "contacts": 47, "deals": 12    (current count)
    IsReadOnly bool              // true when subscription is canceled/expired
    IsGracePeriod bool           // true when payment failed but within grace window
}
```

**Subscription state → entitlement behavior:**

| Subscription State | Features Available | Writes Allowed | UI Behavior |
|-------------------|-------------------|---------------|-------------|
| Active / Trialing | Per-plan map | Yes | Full access |
| Past Due (< 14d) | Per-plan map | Yes | Yellow "Update payment" banner |
| Past Due (> 14d) | Per-plan map | **No** — `IsReadOnly = true` | Read-only + "Reactivate" CTA |
| Canceled | Per-plan map | **No** — `IsReadOnly = true` | Read-only, data export available |
| Unpaid / None | Empty map | No | CRM sidebar hidden, paywall gate |

**Frontend behavior for unavailable features (upgrade preview):**

When a feature is NOT in the current plan but IS in a higher plan:
- Show the tab/UI section grayed out with a padlock icon
- Show an upgrade CTA: "Unlock with Pro"
- API returns 403 for writes; reads are allowed for data visibility

**Alternatives considered:**
- Binary on/off with 403: canceled users can't even export their data; upgrade paths are invisible
- Features only, no quotas: no usage tracking; can't enforce "100 contacts on Starter"

**Rationale:** Real SaaS products have degraded states, not binary gates. The `Entitlement` struct gives every downstream consumer (middleware, services, frontend) enough context to make nuanced decisions.

### D11: Frontend entitlement hook with upgrade preview

**Choice:** A `useEntitlement()` hook (parallel to `usePermissions()`) reads the entitlement from `GET /api/crm/entitlement` and provides `hasFeature(key)`, `usage(quota)`, `isReadOnly`, and `canUpgradeTo(key)` methods.

**Upgrade path model:**
```go
// Defined per module, not in platform
var crmUpgradePaths = map[string][]string{
    "crm_contacts_manage": {"crm_companies"},    // next step from contacts → companies
    "crm_companies":       {"crm_deals", "crm_tags"},
}

// On frontend:
{!hasFeature('crm_deals') && canUpgradeTo('crm_deals') && (
    <Tab label="Deals" disabled icon={Lock}>
      <UpgradeBanner plan="Pro" feature="Deal Pipeline" />
    </Tab>
)}
```

**Rationale:** Upgrade previews drive conversion. A Starter user who can't see Deals will never know to upgrade. Showing the tab with a "Pro Feature" badge creates the upgrade path.

### D12: Colombian market first, LATAM-architected

**Choice:** All CRM defaults, labels, error messages, and pipeline stages use Colombian Spanish. The data model includes Colombian-specific fields (NIT, tipo_documento, departamento). Default currency is COP. Phone validation reuses the existing Colombian `+573XXXXXXXXX` pattern from `pkg/whatsapp/phone.go`.

**LATAM expansion path:**

```
Today (Colombia):                       Future (LATAM):
─────────────────                       ────────────────
Phone: +573XXXXXXXXX (hardcoded)   →    Per-country regex from config
Currency: COP (default)            →    Per-tenant currency field
Tax ID: NIT                         →    Per-country tax ID format + validation
Doc types: CC, NIT, CE, TI, PP     →    Per-country document types
Stages: Spanish (one variant)      →    i18n key system (starts with "es-CO")
```

The architecture uses Spanish strings directly for v1 — no i18n framework. When LATAM expansion happens, string extraction to locale files is mechanical, not architectural.

**Colombian business classifications for Company entity:**

```
Tipo de empresa:
  Microempresa   (1-10 empleados, ingresos < 44.769 UVT)
  Pequeña        (11-50 empleados, ingresos 44.769-431.196 UVT)
  Mediana        (51-200 empleados, ingresos 431.196-2.160.692 UVT)
  Grande         (201+ empleados, ingresos > 2.160.692 UVT)

Sector / Industria (common Colombian categories):
  Tecnología, Manufactura, Comercio, Servicios, Agroindustria,
  Construcción, Salud, Educación, Financiero, Transporte, Alimentos y Bebidas
```

**Rationale:** Colombian businesses are the target market. The existing WhatsApp integration already validates Colombian phones. D12 codifies this as intentional, not accidental. The data model includes LATAM-expandable fields (currency, tax ID format) that are Colombian-initialized but not Colombian-locked.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| ALTER TABLE on contacts may block writes during migration | New columns are nullable with defaults; migration uses `ADD COLUMN` which is typically fast with PostgreSQL |
| Partial unique index on email may reject manual contact creation if duplicate exists | Returns a clear error message; frontend shows "A contact with this email already exists" |
| Activity table grows unbounded | Typical CRM usage is low volume per tenant; future retention policy can be added if needed |
| Contact company FK deletes set company to NULL | Prevents data loss; company deletion doesn't destroy contacts |
| Pipeline/stage deletion with active deals | Pipeline FK uses ON DELETE RESTRICT; must reassign deals first |
| Frontend bundle size increase from new CRM page | CRM page is lazy-loaded via Next.js dynamic import; no new heavy deps |
| Polar.sh metadata format changes break feature parsing | FeatureProvider validates metadata on read; logs warnings for unrecognized keys; defaults to safe state (no features) |
| Subscription webhook missed → entitlement stale | Paywall `RefreshSubscriptionStatus` lazy guard checks on 403; entitlement recalculated on next request |
| Quota usage queries add DB load per request | Usage queries are simple COUNTs on indexed FKs; FeatureProvider can cache Entitlement per request (context-scoped) |
| Grace period logic changes require code deploy | Grace period duration (14 days) is configurable in `billing_provider.go`; could be moved to subscription metadata later |

## Migration Plan

1. Deploy `platform/features/` package — zero dependencies, no DB changes. Inert until wired.
2. Deploy `billing/infra/features/billing_provider.go` — implements FeatureProvider. No behavior change until CRM consumes it.
3. Deploy migration (new tables + ALTER contacts). Zero-downtime: all changes are additive.
4. Deploy backend with CRM domain, repos, services, handlers, routes, feature middleware. WhatsApp flow unchanged.
5. Deploy frontend with CRM dashboard page, components, `useEntitlement` hook.
6. Configure Polar.sh product metadata with `crm_features` for each plan.
7. Rollback: remove frontend route, remove API routes. Tables, FeatureProvider, and platform package can stay — inert without consumers.
