# Design: Restyle del panel según template (dashboard.html + messages-view.html + onboarding.html)

## D1. Referencias (extraídas del template)

- **dashboard.html** (588 líneas): top bar oscura (logo, búsqueda ⌘K, notificaciones, usuario con plan), sidebar `w-64 bg-slate-900` con 6 ítems + grupo "Inteligencia Artificial" + tarjeta "IA Insights" (gradiente emerald, CTA "Ver Sugerencias"); main `bg-slate-50`: saludo + selector de rango + botón "Nueva Conversación", 4 KPIs (`bg-white rounded-2xl p-6 border-slate-200` con icono tintado + delta emerald), gráfico de barras (emerald-200/500 por día), panel "Copiloto IA" oscuro (`bg-gradient-to-br from-slate-900 to-slate-800`) con 3 insights (emerald/amber/blue), tarjetas de conversaciones recientes.
- **messages-view.html** (305 líneas): header ("Mensajes" + subtítulo + "Conectado" + "Nueva campaña"), 4 tarjetas de métricas (Conversaciones hoy 247 / Por responder 18 / Tasa de respuesta 94% / Tiempo promedio 4.2m), toolbar (búsqueda + selects estado/agente), lista `divide-y` con avatares (initials), etiqueta Cliente/Prospecto, snippet, hora, badge de no leídos.
- **onboarding.html** (274 líneas): wizard oscuro multi-paso (`step-page`), barra de progreso, botones `step-prev` (outline) / `step-next` (emerald), tarjetas de tipo de producto `product-type-btn` (comercio/restaurantes/…).

## D2. Arquitectura de cambio (UI-only)

```
components/layout/                     # shell existente — restyle
  dashboard-layout.tsx                 # sidebar oscura + top bar (template D1)
  sidebar.tsx / header.tsx / user-menu.tsx  # adaptados al template
app/dashboard/
  page.tsx                             # overview restilizado (NO es un redirect; ver D2.1)
  inbox/…                              # restyle messages-view sobre la lógica existente
app/signup/…                           # restyle onboarding (wizard oscuro, 3 pasos actuales)
app/not-found.tsx                      # 404 con branding (ya hecho en change marketing)
```

- **Contratos mantenidos**: rutas existentes, permisos RBAC (`usePermissions`), repos/queries TanStack, `use-signup-flow.ts`, componentes Stytch (`StytchProvider`/magic link), copia `lib/copy/ui.ts` (namespaces actuales + claves nuevas es/en).
- **Paleta**: utilidades explícitas `slate-*`/`emerald-*` en el shell/inbox (igual que el sitio de marketing); tokens shadcn (`bg-card`, `bg-primary`) se conservan para el resto de páginas. Tema oscuro (next-themes) sigue funcionando: sidebar/top bar fijas oscuras, contenido respeta el tema.

### D2.1 Contrato crítico preservado — verificación de parámetros de pago en `app/dashboard/page.tsx`

Verificado en el repo: `app/dashboard/page.tsx` es un server component que (1) verifica parámetros de retorno de pago ANTES de renderizar el home y (2) renderiza `<DashboardHome />`:

- `checkout_id` (Polar) → `verifyPayment(checkoutId)` → redirect `/dashboard/settings?view=subscription&payment_verified=true` (éxito) o `payment_error=true` (fallo).
- `payment_id`/`preapproval_id` (MercadoPago): `preapproval_id` sin `payment_id` → redirect `/dashboard/settings?view=subscription` (settlement vía webhook); `payment_id` → `verifyMercadoPagoPayment({ paymentId })` → redirect con `payment_verified=true` o `payment_error=true`.

**El restyle NO debe tocar estas ramas.** El overview del template se compone sobre `DashboardHome` (que ya tiene tarjetas KPI alimentadas por `useConversationsQuery` + queries CRM, quick actions, `FirstRunChecklist`, `AssistantIntro`): reutilizar/restilizar ese contenido, no reemplazarlo ni borrar su lógica. Cualquier cambio de presentación ocurre después de la verificación de pagos.

### D2.2 Navegación del template con RBAC existente

El sidebar actual ya filtra ítems por `ServerPermissions`/entitlement (`permissions.permissions.includes(item.permission)`). Los grupos nuevos del template (incluido "Inteligencia Artificial": Copiloto IA, Entrenamiento, Automatizaciones y la tarjeta "IA Insights") SHALL enlazarse al modelo existente de ítems con permiso (o entitlement): un ítem sin ruta real o sin permiso no se renderiza (o se renderiza deshabilitado con CTA a configurar el asistente si aplica). Prohibido enlazar incondicionalmente secciones gated o inexistentes.

## D3. Overview con datos reales

- KPIs mapeados a las queries que YA consume `DashboardHome` (sin fan-out nuevo):
  - Conversaciones activas → `useConversationsQuery` (inbox store/queries)
  - Ventas semana → analytics/reportes (COP) si existe endpoint; si no, "—"
  - Facturas emitidas → invoicing/billing si existe; si no, "—"
  - Tiempo respuesta IA → métricas de agente/cognitive si existen; si no, "—"
- Gráfico de barras: renderizar con datos del módulo analytics si existe; si no, estado vacío con CTA a reportes (nunca cifras inventadas).
- Panel "Copiloto IA": reutilizar insights existentes (si el módulo agent expone sugerencias); si no, versión estática orientativa con CTA a configurar el asistente.

## D4. Decisiones de implementación

- El restyle es **presentacional**: cada tarea conserva la lógica de datos actual y solo recompone layout/estilos. Los e2e existentes (`inbox-ui.spec.ts`, `auth-passwordless.spec.ts`, etc.) deben pasar; si un page-object depende de clases o textos, se actualiza el page-object (no la lógica).
- Si algún KPI no tiene fuente de datos en el repo, se documenta en tasks.md y se deja "—" (regla del spec).
- **Verificación de pagos**: el gate añade un paso manual/e2e para los retornos de checkout (Polar `checkout_id`, MercadoPago `payment_id`/`preapproval_id`) confirmando que aterrizan en `/dashboard/settings?view=subscription` con `payment_verified=true` / `payment_error=true` según el caso (ver tasks 5.1).
- **Delta spec `app-shell`**: la requirement de dark mode se ajusta para el shell fijo oscuro del template (las superficies de shell usan utilidades slate/emerald explícitas; el contenido conserva tokens de tema y el toggle sigue operativo) y la requirement de dashboard home se extiende con el set de KPIs del template y la regla "—" — registrado como delta MODIFIED para mantener coherencia spec↔código.
