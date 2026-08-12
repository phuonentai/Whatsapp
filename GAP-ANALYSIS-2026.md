# Análisis de brechas: NexoChat como SaaS empresarial moderno 2026

**Fecha:** 2026-08-11 · **Alcance:** monorepo completo (go-b2b-starter + next_b2b_starter + sitio de marketing nuevo)
**Método:** evidencia del repositorio (recon `website/RECON-CODE.md`, specs `openspec/specs/`, configuraciones, dependencias). `[EVIDENCIA]` = verificado en el repo; `[INFERENCIA]` = deducción razonada.

## Resumen ejecutivo

La base es sólida (spec-driven, Stytch B2B, RBAC, billing Polar/MercadoPago, 18 módulos backend, CI con 5 jobs, e2e Playwright, copy español-first). Las brechas principales para "enterprise SaaS 2026":

| # | Área | Brecha clave | Severidad |
|---|------|--------------|-----------|
| 1 | Confianza/Trust | Sin página de estado, sin changelog, sin portal de docs renderizado, OG images ausentes | Alta |
| 2 | Observabilidad | Sin telemetría frontend; backend sin métricas/alertas centralizadas visibles | Alta |
| 3 | Identidad empresarial | Sin SSO/SAML/SCIM ni políticas de sesión avanzadas | Media |
| 4 | Cumplimiento | Sin página de cookies/consentimiento, DPA, subprocesadores, SOC 2 roadmap | Media |
| 5 | Plataforma dev | Sin API keys públicas, webhooks salientes, sandbox, versionado de API | Media |
| 6 | Onboarding/activación | Signup existente no usa el diseño del template (onboarding.html) | Media |
| 7 | Marketing 2026 | Sin analytics/consent, sin A/B, sin i18n de rutas, sin manifest PWA | Media |
| 8 | Contenido | Blog/academia nuevos; docs/ sin renderizar, sin help center | Media |

---

## 1. Sitio de marketing (en construcción en este change)

### Hecho [EVIDENCIA]
- Home, features, pricing, blog (+RSS), academia, páginas legales en `app/(marketing)/`
- Design system del template (slate-900 + emerald-500, Inter + Sora), framer-motion, shadcn reutilizado
- SEO: sitemap, JSON-LD (BlogPosting, Course, FAQPage, Offer), metadata por ruta

### Brechas
- **OG/Twitter images**: `next.config.ts` referencia `/opengraph-image.png`, `/twitter-image.png`, `/screenshot.webp` que NO existen (`public/` solo tiene `icon.png`) [EVIDENCIA]. Hay `scripts/generate-og-images.js` sin correr. → Generar y cachear.
- **`metadataBase`**: `https://yourdomain.com` hardcodeado en root layout y sitemap [EVIDENCIA]. → Env `NEXT_PUBLIC_SITE_URL`.
- **Analytics + consentimiento de cookies**: cero dependencias de analytics en frontend (sentry/posthog/plausible/gtag = 0 matches) [EVIDENCIA]. → Plausible/PostHog + banner de cookies (Ley 1581 / RGPD), sin trackear antes del consentimiento.
- **Página de estado del servicio**: el footer enlaza "Estado del servicio" → `/security` (placeholder). → Instancia de statuspage (BetterStack/Incident.io) + link real.
- **Changelog**: no existe. → `/changelog` alimentado por releases de GitHub.
- **Portal de docs**: `next_b2b_starter/docs/*.md` (11 archivos de dev docs) NO se renderizan como sitio [EVIDENCIA]. → `/docs` con Mintlify/Docusaurus o rutas Next + mdx.
- **i18n de rutas**: copy tiene es/en pero no hay rutas `/en` [EVIDENCIA de `lib/copy/ui.ts`]. → `[locale]` o subdominio cuando se lance inglés.
- **PWA/manifest**: template promete "App Móvil PWA" pero no hay `manifest.ts` ni service worker [EVIDENCIA: `public/` solo icon.png]. → manifest + iconos + (opcional) SW.
- **Formulario de contacto/lead**: CTAs van a `/signup`; sin captura de leads de marketing (newsletter, demo request). → formulario con server action (sin DB nueva: tabla de leads o mailto).
- **A11y/perf presupuestos**: sin `@tailwindcss/typography` (prose hecho a mano — ok), sin audit de WCAG ni Core Web Vitals presupuestos. → `@axe-core` en CI e2e + budgets en next.config.
- **A/B testing / personalización**: nada. → PostHog experiments cuando haya tráfico.

## 2. Aplicación (dashboard)

### Estado [EVIDENCIA]
- Copy español-first tipado (`lib/copy/ui.ts`, 10 namespaces + `marketing` nuevo)
- Bandeja multi-canal, CRM, campañas, knowledge base, reportes, settings — rutas reales en `app/dashboard/*`
- Comando `⌘K`, atajos, RBAC por permisos, e2e Playwright (10+ specs)
- not-found.tsx REDIRIGE a /dashboard o / (no es un 404 real) [EVIDENCIA]

### Brechas
- **404 real**: `app/not-found.tsx` redirige; un SaaS empresarial necesita 404/500 con branding y logging. [EVIDENCIA]
- **Onboarding**: el signup 3-pasos existe (account→organization→business) pero no usa el diseño de `onboarding.html` (wizard oscuro multi-paso con progreso y product-type) ni `onboarding-info.html` (checklist). → Restyling del flujo `/signup` + checklist de primeros pasos con diseño del template; mantener contratos Stytch intactos (gate de gobernanza).
- **Empty states / activación**: verificar estados vacíos en bandeja/CRM (desconocido). → Inventario de empty states + guías inline.
- **Tema**: next-themes presente con dark; el marketing es fijo claro/oscuro por secciones. → Decidir dark mode del dashboard coherente con emerald.
- **Rendimiento**: listas virtualizadas (`@tanstack/react-virtual`) presentes; sin auditar LCP/INP del dashboard. → budgets.
- **i18n runtime**: `copy("namespace", key)` es compile-time es/en; sin selector de idioma en la app. → feature flag de idioma.

## 3. Plataforma enterprise (backend + auth + billing)

### Estado [EVIDENCIA]
- Stytch B2B passwordless, RBAC, multi-tenancy por organización; JWKS + circuit breakers; webhooks firmados (ingress)
- Billing Polar + MercadoPago (PSE/Nequi/tarjetas COP); paywall, feature gating, ai-usage-metering
- Audit log (`admin-panel-audit-log` spec), backup/recovery (`data-backup-recovery` + job `backup-drill` en CI), governance workflow, 18 módulos, migraciones SQLC, swagger target en Makefile
- CI: backend, frontend, spec-validation, backup-drill [EVIDENCIA `.github/workflows/ci.yml`]

### Brechas
- **SSO/SAML/SCIM**: no hay spec (ls de specs no muestra sso/saml/scim) [EVIDENCIA]. → SSO SAML para Enterprise (Stytch lo soporta), SCIM para aprovisionamiento.
- **Audit log**: existe spec `admin-panel-audit-log` — verificar exportación SIEM/API y retención. → Export CSV/API + retención configurable.
- **Telemetría**: go.mod tiene 11 matches de sentry/otel/prometheus/grafana (verificar cuáles están activos; no hay pkg `observability/`) [EVIDENCIA parcial]. → OTel traces + métricas, alertas, SLOs, error tracking frontend (Sentry).
- **API pública para clientes**: docs `05-making-api-requests.md` describen API REST pero no hay API keys por organización ni webhooks salientes con firma. → API keys + scopes, webhooks out (HMAC), rate limits publicados, versionado `/v2`.
- **Sandbox/demo**: sin entorno sandbox con datos de ejemplo. → Sandbox con seed (`seed-e2e` existe).
- **Rate limiting/abuso**: desconocido; el template tiene middleware de sesión. → Limiter por org+IP en API y en webhook ingress.
- **Secrets/entornos**: `app.env` + `app.env.bak` en el repo de go [EVIDENCIA — riesgo]. → vault/SOPS, rotación, .env.bak fuera de git.
- **Resiliencia**: event bus en proceso (doc `event-bus.md`); sin cola externa (solo 1 match redis/kafka/nats en go.mod — verificar) [INFERENCIA]. → Cola (Redis Streams/SQS) para campañas y webhooks si el volumen crece.
- **Escala DB**: pgvector presente; sin particionado/replicas visibles. → Política de retención de mensajes, archivo.

## 4. Cumplimiento y confianza

### Estado [EVIDENCIA]
- Ley 1581 (consentimiento, Habeas Data export/forget, PII masking) en specs; páginas de privacidad/términos/seguridad nuevas
- CSP, HSTS, X-Frame-Options, Referrer-Policy, Permissions-Policy en headers [EVIDENCIA next.config]
- Stytch como única autoridad de identidad (sin credenciales locales)

### Brechas
- **Consentimiento de cookies en el sitio**: necesario con analytics. → Banner + registro de consentimiento.
- **DPA / subprocesadores**: sin página de DPA ni lista de subprocesadores (Stytch, Polar, MercadoPago, Meta, proveedor IA). → Página de cumplimiento.
- **Certificaciones**: sin SOC 2 / ISO 27001 roadmap. → Roadmap + página de confianza con postura.
- **Vulnerability disclosure**: sin SECURITY.md público ni proceso. → SECURITY.md + bug bounty opcional.
- **Retención y borrado**: export/forget existe para datos personales; verificar borrado completo de mensajes/whatsapp. → E2E de borrado.

## 5. Operaciones y entrega

### Estado [EVIDENCIA]
- Dockerfile (frontend standalone), Makefile (migraciones, sqlc, server, test), CI con backup-drill
- E2E Playwright con page-objects y fixtures; vitest para unit; lint

### Brechas
- **Entornos**: sin staging/preview visibles. → Preview por PR (Vercel) + staging.
- **Deploy automatizado**: CI construye; despliegue manual aparente. → CD con rollback automático + blue/green.
- **Monitoreo de errores**: sin Sentry frontend. → Sentry + alertas.
- **Incidentes**: sin runbooks/SLOs. → catálogo de servicios, SLOs, runbooks.
- **Costos**: sin gobernanza de costos IA (ai-usage-metering existe — extender a dashboards de costos por org).

## 6. Contenido y educación

### Estado [EVIDENCIA]
- Blog (5 posts) y academia (3 cursos × 4-6 lecciones) nuevos, contenido español basado en specs
- docs/ markdown dev (11 archivos)

### Brechas
- **Help center / FAQ amplio**: FAQ de 6 ítems; → base de conocimiento de soporte.
- **Changelog** (ver §1).
- **Webinars/casos de éxito**: sin testimonios reales (el template los sugiere). → Sección de clientes cuando existan.
- **Videos/demo**: "Demo en vivo" es CTA a /features; → video de producto o tour interactivo.

## 7. Hoja de ruta sugerida (priorizada)

**Fase 1 — Confianza y medición (2-4 semanas)**
1. OG/twitter images + `NEXT_PUBLIC_SITE_URL` (corregir metadataBase/sitemap)
2. Analytics con consentimiento (Plausible/PostHog + banner cookies)
3. Sentry frontend + error tracking
4. 404/500 reales con branding
5. Página de estado del servicio

**Fase 2 — DevEx y plataforma (4-8 semanas)**
6. Portal de docs renderizado (/docs desde `docs/*.md`)
7. API keys por organización + webhooks salientes firmados + rate limits publicados
8. Changelog alimentado por releases

**Fase 3 — Enterprise (8-16 semanas)**
9. SSO SAML + SCIM (Stytch)
10. DPA/subprocesadores + roadmap SOC 2
11. Sandbox con seed demo
12. SECURITY.md + disclosure

**Fase 4 — Crecimiento (continuo)**
13. i18n de rutas (/en)
14. Onboarding rediseñado con onboarding.html + checklist (onboarding-info.html)
15. A/B testing, testimonios, casos de éxito
16. PWA manifest + (opcional) offline

## Riesgos que requieren decisión
- **Orden de prioridad** de las fases (¿confianza primero o plataforma primero?)
- **Presupuesto de herramientas**: PostHog vs Plausible, BetterStack vs statuspage self-hosted, Mintlify vs Docusaurus
- **Página de estado**: ¿interna (SaaS) o autogestionada en el repo?
- **Idioma de la app**: ¿español-first definitivo o i18n completo en 2026?
