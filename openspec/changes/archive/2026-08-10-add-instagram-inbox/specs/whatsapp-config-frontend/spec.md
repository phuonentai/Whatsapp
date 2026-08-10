## MODIFIED Requirements

### Requirement: WhatsApp section appears in settings overview

The workspace settings overview at `/dashboard/settings` SHALL display a "Messaging" (WhatsApp) card and an "Instagram" card, each visible to users with `org:manage` permission. The Instagram card SHALL link to the Instagram settings view (`/dashboard/settings?view=instagram`).

#### Scenario: User has org:manage and config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has an active WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", the connected phone number as the value, and "Active" as the status

#### Scenario: User has org:manage and no WhatsApp config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has no WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", value "Not connected", and helper text "Connect WhatsApp to receive messages"

#### Scenario: Instagram card mirrors WhatsApp card

- **WHEN** a user with `org:manage` permission views the settings overview
- **THEN** an "Instagram" card SHALL be displayed with the connected IG username (or "Not connected") and status, per the instagram-messaging spec
- **AND** clicking it SHALL navigate to the Instagram settings view

#### Scenario: User lacks org:manage permission

- **WHEN** a user without `org:manage` permission views the settings overview
- **THEN** the overview SHALL NOT display the WhatsApp or Instagram cards
