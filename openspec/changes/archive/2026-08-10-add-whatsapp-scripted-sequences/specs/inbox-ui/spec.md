## ADDED Requirements

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
