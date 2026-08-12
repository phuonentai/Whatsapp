# Tasks: Settings redesign — lenguaje visual del diseño en los 10 módulos

Re-estilo de los 10 módulos de settings al lenguaje del diseño (superficies claras, chips semánticos, jerarquía) con cero cambios de comportamiento. Frontend only (`next_b2b_starter/`).

## 1. Fundaciones visuales [FE-NEXT]

- [x] 1.1 Crear componente `StatusChip` (emerald/amber/red/gray) con icono + texto (sin color-only) en `components/ui/`. Verify: `pnpm build`; inspección visual — PASS (build OK; chips render icon+text en whatsapp/modules/subscription)
- [x] 1.2 Definir convención del lenguaje (superficies slate-50/white, bordes slate-200, rounded-2xl, tiles de color) y copy faltante en `lib/copy/ui.ts`. Verify: `pnpm lint` — PASS (0 errores; convención documentada en doc-comment de `ui.ts` + namespace `ui.settings`)

## 2. Migración de módulos [FE-NEXT]

- [x] 2.1 Profile (profile-section + security-section + mfa-policy-section): migrar `gray-*` → lenguaje; sin cambios de formulario. Verify: `npx vitest run app/dashboard/settings/components/security-section.test.tsx app/dashboard/settings/components/mfa-policy-section.test.tsx`; `pnpm build` — PASS (6 tests; build OK; test de mfa-policy actualizado a `bg-slate-900`)
- [x] 2.2 Subscription (subscription-tab): re-estilo + barras de uso (créditos IA `useAiUsageQuery`, facturas `state.usage`) con umbral amber ≥80%; lógica de cancelación/resumen intacta. Verify: `npx vitest run app/dashboard/settings/components/subscription-tab.test.tsx`; `pnpm build` — PASS (10 tests, incl. 3 nuevos de umbral amber; build OK)
- [x] 2.3 Modules (modules-section): toggles re-estilizados con badge de fuente de plan; contract `module-registry` intacto. Verify: `npx vitest run app/dashboard/settings/components/modules-section.test.tsx`; `pnpm build` — PASS (3 tests con badge "Incluido en plan Pro"; build OK)
- [x] 2.4 Compliance (compliance-section): consent/export/anonymize re-estilizados; traceability intacta. Verify: `pnpm build` — PASS
- [x] 2.5 Audit log (audit-log-view): tabla re-estilizada; contrato append-only intacto. Verify: `pnpm build` — PASS
- [x] 2.6 Messaging WhatsApp (whatsapp-config-section): chips de estado semánticos; tokens Meta siguen en panel avanzado colapsado. Verify: `npx vitest run app/dashboard/settings/components/whatsapp-config-section.test.tsx`; `pnpm build` — PASS (5 tests; chip "Conectado/Pausado" data-driven desde `config.isActive`)
- [x] 2.7 Message templates (templates-section): flujo Meta re-estilizado; sin cambios de contract. Verify: `pnpm build` — PASS
- [x] 2.8 Instagram (instagram-config-section): chips de estado; sin cambios de contract. Verify: `pnpm build` — PASS
- [x] 2.9 Siigo integration (siigo-integration-section): wizard 5 pasos re-estilizado (STEP_ORDER intacto). Verify: `npx vitest run app/dashboard/settings/components/siigo-integration-section.test.tsx`; `pnpm build` — PASS (9 tests; STEP_ORDER intacto)
- [x] 2.10 Siigo admin (siigo-admin-view): vista de operación re-estilizada; gate admin intacto. Verify: `npx vitest run app/dashboard/settings/components/siigo-admin-view.test.tsx`; `pnpm build` — PASS (3 tests)
- [x] 2.11 Overview de settings (settings-content.tsx): tarjetas de sección con icono/valor/helper en el lenguaje; stack de vistas sin cambios. Verify: `pnpm build`; navegación `?view=` intacta — PASS (tiles tintados por módulo; navegación verificada en QA)

## 3. Verificación [OPS-GOV]

- [x] 3.1 Gate estático: `pnpm lint` (0 errores nuevos vs baseline), `pnpm build`, `npx tsc --noEmit`. Verify: los tres comandos PASS — PASS (lint 0 errores / 4 warnings pre-existentes en archivos no tocados; build compiled OK; tsc limpio)
- [x] 3.2 Gate unitario: `npx vitest run app/dashboard/settings/components/` → todos PASS. Verify: comando PASS — PASS (8 archivos, 40 tests)
- [x] 3.3 Gate visual/a11y: capturas Playwright 390x844/768x1024/1440x900 de los 10 módulos → `openspec/changes/settings-redesign/qa/`; checklist contraste + chips con texto. Verify: artefactos en `qa/` — PASS (33 capturas + `qa-report.json`; checklist DOM: chips con texto, tiles tintados, badge de plan, sin errores de consola inesperados; limitaciones de entorno documentadas: templates backend JSON mismatch e instagram no montado en el servidor API — pre-existentes)
- [x] 3.4 Cierre: `openspec validate settings-redesign --type change` PASS; registrar decisión de archivado (ejecutar /opsx-archive o anotar `**Archive deferred:** <razón>`). Verify: comando validate PASS — validate PASS; **Archive deferred:** pendiente de ejecutar `/opsx-archive` tras revisión del council/UIUX (artefactos de QA en `qa/`).
