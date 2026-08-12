# admin-panel-navigation Delta Spec

## MODIFIED Requirements

### Requirement: Navegación de plataforma en shell de operador

La navegación de la superficie de plataforma SHALL presentarse como shell de operador con secciones Organizaciones, Uso IA y Auditoría (en lugar del sidebar parcial de inbox/CRM; SIN sección Siigo — la superficie Siigo permanece en settings del tenant). La navegación del panel de miembros (`/dashboard`) SHALL permanecer sin cambios. Cada sección activa SHALL llevar `aria-current="page"` y el gate de sección SHALL basarse en `platform:operate` con el contrato de fallback de `stytch-authorization` (403 sin permiso; 503 si la política no está disponible y la caché vacía; la UI muestra "política no disponible", nunca un falso "sin permisos").

#### Scenario: Operador navega las secciones

- **WHEN** un operador abre el shell
- **THEN** SHALL ver las secciones de plataforma con enlaces reales
- **AND** la sección activa SHALL marcarse con aria-current

#### Scenario: Navegación de miembros intacta

- **WHEN** un miembro navega `/dashboard`
- **THEN** la sidebar de miembros SHALL permanecer igual (sin enlaces de plataforma)
