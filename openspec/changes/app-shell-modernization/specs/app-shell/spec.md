## ADDED Requirements

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

### Requirement: Dark mode with persisted preference

The app SHALL support light and dark themes. Theme preference SHALL be stored in `localStorage`, SHALL default to the system `prefers-color-scheme`, and SHALL be switchable from the user menu. Shell and auth surfaces SHALL use theme tokens (CSS variables) instead of hardcoded `gray-*`/`bg-white` values so both themes render coherently.

#### Scenario: Toggle switches theme

- **WHEN** user toggles dark mode in the user menu
- **THEN** the app SHALL switch to dark theme and persist the preference

#### Scenario: Preference survives reload

- **WHEN** user has chosen dark mode and reloads the page
- **THEN** the app SHALL render in dark mode

#### Scenario: Default follows system preference

- **WHEN** the user has no stored preference and the OS is in dark mode
- **THEN** the app SHALL render in dark mode

### Requirement: Dashboard home shows KPIs and quick actions

`/dashboard` SHALL render a dashboard home instead of redirecting to settings: KPI cards (e.g., open conversations, contacts, deals by stage), recent activity, and quick actions linking to inbox, CRM, and knowledge. Payment-parameter verification SHALL remain in place before the home renders.

#### Scenario: Dashboard home loads with KPIs

- **WHEN** an entitled user navigates to `/dashboard`
- **THEN** the dashboard home SHALL render KPI cards and quick actions

#### Scenario: KPI values reflect current data

- **WHEN** the dashboard home loads
- **THEN** KPI cards SHALL display values from the org's current data via existing queries

#### Scenario: Quick action navigates to feature

- **WHEN** user clicks a quick action (e.g., "Nueva conversación")
- **THEN** the app SHALL navigate to the corresponding view

### Requirement: Route-level loading states

Shell and list routes SHALL render `loading.tsx` skeletons during server rendering, and lazy views SHALL use Suspense boundaries where route-level streaming applies.

#### Scenario: Dashboard route streams a skeleton

- **WHEN** a user navigates to a route with a `loading.tsx`
- **THEN** the route SHALL render its skeleton while the page loads

### Requirement: Shell accessibility baseline

The shell SHALL provide a skip-to-content link, `aria-current` on the active nav item, and Escape-to-close plus focus return for the mobile drawer.

#### Scenario: Skip link moves focus to content

- **WHEN** user activates the skip-to-content link
- **THEN** focus SHALL move to the main content region, skipping the sidebar

#### Scenario: Active nav item marks aria-current

- **WHEN** a sidebar entry is active
- **THEN** the entry SHALL carry `aria-current="page"`

#### Scenario: Escape closes the mobile drawer

- **WHEN** the mobile sidebar drawer is open and user presses Escape
- **THEN** the drawer SHALL close and focus SHALL return to the hamburger button

### Requirement: Unified brand tokens

The app SHALL use a single brand token set across shell, auth, and not-found surfaces, replacing the "Your App" / "AP Cash" / teal `#0FA8A0` drift.

#### Scenario: Product name renders consistently

- **WHEN** the app shell, auth pages, and not-found page render
- **THEN** they SHALL display the same product name and brand color tokens

#### Scenario: Not-found uses shell tokens

- **WHEN** a user hits a 404 page
- **THEN** the not-found page SHALL use the shell brand tokens instead of the legacy teal
