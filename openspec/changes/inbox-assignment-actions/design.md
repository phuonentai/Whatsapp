# Design: inbox-assignment-actions — las tres jugadas de ownership

## Context

- Base: `conversation-row-scoping` (PATCH assignee con `inbox:reassign` + auditoría; auto-match en ingestión; cola = assignee NULL) y `inbox-scope-views` (gramática de ownership: chips, cola, "nueva" vs no-leído).
- Expert research: claim-before-reply + auto-claim en primer reply (Front/Intercom), feedback inmediato con undo, colisión → "already claimed by X" visible, un solo owner por conversación (golden rule).
- Bandeja actual intacta: las acciones se añaden como capa (chip de cabecera clicable, menú contextual), sin reestructurar.

## Goals / Non-Goals

**Goals:**
- Un picker que cubre las tres jugadas: claim (cola→mío), transfer (mío→otro), release (mío→cola, confirm + undo 5s).
- Auto-claim al primer reply (además del botón explícito).
- Banner de ownership en el poll cuando la conversación abierta cambia de dueño.
- Toast de llegada al receptor + "nueva" dot.
- Menú contextual de fila para supervisores.
- Conflicto atómico 409 visible ("tomado por X").

**Non-Goals:** workload/capacidad → `inbox-workload-balancing`; nota de asignación (reutiliza `conversation-context-note`); bulk checkbox; push/email; typing en vivo.

## Decisions

1. **Picker de tres jugadas como un solo componente** — `AssignmentPicker` disparado por el chip de cabecera: estado actual del assignee, lista de miembros (misma fuente que `equipo-permisos` MemberList — Stytch member API, sin divergencia), acción "Devolver a la cola" con confirm dialog + toast de undo 5s.
   - **Por qué**: claim/transfer/release comparten contexto (quién es el dueño, a quién se lo doy); un componente = una gramática, tres jugadas. Expert: feedback inmediato con undo.
   - **Alternativas**: (a) botones separados por jugada — dispersión de superficie; (b) drag & drop a un rail de miembros — rechazado (sin rail en IA píldoras; hostil en touch).
2. **Auto-claim al primer reply** — al enviar un mensaje en una conversación con `assignee IS NULL`, el POST de mensajes reclama la conversación al remitente en la misma transacción (idempotente: si otro ya la reclamó, el mensaje se envía igual y la respuesta indica el dueño actual). Implementación: el handler de envío realiza el claim condicional (UPDATE ... WHERE assignee IS NULL RETURNING) — sin cambio de contrato de guardrails.
   - **Por qué**: expert (Front/Intercom): el que responde es el dueño; elimina el paso de "reclamar y luego responder".
   - **Alternativa**: solo botón explícito — válida pero añade fricción a la jugada más común.
3. **Banner de ownership en el poll** — en el poll de 5s, si `assignee` de la conversación abierta cambió desde que se abrió, la cabecera muestra "Reasignada a Ana" (o "Reclamada por Ana" si estaba en cola). No bloquea el envío; informa.
   - **Por qué**: anti-colisión sin websockets (expert: indicador de colisión visible); el redactor deja de invertir atención en un chat que no es suyo. El reply ya enviado nunca se pierde.
4. **Llegada al receptor** — toast in-app "Conversación asignada a ti" + punto "nueva" (de `inbox-scope-views`). Sin push/email (sin infra; futuro).
5. **Menú contextual de fila (supervisores)** — ⋮ en fila → "Asignar a…" con el mismo picker. Cubre el triage por exploración desde "Todos"; bulk checkbox queda v2.
6. **Conflicto atómico 409** — claim/transfer concurrentes: el backend devuelve 409 con el ganador; la UI muestra "Tomado por X" (avatar) y refresca el thread. El race se vuelve visible, no frustrante.
7. **Memoria de handoff sin tabla nueva** — "Guardar como nota" (AI context → CRM actividad `nota`, capability existente) es el vehículo; el copy del picker sugiere "guarda contexto antes de transferir" (tooltip), sin construir campo de nota de asignación.
   - **Por qué**: duplicar notas de asignación paralelas a las actividades CRM crearía un segundo sistema; el contacto ya tiene timeline.

## Risks / Trade-offs

- [Auto-claim sorprende si el remitente no quería quedarse el chat] → Mitigación: el toast tras enviar confirma "Chat reclamado por ti" + acción "Devolver a la cola" (undo directo).
- [Undo de release expira y el chat fue reclamado por otro] → Mitigación: undo solo si el chat sigue en cola (si otro lo reclamó, toast "Ya fue reclamada por Ana" — sin estado inconsistente).
- [Banner de ownership spamea en poll] → Mitigación: solo cuando el assignee CAMBIÓ (diff), no en cada poll.
- [Picker sin presencia muestra miembros que no están] → Mitigación: sin indicadores de presencia falsos (no hay infra); se muestra rol, no online.
- [Reasignación a destinatario sobrecargado] → señal de capacidad en el picker es del change futuro `inbox-workload-balancing`; en v1, la transferencia se permite sin advertencia (documentado).

## Migration Plan

1. Depende de `conversation-row-scoping` (endpoint PATCH assignee) e `inbox-scope-views` (chips).
2. Backend mínimo: auto-claim condicional en POST mensajes (si se decide server-side).
3. Frontend: `AssignmentPicker`, menú contextual, banner, toasts, gates `inbox:reassign`.
4. Rollback: git revert; sin migraciones.

## Open Questions

- ¿El auto-claim se implementa server-side (UPDATE condicional en el handler de envío) o client-side (llamada de claim antes del envío)? → server-side es atómico; client-side es más simple pero con ventana de carrera. Decisión: server-side en la misma transacción.
- ¿El undo de release debe ser 5s fijo o configurable? → 5s fijo (consistente, sin configuración nueva).
