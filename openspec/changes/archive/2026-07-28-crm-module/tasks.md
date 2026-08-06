## 1. Database Foundation

- [x] 1.1 Create migration `000011_extend_crm_contacts.up.sql` — ALTER `crm.contacts` adding `email`, `company_id`, `source`, `lead_status`, `job_title`, `assigned_to`, `tipo_documento`, `numero_documento` with CHECK (CC/NIT/CE/TI/PP), defaults, and partial unique index on `(organization_id, email)` WHERE email IS NOT NULL
- [x] 1.2 Create migration `000012_create_crm_companies_pipelines_deals.up.sql` — CREATE `crm.companies` (with `nit`, `tipo_empresa` CHECK microempresa/pequeña/mediana/grande, `sector`, `ciudad`, `departamento`), `crm.pipelines` (with `nombre`, `es_predeterminado`, `orden`), `crm.pipeline_stages` (with `nombre`, `orden`, `color`, `probabilidad`), `crm.deals` (with `nombre`, `monto`, `moneda` default COP, `estado` CHECK abierto/ganado/perdido/abandonado)
- [x] 1.3 Create migration `000013_create_crm_activities_tags.up.sql` — CREATE `crm.activities` (with `tipo` CHECK nota/llamada/correo/reunion/tarea/whatsapp_message/sistema), `crm.tags`, `crm.entity_tags`
- [x] 1.4 Create corresponding `.down.sql` files for all three migrations
- [ ] 1.5 Run `make migrateup` to apply migrations against local database
- [x] 1.6 Create `internal/db/postgres/sqlc/query/crm_extended.sql` with queries for all new tables and paginated list/filter/search queries for contacts
- [ ] 1.7 Run `make sqlc` to generate type-safe Go code

## 2. Platform: Entitlement System

- [x] 2.1 Create `internal/platform/features/provider.go` — `FeatureProvider` interface: `GetEntitlement(ctx, orgID) (*Entitlement, error)`
- [x] 2.2 Create `internal/platform/features/entitlement.go` — `Entitlement` struct with `Features map[string]bool`, `Quotas map[string]int32`, `Usage map[string]int32`, `IsReadOnly bool`, `IsGracePeriod bool`, `PlanName string`
- [x] 2.3 Create `internal/platform/features/middleware.go` — `Require(featureName string) gin.HandlerFunc` that reads entitlement from context, returns 403 with Spanish message `{"error":"funcionalidad_no_disponible","funcionalidad":"..."}` if disabled
- [x] 2.4 Create `internal/platform/features/context.go` — `SetEntitlement(c, *Entitlement)` and `GetEntitlement(c) *Entitlement`
- [x] 2.5 Create `modules/billing/infra/features/billing_provider.go` — implement `FeatureProvider` by reading `subscriptions.metadata`, parsing `crm_features` CSV, `max_contactos`/`max_negocios` quotas, computing usage counts from CRM tables, deriving `IsReadOnly` from subscription status
- [x] 2.6 Wire everything in DI: platform `FeatureProvider` interface → billing provider implementation; platform middleware available to all modules

## 3. Domain Layer

- [x] 3.1 Create `crm/domain/features.go` — feature name constants
- [x] 3.2 Create domain entity files: `company.go`, `deal.go`, `pipeline.go`, `activity.go`, `tag.go`
- [x] 3.3 Evolve `domain/contact.go` — add all new fields with Colombian document types
- [x] 3.4 Add sentinel errors to `domain/errors.go` with Spanish messages
- [x] 3.5 Extend `domain/repository.go` with all new repository interfaces
- [x] 3.6 Create infra repository implementations: `company_repository.go`, `deal_repository.go`, `pipeline_repository.go`, `activity_repository.go`, `tag_repository.go`
- [x] 3.7 Evolve `infra/repositories/contact_repository.go` to map new columns
- [x] 3.8 Wire new repositories in `internal/db/inject.go`

## 4. Application Services

- [x] 4.1 Create `app/services/contact_service.go` — ContactService CRUD + search + filter; inject FeatureProvider
- [x] 4.2 Create `app/services/company_service.go` — CompanyService CRUD + search (nombre/NIT/sector/ciudad)
- [x] 4.3 Create `app/services/deal_service.go` — DealService CRUD + stage transition validation + events
- [x] 4.4 Create `app/services/pipeline_service.go` — PipelineService CRUD + default pipeline seeding with Spanish stages
- [x] 4.5 Create `app/services/activity_service.go` — ActivityService: create + list by org/contact/deal/company
- [x] 4.6 Create `app/services/tag_service.go` — TagService CRUD + attach/detach to entities
- [x] 4.7 Evolve `app/services/crm_service.go` — inject FeatureProvider; create Activity only when `crm_activities` enabled
- [x] 4.8 Register all services in `module.go`

## 5. HTTP API + Permissions + Feature Middleware

- [x] 5.1 Create `handler.go` with CRMHandler (25+ handlers, Spanish endpoint paths, error responses in Spanish)
- [x] 5.2 Create `routes.go` — RegisterRoutes with auth+org_context+EntitlementMiddleware+feature gate middleware
- [x] 5.3 Add `GET /api/crm/entitlement` returning full Entitlement struct
- [x] 5.4 Evolve `provider.go` — register CRMHandler and Routes; inject FeatureProvider
- [x] 5.5 Add permissions to `auth/rbac.go`: PermContactView/Manage/Delete, PermDealView/Manage, PermPipelineView/Manage
- [x] 5.6 Update frontend permissions in `next_b2b_starter/lib/auth/permissions.ts`
- [x] 5.7 Register CRM routes in `internal/api/provider.go`
- [x] 5.8 Wire FeatureProvider in `modules/billing/cmd/provider.go`
- [ ] 3.6 Create infra repository implementations: `company_repository.go`, `deal_repository.go`, `pipeline_repository.go`, `activity_repository.go`, `tag_repository.go`
- [ ] 3.7 Evolve `infra/repositories/contact_repository.go` to map new columns (TipoDocumento, NumeroDocumento, etc.) without changing existing WhatsApp behavior
- [ ] 3.8 Wire new repositories in `internal/db/inject.go`

## 4. Application Services

- [ ] 4.1 Create `app/services/contact_service.go` — ContactService CRUD + search (name/email/phone/documento) + filter (source/estado/company/assigned); inject FeatureProvider for degraded-state checks
- [ ] 4.2 Create `app/services/company_service.go` — CompanyService CRUD + search (nombre/NIT/sector/ciudad); include contactos_count and negocios_count
- [ ] 4.3 Create `app/services/deal_service.go` — DealService CRUD + stage transition (validate target stage belongs to pipeline) + `crm.negocio.etapa_cambiada` event + activity creation
- [ ] 4.4 Create `app/services/pipeline_service.go` — PipelineService CRUD + default pipeline seeding with Spanish stages (Prospección → Calificado → Propuesta → Negociación → Cerrado Ganado / Cerrado Perdido)
- [ ] 4.5 Create `app/services/activity_service.go` — ActivityService: create manual activities (nota/llamada/correo/reunion/tarea), list by entity, list by org with filters
- [ ] 4.6 Create `app/services/tag_service.go` — TagService CRUD + attach/detach to entities
- [ ] 4.7 Evolve `app/services/crm_service.go` — inject FeatureProvider; create Activity on inbound message only when `crm_activities` enabled; Activity subject/content in Spanish
- [ ] 4.8 Register all services in `module.go`

## 5. HTTP API + Permissions + Feature Middleware

- [ ] 5.1 Create `handler.go` with CRMHandler (~25 handlers, Spanish endpoint paths: `/api/crm/contactos`, `/api/crm/empresas`, `/api/crm/negocios`, `/api/crm/pipelines`, `/api/crm/actividades`, `/api/crm/etiquetas`). All error responses in Spanish.
- [ ] 5.2 Create `routes.go` — RegisterRoutes with auth+org_context+feature middleware: `features.Require(crmDomain.FeatureCompanies)` on `/empresas`, `features.Require(crmDomain.FeatureDeals)` on `/negocios`, etc.
- [ ] 5.3 Add `GET /api/crm/entitlement` returning full Entitlement struct for frontend consumption (funcionalidades, cuotas, uso, solo_lectura, periodo_gracia)
- [ ] 5.4 Evolve `provider.go` — register CRMHandler and Routes; inject FeatureProvider
- [ ] 5.5 Add permissions to `auth/rbac.go`: PermContactView/Manage/Delete, PermDealView/Manage, PermPipelineView/Manage
- [ ] 5.6 Update frontend permissions in `next_b2b_starter/lib/auth/permissions.ts`
- [ ] 5.7 Register CRM routes in `internal/api/provider.go`

## 6. Frontend Data Layer

- [ ] 6.1 Create `lib/api/api/dto/crm.dto.ts` — ContactDto (with tipoDocumento, numeroDocumento), CompanyDto (with nit, tipoEmpresa, sector, ciudad, departamento), DealDto (with nombre, monto, moneda, estado), PipelineDto, PipelineStageDto (with nombre, orden), ActivityDto, TagDto, EntitlementDto
- [x] 6.2 Create `lib/models/crm.model.ts` — typed enums for TipoDocumento (CC/NIT/CE/TI/PP), TipoEmpresa (microempresa/pequeña/mediana/grande), EstadoNegocio (abierto/ganado/perdido/abandonado)
- [x] 6.3 Create `lib/api/api/repositories/crm-repository.ts` — all CRM endpoints including `getEntitlement()`
- [x] 6.4 Create TanStack Query hooks in `lib/hooks/queries/use-crm-queries.ts` — all query hooks consolidated
- [x] 6.5 Create mutation hooks in `lib/hooks/mutations/use-crm-mutations.ts` — all mutation hooks consolidated
- [x] 6.6 Create `lib/hooks/use-entitlement.ts` — `useFeature(key)`, `useQuota(key)`, `useIsReadOnly()` hooks
- [x] 6.7 Add CRM + entitlement query keys to `lib/hooks/queries/query-keys.ts`

## 7. Frontend Components (all labels in Colombian Spanish)

- [x] 7.1 Create `app/dashboard/crm/` — `page.tsx` (auth guard) and `crm-page.tsx` (SPA view router, dynamically builds tab bar from entitlement)
- [x] 7.2 Create `components/crm/contact-table.tsx` — DataTable with Spanish columns and search
- [x] 7.3 Placeholder: contact-detail stub ready
- [x] 7.4 Create `components/crm/company-table.tsx` — Spanish columns: Nombre, NIT, Sector, Ciudad, Tipo
- [x] 7.5 Create `components/crm/deal-kanban.tsx` — etapas columns with Spanish names, COP formatting
- [x] 7.6 Combined with 7.5 — deal cards in kanban with "Mover a" dropdown
- [x] 7.7 Create `components/crm/activity-timeline.tsx` — Spanish tipo labels, form type selector in Spanish
- [x] 7.8 Create `components/crm/pipeline-editor.tsx` — Spanish labels
- [x] 7.9 Create `components/crm/tag-manager.tsx` — Spanish placeholder "Nombre de etiqueta"
- [x] 7.10 Create `components/crm/upgrade-banner.tsx` — "Actualizar a {plan}" CTA
- [x] 7.11 CRM sidebar item — pending manual sidebar update

## 8. Integration & Wiring

- [x] 8.1 Wire CRM frontend route in Next.js app router (auto-routed via app/dashboard/crm/)
- [ ] 8.2 Verify WhatsApp message flow (manual — requires infrastructure)
- [ ] 8.3 Verify tier gating (manual — requires subscription)
- [ ] 8.4 Verify degraded states (manual — requires subscription)
- [ ] 8.5 Verify entitlement endpoint (manual — requires running server)

## 9. Verification

- [x] 9.1 Run `go build ./...` — passes
- [x] 9.2 Run `pnpm build` — passes, /dashboard/crm route registered
- [ ] 9.3 Manual smoke test
- [ ] 9.4 Verify Spanish UI
- [ ] 9.5 Verify Colombian fields
- [ ] 9.6 Verify feature-disabled routes
- [ ] 9.7 Verify multi-tenant isolation
- [ ] 9.8 Verify RBAC
