# VERDICT — conversation-row-scoping (re-revisión, rev 2)

STATUS: APPROVED
MARKET: PASS

## Resumen ejecutivo

Re-revisión tras el REJECTED previo (rev 1). La revisión (revision.md, `REVISION: COMPLETE`) aborda las 6 required design changes del verdict anterior con evidencia verificada en repositorio:

1. **Mecanismo de feature-gating** — corregido: `FeatureService.IsEnabled`/`plans.go` NO existen; el flag se define sobre `FeatureProvider.GetEntitlement` → `Entitlement.Features` (metadata de suscripción/módulo/grant base, precedente `defaultGrantedModules` verificado en `billing_provider.go`), con semántica activa/trialing/past_due vs free/inactiva consistente con el código existente (`isGracePeriod` mantiene features; `isReadOnly` no). Nombres de plan declarados como supuesto a validar contra el catálogo Polar. ✓
2. **RLS session vars** — `SET LOCAL` transaccional obligatorio, fail-closed + observabilidad, workers de background (`outbound`, `message-send`, campañas, analytics, durable pipeline) con rol `app_session`/contexto org explícito. ✓
3. **Precedencia de autorización** — política Stytch cacheada (Redis TTL 5 min) como fuente runtime; structs Go como tipos de compilación + fallback dev/mock (espejado en `internal/modules/auth/rbac.go`/`roles.go`, verificado en disco) + versionado de cache key (patrón `export`). ✓
4. **Índice y migración** — `idx_companies_owner` añadido; migración por pasos (ADD COLUMN catalog-only → audit pre-migración con precedente real `000016_pre_migration_audit.sql` → backfill por lotes → `CREATE INDEX CONCURRENTLY` fuera de la transacción de golang-migrate v4.17.1 → RLS aparte). ✓
5. **Directorio de miembros** — contrato Stytch validado en docs oficiales (`POST /v1/b2b/organizations/members/search`, paginación, `statuses: [active]`), circuit-breaker de dos niveles + cache Redis, validación mismo-org, degradación 503 con bandeja funcional. ✓
6. **Market Risk** — R4 (sustitución competitiva), R5 (deriva de política Meta como supuesto monitoreado con owner/trigger), R6 (auto-match cross-tenant) añadidos; Decisión 11 (match org-scoped vía `phone_number_id` UNIQUE) con escenario de rechazo cross-tenant en el delta spec. ✓

`openspec validate` pasa (23/23). No quedan hallazgos bloqueantes; los residuos son de severidad baja y están registrados abajo.

## Market Read

Las condiciones de mercado del verdict anterior quedaron resueltas: sustitución competitiva cubierta (R4, con owner product/GTM y trigger), auto-match org-scoped confirmado (Decisión 11 + escenario cross-tenant en `whatsapp-webhook-ingress`), y deriva de política de Meta registrada como supuesto monitoreado (R5, owner ops/product). El impacto de mercado del change es bajo y en su mayoría positivo: sin costo nuevo por mensaje ni alteración del metering de IA; el scoping es un diferenciador de compliance (Ley 1581: minimización de datos) y de operación de equipos para planes pagos. El riesgo de mercado dominante sigue siendo la cola sin staffing contra la ventana 24h de WhatsApp (R2) — ahora mitigado con audit pre-migración cuantificado y auto-match org-scoped. No se exige condición de mercado pendiente.

## Hallazgos por persona

### 1. Staff Security Engineer

- **[S1 — MEDIA] Mecanismo de detección del fail-closed RLS subespecificado.** La Decisión 9 exige que un path interactivo con RLS activa y vars ausentes devuelva 500 explícito (no lista vacía silenciosa), pero no especifica la señal de detección: distinguir "vars ausentes (bug)" de "cero filas legítimas (fuera de scope)" requiere verificar en el middleware que las vars se setearon (p. ej. `current_setting('app.current_member_id')` no nula al inicio de la transacción) o una política RLS de auditoría. → **Residual aceptado**: el criterio de verificación debe incluir la aserción de vars en el middleware (tasks 3.3/6.1); no bloquea.
- **[S2 — BAJA] Staleness del directorio cacheado (TTL 5 min).** Un miembro desactivado en Stytch puede aparecer como destino válido hasta el próximo refresco. Aceptable: la validación server-side se hace contra el directorio y el destino pertenece al org; residual de ventana corta. → Residual.
- **[S3 — BAJA] "Audit ledger" sin destino concreto.** El delta spec y tasks 4.2 exigen registro actor/origen/destino; existe precedente de ledger append-only org-scoped (`procurement.audit_log`, 000037) y de actividades CRM. El destino exacto (tabla nueva vs `crm.actividades`) es detalle de implementación en tasks. → Residual.
- Webhook: firma HMAC + idempotencia `INSERT ... ON CONFLICT` ya especificadas; bypass `app_session` para INSERT y UPDATE/DELETE solo metadata de sistema — consistente. Sin hallazgos nuevos.

### 2. Staff DBA

- **[D1 — BAJA] Costo del predicado de unión dentro de la política RLS.** La rama de ownership (conversations → contacts → companies → accounts) es un EXISTS correlacionado por fila; con RLS como capa de respaldo y el query layer como primario, el costo se acota con `idx_companies_owner` (nuevo) + `idx_contacts_company`/`idx_contacts_assigned`/`idx_accounts_stytch_member_id` (existentes). → Residual: verificar plan de ejecución (EXPLAIN) en apply; añadir a tasks 6.x si aplica.
- **[D2 — BAJA] Backfill y filas con `accounts.stytch_member_id` NULL.** El backfill por lotes con `COALESCE(assigned_to, owner_account_id)` deja NULL (cola) cuando el account referenciado no tiene `stytch_member_id`; el audit pre-migración lo cuantifica — correctamente tratado como riesgo de cola día uno (R2). → Residual aceptado.
- Migración por pasos, ADD COLUMN catalog-only, backfill idempotente por lotes, `CREATE INDEX CONCURRENTLY` fuera de la transacción del runner, política RLS en migración aparte: correcto y expand-safe. Sin hallazgos bloqueantes.

### 3. SRE

- **[SRE1 — BAJA] Race de claim en la cola.** Dos miembros con `view_unassigned`+`reassign` pueden reclamar la misma conversación de cola simultáneamente; último-write-wins sobre `assignee_stytch_member_id`. Idempotente por naturaleza (PATCH), sin corrupción; la UX lo resuelve con refresh del poll 5s. → Residual para `inbox-assignment-actions` (si se quiere, control de concurrencia con `updated_at`/versión).
- Circuit-breaker de dos niveles + cache Redis para el directorio, degradación 503 `member_directory_unavailable`, bandeja funcional: bien especificado. Rollback dual (Git + política Stytch) + migración down + revert de grant del flag: documentado y completo.
- Observabilidad: métrica de anomalías cero-filas (S1) es el único punto con mecanismo de detección pendiente de precisar en apply. Sin hallazgos bloqueantes.

### 4. Staff Product/GTM

- **[P1 — BAJA] Activación del miembro sin asignaciones.** El empty state "Mis chats vacío → Cola" queda registrado en R3 como superficie de activación y se entrega con `inbox-scope-views`; correcto que el contrato mínimo lo exponga.
- Economía unitaria sin cambio (sin costo por mensaje; flag solo-pagos refuerza upgrade); coherencia de pricing intacta (no toca `paywall`/`plan-pricing-ux`; semántica past_due/grace consistente con el entitlement). Diferenciador competitivo claro y no durable por sí solo — correctamente aceptado como R4 con owner y trigger. Sin hallazgos bloqueantes.

### 5. Colombia IT & Market

- **[C1] Compliance positivo**: menor superficie de lectura de datos personales apoya Ley 1581/Habeas Data; sin cambios en consentimiento/export/forget (non-goal declarado). Deriva de política de Meta registrada como supuesto monitoreado (R5). Sin hallazgos nuevos.

## Required design changes

Ninguna — veredicto aprobatorio. Los residuos no bloqueantes quedan registrados para los criterios de verificación de apply (tasks 3.3/6.1: aserción de vars en middleware; tasks 6.x: EXPLAIN del predicado de unión bajo RLS) y para los changes de seguimiento (`inbox-scope-views`, `inbox-assignment-actions`).

## Notas

- El contract de roles para `view_unassigned` (¿roles de ventas o explícitos?) y la semántica de unread del tab "Todos" permanecen como Open Questions de producto — no bloquean este change.
- Verificación en apply pendiente (con criterios en tasks): versión exacta del SDK Go de Stytch (`stytch/b2b/organizations/members`) y modo no-transaccional del runner de migraciones para `CREATE INDEX CONCURRENTLY`.
