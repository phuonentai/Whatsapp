# Delta Spec: inbox-ui — ux-error-recovery

## MODIFIED Requirements

### Requirement: User sends a reply in an open conversation
The system SHALL send a reply typed into the reply input of the selected conversation via `useSendMessage`, append it to the message thread, and clear the input on success. An empty or whitespace-only draft SHALL NOT be sent. On failure, the system SHALL show an error toast, SHALL keep the draft in the input, SHALL NOT append the message to the thread, and SHALL NOT produce an unhandled promise rejection.

#### Scenario: Typed reply appears in thread
- **WHEN** user opens a conversation and types a reply then submits it
- **THEN** the message appears in the message thread for that conversation

#### Scenario: Empty reply is not sent
- **WHEN** user submits an empty or whitespace-only reply input
- **THEN** no message is sent and no request reaches the messaging API

#### Scenario: Failed reply preserves the draft
- **WHEN** user sends a reply and the request fails
- **THEN** an error toast SHALL be shown and the draft SHALL remain in the input

#### Scenario: Failed reply does not append to the thread
- **WHEN** user sends a reply and the request fails
- **THEN** no message SHALL appear in the thread and the rejection SHALL be handled (no unhandled promise rejection)

## ADDED Requirements

### Requirement: Conversation list shows unread indicators

The conversation list SHALL render an unread indicator for conversations with new inbound messages not yet opened by the current user, and SHALL clear it once the conversation is opened or a reply is sent.

#### Scenario: New inbound message marks item unread

- **WHEN** a conversation receives a new inbound message and the user has not opened it
- **THEN** the conversation item SHALL display an unread indicator distinct from the pending-suggestion badge

#### Scenario: Opening the conversation clears the indicator

- **WHEN** the user opens a conversation that has an unread indicator
- **THEN** the indicator SHALL clear for that conversation

#### Scenario: Sending a reply clears the indicator

- **WHEN** the user sends a reply in a conversation with an unread indicator
- **THEN** the indicator SHALL clear after the reply succeeds
