# Design: conversation-row-scoping — visibilidad por menor privilegio (planes pagos)

> Rev 2 — respuesta al VERDICT.md del council (REJECTED, 6 required design changes). Cambios: mecanismo real de feature-gating (V1), semántica SET LOCAL + workers de RLS (S1/S1b), precedencia de autorización reconciliada (S2), índice y migración por pasos (D1/D2), directorio de miembros para `inbox:reassign` (S3), Market Risk completado (condiciones de mercado 1–3). Ver `revision.md` para el mapeo de cobertura.

## Context

- La fila `crm.conversations` hoy se acota solo por `organization_id` (`lean-data-isolation`, query layer). Cualquier miembro con permiso de bandeja ve todas las conversaciones del tenant.
- El grafo de ownership B2B existe y está verificado en migraciones:
  - `crm.companies.owner_account_id → organizations.accounts(id)` (000012, comentado "Responsable de la empresa")
  - `crm.contacts.company_id → crm.companies(id)` (000011, índice `idx_contacts_company`); `crm.contacts.assigned_to → organizations.accounts(id)` (000011, índice `idx_contacts_assigned`)
  - `crm.tickets.assignee_stytch_member_id` (TEXT, 000017) + índice `(organization_id, assignee_stytch_member_id)` — patrón de asignación por `stytch_member_id` sin tabla local de miembros (Stytch = runtime SSOT)
  - `organizations.accounts.stytch_member_id` (VARCHAR, 000002, índice `idx_accounts_stytch_member_id`) — puente de join accounts(id) → stytch_member_id
- Query surfaces que tocan `crm.conversations` (verificado): `query/crm.sql` (77/81 lista, 90/102 INSERT, 115/123 UPDATE status, 133 stats) y `query/agent.sql` (156, 193 — el AI rail lee conversaciones con SQL directo). Lectores adicionales vía repositorio: `conversation_service.go`, `conversation_repository.go`, `outbound_service.go`, `message_send_handler.go`, `agent_repository.go`, `webhook_service.go` (workers de envío/mensajería/campañas — ver Decisión 9).
- **Feature-gating real (verificado, corrige premisa del draft v1)**: NO existe `FeatureService.IsEnabled` ni `internal/platform/features/plans.go`. El mecanismo es `features.FeatureProvider.GetEntitlement(ctx, orgID)` (`internal/platform/features/provider.go`) implementado por `billingFeatureProvider` (`internal/modules/billing/infra/features/billing_provider.go`): `Features` derivado de metadata de suscripción (`crm_features`, `ai_features`, `module_*`), módulos del catálogo (`modules.modules.granted_features`), con `PlanName` de Polar. Free tier (sin suscripción) → `Features` vacío (flag false por defecto). `isGracePeriod` (past_due) mantiene features; `isReadOnly` (inactiva) no. Precedente de grant base: `defaultGrantedModules = ["analytics"]` (siempre activo para suscripciones activas). **Drift spec↔código preexistente**: el spec vivo de `feature-gating` describe `FeatureService`/`plans.go` que no existen; este change reconcilia el contrato (Decisión 8).
- **RBAC (verificado)**: la política Stytch es el runtime SSOT, cacheada en Redis con TTL 5 min (`stytch_rbac_service.go`/`rbac_policy.go`); 503 si API inalcanzable con cache vacía (spec vivo `stytch-authorization`). Los maps de fallback dev/mock viven en `internal/modules/auth/rbac.go`/`roles.go` (roles Member/Manager/Admin; "fallback when the auth provider doesn't provide explicit permissions"). Precedente de versionado de cache key: acción `export` (spec vivo).
- **RLS (verificado)**: el spec vivo de `lean-data-isolation` define el patrón RLS opt-in (`app.current_organization_id`, rol `app_session` con bypass), pero **ninguna migración implementa RLS hoy**; este change introduce la primera política real y la extiende a nivel miembro. Migraciones vía golang-migrate v4 (cada archivo se envuelve en transacción por defecto → `CREATE INDEX CONCURRENTLY` requiere modo no-transaccional o paso post-deploy).
- **Webhook (verificado)**: `whatsapp.whatsapp_configs.phone_number_id` es UNIQUE y `organization_id` UNIQUE (000010) → el `phone_number_id` del payload resuelve a exactamente un org (base del auto-match org-scoped). Firma HMAC-SHA256 + idempotencia `INSERT ... ON CONFLICT` ya especificadas en el spec vivo.
- `inbox-member-tier` (change en curso) introduce `inbox:view`/`inbox:reply` — capacidad (qué puedes hacer). Este change añade el eje de scope (qué filas puedes ver). Componen.
- Constraints de gobernanza (AGENTS.md): Stytch es runtime SSOT (identidad + RBAC); el mapeo rol→scope local debe ser un contrato tipado con la política cacheada como fuente runtime; todo cambio de política Stytch requiere rollback dual; llamadas outbound a Stytch con circuit-breaker de dos niveles (umbral 5, timeout 10s, probe half-open 2); webhooks verificados por firma; RLS opt-in (defense-in-depth).

## Goals / Non-Goals

**Goals:**
- Scope por fila en `crm.conversations`: assignee ∪ owner de empresa ∪ `view_all` ∪ cola no-asignada (regla de unión, decisión de producto).
- Enforcement en query layer (SQLC) + RLS opt-in (defense-in-depth), ambos derivados del MISMO resolver; RLS con semántica transaccional segura (`SET LOCAL`).
- Permisos Stytch ortogonales: `inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign` — composables con acciones (`inbox:reply`), sin conceder destructivas (`inbox:delete`/`org:manage`).
- Cola de leads visible para `inbox:view_unassigned` + auto-match determinístico org-scoped en ingestión.
- Migración backfill determinística sin grace-period temporal, por pasos, con audit pre-migración.
- AI rail acotado por el scope del miembro.
- Solo planes pagos (feature flag sobre el mecanismo de entitlement real).
- Directorio de miembros para reasignación con circuit-breaker y degradación visible.

**Non-Goals:**
- NO tabla de equipos (`crm.teams`/`team_members`) — iteración futura.
- NO round-robin para leads net-nuevos.
- NO `inbox:delete`, NO ampliar `org:manage`.
- NO cambios en guardrails de envío, streaming, consentimiento.
- NO credenciales/sesiones locales.
- NO almacenar la lista de miembros de Stytch en tablas locales (solo `stytch_member_id` como FK lógico).

## Decisions

1. **Regla de unión para ownership divergente** — una conversación es visible si `assignee = me` **O** `company.owner_account_id` (vía join) = miembro **O** `view_all` **O** (`assignee IS NULL` y `view_unassigned`).
   - **Por qué**: en B2B el AE (`owner_account_id`) requiere visibilidad de supervisión aunque el manejo diario sea de SDR/CSM (`assigned_to`). La unión conserva menor privilegio (no es org-wide).
   - **Alternativas**: (a) solo `assigned_to` — falla el "vacation test"; (b) solo `owner_account_id` — el SDR no ve el chat que maneja. Rechazadas.

2. **Cola de no-asignados como fallback primario + auto-match determinístico org-scoped como routing secundario** — net-new inbound → `assignee = NULL`, visible solo para `inbox:view_unassigned`. Antes de caer a la cola, el webhook intenta match determinístico acotado al org resuelto del `phone_number_id` (ver Decisión 11).
   - **Por qué**: sin cola, los leads net-nuevos mueren invisibles. Round-robin es inapropiado para leads B2B sin calificar.
   - **Alternativa**: ronda round-robin — rechazada.

3. **Composición ortogonal scope × acción, con precedencia de autorización reconciliada** — el middleware resuelve permisos desde la **política Stytch cacheada (fuente runtime)** y materializa session vars (`app.current_member_id`, `app.is_view_all`, `app.is_view_unassigned`); Postgres aplica el predicado de datos (joins de ownership) vía RLS y/o query layer.
   - **Precedencia (corrige v1)**: la política cacheada (Redis TTL 5 min) es la única fuente runtime de rol→permiso; 503 si inalcanzable con cache vacía (spec vivo `stytch-authorization`). Los structs Go tipados del contrato rol→scope son **tipos de compilación + fallback dev/mock** (precedente `rbac.go`/`roles.go`); los permisos nuevos se espejan allí y se versiona la cache key (patrón `export`).
   - **Por qué**: Postgres no puede llamar a Stytch. `view_all` es un vector de scope distinto, composable: "Sales Director" = `view_all` + `reply`/`reassign`, sin destructivas.
   - **Alternativas**: (a) scope acoplado al rol — frágil; (b) array `app.current_member_scopes` precomputado — query extra por request y contrato de frescura; se omite: los joins calculan en vivo.

4. **Columna `assignee_stytch_member_id` TEXT (patrón tickets)** — no FK a `organizations.accounts(id)`.
   - **Por qué**: Stytch es SSOT de identidad; `accounts` puede quedar desincronizada con los miembros reales de Stytch. El patrón tickets ya establece el precedente.
   - **Alternativa**: FK a `accounts(id)` — inconsistente con tickets y frágil si falta la fila local.

5. **Una sola fuente para query layer y RLS: el scope resolver** — el predicado se define una vez (Go domain `conversation-scope`) y se traduce a (a) WHERE de SQLC y (b) política RLS. Si divergen, la UI mostrará listas acotadas con unread phantom.
   - **Por qué**: la fuga más probable es el unread poll y la paginación; un solo origen elimina la deriva.

6. **Ingestión webhook fuera de scope** — INSERT siempre permitido (rol `app_session` con bypass RLS o política RLS que permita INSERT del servicio de webhook); UPDATE/DELETE por webhook solo metadata de sistema (p. ej. `whatsapp_message_id`, estado de entrega), nunca re-asignación ni borrado de scope.
   - **Por qué**: si RLS bloquea la creación, la plataforma WhatsApp muere (todo inbound). Bypass documentado en `lean-data-isolation` (scenario app_session).

7. **AI rail consulta vía repositorio scoped** — `agent.sql` deja de leer `crm.conversations`/`crm.messages` directo; el contexto pasa por el repositorio con predicado de scope.
   - **Por qué**: el contexto LLM debe quedar acotado por la visibilidad del miembro.

8. **Flag `conversation_row_scoping` sobre el mecanismo de entitlement real** — se define en el mapeo plan→features **reconciliado con el entitlement provider existente** (`billingFeatureProvider.GetEntitlement`): el flag es `true` solo para suscripciones **activas/trialing/past_due en plan pago**; free/inactiva → `false`. Canal de grant (decisión de implementación en apply, una de): (a) metadata de suscripción (`crm_features` con `conversation_row_scoping`, push en el catálogo de planes Polar), (b) módulo vendible con `granted_features` (patrón `add-sellable-modules`), o (c) grant base para suscripciones activas (precedente `defaultGrantedModules`). El cambio reconcilia el drift spec↔código (spec vivo describe `plans.go`; el código es metadata-driven): si se adopta el modelo `plans.go` del spec vivo, ese mapa pasa a ser la única fuente y se documenta la reconciliación en el change; si se mantiene el mecanismo actual, se actualiza el spec vivo en un cambio de gobernanza aparte. **Los nombres de plan "Starter/Pro/Enterprise" son un supuesto a validar contra el catálogo de Polar** (no aparecen en el código; `PlanName` llega de la suscripción).
   - **Por qué**: restricción de producto (solo pagos) sin bifurcar esquema: la columna existe siempre, el flag decide si el predicado de scope se aplica.
   - **Riesgo conocido**: RLS no se puede gatear por flag por-request sin session var; se resuelve seteando `app.is_view_all = true` (o `app.scope_enabled`) cuando el plan no incluye la feature — la política RLS se escribe una vez y respeta el flag vía session var. El flag debe resolverse **una vez por request en el middleware de entitlement** (no repetir lecturas de suscripción por query).

9. **RLS con semántica transaccional segura + política explícita de workers** —
   - Las session vars de scope (`app.current_member_id`, `app.is_view_all`, `app.is_view_unassigned`) se setean **exclusivamente con `SET LOCAL` dentro de la transacción del request** (nunca `SET` a nivel sesión sobre el pool de conexiones de database/sql): una conexión reutilizada no puede heredar vars de scope de otro miembro, y cada request parte de un estado limpio. Las queries de lectura (lista, thread, poll 5s, unread, stats) corren dentro de esa transacción de lectura.
   - **Fail-closed**: con RLS activa y sin vars, PostgreSQL devuelve cero filas (spec vivo). Para que la inanición silenciosa de paths interactivos sea detectable, se añade observabilidad: métrica/log de anomalías "política activa con vars ausentes o mal resueltas" (p. ej. request interactivo que devuelve cero filas con RLS activa y vars no seteadas → error 500 explícito, no lista vacía silenciosa).
   - **Workers de background** (outbound send, message-send, campañas, analytics, durable pipeline, cron) que leen conversaciones **sin contexto de miembro**: corren con rol `app_session` (bypass RLS) y su control de aislamiento sigue siendo el query layer org-scoped existente (mismo comportamiento que hoy). Alternativa permitida: contexto org explícito por worker (setear `app.current_organization_id`). Nunca contexto de miembro inventado.
   - **Por qué**: sin esto, un worker bajo RLS devuelve cero filas silenciosamente (envíos muertos) o una conexión del pool hereda scope de otro miembro (fuga). Ambas son fallas de clase de seguridad/disponibilidad que el draft v1 no especificaba.

10. **Directorio de miembros para `inbox:reassign`** — la lista de asignables se obtiene de la **Stytch B2B Members API** (contrato validado en docs oficiales de Stytch: `POST /v1/b2b/organizations/members/search` con `organization_ids` = org del solicitante, query vacía devuelve miembros no-eliminados, paginación por `next_cursor`, filtro `statuses: [active]`; SDK Go `stytch/b2b/organizations/members`), envuelta en el **circuit-breaker de dos niveles** existente para llamadas outbound a Stytch (umbral 5, timeout 10s, probe half-open 2) + **cache Redis** (TTL 5 min, patrón de la política RBAC). Solo se persiste/retorna `stytch_member_id` (FK lógico); nunca credenciales ni datos de sesión. Versión del SDK a verificar en apply.
    - Validación: el destino de una re-asignación DEBE pertenecer al mismo org (el org del miembro solicitante, no un org del payload).
    - Degradación: si el directorio no está disponible (circuit abierto o cache vacía), el picker se oculta con estado de retry (sin ghost) y el endpoint de re-asignación responde 503 con `member_directory_unavailable`; la bandeja sigue funcionando (lectura/respuesta intactos).
    - **Por qué**: sin tabla local de miembros, el picker no tiene de dónde sacar la lista; inventar una tabla local viola el SSOT de Stytch.

11. **Auto-match org-scoped en ingestión** — el match determinístico (teléfono/NIT → `crm.companies`) se acota **dentro del org resuelto del `phone_number_id`** del payload (`whatsapp.whatsapp_configs.phone_number_id` es UNIQUE → un solo org). Nunca se matchea cross-tenant: un teléfono puede existir en varias organizaciones de la plataforma.
    - **Por qué**: sin esto, un inbound podría asignarse al owner de la empresa equivocada (fuga de contexto entre tenants).
    - Normalización de NIT: se mantiene el spike (ver Open Questions) — el inbound trae solo teléfono.

12. **Índices y migración por pasos** —
    - Índice nuevo: `CREATE INDEX idx_companies_owner ON crm.companies(owner_account_id)` (el predicado de unión filtra companies por owner en cada lista/poll/RLS; hoy no existe).
    - Migración por pasos (expand sin bloqueo largo):
      1. `ALTER TABLE crm.conversations ADD COLUMN assignee_stytch_member_id TEXT` (catalog-only, sin default volátil) + `COMMENT` + índice `(organization_id, assignee_stytch_member_id)`.
      2. **Audit pre-migración** (precedente `000016_pre_migration_audit.sql`): consulta que cuantifica filas que caerán a cola — contactos sin `assigned_to` ni `owner_account_id`, y casos `assigned_to`/`owner_account_id` con `accounts.stytch_member_id` NULL — para la comunicación de staffing día uno.
      3. **Backfill por lotes** (UPDATE en batches por rango de id, no un UPDATE full-table único): prioridad `contacts.assigned_to` → fallback `companies.owner_account_id` (vía `accounts.stytch_member_id`) → NULL (cola). Un UPDATE único es transacción larga con WAL alto y contención con el poll de 5s.
      4. **Índice de ownership** (`idx_companies_owner`): `CREATE INDEX CONCURRENTLY` — golang-migrate v4 envuelve cada archivo en transacción, así que se ejecuta en modo no-transaccional del runner o como paso post-deploy del job (documentado), nunca dentro de la transacción envuelta.
      5. Política RLS opt-in en migración aparte (flag de deploy), con grants para el rol `app_session` y semántica `SET LOCAL`.
    - **Por qué**: un solo archivo de migración con ADD COLUMN + backfill full-table + índice no-concurrente bloquea escrituras y arriesga el servicio (poll 5s, inbound).

## Risks / Trade-offs

- [Fuga por unread/paginación sin scope] → Mitigación: auditoría de TODAS las queries SELECT de `crm.conversations` (crm.sql, agent.sql); test de paridad lista-vs-unread; el resolver es la única fuente.
- [RLS bloquea ingestión] → Mitigación: bypass `app_session` para INSERT de webhook; test e2e de inbound bajo RLS activa.
- [RLS inanición de workers (envíos/campañas mueren en silencio)] → Mitigación: workers con rol `app_session`/contexto org explícito (Decisión 9); test de worker bajo RLS activa; métrica de anomalías cero-filas.
- [Fuga de scope por vars de sesión en pool] → Mitigación: `SET LOCAL` transaccional obligatorio (Decisión 9); test de reuso de conexión (dos miembros en el mismo pool).
- [Cliff invisible de conversaciones existentes tras migración (cliente de cola)] → Mitigación: backfill determinístico por lotes + audit pre-migración que cuantifica la cola + comunicación a admins día uno; sin grace-period temporal.
- [Deriva de política Stytch (permisos nuevos)] → Mitigación: política cacheada como fuente runtime; permisos espejados en `rbac.go`/`roles.go` (fallback dev/mock); versionado de cache key (patrón `export`); rollback dual (Git + política Stytch) documentado.
- [Divergencia assignee vs owner genera superficie más amplia] → Mitigación: la unión es intencional y documentada como contrato; auditoría de reasignaciones en audit ledger.
- [Directorio de miembros no disponible (circuit abierto)] → Mitigación: picker oculto con retry (sin ghost), endpoint 503 `member_directory_unavailable`, bandeja funcional; cache Redis 5 min + circuit-breaker de dos niveles.
- [AI rail filtra contexto de conversaciones no visibles] → Mitigación: el repositorio scoped es obligatorio para agent.sql; test de contexto con miembro de scope reducido.
- [Feature flag bifurca comportamiento] → Mitigación: el flag vive en una sola decisión (aplicar predicado o no) resuelta una vez por request en el middleware de entitlement; tests en ambos modos; semántica past_due/grace consistente con el entitlement existente.

## Migration Plan

1. **Política Stytch**: añadir `inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign` y asignación a roles (Stytch dashboard/API, documentado). Espejar en fallback `rbac.go`/`roles.go`. Versionar la cache key de la política (`auth:stytch:rbac:policy:v2`). Rollback: remover de la política + revertir fallback.
2. **Migración SQL por pasos**: (a) `ADD COLUMN assignee_stytch_member_id TEXT` + índice `(organization_id, assignee_stytch_member_id)`; (b) audit pre-migración (cola cuantificada, incl. `stytch_member_id` NULL); (c) backfill por lotes (assigned_to → owner → NULL); (d) `CREATE INDEX CONCURRENTLY idx_companies_owner` (runner no-transaccional o post-deploy); (e) política RLS opt-in en migración aparte (flag de deploy, grants `app_session`, semántica `SET LOCAL`).
3. **Backend**: scope resolver + middleware de entitlement/session vars (`SET LOCAL` transaccional) + predicado en queries SQLC (`make sqlc`) + endpoint de re-asignación (`inbox:reassign`, directorio vía Stytch Members con circuit-breaker + cache) + auto-match org-scoped en ingestión.
4. **Frontend**: tabs de scope, picker de asignación (degradación sin directorio), gates por permiso, copy.
5. **Despliegue**: activar flag en planes pagos (mecanismo de grant elegido en apply); comunicar a admins (cola de no-asignados cuantificada, staffing). Rollback: git revert + política Stytch revertida + migración down (drop columna/índice/política) + revertir grant del flag.

## Market & Unit Economics

Este change **toca el canal WhatsApp/Meta y el modelo de autorización**, y lo declara:

- **Canal WhatsApp (Meta Business Platform): sin costo nuevo por mensaje.** El scoping y el auto-match no alteran el path de envío ni el metering; la cola de no-asignados amplía a quién se muestra un inbound, no cuánto cuesta procesarlo.
- **Autorización: cambio de contrato Stytch.** Se añaden `inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign` (runtime SSOT) + un nuevo outbound (Stytch Members API) con circuit-breaker; sin costo directo, con superficie de deriva de política (R1) y rollback dual documentado.
- **Monetización / planes: el scoping es un diferenciador de plan pago.** `conversation_row_scoping` es un flag solo-pagos (mecanismo: entitlement provider, Decisión 8): el free tier conserva org-scope; el valor percibido (menor privilegio, cola de leads por rol) refuerza el upgrade a Starter/Pro/Enterprise. No se tocan `paywall` ni `plan-pricing-ux`; el flag vive en `feature-gating`. Nombres de plan = supuesto a validar contra el catálogo Polar.
- **Métrica de operación:** menor tiempo de respuesta en cola (leads asignables por rol) y superficie de lectura reducida (compliance) — medibles como follow-up de analytics.

## Market Risk

- **R1 — Deriva de política RBAC (seguridad + compliance).** Permisos de scope nuevos amplían/ajustan la superficie de lectura de datos personales (WhatsApp). **Owner:** security/architecture. **Trigger:** incidente de acceso, auditoría, cambio de roles de Stytch. **Mitigación:** política cacheada como fuente runtime, enforcement server-side (404 out-of-scope), unread/paginación auditados, auditoría de reasignaciones, rollback dual (Git + política Stytch), permisos espejados en fallback dev/mock.
- **R2 — Cola de leads sin staffing (soporte).** Tras la migración, conversaciones sin asignar quedan en cola; si nadie con `inbox:view_unassigned` la vigila, los leads net-nuevos no se responden (ventana 24h de WhatsApp). **Owner:** product/ops. **Trigger:** despliegue del backfill. **Mitigación:** audit pre-migración cuantifica la cola real; comunicación a admins día uno (staffing); auto-match org-scoped reduce el volumen de la cola; el flag solo-pagos acota el cambio a tenants pagos con equipos.
- **R3 — Fricción de visibilidad (usuario).** Miembros que antes veían todo ahora ven menos (menor privilegio); puede frustrar si el rol no incluye `view_all`. **Owner:** product. **Trigger:** feedback o tickets. **Mitigación:** tabs explícitos (Mis chats / Cola / Todos), contrato rol→scope documentado, upgrade path claro vía asignaciones, empty state "Mis chats vacío → Cola" como superficie de activación (contrato UI mínimo).
- **R4 — Sustitución competitiva (aceptado).** Meta native WhatsApp AI y los CRM locales de WhatsApp (incumbentes B2B) pueden presentar scoping/equipos como feature nativa o gratuita; el scoping de este change es diferenciador de compliance/minimización de datos (Ley 1581) y de operación de equipos, pero no es durable por sí solo. **Owner:** product/GTM. **Trigger:** release de Meta con gestión de equipos nativa o presión competitiva en renovaciones. **Mitigación:** el scoping se entrega junto con cola/auto-match (valor operativo completo); se monitorea el posicionamiento competitivo en el roadmap de plan-pricing-ux.
- **R5 — Deriva de política de Meta (supuesto monitoreado).** Pricing de conversaciones, aprobación de templates, términos de IA-agent y native AI messaging son una superficie de riesgo existencial del canal. Este change no altera el path de envío ni el metering; se registra como **supuesto a vigilar**, no premisa. **Owner:** ops/product. **Trigger:** comunicado de política de Meta sobre pricing de conversaciones o términos de IA. **Mitigación:** monitoreo de comunicados; los cambios se gestionan en cambios de canal aparte (`whatsapp-outbound-send`, `whatsapp-compliance`).
- **R6 — Auto-match cross-tenant (seguridad).** Si el match de teléfono/NIT no se acota al org del `phone_number_id`, un inbound puede asignarse al owner equivocado. **Owner:** security. **Trigger:** incidente de enrutamiento. **Mitigación:** Decisión 11 (match org-scoped obligatorio) + test de cross-tenant en la suite de ingestión.

## Open Questions

- ¿El auto-match por NIT requiere normalización previa (el inbound trae solo teléfono)? → Spike: resolución empresa por teléfono vs NIT en metadata. (Se mantiene; el match SIEMPRE org-scoped vía Decisión 11.)
- ¿`view_unassigned` se concede a roles de ventas (`member` con flag de leads) o solo a roles explícitos? → Decisión de producto en tasks.
- ¿El unread badge de "Todos" (view_all) debe incluir conversaciones de la cola? → Semántica de conteo por tab.
- Contrato de la Stytch B2B Members API: validado en docs oficiales (`POST /v1/b2b/organizations/members/search`); queda por verificar en apply la versión exacta del SDK Go pinneado (`stytch/b2b/organizations/members`).
- Nombres de plan "Starter/Pro/Enterprise" y canal de grant del flag (metadata vs módulo vs grant base) → supuesto a validar contra el catálogo Polar (ver Decisión 8).
