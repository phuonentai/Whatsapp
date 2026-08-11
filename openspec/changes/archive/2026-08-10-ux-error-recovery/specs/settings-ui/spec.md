# Delta Spec: settings-ui — ux-error-recovery

## ADDED Requirements

### Requirement: Destructive member and compliance actions use custom dialogs

Member role changes, member removal, and the compliance forget flow SHALL confirm via the custom `ConfirmDialog` component and SHALL NOT use native `window.confirm`.

#### Scenario: Member role change confirms via custom dialog

- **WHEN** an admin changes a member's role
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** no `window.confirm` SHALL be invoked

#### Scenario: Member removal confirms via custom dialog

- **WHEN** an admin removes a member
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** the "last admin" guard SHALL show an inline error, not a native dialog

#### Scenario: Compliance forget confirms via custom dialog

- **WHEN** a user triggers the compliance forget flow
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** no `window.confirm` SHALL be invoked
