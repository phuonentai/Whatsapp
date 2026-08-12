# Proposal: inbox-workload-balancing — visibilidad de capacidad y distribución justa (GATED / futuro)

## Why

La investigación de expertos (Front, Intercom, Zendesk 2025-2026) señala la **visibilidad de capacidad** como el patrón emergente de los shared inbox: los agentes deben ver cuántas conversaciones cargan y cuál es el límite del equipo, y el sistema debe dificultar el acaparamiento ("cherry-picking"). Este change es **deliberadamente gated y posterior**: requiere `conversation-row-scoping` + `inbox-scope-views` + `inbox-assignment-actions` como base (ownership y cola ya existen), y agrega la capa de operación de equipo. Se registra como proposal completo para capturar la decisión, pero NO se implementa antes de los cambios base.

## What Changes

- **Límites de asignación por rol/equipo** (config backend + UI): cada miembro tiene un tope de conversaciones activas asignadas (p. ej. agente=8, supervisor=15); el tope se muestra como progreso en el sidebar/lista ("6/8 conversaciones").
- **Distribución justa / anti-hoarding**: señalización de sobrecapacidad (el badge del picker muestra la carga del destinatario antes de transferir; "Ana está al límite (7/8)" con confirmación explícita para transferir igual). NO round-robin automático para leads B2B (se mantiene la decisión del change base: los leads sin calificar requieren contexto, no rueda).
- **Cola priorizada por urgencia + carga**: el sort de la cola combina ventana 24h (de `inbox-scope-views`) con capacidad del equipo; opcionalmente "Pull" guiado (sugerencia del sistema del siguiente chat, sin forzar).
- **Workload del equipo para supervisores**: vista de carga por miembro (quién está ahogado, quién ocioso) en "Todos" — el mapa de cobertura que los supervisors ya leen en los chips de assignee, ahora con números.
- **Flag `conversation_row_scoping`**: solo planes pagos, igual que el change base.

## Capabilities

### New Capabilities

- `inbox-workload-balancing`: límites de asignación configurables, indicadores de capacidad, distribución justa y vista de carga de equipo.

### Modified Capabilities

- `inbox-ui`: indicadores de capacidad (progreso), aviso de destinatario al límite en el picker, vista de workload del equipo.
- `whatsapp-inbox`: el sort de cola combina urgencia con capacidad; señales de sobrecapacidad.
- `feature-gating`: flag `conversation_row_scoping` cubre también los controles de workload (solo pagos).

## Impact

- **Backend** (`go-b2b-starter/`): tabla/config de límites por rol o miembro (FK lógico `stytch_member_id`), conteo de carga por miembro en queries de lista, endpoints de configuración (solo `org:manage` + permiso nuevo `inbox:manage_limits` en la política Stytch, con rollback dual).
- **Frontend** (`next_b2b_starter/`): badges de capacidad, picker con carga del destinatario, vista de workload del supervisor.
- **Auth**: permiso `inbox:manage_limits` en la política Stytch (runtime SSOT) — contrato de rollback dual documentado.
- **Ops**: `make test`, `pnpm build`/`lint`/`tsc`, vitest, Playwright → `qa/`.
- **Rollback**: git revert + reversión de política Stytch (permiso `inbox:manage_limits`) + migración down (tabla de límites).

## Non-Goals

- NO round-robin automático ni routing por carga para leads net-nuevos (decisión del change base se mantiene).
- NO indicadores de typing en vivo ni presencia real-time (sin websockets; futuro).
- NO reestructurar la bandeja actual (restricción "UI intacta" heredada de los cambios base).
- NO almacenar credenciales/sesiones localmente (todo auth sigue en Stytch B2B).

## Gate

- Este change NO se implementa antes de: `conversation-row-scoping` (archivado), `inbox-scope-views` (archivado) e `inbox-assignment-actions` (archivado). Requiere revisión de council (toca canal WhatsApp + contrato Stytch con permiso nuevo) y su `routing.json` lo refleja.
