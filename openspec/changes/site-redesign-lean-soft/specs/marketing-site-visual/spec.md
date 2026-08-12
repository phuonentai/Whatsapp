# marketing-site-visual Specification

## Purpose

Presentación visual del sitio público de marketing (`app/(marketing)/` + `components/marketing/*`): paleta empresarial suave (neutros cálidos, azul corporativo apagado, verde salvia), composición ligera (hero, features, pricing, FAQ, footer), tipografía Inter-only y motion restringido. Los cambios son de presentación: rutas, copy, SEO-estructura y comportamiento de componentes interactivos (pricing toggle, ROI slider, FAQ accordion) permanecen intactos.

## ADDED Requirements

### Requirement: Paleta empresarial suave en todo el sitio público

El sitio público SHALL usar la paleta empresarial suave definida por los tokens de tema: superficies claras (`background`/`card`/`muted`), texto carbón suave (`foreground`/`muted-foreground`), primario azul corporativo apagado (`primary`) y acento secundario verde salvia suave (`secondary`). Las secciones oscuras `slate-900` y el emerald saturado (`emerald-500`) SHALL NOT usarse en superficies principales ni botones primarios. Los textos de cuerpo SHALL cumplir contraste AA (≥4.5:1) sobre sus superficies.

#### Scenario: Hero con superficie clara

- **WHEN** un visitante abre la homepage
- **THEN** el hero SHALL renderizarse sobre superficie clara sin gradientes oscuros de fondo
- **AND** el badge, el título y los CTAs SHALL usar los tokens de tema (primario azul suave)

#### Scenario: Botones primarios azules suaves

- **WHEN** se renderiza un CTA primario en cualquier página pública
- **THEN** el botón SHALL usar `primary`/`primary-foreground` (azul corporativo apagado) sin sombras emerald ni `hover:scale`

#### Scenario: Contraste AA en texto

- **WHEN** un texto de cuerpo se renderiza sobre una superficie clara
- **THEN** la combinación SHALL cumplir contraste AA (≥4.5:1 para texto normal)

### Requirement: Composición ligera de secciones

Las secciones del sitio público SHALL presentar una composición ligera: padding vertical acotado (`py-16`–`py-24`), alternancia de superficies `card`/`muted` con bordes sutiles en lugar de bloques oscuros, sin sombras grandes decorativas y sin gradientes de fondo pesados. La estructura de secciones (hero, logo strip, feature grid, comparison, pricing, FAQ, CTA, footer) y sus rutas SHALL permanecer sin cambios.

#### Scenario: Secciones alternadas sin bloques oscuros

- **WHEN** un visitante hace scroll por la landing
- **THEN** las secciones SHALL alternar superficies claras suaves con bordes sutiles
- **AND** ninguna sección SHALL renderizar un bloque `bg-slate-900`

#### Scenario: Peso visual reducido

- **WHEN** se compara el hero actual con el rediseñado
- **THEN** el rediseño SHALL tener menor peso visual: sin `animate-pulse` decorativo, sin sombras `shadow-emerald-*`, con tipografía de display una talla menor (máx. `text-5xl` en hero)

### Requirement: Tipografía Inter-only

El sitio público SHALL usar la familia Inter (variable ya cargada en `app/layout.tsx`) para títulos y cuerpo. La fuente display Sora SHALL retirarse de `components/marketing/fonts.ts` y de `next/font/google`. El alias `font-heading` SHALL resolverse a Inter.

#### Scenario: Títulos renderizan con Inter

- **WHEN** un encabezado con `font-heading` se renderiza en una página pública
- **THEN** la fuente aplicada SHALL ser Inter (misma familia que el cuerpo)
- **AND** el paquete de fuentes descargado SHALL NO incluir Sora

### Requirement: Motion restringido

Las animaciones del sitio público SHALL limitarse a fade-up sutil (≤300ms) sin scroll-jacking ni animaciones de entrada que bloqueen LCP. El hero SHALL renderizarse sin animación de entrada. Las animaciones SHALL respetar `prefers-reduced-motion` (desactivadas).

#### Scenario: Hero sin animación de entrada

- **WHEN** la homepage carga
- **THEN** el contenido del hero SHALL estar visible sin animación de entrada
- **AND** el LCP no SHALL bloquearse por motion

#### Scenario: Reduced motion respetado

- **WHEN** el sistema del usuario indica `prefers-reduced-motion: reduce`
- **THEN** las animaciones de revelado SHALL desactivarse

### Requirement: Comportamiento interactivo intacto

Los componentes interactivos (pricing con toggle mensual/anual, ROI slider calculator, FAQ accordion, filtros de blog) SHALL conservar su comportamiento y su lógica; el rediseño SHALL aplicar solo clases/estilos. Las rutas, metadata y structured data de las páginas públicas SHALL NO cambiar.

#### Scenario: Toggle de pricing funcional

- **WHEN** un visitante alterna mensual/anual en `/pricing`
- **THEN** los precios SHALL actualizarse según la lógica existente con el nuevo estilo

#### Scenario: Rutas y metadata intactas

- **WHEN** se navega a cualquier ruta pública
- **THEN** la ruta y su metadata/SEO-estructura SHALL ser las existentes (sin cambios)
