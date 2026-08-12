# Tasks: Consistencia visual de producto — CRM, reportes, procurement, auth, marketing y signup

Barrido de consistencia visual semántica (color+texto, destructivo red, superficies IA ✦) en las superficies restantes, sin cambios de lógica. Frontend only (`next_b2b_starter/`).

## 1. Fundaciones [FE-NEXT]

- [ ] 1.1 Reutilizar/extender el `StatusChip` o helper de estados semánticos (emerald/amber/red/gray) desde `settings-redesign` (o definirlo si no existe); un solo punto de colores semánticos. Verify: `pnpm build`
- [ ] 1.2 Grep baseline: inventariar clases duras residuales (`gray-*`, `emerald-500`, `shadow-emerald-*`, `slate-900`) en las rutas objetivo. Verify: salida grep documentada en tasks

## 2. Migración por superficie [FE-NEXT]

- [ ] 2.1 CRM (`app/dashboard/crm/*`): badges de etapa/etiqueta con color+texto, diálogos destructivos red, botones primarios emerald; queries/acciones intactas. Verify: `npx vitest run app/dashboard/crm/` (si aplica); `pnpm build`
- [ ] 2.2 Reportes (`app/dashboard/reportes/*`): series con `--chart-*`, leyendas con texto, estado vacío honesto, tablas con `th scope="col"`; endpoints intactos. Verify: `pnpm build`; charts renderizan
- [ ] 2.3 Procurement + cronogramas (`app/dashboard/procurement/*`): chips de estado con texto; flujo intacto. Verify: `pnpm build`
- [ ] 2.4 Auth/authenticate (`app/auth/*`, `app/authenticate/*`): superficies al lenguaje; contratos Stytch intactos. Verify: `pnpm build`; flujos renderizan
- [ ] 2.5 Signup (`app/signup/*`): wizard al lenguaje (pasos DIAN visuales); `use-signup-flow.ts` sin cambios. Verify: `npx vitest run app/signup/page.test.tsx`; `pnpm build`
- [ ] 2.6 Marketing (`app/(marketing)/*`, `components/marketing/*`): barrido de clases duras residuales → tokens/lenguaje; rutas/SEO intactos. Verify: grep sin coincidencias; `pnpm build`
- [ ] 2.7 Copy faltante en `lib/copy/ui.ts` (si algún estado nuevo requiere texto). Verify: `pnpm lint`

## 3. Verificación [OPS-GOV]

- [ ] 3.1 Gate estático: `pnpm lint`, `pnpm build`, `npx tsc --noEmit`. Verify: los tres comandos PASS
- [ ] 3.2 Gate unitario: vitest de CRM/reportes/procurement/signup existente → todos PASS (page-objects actualizados si dependen de clases). Verify: comando PASS
- [ ] 3.3 Grep de verificación: sin `gray-*`/`emerald-500`/`shadow-emerald-*` residuales fuera del lenguaje aprobado en las rutas objetivo. Verify: comando grep sin coincidencias
- [ ] 3.4 Gate visual/a11y: capturas Playwright 390x844/768x1024/1440x900 de CRM/reportes/procurement/auth/signup/marketing → `openspec/changes/product-visual-consistency/qa/`; contraste y chips con texto. Verify: artefactos en `qa/`
- [ ] 3.5 Cierre: `openspec validate product-visual-consistency --type change` PASS; registrar decisión de archivado (ejecutar /opsx-archive o anotar `**Archive deferred:** <razón>`). Verify: comando validate PASS
