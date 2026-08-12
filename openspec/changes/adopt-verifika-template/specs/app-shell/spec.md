# app-shell Delta Spec

## MODIFIED Requirements

### Requirement: Dark mode with persisted preference

The app SHALL support light and dark themes. Theme preference SHALL be stored in `localStorage`, SHALL default to the system `prefers-color-scheme`, and SHALL be switchable from the user menu. Content surfaces SHALL use theme tokens (CSS variables) so both themes render coherently. The shell (sidebar, top bar) SHALL render the Verifika template's fixed dark `slate-900` chrome with explicit slate/emerald utilities regardless of the active theme (see `verifika-visual-identity`); the theme toggle SHALL remain operational for content surfaces. This requirement supersedes the soft-light shell direction of `site-redesign-lean-soft`.

#### Scenario: Toggle switches theme

- **WHEN** user toggles dark mode in the user menu
- **THEN** the app SHALL switch the content theme and persist the preference

#### Scenario: Preference survives reload

- **WHEN** user has chosen dark mode and reloads the page
- **THEN** the app SHALL render content in the chosen theme

#### Scenario: Default follows system preference

- **WHEN** the user has no stored preference and the OS is in dark mode
- **THEN** the app SHALL render in dark mode

#### Scenario: Shell stays dark in light theme

- **WHEN** the active theme is light
- **THEN** the sidebar and top bar SHALL still render the fixed dark `slate-900` Verifika chrome surface
