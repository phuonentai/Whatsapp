# Proposal: inbox-assignment-actions — las tres jugadas de ownership (reclamar, transferir, liberar)

## Why

`conversation-row-scoping` entrega el endpoint de reasignación (PATCH assignee, permiso `inbox:reassign`, auditoría) e `inbox-scope-views` la identidad visual (chips, cola, nueva-vs-no-leído). Falta la **cara de escritura**: las interacciones que mueven ownership — reclamar de la cola, transferir a un colega, liberar a la cola — y la coreografía alrededor (auto-claim al responder, banner de colisión, llegada al receptor, undo). Restricción dura: **la UI actual permanece intacta** — estas acciones se añaden como capa (chip de cabecera clicable, menú contextual de fila), sin reestructurar el master-detail ni los tabs existentes.

## What Changes

- **Picker de tres jugadas en un solo componente** (disparador: chip de assignee en la cabecera del thread): (1) **Reclamar** (cola → mío, 1 click), (2) **Transferir** (a un miembro, picker con búsqueda), (3) **Liberar a la cola** (mío → cola, con confirmación Y undo de 5s — feedback inmediato con undo, patrón expert). El mismo componente cubre las tres jugadas.
- **Auto-claim al primer reply**: responder a una conversación sin asignar la reclama implícitamente (modelo Intercom/Front), además del botón explícito. Ambos modelos conviven (decisión tomada).
- **Banner de colisión/ownership** (anti-colisión, patrón expert "already claimed by X"): en el poll de 5s, si la conversación abierta cambió de assignee (reclamada/transferida por otro), la cabecera muestra "Ahora asignada a Ana" — el redactor deja de invertir atención en un chat que no es suyo; si su reply ya viajó, el mensaje se envía igual (sin pérdida) y la UI aclara la propiedad.
- **Llegada al receptor**: toast "Conversación asignada a ti" + punto "nueva" en Mis chats (distinto de no-leído, definido en `inbox-scope-views`). Sin infra de push/email en v1 (solo in-app).
- **Menú contextual de fila para supervisores** (⋮ en la fila → "Asignar a…"): reassign masivo-por-exploración desde "Todos" sin abrir cada thread. Bulk (checkbox-mode) queda como v2.
- **Conflictos atómicos**: claim/transfer concurrentes resueltos server-side (409 con el ganador); la UI muestra "Tomado por X" con su avatar (el race se vuelve visible, no frustrante).
- **Memoria de handoff SIN tabla nueva**: el remitente puede guardar el contexto IA como `nota` de CRM (capability `conversation-context-note` existente) antes de transferir; el receptor la ve en el timeline del contacto. NO se construye campo de nota de asignación en v1.

## Capabilities

### New Capabilities

- (ninguna — deltas sobre capacidades existentes)

### Modified Capabilities

- `inbox-ui`: picker de tres jugadas (claim/transfer/release con confirm + undo), auto-claim al primer reply, banner de ownership en poll, toast de llegada al receptor, menú contextual de fila (reassign), manejo de conflicto 409 ("tomado por X"), gates por `inbox:reassign`.
- `whatsapp-inbox`: el endpoint de envío manual SHALL reclamar implícitamente si la conversación está sin asignar y el remitente tiene permiso para reclamarla (contrato de auto-claim a nivel de comportamiento de API).

## Impact

- **Frontend** (`next_b2b_starter/`): `components/conversation-header.tsx` (chip clicable → picker), nuevo componente `assignment-picker.tsx` (búsqueda de miembros, release con confirm+undo), `components/conversation-list.tsx` (menú contextual ⋮), `components/reply-input.tsx` (auto-claim + toast), `lib/hooks/mutations/` (use-assign, use-release, use-claim), `lib/copy/ui.ts`.
- **Backend**: mínimo — si el auto-claim se implementa como comportamiento del endpoint de envío (no solo del cliente), requiere un pequeño cambio en el handler de POST mensajes (idempotente, transaccional con el envío). El PATCH assignee ya existe en `conversation-row-scoping`.
- **Auth**: sin cambios de contrato Stytch nuevos (reutiliza `inbox:reassign` del change base).
- **Ops**: `pnpm build`/`lint`/`tsc`, vitest de reply-input/picker/header, Playwright → `qa/`, tests de carrera (claim concurrente 409).
- **Rollback**: git revert; sin migraciones; sin estado de Stytch.

## Non-Goals

- NO capacidades de workload (límites de asignación, fairness, anti-hoarding) — `inbox-workload-balancing` (gated/futuro).
- NO indicadores de typing en vivo (sin presencia/websockets; futuro).
- NO campo de nota de asignación (la memoria de handoff reutiliza `conversation-context-note`/CRM actividades).
- NO bulk reassign con checkbox-mode en v1 (menú contextual cubre el caso supervisor).
- NO notificaciones push/email de asignación (solo in-app).
- NO almacenar credenciales/sesiones localmente (todo auth sigue en Stytch B2B).
