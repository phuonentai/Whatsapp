# Tasks: Custom roles editor — edición in-app de la política RBAC de Stytch

Editor de roles personalizados que escribe en la política de Stytch vía API (runtime SSOT), con validación, breaker, auditoría y rollback documentado. **Gated**: requiere council y se implementa después de `equipo-permisos`. Backend Go + frontend Next.

## 1. Spike y política [BE-INFRA]

- [ ] 1.1 Spike: documentar la API de roles de Stytch (métodos para crear/actualizar/archivar roles y permisos), límites, y cómo capturar backup/restauración de la política. Verify: notas en tasks; endpoint/métodos confirmados
- [ ] 1.2 Añadir permiso `roles:manage` a la política Stytch (asignado a admin); documentar rollback de política (backup previo + restauración). Verify: política aplicada; rollback documentado en tasks

## 2. Backend Go [BE-DOMAIN]

- [ ] 2.1 Domain interface `RolePolicyRepository` (CreateRole/UpdateRole/ArchiveRole/ListRoles) sin importar SDK. Verify: `make build`; sin SDK en domain
- [ ] 2.2 Adapter `infrastructure/auth/stytch/role_policy.go` implementando la interface con breaker (threshold 5, timeout 10s, half-open 2) e idempotencia. Verify: `make test`; tests de idempotencia y breaker abierto (rechazo sin escritura)
- [ ] 2.3 Endpoints `/rbac/roles` create/update/archive: validación server-side (permisos del catálogo, roles del sistema protegidos → 403, duplicados), auditoría con diff de permisos. Verify: `make test`; tests 403 y validación
- [ ] 2.4 Backup/versionado de política previo a escritura (para rollback). Verify: `make test`; snapshots de política generados
- [ ] 2.5 Listar roles personalizados en la respuesta de `/rbac/roles` para matriz/asignación (integración con equipo-permisos). Verify: `make test`; roles personalizados en DTO

## 3. Frontend [FE-NEXT]

- [ ] 3.1 Editor UI (`?view=roles` o tab avanzado de access): lista de roles personalizados (activo/archivado), formulario crear/editar (nombre, descripción, permisos por categoría del catálogo), confirmar archivar. Verify: `pnpm build`
- [ ] 3.2 Validación en UI: permisos desconocidos, duplicados, roles del sistema bloqueados (sin llamadas inválidas). Verify: `pnpm build`; errores inline
- [ ] 3.3 Nota de propagación "Los cambios aplican en hasta 5 minutos" tras guardar + refetch de matriz/asignación. Verify: `pnpm build`
- [ ] 3.4 Gate de vista `org:manage` + `roles:manage` (403 sin permiso). Verify: `pnpm build`; 403 sin controles
- [ ] 3.5 Integración en `equipo-permisos`: roles personalizados como columnas de la matriz y opciones del role select de miembros. Verify: `pnpm build`
- [ ] 3.6 Copy nueva en `lib/copy/ui.ts` (español tipado: "Roles personalizados", "Nuevo rol", "Archivar rol", "Cambios aplican en hasta 5 minutos"). Verify: `pnpm lint`

## 4. Verificación [OPS-GOV]

- [ ] 4.1 Gate backend: `make test` (idempotencia, breaker, 403, validación), `make lint`. Verify: comandos PASS
- [ ] 4.2 Gate estático frontend: `pnpm lint`, `pnpm build`, `npx tsc --noEmit`. Verify: los tres comandos PASS
- [ ] 4.3 Gate unitario frontend: `npx vitest run app/dashboard/settings/components/` → todos PASS. Verify: comando PASS
- [ ] 4.4 Gate visual/a11y: capturas Playwright 390x844/768x1024/1440x900 del editor y matriz → `openspec/changes/custom-roles-editor/qa/`. Verify: artefactos en `qa/`
- [ ] 4.5 Cierre: `openspec validate custom-roles-editor --type change` PASS; registrar decisión de archivado (ejecutar /opsx-archive o anotar `**Archive deferred:** <razón>`). Verify: comando validate PASS

**Archive deferred:** change gated — requiere aprobación de council (escritura a la política de Stytch) y se implementa solo después de `equipo-permisos`; el rollback dual (Git + política Stytch) se ejecuta según el procedimiento documentado en tasks 1.2.
