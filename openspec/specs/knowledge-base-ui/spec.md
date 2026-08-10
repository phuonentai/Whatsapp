# Spec: knowledge-base-ui

## Purpose

Defines the knowledge base UI: PDF uploads, document listing, and chat over uploaded knowledge base content.

## Requirements

### Requirement: User uploads a PDF document to the knowledge base
The system SHALL accept a PDF file via the dropzone, upload it through the configured upload handler, and SHALL display the uploaded document in the document list. Non-PDF files SHALL be rejected with an error message and SHALL NOT be uploaded.

#### Scenario: PDF upload adds document to list
- **WHEN** user selects a PDF file and confirms the upload
- **THEN** the document appears in the knowledge-base document list

#### Scenario: Non-PDF file is rejected
- **WHEN** user selects a non-PDF file (e.g. `.txt`)
- **THEN** an error message is shown and no upload request is made

### Requirement: Knowledge base lists uploaded documents
The system SHALL render the list of documents belonging to the workspace, showing the uploaded titles.

#### Scenario: Uploaded document title is visible
- **WHEN** a document exists for the workspace and user views the knowledge base
- **THEN** the document title is visible in the document list

### Requirement: User sends a chat message in the knowledge base
The system SHALL accept a chat message in the chat interface and SHALL append the user's message to the chat thread. An empty message SHALL NOT be sent.

#### Scenario: Sent message appears in chat thread
- **WHEN** user types a message and submits it in the chat interface
- **THEN** the message is appended to the chat thread

#### Scenario: Empty chat message is not sent
- **WHEN** user submits an empty chat message
- **THEN** no message is appended and no request is made

### Requirement: Knowledge base handles empty and failure states
The system SHALL render an empty state when the workspace has no documents, and SHALL surface an upload error message without adding a document when the upload API fails.

#### Scenario: Empty document list renders empty state
- **WHEN** a workspace has no documents and user opens the knowledge base
- **THEN** an empty-state message is shown instead of a list

#### Scenario: Failed upload surfaces an error
- **WHEN** user uploads a valid PDF and the upload API responds with a server error (5xx)
- **THEN** an error message is shown and no document is added to the list
