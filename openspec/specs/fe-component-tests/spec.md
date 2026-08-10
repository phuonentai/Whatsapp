# Spec: fe-component-tests

## Purpose

Defines frontend component test coverage via vitest: form validation through the render harness and table empty and populated states.

## Requirements

### Requirement: Component tests run via vitest
The system SHALL provide a `pnpm test` command in `next_b2b_starter/` that runs the component-test suite with vitest in a `jsdom` environment, with Testing Library matchers loaded from a setup file.

#### Scenario: `pnpm test` runs the suite
- **WHEN** a developer runs `pnpm test` in `next_b2b_starter/`
- **THEN** all component specs execute and the command exits 0 on success

### Requirement: Forms validate and submit through the render harness
The system SHALL cover CRM forms/dialogs (contact, company, deal) with component tests asserting that required-field validation blocks submission, valid submission invokes the mocked server action exactly once, and cancel closes without submitting. Tests SHALL render through a shared `renderWithProviders()` harness that supplies a fresh QueryClient per test and mocks `lib/actions/*` server actions and auth/user hooks.

#### Scenario: Required fields block submit
- **WHEN** a form is rendered with empty required fields and the submit control is activated
- **THEN** validation errors appear and the mocked server action is not called

#### Scenario: Valid submit calls the action once
- **WHEN** a form is filled with valid data and submitted
- **THEN** the mocked server action is called exactly once with the submitted values

#### Scenario: Cancel closes without submitting
- **WHEN** a dialog form is cancelled after editing
- **THEN** the dialog closes and no server action is called

### Requirement: Tables cover empty and populated states
The system SHALL cover CRM tables (contact, company) with component tests asserting the empty state renders, rows render from props, and row actions fire their callbacks.

#### Scenario: Table empty state renders
- **WHEN** a table receives an empty row set
- **THEN** an empty-state message is shown instead of rows

#### Scenario: Table rows render and actions fire
- **WHEN** a table receives rows and a row action is activated
- **THEN** the rows render with their data and the action callback fires

### Requirement: Shared UI behaviours are covered
The system SHALL cover `tag-picker`, `upgrade-banner`, `deal-kanban`, `confirm-dialog`, inbox reply input, knowledge dropzone, and settings module toggle with component tests for selection toggling, gated-content visibility, confirm/cancel semantics, empty-input no-ops, PDF-only upload acceptance, and switch state reflection/persistence via mocked actions.

#### Scenario: Selection and gating behaviours
- **WHEN** picker selection is toggled, gated content is rendered for a privileged context, and a confirm dialog is accepted or cancelled
- **THEN** selection state updates, gated content visibility matches the context, and confirm/cancel semantics hold

#### Scenario: Input and toggle behaviours
- **WHEN** an empty reply or chat input is submitted, a non-PDF file is dropped, and a module switch is toggled
- **THEN** no send occurs, an upload error is shown, and the switch reflects and persists its state via the mocked action
