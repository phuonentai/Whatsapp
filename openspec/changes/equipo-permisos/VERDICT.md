# Verdict: equipo-permisos (revisión 3)

STATUS: REJECTED
MARKET: N/A

## Market Read

Out of scope — sin cambios frente a revisiones previas. La revisión 3 ajusta exclusivamente infraestructura de auth (breaker, consolidación de servicio de política, coherencia de living spec); no toca billing/pricing, metering/LLM cost, canal WhatsApp/Meta, compliance ni funnel. La página sigue siendo una consolidación read-only de settings admin. No se requieren `## Market & Unit Economics` / `## Market Risk`.

## Revisión del veredicto 2 (2 cambios requeridos)

1. **Coherencia "DTOs retained as API contract"** — resuelto. El delta MODIFICA la living requirement (bloque completo, header exacto): eliminación de datos estáticos acotada al data path de producción; fallback dev permitido solo con gate de placeholder-detection; 3 escenarios (incl. el original "API response format unchanged"). Sin contradicción remanente al archivar.
2. **Cobertura de breaker + consolidación** — resuelto en dirección y contrato: decisión 1 (fetch vía `platform/stytch.Client.Run`, 5/10s/2), decisión 2 (única implementación servida, `auth:stytch:rbac:policy:v2`), escenarios de delta ("Fetch de política con circuit breaker", "Un único servicio de política servido"), tasks 1.2 (wiring + breaker con fallo simulado) y 1.4 (retiro del duplicado). **Pero** la premisa de consumidores de la decisión 2/task 1.4 es inexacta (ver hallazgo SRE #1 abajo) y hay un defecto de numeración (ver hallazgo #2).

## Persona Findings

### 1. Staff Security Engineer

- **[INFO — good]** Sin escritura de política; 403 sin datos; gates `org:manage` (vista) y `audit:view` (ledger inline, sin fetch); matriz sin datos por-org (política project-wide por diseño); sin secretos nuevos. El fallback estático queda acotado a dev/placeholders y el delta cierra la contradicción con "DTOs retained as API contract".
- **[INFO — residual]** Shape de roles: `normalizeRoleID` quita el prefijo `stytch_`; si la política solo define roles por defecto, la matriz muestra 2 roles (display-only; gating permission-driven). Sin cambio requerido.

### 2. Staff DBA

- **[PASS]** Sin migraciones, sin SQLC, sin escrituras; el wiring no toca esquema; lecturas reutilizan queries existentes; sin N+1. Sin cambios.

### 3. Staff SRE

- **[LOW-MED] Premisa de consumidores inexacta en decisión 2 y task 1.4.** Verificado en código: `internal/platform/stytch/rbac_policy.go` (`RBACPolicyService`, cache key `stytch:rbac:policy`, retorno `[]string`) NO tiene consumidores — solo su propia provisión DI (`inject.go`, `cmd/provider.go`, `bootstrap/init_mods.go`). El repositorio citado como consumidor (`organizations/.../stytch_role_repository.go`) consume el cliente con breaker `*stytch.Client` (llama él mismo al endpoint de política, línea "stytch fetch rbac policy") y NO el servicio duplicado. La dirección de la consolidación es correcta (el servicio de modules/auth es el único consumido por `StytchRBACService`/`token_verifier`/`StytchAuthAdapter.PolicyService()`), pero el trabajo real es: confirmar cero consumidores por grep, retirar el duplicado + su provisión DI + cache key, conservar el `Client` con breaker (lo usa el org repo y ahora el fetch del servicio consolidado vía `Run`). **Requerido:** corregir la premisa en decisión 2 y task 1.4.
- **[GOOD]** Breaker especificado con umbral/tiempo/half-open y fallo simulado en verificación; contrato 503 preservado; rollback trivial; log en fallo.

### 4. Staff Product/GTM

- **[PASS / N/A]** Consolidación admin read-only; sin palancas de pricing/IA/canal/funnel. Sin análisis de unit economics requerido.

### 5. Colombia IT & Market

- **[PASS / N/A]** Sin PII nueva, sin transferencias, sin DIAN, sin superficie WhatsApp Business, sin retenciones nuevas. Postura Ley 1581/Habeas Data sin cambios.

## Required design changes

1. **Corregir la premisa de consumidores de la consolidación (decisión 2 y task 1.4).** Verificado: el duplicado `platform/stytch` `RBACPolicyService` (cache key `stytch:rbac:policy`) no tiene consumidores — solo su provisión DI; `stytch_role_repository.go` usa el `*stytch.Client` con breaker, no el servicio duplicado. La decisión y la task SHALL reformularse: (a) grep para confirmar cero consumidores del tipo duplicado, (b) retirar el duplicado + su provisión DI (`inject.go`/`cmd/provider.go`) + cache key viejo, (c) conservar el `Client` con breaker (usado por el org repo y por el fetch del servicio consolidado vía `Client.Run`). El texto actual ("migrar sus consumidores... confirmado: stytch_role_repository.go") es factualmente incorrecto y llevaría a trabajo innecesario o a un retiro incompleto.
2. **Renumerar las decisiones de design.md.** Existen DOS "Decisión 2" (consolidación y propagación de descripciones), lo que hace ambiguas las referencias cruzadas ("Decisiones 2+5" en la tabla de compliance, "(decisión 2)" en la decisión de metadata) y el desfase del resto. SHALL renumerarse secuencialmente (propagación pasa a 3 y así sucesivamente) y corregir la referencia interna en la decisión de consolidación ("task 1.2" → "task 1.4").

Ambos son correcciones de una pasada; la arquitectura (FE + wiring mínimo, SSOT Stytch, breaker, una sola implementación servida) queda validada y sin cambios.
