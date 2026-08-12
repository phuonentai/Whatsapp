# Design: Inbox v2 — tiers de acceso + pulido del AI rail

## Context

- Brief: superficie de trabajo; tres tiers (miembro lee/responde, admin/agente todo, plataforma ruta separada); distinción visual IA (✦ violet); escalaciones amber human-only; aprobar = prefill; contexto consent-gated con degradación visible; 402 solo-IA; unread live-region; banner canal desconectado; chip de secuencia.
- Estado actual: bandeja gateada `org:manage` completa (sidebar), master-detail + AI rail ya implementado (sugerencias con approve/reject, escalación `type=escalation`, writing assist 4 modos con 402 toast, quick replies + secuencias, contexto consent-gated, métricas, close/reopen, poll 5s).
- Backend: endpoints de conversaciones/mensajes con auth + org_context; guardrails `agent-governance` (send_message deterministic); política Stytch = runtime SSOT.

## Goals / Non-Goals

**Goals:**
- Tier de miembro: leer + responder manual, sin controles de admin.
- Enforcement server-side (no solo UI).
- AI rail pulido con distinción ✦, escalación human-only, prefill al aprobar.
- 402 solo para IA; unread live-region; banner de canal desconectado; chip de secuencia.

**Non-Goals:**
- NO plataforma operador aquí (admin-panel).
- NO auto-send de sugerencias.
- NO cambios de guardrails/streaming/consent.
- NO duplicar permisos locales (todo en política Stytch).

## Decisions

1. **Permisos nuevos en Stytch** — `inbox:view` + `inbox:reply` en la política, asignados a un rol (decisión: `member` gana `inbox:view`+`inbox:reply`; `agent` y `admin` conservan `org:manage` para todo). Alternativa (reutilizar `org:view`) se descarta: `org:view` es demasiado amplio y acoplaría la bandeja a la gestión de org. El enforcement server-side lee los permisos efectivos del miembro (ya disponibles en el contexto de auth).
2. **Enforcement en endpoints** — lectura (`GET /crm/conversaciones*`) con `inbox:view`; envío manual con `inbox:reply`; close/reopen/quick-replies/sugerencias con `org:manage`. 403 server-side + ocultar controles en UI (regla "hide what you can't use", con la excepción del tooltip de créditos).
3. **Envío manual de miembro por guardrails** — el path de envío no cambia; el miembro es un actor más del snapshot de guardrails (denials en audit). Sin excepción de rol en `agent-governance`.
4. **Distinción visual IA** — componente `AiBadge`/estilo ✦ + tinte violet para drafts/sugerencias; mensajes humanos sin marca. Escalaciones amber con nota "requiere juicio humano"; sin acción de auto-envío.
5. **Aprobar = prefill** — botón "Aprobar y enviar" actualmente envía; se cambia a prefill del composer (draft listo para revisión) + envío explícito. Alternativa (mantener approve-and-send como dos pasos) se documenta: el brief exige prefill; se implementa prefill y se conserva el flujo de dos pasos para autopiloto (sin cambio).
6. **402 solo-IA** — el estado de créditos se lee de `useAiUsageQuery`; sparkles disabled + tooltip; banner en panel de sugerencias; composer manual intacto.
7. **Unread + live-region** — store existente (`use-inbox-store`) + poll 5s; live-region discreta con conteo incremental.
8. **Banner de canal desconectado** — estado de conexión de whatsapp/instagram (queries existentes) → banner sobre composer por canal seleccionado.

## Risks / Trade-offs

- [Nuevos permisos Stytch rompen miembros existentes] → Mitigación: mapeo rol→permiso explícito en el change; rollback dual (Git + política Stytch) documentado.
- [Enforcement duplicado en varios endpoints] → Mitigación: helper de autorización compartido (middleware/check por permiso), tests por rol.
- [Prefill vs approve-and-send esperado por usuarios] → Mitigación: mantener el botón "Aprobar" que precarga + el envío manual explícito; copy aclara "precargado, revisa y envía".
- [Live-region spam en poll] → Mitigación: anunciar solo incrementos, no el estado completo.
- [Bandeja más abierta = más superficie de compliance] → Mitigación: consentimiento y guardrails aplican igual a miembros; auditoría registra actor del envío.

## Market & Unit Economics

Este change **toca el canal WhatsApp/Meta y el modelo de autorización**, y lo declara:

- **Canal WhatsApp (Meta Business Platform): sin costo nuevo por mensaje.** El envío manual de miembro usa el mismo path/guardrails/metering; no hay nuevas llamadas LLM ni cambios de pricing de conversación. El tier de miembro amplía quién puede responder, no cuánto cuesta responder.
- **Autorización: cambio de contrato Stytch.** Se añaden permisos `inbox:view`/`inbox:reply` y su asignación a roles (runtime SSOT); sin costo directo, pero con superficie de deriva de política (R1) y necesidad de rollback dual documentado.
- **Precios / planes / créditos: sin cambio.** No se tocan `paywall`, `plan-pricing-ux` ni `ai-usage-metering`; el 402 solo-IA reutiliza el guard existente.
- **Métrica de operación:** el tier de miembro puede reducir tiempo de respuesta (más operadores pueden responder) — medible como follow-up de analytics, no en este change.

## Market Risk

- **R1 — Deriva de política RBAC (seguridad + compliance).** Nuevos permisos de bandeja asignados a `member` amplían superficie de lectura de datos personales (WhatsApp). **Owner:** security/architecture. **Trigger:** incidente de acceso, auditoría, o cambio de roles de Stytch. **Mitigación:** enforcement server-side (403), guardrails aplicados a miembros, auditoría registra actor del envío, rollback dual documentado (Git + política Stytch).
- **R2 — Términos de Meta (agentes de IA y pricing de conversación).** La distinción ✦ IA y el banner de canal desconectado son superficie de deriva si la copy sugiere capacidades no confirmadas. **Owner:** product/ops. **Trigger:** cambio de términos de Meta sobre agentes de IA; copy que afirme comportamiento autónomo. **Mitigación:** copy en `lib/copy/ui.ts` sin claims de autonomía; el prefill al aprobar mantiene human-in-the-loop (filosofía copilot).
- **R3 — Fricción de miembros con controles ocultos.** Ocultar quick replies/sugerencias a miembros (regla "hide, no ghosts") puede frustrar si el miembro esperaba esas herramientas. **Owner:** product. **Trigger:** feedback de miembros o tickets. **Mitigación:** el tier es explícito y documentado; el 402-only exception (tooltip) evita confusión de sistema vs permiso.

## Migration Plan

1. **Política Stytch**: añadir `inbox:view`/`inbox:reply` y asignar a roles (Stytch dashboard o API, documentado).
2. **Backend**: enforcement por permiso en endpoints + helper compartido + tests 200/403 por rol.
3. **Frontend**: sidebar gate (bandeja visible con `inbox:view` u `org:manage`), ocultar controles admin, pulido AI rail, unread live-region, banner, chip secuencia.
4. **Gates**: `make test`, vitest inbox, lint/build/tsc, Playwright visual/a11y (3 tiers).
5. **Rollback**: revertir Git + revertir política Stytch (restaurar gate `org:manage`).

## Open Questions

- ¿El rol `member` actual existe en la política Stytch con permisos asignables? (spike en tasks 1.1).
- ¿El DTO de sugerencias distingue autoría IA para el ✦ visual? (si no, se marca por origen de la sugerencia, que ya es IA por construcción).
- ¿`useAiUsageQuery` es suficiente para el estado de créditos del composer? (verificar en tasks; sino, extender con el guard 402 existente).
