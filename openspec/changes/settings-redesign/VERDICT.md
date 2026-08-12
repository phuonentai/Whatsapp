# Council Verdict — settings-redesign

STATUS: APPROVED
MARKET: PASS

Fecha de revisión: fresh council (sin `revision.md` ni `VERDICT.md` previos).

## Resumen

Change frontend-only de re-estilo visual de los 10 módulos de settings + overview al lenguaje del diseño (slate-50/white, bordes slate-200, chips semánticos con texto, jerarquía tipográfica). Verificado contra el código: las 10 vistas `?view=` existen en `app/dashboard/settings/components/settings-content.tsx` (`SettingsView` incluye profile|subscription|modules|compliance|audit|whatsapp|templates|instagram|siigo|siigo-admin|overview); `useAiUsageQuery` existe y ya se consume en `subscription-tab.tsx`; `AiUsageDto` porta `credits_used`/`credits_max`/`credits_remaining`; `state?.usage` ya computa `included`/`used`/`usagePercent` en `subscription-tab.tsx`; `lib/copy/ui.ts` existe; 7 suites vitest de settings existen; `gray-*` hardcodeadas presentes en 18 archivos del directorio; compliance-section ya tiene consent/export/anonymize. Las premisas del design/proposal son verificables y verificadas. Market-in-scope (`requires_market_read: true`; delta sobre `billing-provider-ux` + visibilidad de uso = billing/quotas/credits; adyacencia WhatsApp/Meta); ambas secciones requeridas (`## Market & Unit Economics`, `## Market Risk`) están presentes con owners y triggers nombrados.

## Market Read

- **Sin delta de economía unitaria.** Change presentacional: no añade llamadas LLM, metering, ni routing nuevo; no toca `paywall`, `plan-pricing-ux` ni `billing-quota-integrity`; la lógica de cancelación/resumen/cambio de plan en `subscription-tab.tsx` queda intacta. Costo por acción IA, precios (COP) y márgenes por plan no se alteran. El design lo declara explícitamente y el código lo confirma (los datos de uso ya están computados hoy).
- **Superficie de riesgo de límites (churn/upgrade).** Hacer visibles las barras de uso con umbral amber ≥80% puede alertar a clientes de límites previamente ignorados → tickets de soporte o presión de upgrade. Riesgo aceptado y registrado (R1: owner product/GTM, trigger definido, mitigación: umbral + copy "ver plan" + estado neutro "—" sin fabricar porcentajes). Umbral v1 fijo 80% documentado como decisión con pregunta abierta (configurable por plan en follow-up) — residual aceptable.
- **Adyacencia de canal WhatsApp/Meta (riesgo existencial de canal).** Los chips de estado no deben afirmar estados de plataforma Meta que el backend no confirme. Riesgo registrado (R2: owner product/ops, trigger: cambio de términos de Meta o copy que afirme estado sin confirmación; mitigación: chips alimentados de las queries de conexión existentes — verificado: `signupStatusQuery.data?.status` con estados connected/failed/exchanging/registering/verifying). Los tokens Meta permanecen en el panel avanzado colapsado; sin nuevo manejo cliente-side de secretos.
- **Adyacencia Ley 1581.** compliance-section se re-estiliza sin tocar consentimiento/exportación/anonimización (traceability intacta). Sin cambios de retención de datos ni de flujo de transferencia/forget.
- **Adyacencia facturación DIAN/Siigo.** Wizard Siigo 5 pasos re-estilizado con `STEP_ORDER` intacto; sin cambios de lógica de facturación electrónica.
- **Sin premisas de mercado no verificadas.** No se asevera ningún hecho externo como premisa (precios de proveedores, postura regulatoria, claims de competidores). La expectativa de reducción de churn se califica como presentacional y sin métrica nueva (deferido a follow-up de analytics) — honesto.
- **Sustitución competitiva:** no implicada por un change visual; el riesgo residual de canal (Meta native AI / drift de políticas WhatsApp Business) queda cubierto por el trigger de R2.

## Hallazgos por persona

### 1. Staff Security Engineer — sin bloqueos (severity: LOW)
- **S1 (LOW, residual):** Re-estilo no toca auth/RBAC/contratos Stytch/Polar/Siigo/Meta; las vistas `siigo-admin` conservan su gate admin. No hay webhooks en scope (no aplica idempotencia webhook). No se almacenan credenciales localmente (tokens Meta/Siigo siguen en backend). **Residual:** durante 2.6, conservar semántica de los campos de token (placeholder "Leave blank to keep current", sin renderizar secretos en claro) al re-estilizar el panel avanzado. Sin cambio de diseño requerido.

### 2. Staff DBA — sin hallazgos (N/A)
- Sin DB, sin migraciones, sin SQLC, sin índices. Las barras reutilizan `useAiUsageQuery` (ya cableado, staleTime 1 min / gcTime 5 min) y `state?.usage` ya computado — sin queries nuevas, sin N+1. Verificación de la premisa "datos ya disponibles": confirmada en `subscription-tab.tsx` (líneas 98-101).

### 3. Staff SRE — sin bloqueos (severity: LOW)
- **SRE1 (LOW, residual):** Sin endpoints nuevos, sin locks, sin idempotency keys necesarias; rollback = git revert sin estado de DB/Stytch (correcto y suficiente). **Residual:** la hipótesis "mejor comprensión de límites reduce churn" no se mide en este change; el design lo difiere explícitamente a follow-up de analytics — aceptado, pero el negocio debe presupuestar esa medición como follow-up para validar la inversión visual en subscription.

### 4. Staff Product/GTM — sin bloqueos (severity: LOW)
- **P1 (LOW, residual):** Umbral amber fijo 80% para todos los planes es una decisión v1 documentada; la pregunta abierta de configuración por plan debe quedar registrada en `tasks.md` (el design dice "se documenta") y no debe convertirse en sorpresa de plan para planes con límites más bajos. **Residual:** riesgo R1 (presión de upgrade/tickets) con owner product/GTM y trigger definido. Coherencia de pricing intacta (sin cambios de gating por feature ni de fuentes de plan en modules).

### 5. Colombia IT & Market — sin bloqueos (severity: LOW)
- **C1 (LOW, residual):** Re-estilo de compliance (Ley 1581) sin cambio de consentimiento/exportación/anonimización — correcto. Wizard Siigo intacto — sin cambio de lógica de facturación DIAN. Riesgo de canal WhatsApp Business (drift de política, AI messaging nativo, aprobación de templates) como superficie existencial queda cubierto por R2 con trigger explícito. Sin hallazgo de requerimiento de diseño.

## Conclusión

No se identifican defectos REJECT-level. Las secciones `## Market & Unit Economics` y `## Market Risk` están presentes para un change market-in-scope; los riesgos residuales (R1, R2) tienen owner y trigger nombrados y son compatibles con `MARKET: PASS`. Premisas verificadas contra el código; sin hechos externos aseverados como premisas. Se aprueba con los residuales arriba anotados (todos de severidad LOW, sin cambios de diseño requeridos).
