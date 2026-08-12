## ADDED Requirements

### Requirement: Branding configuration surface in settings

The system SHALL provide a "Marca / Documentos" section in dashboard settings where `org:manage` members configure document branding: logo upload, colors, letterhead, terms footer, validity days, default IVA, and numbering prefix, with a live preview of a sample document header.

#### Scenario: Member accesses branding settings
- **WHEN** a member navigates to the branding section in settings
- **THEN** the section SHALL display the current branding configuration with an editable form and a sample document header preview

#### Scenario: Branding saved from settings
- **WHEN** an `org:manage` member edits branding fields and saves
- **THEN** the settings UI SHALL call the branding API
- **AND** SHALL reflect the saved configuration and updated preview
