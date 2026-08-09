## ADDED Requirements

### Requirement: Admin can edit the workspace display name

The workspace settings card SHALL allow a user with `org:manage` permission to edit the organization display name and persist it via `PUT /organizations`, passing through the required `status` field.

#### Scenario: Save workspace name

- **WHEN** an admin edits the workspace name and saves
- **THEN** the system SHALL send `PUT /organizations` with the new `name` and the current org `status`
- **AND** on success SHALL show a confirmation toast and refresh the displayed workspace name

#### Scenario: Save fails on invalid input

- **WHEN** an admin submits a blank or invalid workspace name
- **THEN** the system SHALL display an inline error and NOT call the update endpoint

#### Scenario: Non-admin cannot edit workspace name

- **WHEN** a user without `org:manage` permission views the workspace settings
- **THEN** the workspace name SHALL render read-only with no edit control

### Requirement: Stytch sync is circuit-breaker guarded

Every outbound Stytch B2B call made by the org display-name and member-role update flows SHALL be guarded by the shared Stytch circuit breaker. When the breaker is open, the update SHALL be rejected and no local write SHALL occur, so both SSOTs stay in phase.

#### Scenario: Breaker open rejects the update without local write

- **WHEN** the Stytch circuit breaker is open during a `PUT /organizations` or `PUT /accounts/:id` call
- **THEN** the system SHALL return an error to the caller
- **AND** the local organization or account row SHALL NOT be modified

#### Scenario: Breaker recovers and updates resume

- **WHEN** the Stytch circuit breaker transitions back to closed (half-open probe succeeds)
- **THEN** subsequent `PUT /organizations` and `PUT /accounts/:id` calls SHALL sync to Stytch and persist locally as normal

### Requirement: Admin can change a member's role

The member list SHALL allow a user with `org:manage` permission to change a member's role via `PUT /auth/members/:member_id/role`, using the role vocabulary accepted by the backend (`admin`, `approver`, `member`). Member roles are governed by the Stytch B2B member management contract (the endpoint syncs the Stytch member role before the local write); no local credential or role store is introduced.

#### Scenario: Change member role

- **WHEN** an admin changes a member's role to a valid backend role and confirms
- **THEN** the system SHALL send `PUT /auth/members/:member_id/role` with the new role
- **AND** on success SHALL update the row in place and show a confirmation toast

#### Scenario: Role change fails

- **WHEN** the `PUT /auth/members/:member_id/role` call returns an error
- **THEN** the system SHALL show an error toast and leave the member's displayed role unchanged

#### Scenario: Admin cannot change own role

- **WHEN** an admin views their own member row
- **THEN** the role control SHALL be disabled for that row

#### Scenario: Guard against demoting the last admin

- **WHEN** an admin attempts to change the role of the last `admin` member of the organization
- **THEN** the system SHALL reject the change with an inline error explaining that at least one admin must remain
