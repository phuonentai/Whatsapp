# Spec: ui-copy

## Purpose

TBD. Defines the typed copy layer: all user-facing copy in the onboarding, billing, WhatsApp-config, inbox, dashboard, and agent-settings surfaces resolves from `lib/copy/ui.ts` in Spanish-first voice with English fallback, developer jargon is confined to the advanced WhatsApp panel, and connect progress is expressed in user language.

## Requirements

### Requirement: Typed copy layer routes all user-facing copy

The system SHALL provide a typed copy layer at `lib/copy/ui.ts` exporting a copy namespace grouped by surface (`auth`, `billing`, `whatsapp`, `inbox`, `dashboard`, `agent`, `common`). All user-facing strings in the onboarding, billing, WhatsApp-config, inbox, dashboard, and agent-settings surfaces SHALL resolve from this layer rather than being hardcoded inline in JSX. Each key SHALL be TypeScript-typed so misspelled or unknown keys fail compilation.

#### Scenario: Component copy resolves from the copy layer

- **WHEN** a user-facing label, description, button, empty-state, or error is rendered in the affected surfaces
- **THEN** the string SHALL be sourced from `lib/copy/` via a typed key
- **AND** no new hardcoded user-facing string SHALL be introduced in those surfaces

#### Scenario: Unknown copy key fails the build

- **WHEN** a component references a copy key that does not exist in the namespace
- **THEN** the TypeScript build SHALL fail

### Requirement: Spanish-first strings with English fallback

Copy keys SHALL default to Spanish (the primary market language for Colombia). Each key SHALL carry a Spanish string; an English fallback SHALL exist only where a migration is incomplete, and the fallback SHALL never be an empty string. Primary onboarding, billing, and WhatsApp-connect flows SHALL ship fully in Spanish with no key resolving to its fallback.

#### Scenario: Primary flows render Spanish

- **WHEN** a user completes signup, opens the plans modal, or connects WhatsApp
- **THEN** every rendered string SHALL be Spanish and SHALL NOT resolve to the English fallback

#### Scenario: Fallback never renders empty

- **WHEN** a copy key is referenced
- **THEN** it SHALL resolve to a non-empty string (Spanish or, for incomplete migrations only, English)

### Requirement: Developer jargon confined to advanced settings

Meta developer tokens and protocol terms (webhook secret, verify token, WABA ID, permanent access token, API version, Graph API URL, Embedded Signup) SHALL appear only inside the collapsed advanced panel of the WhatsApp configuration surface. Primary user-facing copy SHALL describe the action and outcome in plain language without exposing mechanism names.

#### Scenario: Connect flow uses plain language

- **WHEN** a user connects WhatsApp via the primary connect action
- **THEN** the visible copy SHALL describe connecting the business WhatsApp and receiving/managing messages
- **AND** SHALL NOT display Meta developer token labels or protocol terms in the primary view

#### Scenario: Advanced panel retains technical fields

- **WHEN** a user expands the advanced settings panel in the WhatsApp configuration surface
- **THEN** the technical token and identifier fields remain available with their developer labels

### Requirement: Connect micro-status steps are user-facing

The WhatsApp connect progress steps SHALL be expressed in user language (e.g., "Conectando tu WhatsApp…", "Validando la conexión…", "Todo listo") and SHALL NOT use internal protocol terminology.

#### Scenario: In-progress steps render plain progress copy

- **WHEN** the embedded-signup connect flow is in progress
- **THEN** the progress messages SHALL be the Spanish user-facing strings
- **AND** SHALL NOT reference sessions, tokens, or webhooks
