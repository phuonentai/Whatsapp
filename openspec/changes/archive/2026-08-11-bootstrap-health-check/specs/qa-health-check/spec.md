## ADDED Requirements

### Requirement: Health check verification procedure

The repository SHALL provide a documented, credential-free health check verification workflow that confirms the health endpoint behaves as specified before the stack is exercised.

#### Scenario: Health endpoint responds OK

- **WHEN** `GET /healthz` is requested from `go-b2b-starter/cmd/mock-siigo/main.go`
- **THEN** the endpoint SHALL return HTTP 200 with a JSON body of `{"status":"ok"}`

#### Scenario: QA server waits for health before use

- **WHEN** `scripts/run-qa-server.sh` starts the dev server
- **THEN** the script SHALL be executable
- **AND** it SHALL poll candidate health URLs (including `/healthz`) until one responds before declaring the server ready

#### Scenario: Bootstrap task declares low risk

- **WHEN** the bootstrap health-check change is evaluated
- **THEN** its `routing.json` SHALL declare `requires_council`, `requires_playwright`, `requires_migration`, `requires_feature_flag`, `security_impact`, `data_model_change`, and `payment_impact` as `false`, and `complexity` as `low`
