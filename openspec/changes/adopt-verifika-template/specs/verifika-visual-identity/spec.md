# verifika-visual-identity Specification

## Purpose

Define la identidad visual adoptada de la plantilla Verifika/ChatFlow CRM (export Shuffle): chrome oscuro `slate-900`/`slate-800`, acentos `emerald-500`, superficies de contenido claras `slate-50`/`white`, tipografía display + Inter, e idioms de componente (botones, inputs, tarjetas KPI, badges, navegación). Gobierna todas las superficies — marketing y producto — y reemplaza la dirección visual "soft-light" de `site-redesign-lean-soft` (supersedido).

## ADDED Requirements

### Requirement: Tokens de color de la identidad Verifika

El sistema SHALL usar los tokens de la identidad Verifika: chrome oscuro `slate-900` (nav, hero, sidebar, top bar, footer) con superficies `slate-800`/bordes `slate-700`/`slate-800`, primario `emerald-500`-family (CTAs, acentos, badges, estados activos), y superficies de contenido claras `slate-50`/`white` con bordes `slate-200` y texto `slate-900`/`slate-600`. El sistema de temas claro/oscuro SHALL conservarse operativo para las superficies de contenido (el chrome oscuro es fijo).

#### Scenario: Chrome oscuro en ambos temas

- **WHEN** un usuario navega en tema claro u oscuro
- **THEN** nav/hero/sidebar/top bar/footer SHALL renderizar la superficie oscura `slate-900` de la identidad
- **AND** las superficies de contenido SHALL renderizar con los tokens del tema activo

#### Scenario: CTAs y acentos emerald

- **WHEN** se renderiza un botón primario o un acento de estado (badge activo, indicador de no leído, enlace de CTA)
- **THEN** SHALL usar la familia `emerald-500` de la identidad (botones `bg-emerald-500`, acentos `text-emerald-400/600`, fondos `bg-emerald-50/100`)

### Requirement: Tipografía display + Inter

El sistema SHALL usar tipografía display (`font-heading`) para títulos y encabezados, cargada vía `next/font/google`, e Inter para el texto de cuerpo. La variable de fuente display SHALL estar disponible en `tailwind.config.ts` bajo el alias `font-heading` para que los usos existentes de la clase `font-heading` (64+ usos en la plantilla) resuelvan correctamente.

#### Scenario: Títulos con fuente display

- **WHEN** un encabezado (h1/h2/h3) con clase `font-heading` se renderiza
- **THEN** SHALL usar la fuente display cargada, no Inter
- **AND** el body SHALL usar Inter

#### Scenario: Fuente display disponible en build

- **WHEN** `pnpm build` se ejecuta
- **THEN** la fuente display SHALL estar incluida vía next/font y la clase `font-heading` SHALL resolver sin error de tipografía

### Requirement: Idioms de componente de la plantilla

Las superficies SHALL usar los idioms de componente de la plantilla: tarjetas claras `rounded-xl border border-slate-200` (contenido) u oscuras con `border-slate-800` (chrome); tarjetas KPI con chip de icono en fondo tintado (emerald/amber/blue/purple `-50`); botones primarios `bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg`; inputs con `focus:border-emerald-500 focus:ring-emerald-500`; badges de estado con punto de color (verde = activo/conectado, ámbar = pendiente, rojo = urgente); avatares con gradiente `from-emerald-400 to-emerald-600`.

#### Scenario: Tarjeta KPI con chip de icono

- **WHEN** una tarjeta KPI se renderiza en el dashboard o la bandeja
- **THEN** SHALL incluir chip de icono con fondo tintado (`bg-emerald-50`/`bg-amber-50`/`bg-blue-50`/`bg-purple-50`) e icono en color de acento correspondiente

#### Scenario: Botón primario emerald

- **WHEN** un botón primario se renderiza en cualquier superficie
- **THEN** SHALL usar `bg-emerald-500` con hover `bg-emerald-600`, esquinas `rounded-lg` y texto blanco

### Requirement: Idioma del documento y copy en español

El documento HTML SHALL declarar `lang="es"` en el root layout. Los strings de UI SHALL resolverse desde la capa tipada `lib/copy/ui.ts` (español-first con espejo en inglés), incluyendo la copia fusionada de la plantilla; los errores ortográficos de la plantilla ("Inciciar session", "session") SHALL corregirse.

#### Scenario: lang es en el documento

- **WHEN** cualquier página renderiza el documento HTML
- **THEN** el elemento `<html>` SHALL tener `lang="es"`

#### Scenario: Copia de plantilla fusionada y corregida

- **WHEN** la landing y las páginas de onboarding renderizan copy proveniente de la plantilla
- **THEN** los strings SHALL provenir de `lib/copy/ui.ts` con los typos corregidos ("Iniciar sesión", no "Inciciar session")

### Requirement: Assets de la plantilla como placeholders

Los assets de la plantilla (logos `plain-*`/`welytics`/`resecurb`, avatares genéricos, `map.png`) SHALL usarse solo como placeholders decorativos cuando aplique; la marca del producto (NexoChat) y los datos reales SHALL prevalecer sobre cualquier asset stock. `react-apexcharts` de la plantilla SHALL NOT adoptarse (los charts usan `recharts` ya presente).

#### Scenario: Marca del producto en chrome

- **WHEN** nav, sidebar o footer renderizan el logo/nombre
- **THEN** SHALL mostrar la identidad NexoChat, no "Verifika"/"ChatFlow" ni logos stock

#### Scenario: Charts con recharts

- **WHEN** un chart se renderiza (dashboard/reportes)
- **THEN** SHALL usar `recharts` existente, no `react-apexcharts`
