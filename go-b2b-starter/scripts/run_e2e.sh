#!/bin/bash

# File: scripts/run_e2e.sh
# One-command offline e2e runner (invoked via `make test-e2e`).
#
# Requires:
#   - Docker (root docker-compose.yml: postgres + redis + migrate services)
#   - Playwright chromium installed: pnpm --dir ../next_b2b_starter exec playwright install chromium
#
# Boots the canonical offline test environment:
#   backend  :8080, frontend :3001, DB saas_db_test, AUTH_MOCK_ENABLED=true

set -euo pipefail

COMPOSE_FILE="../docker-compose.yml"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-saas_db_test}"
MIGRATIONS_DIR="internal/db/postgres/sqlc/migrations"
FRONTEND_DIR="../next_b2b_starter"
MOCK_SIIGO_ADDR=":8090"
MOCK_SIIGO_URL="http://localhost:8090"
BACKEND_PID=""
FRONTEND_PID=""
MOCK_SIIGO_PID=""

cleanup() {
  echo "Cleaning up spawned processes..."
  [ -n "$BACKEND_PID" ] && kill "$BACKEND_PID" 2>/dev/null || true
  [ -n "$FRONTEND_PID" ] && kill "$FRONTEND_PID" 2>/dev/null || true
  [ -n "$MOCK_SIIGO_PID" ] && kill "$MOCK_SIIGO_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "==> Ensuring postgres + redis are up (docker compose)..."
docker compose -f "$COMPOSE_FILE" up -d postgres redis

echo "==> Resetting test database '$POSTGRES_DB' for a deterministic run..."
# Wait for postgres to accept connections (container start is not readiness)
for i in $(seq 1 30); do
  if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" >/dev/null 2>&1; then
    break
  fi
  [ "$i" -eq 30 ] && { echo "ERROR: postgres not ready" >&2; exit 1; }
  sleep 1
done
# --force terminates lingering pooled connections (e.g. a leftover backend)
docker compose -f "$COMPOSE_FILE" exec -T postgres \
  dropdb -U "$POSTGRES_USER" --if-exists --force "$POSTGRES_DB" 2>/dev/null || true
docker compose -f "$COMPOSE_FILE" exec -T postgres \
  createdb -U "$POSTGRES_USER" "$POSTGRES_DB"

echo "==> Applying migrations to '$POSTGRES_DB'..."
POSTGRES_DB="$POSTGRES_DB" docker compose -f "$COMPOSE_FILE" run --rm migrate

echo "==> Seeding e2e organizations..."
POSTGRES_HOST=localhost POSTGRES_PORT=5432 \
  POSTGRES_USER="$POSTGRES_USER" POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  POSTGRES_DB="$POSTGRES_DB" SKIP_MIGRATIONS=true go run ./cmd/seed-e2e

echo "==> Starting mock Siigo provider on :8090..."
go run ./cmd/mock-siigo -addr "$MOCK_SIIGO_ADDR" > /tmp/e2e-mock-siigo.log 2>&1 &
MOCK_SIIGO_PID=$!

echo "==> Starting Go backend on :8080 (mock auth)..."
AUTH_MOCK_ENABLED=true SERVER_ADDRESS=:8080 \
  SIIGO_BASE_URL="$MOCK_SIIGO_URL" \
  SIIGO_TOKEN_URL="$MOCK_SIIGO_URL/token" \
  SIIGO_WEBHOOK_SECRET="test_webhook_secret_for_e2e" \
  SIIGO_SANDBOX=true \
  go run ./cmd/api/main.go > /tmp/e2e-backend.log 2>&1 &
BACKEND_PID=$!

echo "==> Starting Next.js frontend on :3001..."
pnpm --dir "$FRONTEND_DIR" dev -p 3001 > /tmp/e2e-frontend.log 2>&1 &
FRONTEND_PID=$!

echo "==> Waiting for services to become healthy..."
for i in $(seq 1 90); do
  if curl -fsS http://localhost:8080/readyz >/dev/null 2>&1 \
     && curl -fsS -o /dev/null http://localhost:3001 >/dev/null 2>&1; then
    echo "  backend + frontend healthy after ${i}s"
    break
  fi
  if [ "$i" -eq 90 ]; then
    echo "ERROR: timed out waiting for backend/frontend" >&2
    echo "--- /tmp/e2e-backend.log ---" >&2
    cat /tmp/e2e-backend.log >&2
    echo "--- /tmp/e2e-frontend.log ---" >&2
    cat /tmp/e2e-frontend.log >&2
    exit 1
  fi
  sleep 1
done

# No-progress watchdog: if the Playwright runner emits no output for 180s (a hang),
# kill it, clear leaked chromium/playwright processes, and rerun the suite once.
# Real test failures (non-124 exits) are NOT retried.
run_attempt() {
  local attempt="$1"
  local logfile="/tmp/e2e-attempt-$attempt.log"
  local statusfile="/tmp/e2e-attempt-$attempt.status"
  echo "==> Running Playwright e2e suite (attempt $attempt)..."
  script -q -c "pnpm --dir \"$FRONTEND_DIR\" test:e2e; echo \$? > \"$statusfile\"" "$logfile" &
  local pid=$!
  while kill -0 "$pid" 2>/dev/null; do
    sleep 5
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    local mtime
    mtime=$(stat -c %Y "$logfile")
    if [ "$(( $(date +%s) - mtime ))" -gt 180 ]; then
      echo "!! no e2e output for 180s — killing attempt $attempt" >&2
      kill -TERM "$pid" 2>/dev/null
      wait "$pid" 2>/dev/null
      pkill -f "chromium|playwright" 2>/dev/null || true
      return 124
    fi
  done
  wait "$pid"
  local runner_status
  runner_status=$(cat "$statusfile" 2>/dev/null || echo 124)
  rm -f "$statusfile"
  return "$runner_status"
}

run_attempt 1 || e2e_status=$?
if [ "${e2e_status:-0}" -eq 124 ]; then
  echo "!! e2e runner hung — retrying suite once" >&2
  run_attempt 2 || e2e_status=$?
fi
exit "${e2e_status:-0}"
