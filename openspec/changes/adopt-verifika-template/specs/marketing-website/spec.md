# marketing-website Delta Spec

## MODIFIED Requirements

### Requirement: Verifika template design system

Marketing pages SHALL use the approved Verifika/ChatFlow template design: dark `slate-900` hero, nav and footer with `emerald-500` accent, Sora headings (`font-heading`) plus Inter body, `max-w-7xl` containers, and the Verifika section compositions: nav (logo, enlaces `#caracteristicas`/`#precios`/`#faq`, CTA "Iniciar sesión"), hero with integrations strip (Meta WhatsApp, Siigo, MercadoPago, Nequi, Efecty), features section ("Integración Oficial Meta & Siigo"), comparison "El Proceso Tradicional vs ChatFlow IA" (two-column before/after), stats strip (conversaciones diarias, horas perdidas, COP en ventas perdidas), pricing cards, FAQ accordion, and footer. The existing shadcn theme tokens (CSS variables) SHALL NOT be altered for the app shell; marketing components SHALL use explicit Tailwind utilities consistent with the Verifika identity (see `verifika-visual-identity`).

#### Scenario: Landing renders Verifika composition

- **WHEN** the homepage renders
- **THEN** it SHALL include the dark hero with integrations strip, the features section, the "Proceso Tradicional vs ChatFlow IA" comparison, the stats strip, pricing, FAQ and footer in the Verifika template structure

#### Scenario: App shell tokens untouched

- **WHEN** the marketing site is built
- **THEN** `app/globals.css` CSS variables and `tailwind.config.ts` color tokens SHALL remain the theme token set of the app (chrome oscuro definido por la identidad Verifika), with the heading font family added

## ADDED Requirements

### Requirement: Onboarding-info page

The system SHALL serve a public marketing page at `/onboarding-info` explaining the product onboarding process: overview of steps and requirements ("Pasos" / "Requisitos"), activation promise ("Activación en menos de 24 horas", "Tiempo promedio de activación"), the step-by-step process with live-session accompaniment, an interactive readiness checklist ("Antes de empezar" — NIT y RUT, WhatsApp Business, etc.), a typical schedule (Día 1 kickoff y conexión de cuentas, Día 2, ...), and an FAQ (duration, cost, number/message portability, non-technical teams). The page SHALL export `generateMetadata` (title `%s | NexoChat`, description, OG) and SHALL be listed in `app/sitemap.ts`. All strings SHALL resolve from the Spanish-first copy layer.

#### Scenario: Page renders at /onboarding-info

- **WHEN** a visitor requests `/onboarding-info`
- **THEN** the route SHALL render HTTP 200 with the marketing shell and the onboarding-info sections in the Verifika template language

#### Scenario: Metadata and sitemap

- **WHEN** the page renders
- **THEN** the document SHALL include title, description and OG metadata
- **AND** `/sitemap.xml` SHALL include `/onboarding-info`
