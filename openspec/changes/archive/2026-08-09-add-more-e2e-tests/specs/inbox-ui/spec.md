# Spec: inbox-ui

## ADDED Requirements

### Requirement: User sends a reply in an open conversation
The system SHALL send a reply typed into the reply input of the selected conversation via `useSendMessage`, append it to the message thread, and clear the input on success. An empty or whitespace-only draft SHALL NOT be sent.

#### Scenario: Typed reply appears in thread
- **WHEN** user opens a conversation and types a reply then submits it
- **THEN** the message appears in the message thread for that conversation

#### Scenario: Empty reply is not sent
- **WHEN** user submits an empty or whitespace-only reply input
- **THEN** no message is sent and no request reaches the messaging API

### Requirement: User filters the conversation list by status
The system SHALL filter the conversation list by conversation status (active/closed) via the status filter control, and SHALL reload conversations scoped to the selected status.

#### Scenario: Status filter narrows the list
- **WHEN** user selects a status filter
- **THEN** the conversation list shows only conversations matching that status

### Requirement: Quick replies insert playbook guion text
The system SHALL render one pill per applied playbook guion for the workspace, and SHALL insert the selected guion's message text into the reply input when a pill is clicked. When no applied playbook guions exist, the quick-replies row SHALL NOT render.

#### Scenario: Clicking a guion pill fills the reply input
- **WHEN** an applied playbook with guiones exists and user clicks a quick-reply pill
- **THEN** the reply input is pre-filled with the guion message text

#### Scenario: No applied playbooks hides quick replies
- **WHEN** the workspace has no applied playbook guiones
- **THEN** the quick-replies row is not visible

### Requirement: User approves a pending agent suggestion
The system SHALL render pending agent suggestions for the selected conversation, allow the user to approve a pending suggestion, and SHALL remove it from the pending list on approval via `POST /api/agent/suggestions/:id/approve`.

#### Scenario: Approving a suggestion removes it from pending
- **WHEN** a pending suggestion exists for the conversation and user clicks approve
- **THEN** the suggestion is removed from the pending suggestions panel

#### Scenario: Rejecting a suggestion dismisses it
- **WHEN** a pending suggestion exists for the conversation and user clicks reject
- **THEN** the suggestion is dismissed from the pending suggestions panel

### Requirement: Non-admin members see no privileged inbox controls
The system SHALL hide quick-replies and the agent-suggestion panel from members without ORG_MANAGE/ORG_AGENT permission, and SHALL reject privileged inbox API calls from non-admin identities with 403.

#### Scenario: Member sees no quick replies
- **WHEN** a member identity opens a conversation in a workspace that has applied playbook guiones
- **THEN** the quick-replies row is not visible

#### Scenario: Member sees no suggestion panel
- **WHEN** a member identity opens a conversation that has a pending suggestion
- **THEN** the pending-suggestion panel is not visible

#### Scenario: Member approve call is rejected
- **WHEN** a member identity calls `POST /api/agent/suggestions/:id/approve`
- **THEN** the API responds 403 and the suggestion remains pending

### Requirement: Inbox handles empty and failure states
The system SHALL render an empty state when the conversation list has no conversations, SHALL surface an error toast when sending a reply fails, and SHALL NOT mutate the thread on failure.

#### Scenario: Empty conversation list renders empty state
- **WHEN** an org has no conversations and user opens the inbox
- **THEN** an empty-state message is shown instead of a list

#### Scenario: Failed reply surfaces an error toast
- **WHEN** user sends a reply and the messaging API responds with a server error (5xx)
- **THEN** an error toast is shown and the failed message is not appended to the thread

#### Scenario: Long reply sends intact
- **WHEN** user submits a reply containing several thousand characters
- **THEN** the full message appears in the thread un-truncated

#### Scenario: Unicode reply round-trips
- **WHEN** user submits a reply containing non-ASCII text
- **THEN** the text renders in the thread exactly as sent
