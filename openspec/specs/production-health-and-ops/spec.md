## Purpose

Defines production health: a dependency-probing health endpoint, compose migration job, and container healthchecks.
## Requirements
### Requirement: Health endpoint probes dependencies

The system SHALL expose a health endpoint whose production behavior verifies connectivity to Postgres and Redis, and SHALL report readiness separately from liveness.

#### Scenario: All dependencies healthy

- **WHEN** Postgres and Redis are reachable
- **THEN** `/health` SHALL return HTTP 200 with status indicating ready

#### Scenario: Dependency down reports unhealthy

- **WHEN** Postgres or Redis is unreachable
- **THEN** `/health` SHALL return HTTP 503
- **AND** SHALL include which dependency failed

#### Scenario: Liveness remains available

- **WHEN** dependencies are down but the process is running
- **THEN** the liveness endpoint SHALL still return HTTP 200 so orchestrators do not kill a process mid-recovery

### Requirement: Production compose migration job

The production deployment SHALL apply database migrations as a one-shot job before the backend starts, rather than requiring manual migration.

#### Scenario: Migrate job runs before backend

- **WHEN** `docker compose -f docker-compose.production.yml up` runs
- **THEN** the `migrate` service SHALL apply pending migrations and exit
- **AND** the backend SHALL depend on the migration job completing successfully

#### Scenario: Migration failure blocks startup

- **WHEN** a migration fails
- **THEN** the backend SHALL NOT start
- **AND** the failure SHALL be visible in the migrate service logs

### Requirement: Container healthchecks in production compose

The production compose file SHALL define healthchecks for backend and Redis in addition to Postgres.

#### Scenario: Backend healthcheck

- **WHEN** the backend container is running
- **THEN** its healthcheck SHALL probe the backend health endpoint and mark the container unhealthy when the probe fails

#### Scenario: Redis healthcheck

- **WHEN** the Redis container is running
- **THEN** its healthcheck SHALL probe `redis-cli ping` and mark the container unhealthy when it fails

#### Scenario: Startup ordering uses health

- **WHEN** the backend depends on Redis and Postgres
- **THEN** the dependencies SHALL use `condition: service_healthy` so the backend waits for healthy dependencies

### Requirement: Secrets hygiene for tracked environment files

The repository SHALL NOT track backup or plaintext secret-bearing environment files.

#### Scenario: Env backup files are ignored

- **WHEN** a file matching `*.env.bak` or similar backup env patterns is created
- **THEN** the repository SHALL ignore it via `.gitignore`

#### Scenario: No secret-bearing env file in history

- **WHEN** the repository is scanned for tracked env files containing secret-shaped values
- **THEN** the scan SHALL find none
- **AND** CI SHALL fail if one is introduced

### Requirement: Response compression

The backend SHALL compress HTTP responses with gzip when the client indicates support via `Accept-Encoding` and the response content type is compressible. Streaming responses MUST NOT be buffered or compressed.

#### Scenario: Compressible response is gzipped

- **WHEN** a client sends `Accept-Encoding: gzip`
- **AND** the endpoint returns a JSON or text response
- **THEN** the response SHALL carry `Content-Encoding: gzip`

#### Scenario: SSE streaming response is not compressed

- **WHEN** a request targets the SSE chat endpoint
- **THEN** the response SHALL NOT be gzip-compressed
- **AND** tokens SHALL stream without buffering

#### Scenario: Client without gzip support

- **WHEN** a client does not send `Accept-Encoding: gzip`
- **THEN** the response SHALL be returned uncompressed

### Requirement: Prometheus metrics endpoint

The backend SHALL expose a `/metrics` endpoint serving Prometheus text format, including an HTTP request counter and a request latency histogram.

#### Scenario: Metrics endpoint responds

- **WHEN** `/metrics` is requested
- **THEN** it SHALL return HTTP 200 in Prometheus text format
- **AND** SHALL include the `http_requests_total` counter and `http_request_duration_seconds` histogram

#### Scenario: Metrics exclude streaming endpoints

- **WHEN** a request hits the SSE chat endpoint or `/metrics` itself
- **THEN** that request SHALL NOT inflate the request histogram

### Requirement: Profiling endpoint gated by configuration

The backend SHALL register pprof handlers under `/debug/pprof` only when explicitly enabled by configuration. The default SHALL be disabled.

#### Scenario: Profiling enabled

- **WHEN** `PPROF_ENABLED` is true
- **THEN** `/debug/pprof/` SHALL be registered and return HTTP 200

#### Scenario: Profiling disabled by default

- **WHEN** `PPROF_ENABLED` is unset or false
- **THEN** `/debug/pprof` SHALL NOT be registered

