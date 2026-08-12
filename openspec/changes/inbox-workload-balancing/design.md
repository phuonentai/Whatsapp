# Design: inbox-workload-balancing — capacidad y distribución justa (GATED / futuro)

## Context

- Base requerida: `conversation-row-scoping` (ownership, cola, permisos, flag pagos), `inbox-scope-views` (chips, urgencia 24h), `inbox-assignment-actions` (picker, auto-claim, release).
- Expert research (Front/Intercom/Zendesk 2025-26): **visibilidad de capacidad** como patrón emergente (Front: límites de asignación por teammate visibles como progreso en sidebar; Intercom: balanced assignment con Inbox/Teammate capacity; Zendesk: workload badges + recommended sort). Anti-hoarding: "Keep assignments when teammate is OOO", "Pull conversation" opcional.
- Este change está GATED: no se implementa antes de archivar los tres cambios base; requiere council (canal WhatsApp + permiso Stytch nuevo).

## Goals / Non-Goals

**Goals:**
- Límites de asignación configurables por rol/miembro (conteo de activas).
- Indicadores de capacidad: progreso por miembro, aviso de destinatario al límite en el picker.
- Distribución justa / anti-hoarding con señalización (no round-robin para leads B2B).
- Vista de workload de equipo para supervisores.
- Solo planes pagos (flag `conversation_row_scoping`).

**Non-Goals:** round-robin automático para leads; typing en vivo; reestructurar la bandeja actual; credenciales locales.

## Decisions

1. **Límites como configuración por rol (defaults) con override por miembro** — tabla local `inbox_assignment_limits(organization_id, stytch_member_id NULL=role default, role, max_active)`; el conteo de activas = conversaciones `assignee = miembro AND status='active'`. Defaults por rol en la política (admin=∞, supervisor=15, agente=8) pero configurables por org.
   - **Por qué**: Front usa límites por teammate con defaults por inbox; un tope global por rol es lo más simple y cubre el 80%.
   - **Alternativa**: límites por equipo — no hay equipos todavía (decisión del change base: sin `crm.teams`); se documenta como futuro.
2. **Indicadores de capacidad** — progreso "6/8" en el chip del miembro dentro del picker (de `inbox-assignment-actions`) y en la fila del asignado en "Todos"; al límite: color ámbar + confirmación explícita para transferir igual ("Ana está al límite (8/8) — transferir de todas formas").
   - **Por qué**: expert — la capacidad debe ser visible ANTES de la acción, no después del error.
3. **Anti-hoarding sin prohibir** — señalización, no bloqueo: sobrecapacidad se muestra, transferir igual es posible (excepción explícita con confirm). "Keep assignments when OOO" (no re-asignar automáticamente al volver) queda como regla documentada, no implementada (no hay presencia).
   - **Por qué**: los leads B2B requieren contexto; el bloqueo duro castigaría al equipo. El "Pull" guiado (sugerencia del sistema del siguiente chat por urgencia+capacidad) queda opcional/futuro.
4. **Vista de workload de equipo (supervisor)** — en "Todos", columna/grupo por miembro con conteo y progreso; el mapa de cobertura que ya se lee en chips se vuelve numérico. Read-only, gated por `inbox:view_all`.
5. **Permiso `inbox:manage_limits`** (política Stytch, solo `admin`) para configurar límites; rollback dual documentado (Git + política).

## Risks / Trade-offs

- [Límites mal calibrados frenan el trabajo] → Mitigación: defaults conservadores + override por miembro + el límite nunca bloquea la respuesta urgente (el auto-claim de una cola urgente ignora el límite con registro en audit).
- [Fuga de datos de carga a roles sin permiso] → Mitigación: vista de workload gated por `inbox:view_all`; el progreso individual en el picker es visible solo al transferir.
- [Costo de mantenimiento de configuración] → Mitigación: defaults por rol cubren el caso común; el override es excepción.
- [Cambio de contrato Stytch (permiso nuevo)] → Mitigación: mapeo rol→permiso explícito; rollback dual; cache de política versionada.

## Migration Plan

1. Backend: tabla de límites + migración + conteo de activas + endpoints CRUD (gate `inbox:manage_limits`).
2. Política Stytch: `inbox:manage_limits` (admin).
3. Frontend: indicadores en picker, progreso en filas, vista de workload.
4. Rollback: git revert + política revertida + migración down.

## Open Questions

- ¿El límite aplica a conversaciones activas (status='active') o a todas las visibles en "Mis chats"? → propuesta: activas (las cerradas/archivadas no cargan).
- ¿El auto-claim urgente ignora el límite siempre o con umbral? → propuesta: ignora con registro en audit (la ventana 24h manda).
