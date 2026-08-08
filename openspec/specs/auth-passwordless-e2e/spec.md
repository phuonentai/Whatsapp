## Purpose

Define the end-to-end behavior of the passwordless authentication flow: signup and login pages expose no password input, the signup payload carries no password, no password login endpoint exists, and the magic link landing page renders without a password field.

## Requirements

### Requirement: Signup page exposes no password field

The `/signup` page SHALL render the account and organization steps without any password input. The E2E test SHALL assert that no `input[type="password"]` element exists on the page.

#### Scenario: Signup page shows no password input

- **WHEN** a browser visits `/signup`
- **THEN** the page SHALL render Full Name, Email, Organization Name, and Industry fields
- **AND** the page SHALL NOT contain any `input[type="password"]` element

### Requirement: Login page is email-only

The `/auth` login page SHALL expose only an email-based magic link flow with no password input. The E2E test SHALL assert that no `input[type="password"]` element exists on the page.

#### Scenario: Login page shows no password input

- **WHEN** a browser visits `/auth`
- **THEN** the page SHALL render an email input and a "Email me a sign-in link" submit control
- **AND** the page SHALL NOT contain any `input[type="password"]` element

### Requirement: Signup payload excludes owner_password

The frontend signup request to `POST /auth/signup` SHALL NOT include an `owner_password` field. The E2E test SHALL intercept the request and assert the payload lacks the key.

#### Scenario: Intercepted signup payload has no owner_password

- **WHEN** a browser submits the signup form on `/signup`
- **AND** the outgoing `POST /auth/signup` request is intercepted
- **THEN** the request payload SHALL NOT contain an `owner_password` key

### Requirement: No password login endpoint exists

The system SHALL NOT expose a `POST /auth/login` (or equivalent password authentication) endpoint. The E2E test SHALL assert that such a request is rejected with 404 or 405.

#### Scenario: Password login endpoint is absent

- **WHEN** a request is sent to `POST /auth/login`
- **THEN** the response status SHALL be 404 or 405

### Requirement: Magic link landing page exists

The `/authenticate` page SHALL handle the magic link token exchange flow without any password input.

#### Scenario: Authenticate page renders without password input

- **WHEN** a browser visits `/authenticate`
- **THEN** the page SHALL render the verification flow
- **AND** the page SHALL NOT contain any `input[type="password"]` element