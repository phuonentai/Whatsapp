# Delta Spec: settings-ui — app-shell-modernization

## ADDED Requirements

### Requirement: Settings views are reachable from the command palette

Every settings detail view (`?view=...`) SHALL be registered as a command-palette destination with a display name and target URL.

#### Scenario: Palette lists settings views

- **WHEN** the command palette opens and the user types "settings"
- **THEN** the palette SHALL list settings destinations including Account, Team, Subscription, Modules, AI Copilot, Compliance, Messaging, and Audit log

#### Scenario: Palette entry opens a settings view

- **WHEN** user selects a settings destination in the palette and presses Enter
- **THEN** the app SHALL navigate to `/dashboard/settings?view=<selected>`
