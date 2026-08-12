# Proposal: inbox-scope-views — superficie de lectura del scoping (UI aditiva, sin romper la bandeja actual)

## Why

El change `conversation-row-scoping` entrega el contrato backend (scope por fila, permisos `inbox:view_all`/`inbox:view_unassigned`, assignee en el API). Este change construye la **cara de lectura** en la bandeja: el miembro debe *ver* su scope (mis chats, cola, todos) sin que la UI actual se reestructure. Restricción dura: **la bandeja actual (master-detail, tabs de estado All/Active/Closed/Archived, tabs de canal WhatsApp/Instagram, AI rail) permanece intacta** — el scope se añade como capa aditiva, nunca como cuarta fila de tabs.

## What Changes

- **Selector de scope aditivo** (píldoras compactas con contadores: "Mis chats (n) · Cola (n) · Todos (n)") por ENCIMA de los tabs existentes, visualmente distinto de los tabs (píldoras, no tabs) para que lea como dimensión nueva sin competir con estado/canal. Las filas de tabs existentes NO se tocan. En móvil (390px), el selector colapsa a un control único y los filtros de estado/canal pasan a un sheet "Filtrar" (sin eliminar rutas ni vistas actuales).
- **Identidad de asignación en la lista y cabecera**: chip de avatar (iniciales, color estable) por fila; conversación sin asignar = slot vacío con anillo ámbar (nunca confundible con "abandonada"); cabecera de thread muestra el assignee actual; el mismo render de avatar en lista/cabecera/picker (una gramática de ownership).
- **Distinción "nueva" vs "no leído"** (estado de trabajo, no read/unread — patrón expert de los shared inbox): punto "nueva" = conversación recién asignada a mí (llegó ownership), badge de no leído = llegaron mensajes. Se renderizan distinto y no se confunden.
- **Estados vacíos con tres verdades**: (1) sin chats en mi scope pero con cola → "No tienes chats asignados — reclama de la cola (n)" + CTA; (2) cola vacía → "La cola está vacía, todo asignado"; (3) sin permiso → el tab/control simplemente no existe (nunca "no hay datos" cuando es "no es tu data").
- **Urgencia en la cola (priorización antes de asignación)**: cuenta regresiva de la ventana WhatsApp 24h ("responder en 16h", ámbar→rojo) + ordenamiento por urgencia (SLA), no solo por antigüedad. El badge de cola es discreto (punto sutil) hasta que hay urgencia, entonces pulsa.
- **Métricas con audiencia por scope** (`inbox-metrics.tsx`): el strip org-wide solo para `view_all`/`org:manage`; el resto ve mini-stats personales ("tus chats: 4 · tu tiempo de respuesta: 2h"). Mismos datos, audiencias distintas.
- **Flag `conversation_row_scoping`**: free tier NO ve ninguno de estos controles (bandeja 100% actual); al activar el plan pago, si hay conversaciones sin asignar, la vista inicial es la Cola (para que el upgrade se sienta como ganancia, no pérdida).

## Capabilities

### New Capabilities

- (ninguna — deltas sobre capacidades existentes)

### Modified Capabilities

- `inbox-ui`: selector de scope aditivo con contadores; identidad de asignación (chips, slot ámbar); distinción "nueva" vs no-leído; estados vacíos por scope; urgencia de cola (countdown 24h + sort SLA); métricas por audiencia; comportamiento free-tier (flag) y primer-run de upgrade.
- `feature-gating`: la UI de scope se oculta con el flag `conversation_row_scoping` (free tier intacto); primer-run de upgrade con aterrizaje en Cola si hay no-asignadas.

## Impact

- **Frontend** (`next_b2b_starter/`): `app/dashboard/inbox/page.tsx` (selector de scope + integración aditiva), `components/conversation-list.tsx` (chips de assignee, punto "nueva", estados vacíos, orden por urgencia), `components/conversation-header.tsx` (assignee en cabecera), `components/inbox-metrics.tsx` (split de audiencia), `lib/models/conversation.model.ts` (assignee en el DTO de lista), `lib/copy/ui.ts`.
- **Backend**: NINGUNO nuevo — consume el contrato de `conversation-row-scoping` (lista con `scope` param, assignee en el DTO). Si el parámetro `scope` no existe aún en el API, este change espera o coordina con el change base (dependencia explícita).
- **Auth**: sin cambios de contrato Stytch (los permisos de scope viven en `conversation-row-scoping`).
- **Ops**: `pnpm build`/`lint`/`tsc`, vitest de conversation-list/header/metrics, Playwright visual/a11y → `qa/`.
- **Rollback**: git revert; sin migraciones; la UI vuelve a la bandeja actual.

## Non-Goals

- NO interacciones de escritura (claim/release/transfer, auto-claim, banner de colisión) — viven en `inbox-assignment-actions`.
- NO reestructurar tabs de estado/canal existentes ni el master-detail (la restricción "UI intacta" es un requisito, no una preferencia).
- NO capacidades de workload (límites de asignación, fairness) — `inbox-workload-balancing` (gated/futuro).
- NO indicadores de typing en vivo (sin infraestructura de presencia/websockets; queda como iteración futura).
- NO almacenar credenciales/sesiones localmente (todo auth sigue en Stytch B2B).
