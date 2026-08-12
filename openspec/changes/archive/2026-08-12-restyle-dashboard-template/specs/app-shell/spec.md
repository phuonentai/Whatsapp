## Purpose

El restyle del panel al template aprobado (change `restyle-dashboard-template`) ajusta dos requirements del shell: el shell pasa a una superficie fija oscura con utilidades slate/emerald explícitas (identidad del template) manteniendo el toggle de tema para el contenido, y el dashboard home extiende su set de KPIs al del template con la regla "—" para fuentes ausentes. La verificación de parámetros de pago previa al home se mantiene (reforzada explícitamente).

## MODIFIED Requirements

### Requirement: Dark mode with persisted preference

The app SHALL support light and dark themes. Theme preference SHALL be stored in `localStorage`, SHALL default to the system `prefers-color-scheme`, and SHALL be switchable from the user menu. Content surfaces SHALL use theme tokens (CSS variables) so both themes render coherently. The shell (sidebar, top bar) SHALL render the template's fixed dark `slate-900` surface with explicit slate/emerald utilities regardless of the active theme; the theme toggle SHALL remain operational for content surfaces.

#### Scenario: Toggle switches theme

- **WHEN** user toggles dark mode in the user menu
- **THEN** the app SHALL switch the content theme and persist the preference

#### Scenario: Preference survives reload

- **WHEN** user has chosen dark mode and reloads the page
- **THEN** the app SHALL render content in the chosen theme

#### Scenario: Shell stays dark in light theme

- **WHEN** the active theme is light
- **THEN** the sidebar and top bar SHALL still render the fixed dark `slate-900` template surface

### Requirement: Dashboard home shows KPIs and quick actions

`/dashboard` SHALL render a dashboard home instead of redirecting to settings: KPI cards (open conversations, week sales, invoices issued, AI response time), recent activity, quick actions linking to inbox, CRM, and knowledge, a sales chart (real data or empty state with CTA to reports), and the "Copiloto IA" panel (existing insights or static guidance with CTA). KPIs SHALL reuse the existing queries consumed by `DashboardHome` (no new fan-out); a KPI without a data source SHALL render "—" and never a fabricated number. Payment-parameter verification SHALL remain in place before the home renders.

#### Scenario: Dashboard home loads with KPIs

- **WHEN** an entitled user navigates to `/dashboard`
- **THEN** the dashboard home SHALL render KPI cards and quick actions

#### Scenario: KPI values reflect current data

- **WHEN** the dashboard home loads
- **THEN** KPI cards SHALL display values from the org's current data via existing queries

#### Scenario: Missing data source shows placeholder

- **WHEN** a KPI has no data source in the repo
- **THEN** the card SHALL display "—" without fabricating a value

#### Scenario: Payment parameters verified before home

- **WHEN** the URL carries `checkout_id`, `payment_id` or `preapproval_id`
- **THEN** the page SHALL run the corresponding payment verification (Polar `verifyPayment` / MercadoPago `verifyMercadoPagoPayment`) and redirect to `/dashboard/settings?view=subscription` with `payment_verified=true` or `payment_error=true` before rendering the home

#### Scenario: Quick action navigates to feature

- **WHEN** user clicks a quick action (e.g., "Nueva conversación")
- **THEN** the app SHALL navigate to the corresponding view
