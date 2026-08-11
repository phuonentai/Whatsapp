# Spec: inbox-ui

## Purpose

Defines the inbox UI: replying in open conversations, filtering the conversation list by status, and quick replies with playbook guion text.

## Requirements

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
The system SHALL render an empty state when the conversation list has no conversations, SHALL surface an error toast when sending a reply fails, and SHALL NOT mutate the thread on failure. Empty and failure states SHALL render using the typed copy layer in Spanish-first voice, replacing the current English empty-state strings.

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

#### Scenario: Channel empty states render Spanish

- **WHEN** the conversation list is empty for a channel (WhatsApp or Instagram) or for all channels
- **THEN** the empty-state message SHALL be a Spanish string resolved from the copy layer that explains how to get started (connect the channel in Settings)

#### Scenario: Thread empty state renders Spanish

- **WHEN** an open conversation has no messages
- **THEN** the thread empty-state hint SHALL be a Spanish string resolved from the copy layer

### Requirement: Scripted sequence quick replies auto-advance in the composer
The quick-replies row SHALL render scripted-sequence guiones (those carrying a `pasos` array) as distinct pills showing the sequence title and its step count. Clicking a sequence pill SHALL start sequence mode: the composer SHALL be pre-filled with the first step's message; after the message is sent successfully, the composer SHALL be pre-filled with the next step's message; after the final step is sent, sequence mode SHALL end and the progress indicator SHALL disappear. A progress indicator SHALL show the current step ("Paso k de n") while sequence mode is active. The platform SHALL NOT auto-send any step; every step SHALL be sent by the human via the existing conversation send path. Sequence mode SHALL reset when the selected conversation changes. Single-shot guiones SHALL keep the current one-click fill behavior unchanged.

#### Scenario: Clicking a sequence pill fills the first step
- **WHEN** an applied playbook exposes a sequence guion and the user clicks its pill
- **THEN** the reply input SHALL be pre-filled with the first step's message
- **AND** a progress indicator SHALL show "Paso 1 de n"

#### Scenario: Sending a step auto-advances to the next
- **WHEN** sequence mode is active and the user sends the current step's message successfully
- **THEN** the reply input SHALL be pre-filled with the next step's message
- **AND** the progress indicator SHALL update to the next step number

#### Scenario: Sending the final step ends sequence mode
- **WHEN** sequence mode is active and the user sends the last step's message successfully
- **THEN** sequence mode SHALL end and the progress indicator SHALL disappear

#### Scenario: Sequence steps are never auto-sent
- **WHEN** a sequence is active
- **THEN** no step SHALL be sent without an explicit human send action

#### Scenario: Changing conversation resets sequence mode
- **WHEN** sequence mode is active and the user selects a different conversation
- **THEN** sequence mode SHALL reset and no further step SHALL be pre-filled

#### Scenario: Failed send does not advance the sequence
- **WHEN** the user sends a step and the send fails
- **THEN** the sequence SHALL NOT advance and the same step SHALL remain pre-filled

### Requirement: User filters the conversation list by channel

The system SHALL render channel filter tabs (All / WhatsApp / Instagram) on the inbox page, and SHALL reload conversations scoped to the selected channel by passing `channel` to the conversation list query. The selected channel SHALL be reflected in the URL query parameter (`?channel=all|whatsapp|instagram`), and the filter SHALL default to All.

#### Scenario: Channel filter narrows the list

- **WHEN** user selects the "WhatsApp" tab
- **THEN** the conversation list shows only conversations with `channel = 'whatsapp'`

#### Scenario: Instagram tab shows only Instagram conversations

- **WHEN** user selects the "Instagram" tab
- **THEN** the conversation list shows only conversations with `channel = 'instagram'`

#### Scenario: Default filter shows all channels

- **WHEN** the inbox page first loads without a channel query parameter
- **THEN** the conversation list query SHALL be made without a channel filter, showing conversations of all channels

#### Scenario: Channel filter persists across refresh

- **WHEN** a channel tab is selected
- **THEN** the URL query parameter SHALL be set so a page refresh keeps the selected channel

#### Scenario: Empty per-channel inbox shows channel-specific empty state

- **WHEN** an organization has no conversations for the selected channel
- **THEN** the system SHALL display an empty state referencing the channel (e.g., "No Instagram messages yet — connect Instagram in Settings to get started") with a link to the settings page

### Requirement: Conversation items render channel identity

The system SHALL render each conversation item with its channel: Instagram conversations SHALL show the contact's IG username (or display name fallback) and avatar image when available; WhatsApp conversations SHALL retain the phone-number-first display. A channel icon SHALL distinguish Instagram from WhatsApp items.

#### Scenario: Instagram conversation shows username and avatar

- **WHEN** a conversation with `channel = 'instagram'` and a contact with `instagram_username` and `avatar_url` is listed
- **THEN** the item SHALL display the username (falling back to display name) and render the avatar image

#### Scenario: WhatsApp conversation shows phone

- **WHEN** a conversation with `channel = 'whatsapp'` is listed
- **THEN** the item SHALL display the contact's phone number or display name as today

#### Scenario: Thread header shows channel badge

- **WHEN** a conversation is selected in the thread header
- **THEN** the header SHALL display a channel badge (Instagram or WhatsApp) alongside the contact identity

### Requirement: WhatsApp delivery ticks render only on WhatsApp threads

The message thread SHALL render the delivery status ticks (`✓` / `✓✓`) only for WhatsApp-channel messages; Instagram messages SHALL render delivery status without the tick glyphs.

#### Scenario: Ticks on WhatsApp messages

- **WHEN** a WhatsApp-channel outbound message is rendered
- **THEN** its status SHALL be shown with the existing tick glyphs

#### Scenario: No ticks on Instagram messages

- **WHEN** an Instagram-channel outbound message is rendered
- **THEN** the status SHALL be shown without tick glyphs

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
