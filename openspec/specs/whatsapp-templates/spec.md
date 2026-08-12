## Purpose

Defines the org-scoped WhatsApp message template registry: CRUD lifecycle (draft/submitted/approved/rejected/paused), submission to Meta, approval status sync via webhook, manual refresh, and template message sending.

## Requirements


### Requirement: Org-scoped WhatsApp template registry

The system SHALL maintain an org-scoped registry of WhatsApp message templates in `whatsapp.templates` with columns `id`, `organization_id` (FK), `name`, `category`, `language`, `body` (containing `{{N}}` placeholders), `param_count`, `status` (`draft | submitted | approved | rejected | paused`), `meta_template_id` (nullable, set after Meta submission), `rejection_reason` (nullable), `is_active`, `created_at`, and `updated_at`. The registry SHALL enforce uniqueness on `(organization_id, name, language)` matching Meta's constraint and SHALL compute `param_count` from the `{{N}}` placeholders in `body`. Creating, updating, and deleting templates SHALL require the `org:manage` permission; listing SHALL require `org:view`.

#### Scenario: Create a draft template

- **WHEN** a user with `org:manage` permission sends `POST /api/whatsapp/templates` with `{"name": "confirmacion_pedido", "category": "UTILITY", "language": "es", "body": "Hola {{1}}, tu pedido {{2}} fue confirmado."}`
- **THEN** the system SHALL create a row in `whatsapp.templates` scoped to the organization with `status = 'draft'`, `param_count = 2`, and `is_active = true`
- **AND** SHALL return HTTP 200 with the created template

#### Scenario: Validation failure returns a Spanish error

- **WHEN** a user sends `POST /api/whatsapp/templates` with an empty `name`, `category`, `language`, or `body`
- **THEN** the system SHALL return HTTP 400 with a Spanish validation message (e.g., "El nombre de la plantilla es obligatorio")

#### Scenario: Duplicate name and language is rejected

- **WHEN** a user sends `POST /api/whatsapp/templates` with a `name` and `language` combination that already exists for the organization
- **THEN** the system SHALL return HTTP 409 with error code `template_name_conflict`

#### Scenario: List is org-scoped

- **WHEN** a user with `org:view` permission sends `GET /api/whatsapp/templates`
- **THEN** the system SHALL return only the templates belonging to the user's organization

#### Scenario: Update a draft template

- **WHEN** a user with `org:manage` permission sends `PATCH /api/whatsapp/templates/:id` with updated `name`, `category`, `language`, or `body`
- **AND** the template has `status = 'draft'`
- **THEN** the system SHALL update the editable fields and recompute `param_count` from the new `body`
- **AND** SHALL return HTTP 200 with the updated template

#### Scenario: Update is rejected for non-draft templates

- **WHEN** a user with `org:manage` permission sends `PATCH /api/whatsapp/templates/:id` for a template whose status is `submitted`, `approved`, `rejected`, or `paused`
- **THEN** the system SHALL return HTTP 409 with error code `template_not_draft`

#### Scenario: Delete a draft template

- **WHEN** a user with `org:manage` permission sends `DELETE /api/whatsapp/templates/:id` for a template with `status = 'draft'`
- **THEN** the system SHALL delete the template and return HTTP 200

#### Scenario: Delete is rejected for approved templates

- **WHEN** a user with `org:manage` permission sends `DELETE /api/whatsapp/templates/:id` for a template with `status = 'approved'` or `submitted`
- **THEN** the system SHALL return HTTP 409 with error code `template_not_draft`
- **AND** SHALL keep the template (pause instead of delete)

### Requirement: Template submission to Meta

The system SHALL submit a draft template to Meta by calling `POST {graph_api_url}/{api_version}/{phone_number_id}/message_templates` using the organization's stored WhatsApp config credentials (`access_token`, `phone_number_id`), SHALL store the returned `meta_template_id`, and SHALL set the local status to `submitted`. Submission SHALL be idempotent: re-submitting a template already in `submitted` status SHALL return the current state without making a second Meta call. Submission SHALL require the `org:manage` permission and SHALL NOT introduce local credential storage.

#### Scenario: Successful submission

- **WHEN** a user with `org:manage` permission sends `POST /api/whatsapp/templates/42/submit` for a draft template with name `confirmacion_pedido`, language `es`, category `UTILITY`, and a body with 2 parameters
- **AND** the organization has an active WhatsApp config with valid `access_token` and `phone_number_id`
- **THEN** the system SHALL call `POST {graph_api_url}/{api_version}/{phone_number_id}/message_templates` with the template name, language, category, and body components
- **AND** on success SHALL store the returned template ID in `meta_template_id`, set `status = 'submitted'`, and return HTTP 200 with the updated template

#### Scenario: Re-submission is idempotent

- **WHEN** a user sends `POST /api/whatsapp/templates/42/submit` for a template whose status is already `submitted`
- **THEN** the system SHALL NOT call the Meta API again
- **AND** SHALL return HTTP 200 with the current template state

#### Scenario: WhatsApp config missing or inactive

- **WHEN** a user submits a template but the organization has no WhatsApp config or `is_active = false`
- **THEN** the system SHALL return HTTP 400 with error code `whatsapp_not_configured`

#### Scenario: Meta API error during submission

- **WHEN** the Meta API returns an error response during submission
- **THEN** the system SHALL NOT change the local template status or `meta_template_id`
- **AND** SHALL return HTTP 502 with error code `whatsapp_api_error` and the Meta error details

### Requirement: Template approval status sync via webhook

The existing WhatsApp webhook ingress (`POST /api/v1/webhooks/whatsapp`, spec `whatsapp-webhook-ingress`) SHALL process Meta's `message_template_status_update` webhook field: it SHALL look up the template by `meta_template_id` and the resolved organization, SHALL update the local status (`approved`, `rejected`, or `paused`), SHALL store `rejection_reason` when provided, and SHALL record the audit event `template_status_changed`. The update SHALL be idempotent using a transaction-isolated state check (re-applying an identical status SHALL be a no-op) and SHALL NOT create `whatsapp.message.received` events. A status update for an unknown `meta_template_id` SHALL be logged and ignored without returning an error.

#### Scenario: Template approved

- **WHEN** a `message_template_status_update` webhook arrives with `event = "APPROVED"` for a `meta_template_id` matching a template of the resolved organization
- **THEN** the system SHALL set the template's local `status = 'approved'` and `is_active = true`
- **AND** SHALL record the `template_status_changed` audit event

#### Scenario: Template rejected with reason

- **WHEN** a `message_template_status_update` webhook arrives with `event = "REJECTED"` and a `reason` for a known template
- **THEN** the system SHALL set `status = 'rejected'`, store the reason in `rejection_reason`, and set `is_active = false`
- **AND** SHALL record the audit event including the reason

#### Scenario: Duplicate status update is a no-op

- **WHEN** a webhook redelivers a status update whose target status equals the template's current local status
- **THEN** the system SHALL NOT modify the template row and SHALL NOT create a duplicate audit event
- **AND** SHALL return HTTP 200

#### Scenario: Unknown template is ignored

- **WHEN** a `message_template_status_update` webhook arrives with a `meta_template_id` that does not match any template of the resolved organization
- **THEN** the system SHALL log the event for audit
- **AND** SHALL return HTTP 200 without error

### Requirement: Manual template status refresh

The system SHALL provide `POST /api/whatsapp/templates/:id/sync` that fetches the template's current status from Meta (using the organization's stored WhatsApp config) and updates the local row to match, including `status`, `rejection_reason`, and `meta_template_id`. Refresh SHALL require the `org:manage` permission, SHALL be idempotent, and SHALL reconcile local/Meta drift.

#### Scenario: Refresh pulls the latest status

- **WHEN** a user with `org:manage` permission sends `POST /api/whatsapp/templates/42/sync`
- **AND** Meta reports the template as `APPROVED`
- **THEN** the system SHALL set the local status to `approved` and return HTTP 200 with the updated template

#### Scenario: Template missing at Meta

- **WHEN** a user syncs a template that no longer exists at Meta
- **THEN** the system SHALL return HTTP 404 with error code `template_not_found_at_meta`
- **AND** SHALL NOT change the local status

### Requirement: Template message send endpoint

The system SHALL expose `POST /crm/conversaciones/:id/mensajes/template` accepting `{"template_id": <id>, "params": ["<string>", ...]}` that sends a template message via the WhatsApp Cloud API using a `type: "template"` payload. The endpoint SHALL resolve the template scoped to the organization, SHALL require local `status = 'approved'` and `is_active = true`, SHALL require `len(params)` to equal the template's `param_count`, SHALL send through the circuit-breakered client's `SendTemplateMessage`, and SHALL persist the outbound message in `crm.messages` with `direction = 'outbound'`, `status = 'sent'`, and `whatsapp_message_id`. Template sends SHALL NOT be subject to the 24-hour messaging window. Sending SHALL require the `org:manage` permission and SHALL reuse the organization's stored WhatsApp config credentials without introducing local credential storage.

#### Scenario: Send an approved template outside the 24-hour window

- **WHEN** a user with `org:manage` permission sends `POST /crm/conversaciones/42/mensajes/template` with `{"template_id": 7, "params": ["María", "Pedido #1234"]}`
- **AND** template 7 belongs to the organization, has `status = 'approved'`, `is_active = true`, and `param_count = 2`
- **AND** the conversation's `last_message_at` is more than 24 hours in the past
- **THEN** the system SHALL send the Cloud API payload `{"messaging_product": "whatsapp", "recipient_type": "individual", "to": "<contact_phone>", "type": "template", "template": {"name": "confirmacion_pedido", "language": {"policy": "deterministic", "code": "es"}, "components": [{"type": "body", "parameters": [{"type": "text", "text": "María"}, {"type": "text", "text": "Pedido #1234"}]}]}}`
- **AND** on success SHALL persist the message in `crm.messages` with `direction = 'outbound'`, `status = 'sent'`, and `whatsapp_message_id` set to the returned message ID
- **AND** SHALL return HTTP 200 with the created message and WITHOUT the `outside_24h_window` warning

#### Scenario: Unknown or foreign template

- **WHEN** a user sends a template message with a `template_id` that does not exist or belongs to another organization
- **THEN** the system SHALL return HTTP 404 with error code `template_not_found`

#### Scenario: Template not approved

- **WHEN** a user sends a template message for a template whose local status is not `approved` (e.g., `draft`, `submitted`, `rejected`, or `paused`)
- **THEN** the system SHALL return HTTP 400 with error code `template_not_approved`

#### Scenario: Parameter count mismatch

- **WHEN** a user sends a template message whose `params` array length differs from the template's `param_count`
- **THEN** the system SHALL return HTTP 400 with error code `template_param_count_mismatch` and a Spanish error message

#### Scenario: WhatsApp API error during template send

- **WHEN** the WhatsApp Cloud API returns an error response during a template send
- **THEN** the system SHALL NOT persist the message
- **AND** SHALL return HTTP 502 with error code `whatsapp_api_error` and the API error details

#### Scenario: Rate limit exceeded

- **WHEN** the organization exceeds the send rate limit (10 messages per 10 seconds)
- **THEN** the system SHALL return HTTP 429 with error code `rate_limit`

### Requirement: Template routes are RBAC-protected and org-scoped

The system SHALL register the template routes (`POST /api/whatsapp/templates`, `PATCH /api/whatsapp/templates/:id`, `DELETE /api/whatsapp/templates/:id`, `POST /api/whatsapp/templates/:id/submit`, `POST /api/whatsapp/templates/:id/sync`, `GET /api/whatsapp/templates`, and `POST /crm/conversaciones/:id/mensajes/template`) behind the existing `auth` + `org_context` + `subscription` middleware chain. Write operations SHALL require the `org:manage` permission; the list operation SHALL require `org:view`. All operations SHALL operate exclusively within the resolved organization.

#### Scenario: Manage permission required for writes

- **WHEN** a user without the `org:manage` permission attempts to create, update, delete, submit, sync, or send a template
- **THEN** the system SHALL return HTTP 403

#### Scenario: View permission allows listing

- **WHEN** a user with the `org:view` permission (and without `org:manage`) sends `GET /api/whatsapp/templates`
- **THEN** the system SHALL return the organization's templates

#### Scenario: View permission denied for writes

- **WHEN** a user with only the `org:view` permission attempts to create or update a template
- **THEN** the system SHALL return HTTP 403

#### Scenario: Unauthenticated request

- **WHEN** a request arrives at a template route without a valid session token
- **THEN** the system SHALL return HTTP 401
