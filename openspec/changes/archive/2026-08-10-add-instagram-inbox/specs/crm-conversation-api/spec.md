## MODIFIED Requirements

### Requirement: Conversation list API returns paginated conversations

The system SHALL expose a `GET /crm/conversaciones` endpoint that returns conversations scoped to the authenticated organization, ordered by `last_message_at` descending, with contact display name and last message preview. The endpoint SHALL accept an optional `channel` query parameter (`whatsapp`, `instagram`, or empty for all channels) and SHALL return the `channel` on each conversation.

#### Scenario: List conversations with results

- **WHEN** an authenticated user requests `GET /crm/conversaciones?page=1&page_size=20`
- **THEN** the system SHALL return HTTP 200 with a JSON object containing `conversations` (array) and `pagination` (page, page_size, total, total_pages)
- **AND** each conversation SHALL include `id`, `channel`, `contact` (id, phone_number, display_name, instagram_username, avatar_url), `status`, `last_message_at`, `last_message_preview` (content truncated to 100 chars), and `created_at`

#### Scenario: Empty conversation list

- **WHEN** the organization has no conversations
- **THEN** the system SHALL return HTTP 200 with an empty `conversations` array and `total: 0`

#### Scenario: Unauthenticated request

- **WHEN** a request arrives without a valid auth token
- **THEN** the system SHALL return HTTP 401

#### Scenario: Conversations are org-scoped

- **WHEN** an authenticated user requests their organization's conversations
- **THEN** the system SHALL only return conversations where `organization_id` matches the authenticated user's organization

#### Scenario: Filter conversations by status

- **WHEN** an authenticated user requests `GET /crm/conversaciones?status=active`
- **THEN** the system SHALL return only conversations with `status = 'active'`

#### Scenario: Filter conversations by channel

- **WHEN** an authenticated user requests `GET /crm/conversaciones?channel=instagram`
- **THEN** the system SHALL return only conversations with `channel = 'instagram'`
- **AND** when `channel=whatsapp`, SHALL return only conversations with `channel = 'whatsapp'`
- **AND** when `channel` is absent or empty, SHALL return conversations of all channels

### Requirement: Message list API returns paginated messages for a conversation

The system SHALL expose a `GET /crm/conversaciones/:id/mensajes` endpoint returning messages for a conversation, scoped to the organization, paginated and ordered by `created_at` ascending (oldest first).

#### Scenario: List messages in conversation

- **WHEN** an authenticated user requests `GET /crm/conversaciones/42/mensajes?page=1&page_size=50`
- **AND** conversation 42 belongs to the user's organization
- **THEN** the system SHALL return HTTP 200 with a JSON object containing `messages` (array) and `pagination`
- **AND** each message SHALL include `id`, `channel`, `direction` (inbound/outbound), `message_type`, `content`, `status`, `provider_message_id`, `chat_timestamp`, and `created_at`

#### Scenario: Conversation not found

- **WHEN** a conversation ID does not exist or belongs to a different organization
- **THEN** the system SHALL return HTTP 404 with error code `conversation_not_found`

#### Scenario: Empty message list

- **WHEN** the conversation exists but has no messages
- **THEN** the system SHALL return HTTP 200 with an empty `messages` array and `total: 0`

### Requirement: Conversation status update API

The system SHALL expose a `PATCH /crm/conversaciones/:id/status` endpoint to change a conversation's status.

#### Scenario: Close an active conversation

- **WHEN** an authenticated user sends `PATCH /crm/conversaciones/42/status` with body `{"status": "closed"}`
- **AND** conversation 42 belongs to the user's organization
- **THEN** the system SHALL update the conversation status to `closed`
- **AND** return HTTP 200 with the updated conversation

#### Scenario: Reopen a closed conversation

- **WHEN** an authenticated user sends `PATCH /crm/conversaciones/42/status` with body `{"status": "active"}`
- **AND** the conversation exists and belongs to the user's organization
- **THEN** the system SHALL update the conversation status to `active`
- **AND** return HTTP 200 with the updated conversation

#### Scenario: Invalid status value

- **WHEN** a user sends a status value not in `[active, closed, archived]`
- **THEN** the system SHALL return HTTP 400 with a validation error message

### Requirement: Conversation routes are registered under /crm

The system SHALL register conversation endpoints under the existing `/crm` route group with the same auth, org_context, and feature middleware.

#### Scenario: Conversation endpoints are reachable

- **WHEN** the API server starts
- **THEN** `GET /crm/conversaciones`, `GET /crm/conversaciones/:id/mensajes`, and `PATCH /crm/conversaciones/:id/status` SHALL be reachable with auth middleware applied
- **AND** `GET /crm/conversaciones/:id/mensajes` SHALL enforce organization scoping (messages only from the authenticated user's organization)
