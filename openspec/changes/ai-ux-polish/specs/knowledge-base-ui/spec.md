# Delta Spec: knowledge-base-ui — ai-ux-polish

## MODIFIED Requirements

### Requirement: User sends a chat message in the knowledge base
The system SHALL accept a chat message in the chat interface and SHALL append the user's message to the chat thread. An empty message SHALL NOT be sent. Assistant responses SHALL stream via the existing SSE path and SHALL render as GitHub-flavored markdown (headings, lists, bold, italic, code, links) with HTML sanitized, and citation cards SHALL display the referenced documents' titles instead of raw internal ids.

#### Scenario: Sent message appears in chat thread
- **WHEN** user types a message and submits it in the chat interface
- **THEN** the message is appended to the chat thread

#### Scenario: Empty chat message is not sent
- **WHEN** user submits an empty chat message
- **THEN** no message is appended and no request is made

#### Scenario: Assistant response renders markdown
- **WHEN** the assistant response contains markdown structure
- **THEN** it SHALL render formatted (lists, bold, code) and SHALL NOT show raw syntax

#### Scenario: Citation shows the document title
- **WHEN** the assistant response references a workspace document
- **THEN** the citation card SHALL display the document's title, not a raw "Document #id" label

#### Scenario: Streaming text is announced via live region
- **WHEN** the assistant response is streaming
- **THEN** the message container SHALL be a live region (`aria-live="polite"`) announcing the accumulating text

## ADDED Requirements

### Requirement: Assistant messages offer copy action

Each assistant message SHALL provide a copy button that copies the message text to the clipboard.

#### Scenario: Copy button copies the response

- **WHEN** user clicks the copy button on an assistant message
- **THEN** the full message text SHALL be copied to the clipboard and a confirmation SHALL be shown
