# Proposal: Settings redesign — lenguaje visual del diseño en los 10 módulos de configuración

## Why

El equipo UX/UI definió (sesión de diseño, export Shuffle + briefs) un lenguaje visual para las superficies de producto: chrome oscuro slate-900, superficies de contenido claras `slate-50`/white con bordes `slate-200`, acentos emerald para acciones/activo, violet ✦ para superficies IA, rojo solo para destructivo. Los 10 módulos de settings existen hoy como rutas (`?view=profile|subscription|modules|compliance|audit|whatsapp|templates|instagram|siigo|siigo-admin`) pero usan clases `gray-*` hardcodeadas (gris neutro, sin semántica) y composición desigual. Esta propuesta re-estiliza los 10 módulos al lenguaje del diseño, cierra gaps de contenido existentes (usage limits en subscription, estados de conexión en messaging/instagram, wizard Siigo 5 pasos) y conserva todo el comportamiento (RBAC, gates, contratos Stytch/Polar/Siigo).

## What Changes

- **Lenguaje visual unificado** en `app/dashboard/settings/components/*` + `settings-content.tsx`: superficies `slate-50`/white con bordes `slate-200`, tarjetas con `rounded-2xl` y sombras suaves, chips de estado emerald/amber/red con texto (nunca color-only), iconos con tile de color por módulo, jerarquía tipográfica consistente (título 2xl, descripción, secciones). Migración de clases `gray-100/200/600/900` hardcodeadas → tokens/utiles del lenguaje, sin cambiar lógica.
- **Account / Profile**: tarjeta de perfil + seguridad (passkeys/MFA existente) con el lenguaje; sin cambios de formulario.
- **Subscription & billing**: ya muestra plan, uso (créditos IA + facturas incluidas/usadas con barras), cancelación/resumen — se re-estiliza y se hace visible el **límite de uso** como barras con umbral amber ≥80% (datos ya disponibles vía `useAiUsageQuery` + `state.usage`); sin cambios de lógica Polar.
- **Modules**: toggles existentes re-estilizados al lenguaje, con fuente de plan (badge "Incluido en plan X"); sin cambios de contract `module-registry`.
- **Compliance (Ley 1581)**: consentimiento, exportación y anonimización existentes re-estilizados; traceability intacta.
- **Audit log**: tabla re-estilizada; contrato de auditoría append-only intacto.
- **Messaging (WhatsApp)**: estados de conexión (conectado/pausado) con chips semánticos; Meta developer tokens siguen en el panel avanzado colapsado; sin cambios de contract `whatsapp-config`.
- **Message templates**: flujo de aprobación Meta existente re-estilizado; sin cambios de contract `whatsapp-templates`.
- **Instagram config**: estados de conexión con chips; sin cambios de contract.
- **Siigo integration + wizard 5 pasos**: wizard existente re-estilizado; sin cambios de contract Siigo.
- **Siigo admin onboarding**: vista de operación re-estilizada; gate admin intacto.
- **Overview de settings**: la lista de secciones adopta el lenguaje (tarjetas de sección con icono, valor, helper); sin cambios de navegación/stack de vistas.

## Capabilities

### New Capabilities

- (ninguna — son deltas sobre capacidades existentes)

### Modified Capabilities

- `settings-ui`: la requirement de presentación de settings cambia — los 10 módulos SHALL presentar el lenguaje visual del diseño (superficies, chips de estado semánticos, jerarquía) manteniendo todos los contratos de comportamiento (invite, toggles, playbooks, perfil, plan, whatsapp).
- `workspace-settings-management`: la presentación de perfil/nombre del workspace adopta el lenguaje visual; los contratos de sync Stytch circuit-breaker-guarded y RBAC intactos.
- `billing-provider-ux`: la presentación de subscription adopta el lenguaje y hace visible el uso (créditos IA + facturas) con umbrales; la lógica de cancelación/resumen/cambio de plan intacta.

## Impact

- **Frontend only**: `next_b2b_starter/app/dashboard/settings/components/*` (re-estilo), `settings-content.tsx` (overview), `lib/copy/ui.ts` (copy de chips/títulos faltantes). Sin backend, sin migraciones, sin SQLC, sin cambios de API.
- **Auth**: cero cambios Stytch; RBAC de vistas (`?view=` gates) intacto; `siigo-admin` sigue admin-only.
- **Billing**: ramas de cancelación/resumen/cambio de plan intactas byte a byte en comportamiento; solo clases/copy.
- **Dependencias**: ninguna nueva.
- **Ops**: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`; vitest existente de settings (modules-section, whatsapp-config-section, siigo-integration-section, siigo-admin-view, subscription-tab, security-section, mfa-policy-section) debe pasar; visual/a11y Playwright → `qa/`.
- **Rollback**: git revert; sin estado de DB ni de Stytch.
- **Non-Goals**: sin nuevas funcionalidades de settings (equipo/permisos consolidado es change aparte `equipo-permisos`); sin cambios de rutas/URLs; sin cambios de contract (Stytch/Polar/Siigo/Meta/Instagram); sin almacenar credenciales localmente (tokens Meta/Siigo siguen en el backend).

## Assumptions

- Los 10 módulos listados por el equipo ya existen como vistas (`?view=*`) — verificado en `settings-content.tsx`; el trabajo es lenguaje visual + gaps de presentación (usage limits ya computados pero menos visibles).
- "Uso/límites" en subscription se refiere a créditos IA (`ai_usage`) e facturas incluidas/usadas (`usage`) — ambos ya disponibles; no se piden nuevos endpoints.
- Los chips de estado usan color + texto (regla de a11y: sin color-only).
