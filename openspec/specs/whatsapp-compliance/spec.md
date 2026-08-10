## Purpose

Defines WhatsApp compliance: consent state machine per Ley 1581, PII masking before third-party AI calls, and Habeas Data export.

## Requirements

### Requirement: Consent state machine for data treatment (Ley 1581)

The system SHALL track a consent state machine per contact on `crm.contacts`: `consent_status` in (`none`, `requested`, `granted`, `withdrawn`) and `consented_at`. Transitions SHALL be: `none → requested → granted`, with `withdrawn` reachable from any state. A contact with `withdrawn` SHALL NOT receive autonomous replies.

#### Scenario: First contact requests consent automatically

- **WHEN** an organization has `consent_required = true` and an inbound message arrives from a contact with `consent_status = 'none'`
- **THEN** the system SHALL send the configured `consent_template` as a session message to that contact
- **AND** SHALL set the contact's `consent_status` to `requested`
- **AND** SHALL NOT send any other autonomous reply to that contact

#### Scenario: Affirmative reply grants consent

- **WHEN** a contact with `consent_status = 'requested'` replies with an affirmative message (deterministic terms such as "sí", "acepto", "ok", or LLM-classified consent)
- **THEN** the system SHALL set `consent_status = 'granted'` and `consented_at` to the current time

#### Scenario: Withdrawal blocks autonomous sends

- **WHEN** a contact has `consent_status = 'withdrawn'`
- **THEN** the system SHALL NOT send autonomous replies to that contact
- **AND** autonomous drafts SHALL fall back to pending human-review suggestions

### Requirement: PII masking before third-party AI calls

The system SHALL redact personally identifiable information from prompts before they reach third-party AI providers, via a masking decorator on the LLM client seam. At minimum, `numero_documento` values, E.164 phone numbers, and contact display names SHALL be replaced with placeholders (`[DOCUMENTO]`, `[TELEFONO]`, `[NOMBRE]`) in outbound prompt payloads. Original values SHALL remain in local PostgreSQL unchanged.

#### Scenario: Prompt sent to provider contains no raw PII

- **WHEN** an agent analysis call is made for a contact with a document number, phone, and name
- **THEN** the payload delivered to the AI provider SHALL contain the placeholder tokens instead of the raw values

#### Scenario: Masking failure does not fail the call

- **WHEN** masking cannot be applied (e.g. contact facts unavailable)
- **THEN** the call SHALL proceed unmasked
- **AND** the event SHALL be logged as a warning

### Requirement: Admin data export for Habeas Data requests

The system SHALL provide `POST /api/compliance/export` (body `{contact_id}`) that returns a structured bundle of the contact's profile, conversations, and messages for data-portability requests, accessible only with `org:manage` permission. When the contact's consent is `withdrawn`, the export SHALL mask PII fields.

#### Scenario: Export returns full contact history

- **WHEN** a user with `org:manage` permission requests an export for a contact
- **THEN** the system SHALL return a CSV/JSON bundle with the contact profile, all conversations, and all messages

#### Scenario: Export for withdrawn contact masks PII

- **WHEN** the exported contact has `consent_status = 'withdrawn'`
- **THEN** PII fields (phone, name, document, email) SHALL be masked in the export payload

#### Scenario: Export denied without permission

- **WHEN** a user without `org:manage` permission requests an export
- **THEN** the system SHALL return HTTP 403

### Requirement: Admin forget (right to erasure) anonymizes contact data

The system SHALL provide `POST /api/compliance/forget` (body `{contact_id}`) that anonymizes the contact's PII columns (phone, display name, document type/number, email → placeholders), sets `consent_status = 'withdrawn'`, and SHALL keep the contact, conversations, and messages rows for audit purposes. Accessible only with `org:manage` permission.

#### Scenario: Forget anonymizes but preserves records

- **WHEN** a user with `org:manage` permission requests forget for a contact
- **THEN** the contact's PII columns SHALL be replaced with placeholders
- **AND** `consent_status` SHALL be `withdrawn`
- **AND** conversations and messages SHALL remain present

#### Scenario: Forget is idempotent

- **WHEN** forget is requested twice for the same contact
- **THEN** the second request SHALL succeed without error and change nothing further
