#!/usr/bin/env bash
# run-qa-server.sh - Start the local dev server for QA, wait for health, stop cleanly.
# Usage: scripts/run-qa-server.sh [PORT]   (default port 3000)
set -Eeuo pipefail

PORT="${1:-3000}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${ROOT_DIR}/logs"
LOG_FILE="${LOG_DIR}/qa-server-${PORT}.log"
mkdir -p "${LOG_DIR}"

echo "[INFO] Starting QA server on port ${PORT}"
echo "[INFO] Log file: ${LOG_FILE}"

# Locate the dev server. Monorepo convention: Next.js frontend in next_b2b_starter (pnpm).
DEV_CWD="${ROOT_DIR}"
DEV_CMD=(npm run dev)
if [[ ! -f "${ROOT_DIR}/package.json" && -d "${ROOT_DIR}/next_b2b_starter" && -f "${ROOT_DIR}/next_b2b_starter/package.json" ]]; then
  DEV_CWD="${ROOT_DIR}/next_b2b_starter"
  if command -v pnpm >/dev/null 2>&1; then
    DEV_CMD=(pnpm run dev)
  else
    DEV_CMD=(npx next dev)
  fi
fi

# Kill any process left on the port
if command -v lsof >/dev/null 2>&1; then
  LEFTOVER="$(lsof -t -i :"${PORT}" 2>/dev/null || true)"
  if [[ -n "${LEFTOVER}" ]]; then
    echo "[WARN] Killing leftover process(es) on port ${PORT}: ${LEFTOVER}"
    # shellcheck disable=SC2086
    kill ${LEFTOVER} 2>/dev/null || true
    sleep 1
  fi
fi

echo "[INFO] Dev command: ${DEV_CMD[*]} (cwd=${DEV_CWD})"
( cd "${DEV_CWD}" && exec "${DEV_CMD[@]}" ) >"${LOG_FILE}" 2>&1 &
SERVER_PID=$!
echo "[INFO] Dev server PID ${SERVER_PID}"

cleanup() {
  echo "[INFO] Stopping dev server (PID ${SERVER_PID})"
  kill "${SERVER_PID}" 2>/dev/null || true
  wait "${SERVER_PID}" 2>/dev/null || true
  if command -v lsof >/dev/null 2>&1; then
    LEFTOVER="$(lsof -t -i :"${PORT}" 2>/dev/null || true)"
    if [[ -n "${LEFTOVER}" ]]; then
      echo "[WARN] Killing leftover process(es) on port ${PORT}"
      # shellcheck disable=SC2086
      kill ${LEFTOVER} 2>/dev/null || true
    fi
  fi
  echo "[INFO] QA server stopped"
}
trap cleanup EXIT INT TERM

CANDIDATES=(
  "http://localhost:${PORT}/health"
  "http://localhost:${PORT}/api/health"
  "http://localhost:${PORT}/healthz"
  "http://localhost:${PORT}/api/healthz"
)
WAIT_SECONDS=30
START="$(date +%s)"
READY=""
while [[ -z "${READY}" ]]; do
  for url in "${CANDIDATES[@]}"; do
    if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
      READY="${url}"
      break
    fi
  done
  if [[ -n "${READY}" ]]; then break; fi
  NOW="$(date +%s)"
  if (( NOW - START >= WAIT_SECONDS )); then break; fi
  sleep 1
done

if [[ -n "${READY}" ]]; then
  echo "[INFO] Server ready at ${READY}"
else
  echo "[ERROR] No health endpoint responded within ${WAIT_SECONDS}s. Check ${LOG_FILE}"
  exit 1
fi

# Keep running until interrupted; cleanup trap stops the server.
wait "${SERVER_PID}"
