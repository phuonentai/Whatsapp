## ADDED Requirements

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
