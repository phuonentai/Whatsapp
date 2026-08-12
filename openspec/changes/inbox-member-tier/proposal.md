# Proposal: Inbox v2 — tiers de acceso (miembro lee/responde) + pulido del AI rail

## Why

El brief de Inbox define la bandeja como superficie de trabajo ("responder personas correctamente y rápido") con **tres tiers de acceso** — hoy toda la bandeja se gatea con `org:manage`, lo que excluye a miembros que deberían poder leer y responder. El brief especifica además el pulido del AI rail: distinción visual de contenido IA (✦ violet), sugerencias de escalación amber (human-only), contexto consent-gated con degradación visible, composer con créditos agotados (402) que deshabilita solo el asistente IA (nunca el envío manual), unread con live-region, y banner de canal desconectado. Los contratos de guardrails (agent-governance), streaming y consentimiento siguen intactos.

## What Changes

- **Tiers de acceso en la bandeja** (nuevo contrato de permisos):
  - `ORG_MANAGE`/agente: todo (responder, cerrar/reabrir, quick replies + secuencias, sugerencias IA con aprobar/rechazar, writing assist, contexto).
  - Miembro (`org:view` u otro rol de lectura): **leer lista + thread y responder manualmente**; sin cerrar/reabrir, sin quick replies/secuencias, sin panel de sugerencias, sin writing assist (ocultar, no renderizar ghosts). Excepción UX: el menú de transformación IA del composer muestra disabled + tooltip cuando créditos = 0 (condición de sistema, no permiso).
  - Plataforma (operador): ruta separada (admin panel, change aparte).
- **Backend**: el endpoint de mensajes/conversaciones acepta lectura y envío manual para miembros sin `org:manage` (nuevo permiso Stytch `inbox:reply`/`inbox:view` o equivalente, definido en la política); close/reopen/quick-replies/sugerencias siguen en `org:manage`. Enforcement server-side (no solo UI). Circuit-breaker y guardrails de envío sin cambios (el envío manual de miembro pasa por el mismo path con guardrails: denials registrados en audit).
- **AI rail pulido** (frontend):
  - Contenido IA distinto visualmente: drafts con ✦ + tinte violet suave; mensajes humanos sin marca.
  - Escalaciones (`type=escalation`) amber "requiere juicio humano", nunca auto-send.
  - Aprobar sugerencia NUNCA envía en silencio: prefill del composer (o dos pasos explícitos).
  - Contexto consent-gated: si el contacto retiró consentimiento, tab Contexto muestra tarjeta estructural ("mensajes: 14 · consentimiento retirado") sin hechos pre-generados; degradación visible, no oculta.
  - Composer: créditos = 0 → sparkles disabled + tooltip "Créditos agotados — ver plan"; composer manual siempre funcional.
  - Unread: dot + bold en lista; anuncio live-region "2 nuevos" en el poll de 5s.
  - Banner de canal desconectado ("WhatsApp desconectado — los envíos fallarán") sobre el composer.
  - Secuencias: chip "Paso k de n" sobre el input; avanza solo en envío exitoso; reset al cambiar conversación.
  - Estados: skeleton (5 ítems + thread), empty por canal con link a settings, error + retry.
- **Sin cambios de contrato**: guardrails (send_message), streaming/SSE, contexto (ai-context-intelligence), writing-assist (402), quick replies/playbooks.

## Capabilities

### New Capabilities

- (ninguna — el tier de acceso se gobierna por deltas en `stytch-authorization` + `whatsapp-inbox` + `inbox-ui`)

### Modified Capabilities

- `stytch-authorization`: se introducen permisos de bandeja (leer/responder) distintos de `org:manage`; la política Stytch define los roles que los poseen; el enforcement server-side SHALL aplicarlos en los endpoints de conversaciones/mensajes.
- `whatsapp-inbox`: la bandeja acepta lectura + envío manual para miembros; close/reopen/quick-replies/AI siguen en `org:manage`; el endpoint de envío manual de miembro pasa por los guardrails existentes.
- `inbox-ui`: AI rail pulido (✦ visual, escalaciones amber, prefill al aprobar, contexto consent-gated degradado, composer 402 solo-IA, unread live-region, banner canal desconectado, chip de secuencia).
- `agent-governance`: se aclara que el envío manual de miembro atraviesa los mismos guardrails (kill switch, consent, ventana, límite diario) con denials en audit.

## Impact

- **Backend**: `go-b2b-starter/` — permisos Stytch nuevos (política + endpoints de conversaciones/mensajes con enforcement por rol), tests de 403/200 por rol. Sin migración de datos; la política de roles vive en Stytch (runtime SSOT) y su rollback es vía Stytch dashboard/policy.
- **Frontend**: `next_b2b_starter/app/dashboard/inbox/*` (AI rail, composer, unread, banner), `components/layout/sidebar.tsx` (gate de la bandeja: ver con `inbox:view` u `org:manage`), `lib/auth/permissions.ts` (constantes), `lib/copy/ui.ts`.
- **Auth**: cambio de contrato Stytch real — nuevos permisos y roles; SHALL documentarse el rollback de la política en Stytch y el mapeo rol→permisos en el change.
- **Dependencias**: ninguna nueva.
- **Ops**: `make test` (Go), `pnpm build`/`lint`/`tsc`, vitest de inbox (conversation-list, reply-input, agent-suggestions-panel), Playwright visual/a11y → `qa/`.
- **Rollback**: git revert + reversión de la política Stytch (restaurar gate `org:manage` en la bandeja); sin estado de DB.
- **Non-Goals**: sin plataforma operador en este change (ruta separada en admin-panel); sin auto-send de sugerencias (prefill siempre); sin cambios de guardrails/streaming/consent; sin almacenar credenciales localmente (todo auth sigue en Stytch B2B).

## Assumptions

- La política Stytch actual gatea la bandeja con `org:manage` (verificado en sidebar/`whatsapp-inbox`); el nuevo permiso `inbox:reply` (o equivalente) se añade a la política y se asigna a un rol existente o nuevo (decisión de producto en design.md).
- El poll de 5s y el store de inbox (`use-inbox-store`) ya existen; el anuncio live-region se añade sin cambiar el poll.
- `type=escalation` ya existe en el DTO de sugerencias (verificado); el tratamiento amber es presentacional + regla de no auto-send.
- El contexto consent-gated ya distingue consentimiento retirado (ai-context-intelligence); la degradación visible es UI.
