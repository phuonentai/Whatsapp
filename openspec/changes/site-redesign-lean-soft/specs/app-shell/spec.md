# app-shell Specification

## Purpose

Definición del shell de la aplicación (sidebar, top bar, command palette, búsqueda global, atajos de teclado, tema, dashboard home).

**Actualización (site-redesign-lean-soft):** el shell ya no renderiza una superficie fija oscura `slate-900`; sigue los tokens del tema activo con la paleta empresarial suave (claro y oscuro), manteniendo persistencia y toggle del tema.

## Requirements

### Requirement: Command palette navigates anywhere

The app SHALL provide a command palette opened with Ctrl/Cmd+K (or the `?` help overlay) from anywhere in the dashboard. The palette SHALL list all sidebar destinations and settings views, filter by fuzzy text match, support keyboard navigation (arrows + Enter), close on Escape, and be accessible (dialog semantics, focus management).

#### Scenario: Palette opens with Ctrl+K

- **WHEN** user presses Ctrl+K (Cmd+K on macOS) in the dashboard
- **THEN** the command palette SHALL open with keyboard focus in its input

#### Scenario: Typing filters navigation targets

- **WHEN** user types "contactos" in the palette
- **THEN** the palette SHALL show the Contactos destination as the top result

#### Scenario: Enter navigates to selected target

- **WHEN** user selects a destination and presses Enter
- **THEN** the app SHALL navigate to that destination and the palette SHALL close

#### Scenario: Escape closes the palette

- **WHEN** the palette is open and user presses Escape
- **THEN** the palette SHALL close and focus SHALL return to the trigger

### Requirement: Global search finds contacts from the header

The header SHALL expose a global search that opens the palette in search mode. Searching SHALL query the existing `searchContacts` server route and render results keyboard-navigable, with Enter opening the contact detail view.

#### Scenario: Search returns matching contacts

- **WHEN** user types a name or phone in global search
- **THEN** the palette SHALL display matching contacts from the `searchContacts` route

#### Scenario: Enter opens the contact

- **WHEN** user selects a contact result and presses Enter
- **THEN** the app SHALL navigate to the contact detail view

#### Scenario: No results show empty search state

- **WHEN** the search returns no matches
- **THEN** the palette SHALL display a "No results" state

### Requirement: Global keyboard shortcuts

The app SHALL support global keyboard shortcuts: `g d` (dashboard), `g i` (inbox), `g c` (CRM), `g k` (knowledge), `g s` (settings), `?` (shortcuts help). Shortcuts SHALL NOT trigger while typing in an input, textarea, or contenteditable.

#### Scenario: g then i opens inbox

- **WHEN** user presses `g` then `i` outside an input
- **THEN** the app SHALL navigate to `/dashboard/inbox`

#### Scenario: Shortcuts suppressed while typing

- **WHEN** the user is typing in a text input
- **THEN** pressing `g` SHALL NOT trigger navigation

#### Scenario: Question mark shows shortcut help

- **WHEN** user presses `?` outside an input
- **THEN** a shortcuts overlay SHALL open listing available shortcuts

## MODIFIED Requirements

### Requirement: Dark mode with persisted preference

The app SHALL support light and dark themes. Theme preference SHALL be stored in `localStorage`, SHALL default to the system `prefers-color-scheme`, and SHALL be switchable from the user menu. Content surfaces and the shell (sidebar, top bar) SHALL use theme tokens (CSS variables) so both themes render coherently with the soft business palette; the shell SHALL NOT render a fixed dark `slate-900` surface in light theme. The theme toggle SHALL remain operational.

#### Scenario: Toggle switches theme

- **WHEN** user toggles dark mode in the user menu
- **THEN** the app SHALL switch the content and shell theme and persist the preference

#### Scenario: Preference survives reload

- **WHEN** user has chosen dark mode and reloads the page
- **THEN** the app SHALL render content and shell in the chosen theme

#### Scenario: Default follows system preference

- **WHEN** the user has no stored preference and the OS is in dark mode
- **THEN** the app SHALL render in dark mode

#### Scenario: Shell follows theme in light mode

- **WHEN** the active theme is light
- **THEN** the sidebar and top bar SHALL render soft light theme-token surfaces (no fixed `slate-900` shell)

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
