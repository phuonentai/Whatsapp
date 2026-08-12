# settings-ui Delta Spec

## MODIFIED Requirements

### Requirement: Settings presenta el lenguaje visual del diseño

Los módulos de configuración (`app/dashboard/settings/components/*` + overview en `settings-content.tsx`) SHALL presentar el lenguaje visual del diseño: superficies de contenido claras (`slate-50`/white) con bordes `slate-200` y `rounded-2xl`, chips de estado semánticos (emerald = activo/conectado, amber = pendiente/umbral, red = error/desconectado) SIEMPRE con texto (nunca color-only), iconos con tile de color por módulo, y jerarquía tipográfica consistente (título, descripción, secciones). Los 10 módulos (profile, subscription, modules, compliance, audit, whatsapp, templates, instagram, siigo, siigo-admin) SHALL migrar sus clases `gray-*` hardcodeadas al lenguaje sin cambiar ningún contrato de comportamiento (invite, toggles, playbooks, perfil, plan, conexiones).

#### Scenario: Overview con lenguaje del diseño

- **WHEN** un usuario abre `/dashboard/settings`
- **THEN** la lista de secciones SHALL renderizar tarjetas con icono, valor y helper en el lenguaje del diseño
- **AND** la navegación por stack de vistas (`?view=`) SHALL seguir funcionando sin cambios

#### Scenario: Módulo re-estilizado sin cambio de contrato

- **WHEN** un usuario abre cualquiera de los 10 módulos
- **THEN** la vista SHALL presentar el lenguaje visual
- **AND** las acciones (invite, toggle, guardar, conectar, cancelar) SHALL comportarse idéntico al estado previo

#### Scenario: Chips de estado semánticos con texto

- **WHEN** un estado de conexión o módulo se muestra (conectado/pausado/pendiente/error)
- **THEN** el chip SHALL combinar color y texto legible, sin depender solo del color

## ADDED Requirements

### Requirement: Uso y límites visibles en subscription

La vista de subscription SHALL mostrar el uso actual frente al límite del plan con barras de progreso: créditos IA (`ai_usage`: `credits_used`/`credits_max`) y facturas incluidas/usadas (`usage`/`includedInvoices`). Las barras SHALL cambiar a umbral amber cuando el uso alcance ≥80% del límite. La lógica de cancelación/resumen/cambio de plan SHALL permanecer intacta.

#### Scenario: Barras de uso con umbral

- **WHEN** el usuario abre la vista de subscription con datos de uso disponibles
- **THEN** SHALL renderizar barras de créditos IA y facturas con el porcentaje real
- **AND** cuando el uso ≥80% la barra SHALL mostrar el estado amber con texto

#### Scenario: Sin datos de uso

- **WHEN** no hay datos de uso disponibles
- **THEN** las barras SHALL mostrar un estado neutro o "—" sin fabricar porcentajes
