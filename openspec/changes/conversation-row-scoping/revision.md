# Revision — conversation-row-scoping (rev 2)

REVISION: COMPLETE

Revisión de los artifacts de planificación en respuesta al VERDICT.md del council (STATUS: REJECTED, MARKET: CONDITIONAL). Se revisaron: `proposal.md`, `design.md`, `tasks.md` y los delta specs `feature-gating`, `lean-data-isolation`, `stytch-authorization`, `whatsapp-webhook-ingress`, `whatsapp-inbox`, `inbox-ui`. No se tocó código de aplicación. Lista numerada del verdict → cobertura:

## 1. Mecanismo de feature-gating real (V1)

- **Hecho**: verificado en código — `FeatureService.IsEnabled` y `internal/platform/features/plans.go` NO existen; el mecanismo real es `FeatureProvider.GetEntitlement` → `Entitlement.Features` (metadata de suscripción `crm_features`/`ai_features`/`module_*` + módulos del catálogo `granted_features` + `PlanName` de Polar; free tier → Features vacío; precedente `defaultGrantedModules`).
- **Cambios**: design.md Decisión 8 reescrita (flag sobre entitlement real, semántica activa/trialing/past_due vs free/inactiva, canal de grant a elegir en apply, drift spec↔código declarado); delta spec `feature-gating` reescrito (flag en entitlement, escenarios free/inactiva/past_due, resolución única por request); proposal.md (What Changes, Impact, Assumptions); tasks.md 3.4 (elegir y documentar el canal de grant; validar plan names vs catálogo Polar).
- **Pendiente (supuesto)**: nombres de plan "Starter/Pro/Enterprise" y canal de grant → validar contra catálogo Polar en apply.

## 2. Semántica de session vars RLS + workers (S1/S1b)

- **Cambios**: design.md Decisión 9 nueva — `SET LOCAL` obligatorio dentro de la transacción del request (nunca `SET` a nivel sesión sobre el pool), fail-closed con observabilidad (500 explícito, no lista vacía silenciosa), workers de background (outbound, message-send, campañas, analytics, pipeline durable) con rol `app_session` o contexto org explícito, nunca contexto de miembro inventado. Delta spec `lean-data-isolation` reescrito: requisito RLS con semántica transaccional + escenario de reuso de conexión + requisito de workers sin inanición; nota de que es la primera implementación real de RLS. Risks: añadidos riesgos de inanición de workers y fuga por pool. tasks.md 3.3 (middleware SET LOCAL + observabilidad), 3.5 (workers), 6.1 (tests de pool reuse y worker bajo RLS).

## 3. Precedencia de autorización (S2)

- **Cambios**: design.md Decisión 3 reescrita — la política Stytch cacheada (Redis TTL 5 min) es la fuente runtime; 503 si API inalcanzable con cache vacía; structs Go tipados = tipos de compilación + fallback dev/mock. Delta spec `stytch-authorization`: requisito nuevo "Precedencia de autorización" con escenarios (resolución desde política cacheada, espejado en `rbac.go`/`roles.go`, versionado de cache key patrón `export`). tasks.md 1.2/1.3 (espejo en fallback + versionado cache key). Verificado: `internal/modules/auth/rbac.go`/`roles.go` existen como fallback dev/mock.

## 4. Índice y migración segura (D1/D2)

- **Cambios**: design.md Decisión 12 nueva — `CREATE INDEX idx_companies_owner ON crm.companies(owner_account_id)`; migración por pasos (ADD COLUMN catalog-only → audit pre-migración → backfill por lotes → `CREATE INDEX CONCURRENTLY` fuera de la transacción del runner → política RLS en migración aparte). Migration Plan reescrito. tasks.md 2.1–2.6 (pasos, audit pre-migración cuantificando cola incl. `stytch_member_id` NULL, backfill idempotente por lotes, índice concurrente, migración RLS opt-in). Verificado: golang-migrate v4.17.1 (envuelve archivos en transacción); `idx_contacts_company`/`idx_contacts_assigned`/`idx_accounts_stytch_member_id` existen; falta `idx_companies_owner`.

## 5. Directorio de miembros para `inbox:reassign` (S3)

- **Hecho**: contrato validado en docs oficiales de Stytch — `POST /v1/b2b/organizations/members/search` (query vacía + `organization_ids`, paginación `next_cursor`, `statuses: [active]`; SDK Go `stytch/b2b/organizations/members`; no existe GET de listado directo).
- **Cambios**: design.md Decisión 10 reescrita (contrato validado, circuit-breaker de dos niveles + cache Redis, validación mismo-org, degradación); delta spec `stytch-authorization` requisito "Directorio de miembros" (escenarios picker, degradación 503, destino mismo-org); delta spec `whatsapp-inbox` requisito "Re-asignación (endpoint)" (403/404/503 + audit); delta spec `inbox-ui` escenario de picker no-disponible con retry; tasks.md 1.4, 4.2, 5.4; Open Questions actualizada (contrato resuelto; versión del SDK a verificar en apply).

## 6. Market Risk completo (condiciones de mercado 1–3)

- **Cambios**: design.md `## Market Risk` — añadidos R4 (sustitución competitiva: Meta native WhatsApp AI, CRM locales WhatsApp), R5 (deriva de política de Meta como supuesto monitoreado con owner/trigger), R6 (auto-match cross-tenant como riesgo de seguridad con mitigación); R2 actualizado con audit pre-migración y auto-match org-scoped. Decisión 11 nueva (auto-match org-scoped vía `phone_number_id` UNIQUE de `whatsapp.whatsapp_configs`, nunca cross-tenant). Delta spec `whatsapp-webhook-ingress` reescrito con escenario de rechazo cross-tenant. proposal.md What Changes (auto-match org-scoped).

## Residuales (no bloqueantes, registrados)

- Empty state "Mis chats vacío → Cola" como superficie de activación (P1): registrado en design.md R3 y contrato UI mínimo; se entrega con `inbox-scope-views`.
- Semántica de conteo del unread de "Todos" y asignación de `view_unassigned` a roles: permanecen en Open Questions (decisión de producto en tasks).
- Verificación de la versión exacta del SDK Go de Stytch y del modo no-transaccional del runner de migraciones: pasos de apply con criterios de verificación en tasks 1.4/2.4.
