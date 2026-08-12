## ADDED Requirements

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
