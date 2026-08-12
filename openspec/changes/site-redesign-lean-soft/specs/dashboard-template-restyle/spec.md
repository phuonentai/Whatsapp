# dashboard-template-restyle Specification

## Purpose

Define el restyle del panel del producto al template de referencia aprobado (dashboard.html + messages-view.html + onboarding.html): shell con sidebar oscura slate-900, overview con KPIs reales y verificación de pagos preservada, bandeja con composición messages-view y wizard de onboarding oscuro. Cero cambios de lógica de negocio (auth Stytch, RBAC, API y datos intactos).

**Actualización (site-redesign-lean-soft):** el shell y las superficies del producto pasan de la identidad oscura slate-900/emerald del template a la paleta empresarial suave de los tokens de tema. Los contratos de comportamiento (navegación RBAC, verificación de pagos, filtros de bandeja, flujo Stytch) permanecen intactos.

## Requirements

## MODIFIED Requirements

### Requirement: Dashboard shell con identidad del template

El panel del producto (`app/dashboard/*`) SHALL presentar el shell con la paleta empresarial suave: sidebar sobre `card`/`background` con `border-r` de tema, ítems de navegación (Dashboard, Conversaciones, Contactos, Facturas, Pagos, Analíticas; grupo "Inteligencia Artificial": Copiloto IA, Entrenamiento, Automatizaciones), tarjeta "IA Insights" con tintes salvia suaves (`secondary`), y top bar con búsqueda, notificaciones y usuario. El shell SHALL seguir los tokens del tema activo (claro y oscuro suave); las superficies `slate-900` fijas y los acentos `emerald-500` SHALL NOT usarse. La navegación SHALL mantener las rutas y permisos existentes; los tokens shadcn del tema SHALL gobernar el contenido.

#### Scenario: Shell suave con navegación existente

- **WHEN** un usuario autenticado abre cualquier página de `/dashboard`
- **THEN** el shell SHALL renderizar la sidebar con superficies de tema suave y los ítems de navegación con las rutas reales de la app
- **AND** la navegación existente (bandeja, CRM, reportes, knowledge, settings) SHALL seguir funcionando con los permisos RBAC actuales

#### Scenario: Shell sigue el tema activo

- **WHEN** el tema activo es claro u oscuro
- **THEN** la sidebar y la top bar SHALL renderizar las superficies suaves correspondientes al tema (nunca un shell oscuro fijo `slate-900` en tema claro)

#### Scenario: Grupos del template respetan RBAC

- **WHEN** el grupo "Inteligencia Artificial" o la tarjeta "IA Insights" se renderizan en la sidebar
- **THEN** cada ítem SHALL mostrarse solo si el usuario tiene el permiso/entitlement correspondiente según el modelo de navegación filtrada existente
- **AND** los ítems sin ruta real o sin permiso SHALL NOT enlazarse incondicionalmente

### Requirement: Onboarding con wizard del template

El flujo de signup (`app/signup`) SHALL presentar el wizard con la paleta empresarial suave (progreso, navegación prev/next, selección de tipo de negocio) sobre la lógica existente (`use-signup-flow.ts`, componentes Stytch). El contenedor del wizard SHALL usar superficies de tema (sin fondo oscuro fijo); los componentes embedded de Stytch conservan su propio theming. Los contratos Stytch (magic link, redirects, RBAC) SHALL permanecer sin cambios.

#### Scenario: Flujo completo sin cambios de contrato

- **WHEN** un nuevo cliente completa el signup
- **THEN** el wizard SHALL pasar por los pasos account→organization→business con las validaciones actuales
- **AND** las llamadas a Stytch (`sendMagicLink`) y los redirects SHALL ser idénticos a los actuales

#### Scenario: Contenedor del wizard con superficie suave

- **WHEN** un usuario ve el wizard de signup en tema claro
- **THEN** el contenedor SHALL renderizar una superficie clara suave (sin bloque oscuro fijo)
