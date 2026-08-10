## MODIFIED Requirements

### Requirement: Raw webhook payload logging

The system SHALL store the raw webhook request body, headers, and processing metadata in `whatsapp.webhook_logs` BEFORE publishing the event to the eventbus. Failed webhooks SHALL also be logged: a row with `status = 'failed'` and the error message SHALL be inserted BEFORE the error is returned to the caller. `organization_id` SHALL be NULL when the failure occurs before organization resolution (e.g., unknown or missing `phone_number_id`); it SHALL carry the resolved organization id when the config was found but a later step failed (e.g., invalid signature).

#### Scenario: Successful webhook logged and processed

- **WHEN** a webhook passes signature validation and organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'received'`, the raw payload, and the resolved `organization_id`
- **AND** then SHALL publish the `whatsapp.message.received` event

#### Scenario: Failed webhook logged with unresolvable org

- **WHEN** a webhook fails organization resolution (unknown or missing `phone_number_id`)
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'failed'` and the error message
- **AND** the row SHALL have `organization_id` set to NULL
- **AND** the failure SHALL then be returned to the caller (HTTP 404 for unknown phone)

#### Scenario: Failed webhook logged with resolved org

- **WHEN** a webhook passes config lookup but fails signature validation
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'failed'` and the error message
- **AND** the row SHALL have `organization_id` set to the resolved organization
- **AND** the failure SHALL then be returned to the caller (HTTP 401)
