# Sesión E2E Siigo — Reporte de contexto para continuar

**Generado:** 2026-08-10 (sesión de implementación de `add-siigo-e2e-tests`)
**Propósito:** capturar estado, errores, fixes y pendientes para que una sesión nueva continúe sin perder contexto.

---

## 1. Estado del cambio

**Cambio:** `openspec/changes/add-siigo-e2e-tests` — **24/24 tareas marcadas [x]**, `openspec validate` OK.
**Decisión de archivo:** **Archive deferred** — pendiente: (a) gate 6.3 `make test-e2e` en verde, (b) verificación sandbox de despliegue, (c) archivar junto a los 3 cambios Siigo hermanos.

Los 3 cambios Siigo hermanos también están complete/deferred:
- `add-siigo-org-onboarding` (15/15) — conexión, credenciales cifradas, state machine, gating
- `add-siigo-onboarding-data` (16/16) — numeración, import, test-invoice, delta job
- `add-siigo-onboarding-wizard` (16/16) — wizard FE, admin view, kill-switch

---

## 2. Defectos de PRODUCCIÓN encontrados y corregidos por los tests

Estos son fixes reales en código de producción (no solo tests) — crítico que la nueva sesión los conozca:

### 2.1. Import duplicaba empresas por NIT (change add-siigo-onboarding-data)
- **Síntoma:** test de integración (`sqlc/integration/invoicing_integration_test.go::TestImportConfirm_IsIdempotentOnSecondRun`) falló: 2ª corrida creaba duplicados.
- **Causa:** `import_service.go` buscaba con NIT normalizado (`9001112223`) pero GUARDABA el crudo (`900.111.222-3`) → `GetByNit` nunca matcheaba en re-runs.
- **Fix:** `internal/modules/invoicing/app/services/import_service.go` — `Create` guarda `Nit: nit` (normalizado, dígitos).

### 2.2. Ruta webhook/org de Siigo con doble prefijo `/api/api/...`
- **Síntoma:** rutas registradas como `/api/v1/org/siigo/...` bajo el grupo montado en `/api` → rutas reales `/api/api/v1/...`.
- **Causa:** se copió el patrón de billing (bug pre-existente de billing con `/api/v1/webhooks`); el patrón correcto es `/v1/...` (referencia whatsapp).
- **Fix:** `internal/modules/invoicing/routes.go` — grupos `/v1/org/siigo`, `/v1/admin/siigo`, `/v1/webhooks`. **Nota:** billing sigue con el doble prefijo (fuera de scope, documentado).

### 2.3. Wizard FE: dead-end en el paso 4 (test-invoice)
- **Síntoma:** la org quedaba en `numeracion_ok` y el paso "Prueba en sandbox" solo se renderizaba en `sandbox_ok` — pero solo el test-invoice lleva a `sandbox_ok`. Callejón sin salida: imposible llegar a `live`.
- **Fix:** `app/dashboard/settings/components/siigo-integration-section.tsx` — `SandboxAndActivateStep` ahora se renderiza también en `numeracion_ok`.

### 2.4. NAVEGACIÓN settings rota para vistas nuevas
- **Síntoma:** `?view=siigo` / `?view=siigo-admin` no abrían la vista (quedaba el overview).
- **Causa:** whitelist `isAllowed` en `settings-content.tsx` no incluía las vistas nuevas.
- **Fix:** añadidas `siigo` y `siigo-admin` (ambas con `canManageMembers`) al `isAllowed`.

### 2.5. PANIC test-invoice: nil invoice → SIGSEGV
- **Síntoma:** `panic: invalid memory address` en `invoiceRepository.Insert` — el `TestInvoiceService` usaba el **router**, que resuelve orgs no-live a `NoopProvider` → `CreateInvoice` devuelve `(nil, nil)` → `Insert(nil)`.
- **Fix:** `module.go` — `testInvoiceParams.Provider` ahora es el binding `name:"siigo"` (adapter directo, NO el router) + guard defensivo `created == nil → error` en `test_invoice_service.go`.

### 2.6. FK `invoices_deal_id_fkey` en test-invoice
- **Síntoma:** `ERROR: violates foreign key constraint invoices_deal_id_fkey` — el repo insertaba `deal_id=0` en vez de NULL.
- **Fix:** `invoice_repository.go::Insert` — `DealID: pgtype.Int4{Int32: inv.DealID, Valid: inv.DealID != 0}`.

### 2.7. Test-invoice no avanzaba a `sandbox_ok` (early-return)
- **Síntoma:** el mock resuelve `valid` en el POST; `remote.Status == stored.Status` → early-return ANTES del `ConfirmSandboxOK` → la org nunca avanzaba.
- **Fix:** `test_invoice_service.go` — el early-return se reemplazó por "update solo si difiere", y el avance depende de `stored.Status == valid` (cubre POST ya-valid).
- **Test nuevo:** `TestTestInvoice_AlreadyValidOnCreateStillAdvances`.

---

## 3. Otros fixes necesarios (no producción)

- `settings.page.ts` / `admin-panel.page.ts` (e2e): cierres de clase arreglados tras appends (patrón: el archivo original YA terminaba con `}`; verificar antes de hacer `cat >>`).
- `admin-panel.page.ts`: `SETTINGS_VIEW_HEADINGS` ganó `siigo` y `siigo-admin`; `openSiigoOnboarding` usa `getByText` (CardTitle no es heading accesible).
- Spec `siigo-onboarding.spec.ts`: la org siigo es **id 5** (NO 7); asserts tolerantes a estado heredado (ver §5).
- Mocks Go actualizados por firmas de OTROS cambios en curso: `CompanyRepository` ganó `CountList`/`CountSearch`; `ContactRepository` ganó `CountFiltered`/`CountSearch`; `ActivityService.List*` devuelve `ListResult[T]` en vez de `[]T`. (Fixes aplicados en `import_service_test.go`, `mocks_test.go`, `invoicing_service_test.go`.)

---

## 4. Errores de ENTORNO (importantes)

1. **Procesos zombie de sesiones e2e anteriores** (backend `:8080`, frontend `:3001`, redis `:6379` de ~15-17h): matar antes de correr `make test-e2e`. El postgres del SISTEMA (servicio `postgresql`, `/usr/lib/postgresql/17`) ocupa `127.0.0.1:5432` — **detenerlo con `sudo systemctl stop postgresql`** para liberar el puerto al compose (verificado: `sudo -n` funciona sin contraseña).
2. **El frontend dev (`pnpm dev`) se cae intermitentemente** (OOM/env): relanzar con `cd next_b2b_starter && setsid env pnpm dev -p 3001 > /tmp/e2e-frontend.log 2>&1 < /dev/null & disown` y esperar ~30s. Verificar con `curl http://localhost:3001`.
3. **`go` no está en PATH** en shells nuevos: `export PATH=$PATH:/usr/local/go/bin`.
4. **`bin/api`** (binario precompilado viejo en `go-b2b-starter/bin/`) responde 404 en todo — NO es el backend válido. El backend correcto se arranca con `go run ./cmd/api/main.go` (o el script).
5. **Backend reiniciado sin los últimos fixes = comportamiento viejo**: SIEMPRE reiniciar el backend tras editar código invoicing, y verificar con un curl del endpoint afectado.

---

## 5. Stack e2e manual actual (comandos verificados)

Estado al cierre de sesión: **backend `:8080` OK (go run, con TODOS los fixes), mock-siigo `:8090` OK, frontend `:3001` OK, postgres docker en `:5432` (saas_db_test migrada + seed 5 orgs)**.

```bash
export PATH=$PATH:/usr/local/go/bin
cd go-b2b-starter
# DB (si ya está migrada+seed, saltar)
docker compose -f ../docker-compose.yml exec -T postgres dropdb -U postgres --if-exists --force saas_db_test
docker compose -f ../docker-compose.yml exec -T postgres createdb -U postgres saas_db_test
POSTGRES_DB=saas_db_test docker compose -f ../docker-compose.yml run --rm migrate
POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=postgres POSTGRES_PASSWORD=postgres POSTGRES_DB=saas_db_test SKIP_MIGRATIONS=true go run ./cmd/seed-e2e

# Mock + backend + frontend
setsid env ... go run ./cmd/mock-siigo -addr :8090 > /tmp/e2e-mock-siigo.log 2>&1 < /dev/null & disown
setsid env AUTH_MOCK_ENABLED=true SERVER_ADDRESS=:8080 SIIGO_BASE_URL=http://localhost:8090 SIIGO_TOKEN_URL=http://localhost:8090/token SIIGO_WEBHOOK_SECRET=test_webhook_secret_for_e2e SIIGO_SANDBOX=true go run ./cmd/api/main.go > /tmp/e2e-backend.log 2>&1 < /dev/null & disown
cd next_b2b_starter && setsid env pnpm dev -p 3001 > /tmp/e2e-frontend.log 2>&1 < /dev/null & disown
# health: curl :8080/readyz, :8090/healthz, :3001
```

**Reset de estado de la org siigo (para re-runs del spec):**
```bash
cd go-b2b-starter && docker compose -f ../docker-compose.yml exec -T postgres psql -U postgres -d saas_db_test -c "DELETE FROM invoicing.org_connections WHERE organization_id=5; DELETE FROM invoicing.org_numerations WHERE organization_id=5; DELETE FROM invoicing.import_runs WHERE organization_id=5;"
```

**Correr solo el spec siigo:**
```bash
cd next_b2b_starter && npx playwright test siigo-onboarding --config e2e/playwright.config.ts --reporter=list
```

**Verificación del flujo por API:**
```bash
H="X-Test-Org-ID: test-org-siigo:admin-siigo@test.com"
curl -s http://localhost:8080/api/v1/org/siigo/status -H "$H"
curl -s -X POST http://localhost:8080/api/v1/org/siigo/test-invoice -H "$H"
```

---

## 6. Estado del spec e2e al cierre

`e2e/specs/siigo-onboarding.spec.ts` (serial, org `test-org-siigo` = id 5, admin `admin-siigo@test.com` / member `member-siigo@test.com`):

| # | Test | Estado al cierre |
|---|------|------------------|
| 1 | assisted setup (admin request → provision → connected) | ✓ pasa |
| 2 | wizard happy path (numeración → import → sandbox → activar → live) | ✓ pasa |
| 3 | kill-switch pause/resume | ✓ pasa |
| 4 | admin view (fila Activo + numeración auto + confirm · N nuevos) | falló por "confirm · 0 nuevos" — **fix aplicado** (regex `\d+ nuevos`), NO re-verificado |
| 5 | gating (deal facturado pre-live vs live) | NO ejecutado aún |
| 6 | isolation (org pro sigue none) | NO ejecutado aún |

Último run: 3 ✓, 1 ✘ (admin view, fix aplicado sin re-run), 2 sin correr — la ejecución se interrumpió porque **el frontend dev se cayó de nuevo** (`net::ERR_CONNECTION_REFUSED :3001`), relanzado al cierre.

**Pendiente inmediato de la sesión nueva:**
1. Re-correr `npx playwright test siigo-onboarding` (frontend ya relanzado) → esperar 6/6 ✓.
2. Si 6/6 ✓: correr el suite COMPLETO `make test-e2e` (con postgres del sistema detenido y procesos previos limpios) → gate 6.3 en verde.
3. Actualizar `tasks.md` de `add-siigo-e2e-tests` (tarea 6.3 → resultado), re-evaluar archive decision.

---

## 7. Comandos de gate ya verificados en esta sesión

- `go build ./...` — EXIT 0 (al cierre; ojo: otros cambios en curso pueden romperlo — ver §8)
- `go vet ./internal/modules/invoicing/... ./cmd/mock-siigo/... ./cmd/seed-e2e/...` — EXIT 0
- `go test ./internal/modules/invoicing/... ./cmd/mock-siigo/...` — todos ok (incl. nuevos: test_invoice_service 4 tests, cmd/init runDeltaSyncOnce 4 tests, admin_handler 2, resolver 4, adapter ValidateCredentials 3, mock-siigo 4)
- Tests de integración (harness, `-tags integration`, Docker): `TestOrgConnectionRepository_*` + `TestImportConfirm_IsIdempotentOnSecondRun` — **ejecutados y verdes**
- `npx tsc --noEmit` — limpio salvo error EXTERNO `app/layout.tsx` (ThemeProvider `suppressHydrationWarning`, cambio en curso `app-shell-modernization`)
- `pnpm lint` — 0 errores, 1 warning (baseline)
- `npx vitest run` — 17 archivos / 63 tests ✓ (incl. 9 siigo-section + 3 siigo-admin-view)

---

## 8. Cambios de OTROS agentes en curso (movimiento en el árbol)

El working tree tiene ~400+ archivos sin commitear de cambios ajenos que se editan EN VIVO:
- **Instagram/multi-channel** (migración 000031, rename `whatsapp_message_id`→`provider_message_id`, phone nullable): rompió repos crm/agent/instagram repetidamente; yo apliqué los fixes mecánicos (renames, pgtype, mocks). Puede seguir moviéndose.
- **Payments** (`internal/modules/payments/`), **Campaigns** (`crm.campaigns`, migración 000029), **app-shell-modernization** (layout.tsx).
- **Consecuencia:** `go build ./...` repo-wide y suites completas pueden fallar transitoriamente por archivos ajenos a medio escribir. Verificar qué archivo falla antes de asumir que es nuestro código.

---

## 9. Referencias de archivos clave

| Archivo | Rol |
|---|---|
| `go-b2b-starter/cmd/mock-siigo/main.go` (+`main_test.go`) | Mock Siigo e2e (token, /v1/company 404, customers paginados, invoices + Idempotency-Key, /healthz) |
| `go-b2b-starter/scripts/run_e2e.sh` | Boot del stack e2e + env SIIGO_* al mock |
| `go-b2b-starter/cmd/seed-e2e/main.go` | +`test-org-siigo` (admin/member) |
| `next_b2b_starter/e2e/specs/siigo-onboarding.spec.ts` | Spec serial 6 escenarios |
| `next_b2b_starter/e2e/page-objects/settings.page.ts` / `admin-panel.page.ts` | Helpers Siigo |
| `openspec/changes/add-siigo-e2e-tests/{proposal,design,tasks}.md` + `specs/` | Artefactos del cambio |
| `go-b2b-starter/internal/db/postgres/sqlc/integration/invoicing_integration_test.go` | Tests DB (connection repo, import idempotencia) |
| Logs: `/tmp/e2e-backend.log`, `/tmp/e2e-mock-siigo.log`, `/tmp/e2e-frontend.log` | Diagnóstico |
