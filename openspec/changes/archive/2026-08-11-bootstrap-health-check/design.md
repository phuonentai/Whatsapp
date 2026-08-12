# Design: Bootstrap Health Check Verification (Documentation-Only)

## Overview

Documentation-only change. The deliverable is a recorded verification procedure and this proposal/design/tasks/routing quartet; no runtime code changes are made.

## Existing Health Endpoint

- `GET /healthz` in `go-b2b-starter/cmd/mock-siigo/main.go` responds `200` with `{"status":"ok"}`.
- Frontend QA server wrapper `scripts/run-qa-server.sh` waits on `/health` (fallbacks: `/api/health`, `/healthz`, `/api/healthz`).

## Verification Procedure

1. Start the backend with `make server` (go-b2b-starter).
2. `curl -fsS http://localhost:PORT/healthz` returns HTTP 200 and body `{"status":"ok"}`.
3. Or start the QA wrapper: `scripts/run-qa-server.sh <port>` and confirm it reports "Server ready".

## Constraints

- No source code changes. No migrations. No feature flags. No external credentials.
- Multi-tenancy, auth, billing, and payment flows are out of scope by design.
