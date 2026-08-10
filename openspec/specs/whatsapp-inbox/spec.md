## Purpose

Defines the WhatsApp inbox UI: conversation list, message thread view, and conversation status management.

## Requirements

### Requirement: Inbox page displays conversation list

The system SHALL provide an inbox page at `/dashboard/inbox` that displays a list of conversations scoped to the organization, ordered by most recent message.

#### Scenario: Inbox loads conversation list

- **WHEN** a user with `org:manage` permission navigates to `/dashboard/inbox`
- **THEN** the system SHALL fetch `GET /crm/conversaciones` and display conversations in a scrollable list
- **AND** each conversation item SHALL show the contact's phone number (or display name), last message preview (truncated to 60 chars), and relative timestamp
- **AND** active conversations SHALL be visually distinguished from closed/archived ones

#### Scenario: Empty inbox

- **WHEN** an organization has no conversations
- **THEN** the inbox SHALL display an empty state with text "No messages yet — connect WhatsApp in Settings to get started" and a link to the settings page

#### Scenario: Loading state

- **WHEN** the conversation list query is loading
- **THEN** the system SHALL display skeleton placeholders for 5 conversation items

#### Scenario: Error state

- **WHEN** the conversation list query fails
- **THEN** the system SHALL display an error message with a "Retry" button

### Requirement: Message thread view for selected conversation

When a conversation is selected from the inbox list, the system SHALL display the message thread with a reply input at the bottom.

#### Scenario: Selecting a conversation loads its messages

- **WHEN** a user clicks on a conversation in the inbox list
- **THEN** the system SHALL fetch `GET /crm/conversaciones/:id/mensajes` and display messages in chronological order
- **AND** inbound messages SHALL be left-aligned with a gray background
- **AND** outbound messages SHALL be right-aligned with a blue background

#### Scenario: Message types render correctly

- **WHEN** the message thread contains messages of different types
- **THEN** text messages SHALL display the content as plain text
- **AND** media messages (image, video, audio, document) SHALL display a placeholder icon with the media type label and a link to the media URL if available
- **AND** location messages SHALL display a "Location: lat,lng" text

#### Scenario: Reply input sends a message

- **WHEN** a user types text in the reply input and presses Enter or clicks Send
- **THEN** the system SHALL send `POST /crm/conversaciones/:id/mensajes` with the message content
- **AND** on success, SHALL append the sent message to the thread
- **AND** if the API returns an error, SHALL display an inline error toast

#### Scenario: Reply input is disabled when no conversation is selected

- **WHEN** no conversation is selected
- **THEN** the reply input SHALL be disabled with placeholder text "Select a conversation to reply"

#### Scenario: Polling refreshes the message thread

- **WHEN** a conversation is selected and the message thread is visible
- **THEN** the system SHALL poll `GET /crm/conversaciones/:id/mensajes` every 5 seconds
- **AND** new inbound messages SHALL appear in the thread without manual refresh

### Requirement: Conversation status management from inbox

The system SHALL allow changing a conversation's status from the inbox message thread header.

#### Scenario: Close conversation from thread header

- **WHEN** a user clicks the "Close" action in the conversation thread header
- **THEN** the system SHALL send `PATCH /crm/conversaciones/:id/status` with `{"status": "closed"}`
- **AND** on success, the conversation status badge SHALL update and the conversation SHALL move to the closed section in the list

#### Scenario: Reopen conversation from thread header

- **WHEN** a user clicks the "Reopen" action on a closed conversation
- **THEN** the system SHALL send `PATCH /crm/conversaciones/:id/status` with `{"status": "active"}`
- **AND** the conversation SHALL reappear in the active section of the list

### Requirement: Inbox navigation in sidebar

The sidebar SHALL display an "Inbox" navigation entry visible to users with `org:manage` permission.

#### Scenario: User has org:manage permission

- **WHEN** a user with `org:manage` permission views the sidebar
- **THEN** an "Inbox" entry SHALL be visible with a MessageCircle icon
- **AND** clicking it SHALL navigate to `/dashboard/inbox`

#### Scenario: User lacks org:manage permission

- **WHEN** a user without `org:manage` permission views the sidebar
- **THEN** the "Inbox" entry SHALL NOT be displayed

### Requirement: Conversation list filters by status

The inbox SHALL allow filtering conversations by status (all, active, closed, archived).

#### Scenario: Filter by active

- **WHEN** a user selects the "Active" filter tab
- **THEN** the system SHALL fetch `GET /crm/conversaciones?status=active` and display only active conversations

#### Scenario: Default to all conversations

- **WHEN** the inbox page first loads
- **THEN** the system SHALL fetch conversations without a status filter, showing all statuses
