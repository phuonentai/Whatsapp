## Why

The existing CRM module silently captures WhatsApp messages into contacts/conversations/messages but has no HTTP API, web UI, or relationship-management capabilities. There is no way for users to view their contacts, manage sales pipelines, track deals, or log activities. This change transforms the WhatsApp inbox into a modern, full-featured CRM usable from the dashboard — modularly gated by subscription tier so tenants only see what their plan includes.

**Market focus:** Colombian market first, expandable to LATAM. NIT tax identification, COP currency, Colombian Spanish UI/API, and Colombian mobile phone validation (already present at `pkg/whatsapp/phone.go` enforcing `+573XXXXXXXXX`). Architecture supports per-country configuration for later LATAM expansion (currency, phone patterns, tax ID formats).

## What Changes

- **Evolve Contact entity** with new fields: email, company_id, source, lead_status, job_title, assigned_to, tipo_documento (CC/NIT/CE/TI/PP), numero_documento. Colombian phone validation preserved.
- **New Company entity** for CRM organizations (distinct from tenant organizations) with name, nit (NIT + dígito de verificación), tipo_empresa (microempresa/pequeña/mediana/grande), sector, ciudad, departamento, website, notes
- **New Deal entity** for sales opportunities with amount, currency (default COP — pesos colombianos), stage, pipeline, contact/company associations
- **New Pipeline + PipelineStage entities** for configurable per-tenant sales pipelines. Default stages in Colombian Spanish: Prospección, Calificado, Propuesta, Negociación, Cerrado Ganado, Cerrado Perdido
- **New Activity entity** for a unified timeline: notas, llamadas, correos, reuniones, tareas, mensajes WhatsApp, eventos del sistema
- **New Tag entity** with many-to-many entity_tags junction for labeling contacts, companies, and deals (etiquetas)
- **New HTTP API** (~25 endpoints) for full CRUD on contacts, companies, deals, pipelines, activities, and tags. Error messages in Colombian Spanish.
- **New CRM dashboard page** with contact list, company list, deal kanban board, activity timeline, and pipeline editor. Full UI in Colombian Spanish.
- **New RBAC permissions**: contact:view/manage/delete, deal:view/manage, pipeline:view/manage
- **WhatsApp bridge**: inbound messages auto-create Activity records on the timeline (when feature enabled)
- **Auto-seeded default pipeline** with standard sales stages on first tenant access (stages in Spanish)
- **Platform entitlement system** (`platform/features/`) defining `FeatureProvider` interface and `Entitlement` struct (features, quotas, usage, degraded-state flags). The billing module implements `FeatureProvider`, reading subscription metadata synced from Polar.sh. CRM module defines its own feature name constants. Feature middleware gates API routes. Zero platform-to-module coupling.
- **Subscription-aware degraded states**: periodo de gracia (payment failed, features still active with nag), solo lectura (canceled, data viewable but not mutable), deshabilitado (expired, 403). Vista previa de mejoras for features not in current plan.
- **Multi-tenant isolation**: all CRM data scoped per organization_id

## Capabilities

### New Capabilities

- `entitlement-system`: Platform-level `FeatureProvider` interface + `Entitlement` struct (features, quotas, usage, read-only flag). The billing module implements `FeatureProvider` by reading subscription metadata from Polar.sh product config. CRM module owns its feature name constants. Middleware enforces entitlement at route level.
- `contact-management`: Full CRUD for contacts with email, company association, lead status, source tracking, assignment to team members. Supports Colombian document types (CC, NIT, CE, TI, PP).
- `company-management`: Full CRUD for CRM companies with NIT (tax ID), Colombian business size categories (microempresa/pequeña/mediana/grande), sector, ciudad, departamento, and owner assignment
- `deal-management`: Full CRUD for deals with pipeline/stage tracking, amount in COP (pesos colombianos), status lifecycle (abierto/ganado/perdido/abandonado), stage transitions
- `pipeline-management`: Per-tenant configurable sales pipelines with reorderable stages in Colombian Spanish, colors, and probability percentages; auto-seeded default pipeline (Prospección → Calificado → Propuesta → Negociación → Cerrado Ganado / Cerrado Perdido)
- `activity-timeline`: Unified activity log (notas, llamadas, correos, reuniones, tareas, mensajes WhatsApp, eventos del sistema) linkable to contacts, companies, deals; create and query timeline per entity
- `tag-management`: Per-tenant tags (etiquetas) with colors, attachable to contacts/companies/deals via many-to-many junction
- `crm-frontend`: Dashboard SPA in Colombian Spanish with contact list, company list, deal kanban board, activity timeline, pipeline editor, and sidebar navigation entry. UI dynamically adapts to enabled features per subscription tier, shows upgrade previews (vista previa de mejoras) for unavailable features.

### Modified Capabilities

- `crm-core-data`: Contact entity gains email, company_id, source, lead_status, job_title, assigned_to, tipo_documento, numero_documento fields. CRMService.ProcessInboundMessage creates an Activity record only when `crm_activities` feature is enabled. Existing WhatsApp flow semantics are preserved.

## Impact

- **Platform**: New `internal/platform/features/` package — defines `FeatureProvider` interface, `Entitlement` struct, `Require` middleware, context helpers. No imports from any module. Reusable by any future module.
- **Billing module**: New `modules/billing/infra/features/billing_provider.go` — implements `FeatureProvider`. Reads subscription metadata (synced from Polar.sh product config), parses CSV feature lists and quota values, computes current usage from CRM tables.
- **CRM module** (`modules/crm/`): New domain entities, repositories, services, handlers, routes, events. `crm/domain/features.go` defines feature name constants. CRMService injects `FeatureProvider`.
- **Subscription metadata**: Polar.sh products carry CRM feature lists and quotas in metadata (e.g., `crm_features: contacts_manage,companies,deals,activities`, `max_contactos: 1000`, `max_negocios: 100`). Synced via webhook to `subscriptions.metadata`.
- **Database**: New migration adding 6 tables and ALTERing `crm.contacts`. No new tables for entitlement storage (derived from subscription).
- **SQLC**: New query file `crm_extended.sql`; regenerate generated code
- **RBAC**: New permissions added to `auth/rbac.go` (contact:*, deal:*, pipeline:*)
- **Frontend**: New `app/dashboard/crm/` page, ~15 components, DTOs, repositories, models, query hooks, mutation hooks; `useEntitlement` hook for feature/usage checks and upgrade preview renders. All UI labels, buttons, placeholders, errors in Colombian Spanish.
- **No breaking changes**: Existing WhatsApp flow, repositories, SQLC queries, and event bus integration remain intact
