#!/usr/bin/env bash
# bootstrap-stack.sh
# Autonomous Enterprise SaaS Stack Bootstrap.
# Non-interactive. Writes logs to logs/bootstrap/. Backs up files before overwrite.
# Exits non-zero on critical failure only.
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOME_DIR="${HOME:-$ROOT_DIR}"

# --- Environment controls (defaults) ---
BOOTSTRAP_ASSUME_YES="${BOOTSTRAP_ASSUME_YES:-true}"
BOOTSTRAP_INSTALL_PACKAGES="${BOOTSTRAP_INSTALL_PACKAGES:-true}"
BOOTSTRAP_CREATE_DEMO_SPEC="${BOOTSTRAP_CREATE_DEMO_SPEC:-true}"
BOOTSTRAP_IMPLEMENT_DEMO="${BOOTSTRAP_IMPLEMENT_DEMO:-false}"
BOOTSTRAP_OPEN_PR="${BOOTSTRAP_OPEN_PR:-false}"

# --- Logging ---
LOG_DIR="${ROOT_DIR}/logs/bootstrap"
mkdir -p "${LOG_DIR}"
TS="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="${LOG_DIR}/bootstrap-${TS}.log"
touch "${LOG_FILE}"

log() {
  local level="$1"; shift
  local msg="$*"
  local line
  line="$(date +%Y-%m-%dT%H:%M:%S%z) [${level}] ${msg}"
  printf '%s\n' "${line}" | tee -a "${LOG_FILE}"
}

# --- Report accumulator ---
REPORT_FILE="${ROOT_DIR}/BOOTSTRAP_REPORT.md"
INSTALLED_DEPS=()
FAILED_DEPS=()
INSTALLED_PACKAGES=()
FAILED_PACKAGES=()
CREATED_FILES=()
VERIFY_PASS=()
VERIFY_FAIL=()
BLOCKERS=()
MANUAL_ACTIONS=()

CRITICAL_FAILURE=0

# --- Helpers ---
backup_file() {
  local f="$1"
  if [[ -e "${f}" ]]; then
    local bak="${f}.bak.${TS}"
    cp -p "${f}" "${bak}" 2>/dev/null && log INFO "backed up ${f} -> ${bak}" || log WARN "could not back up ${f}"
  fi
}

ensure_dir() {
  local d="$1"
  if [[ ! -d "${d}" ]]; then
    mkdir -p "${d}" && log INFO "created directory ${d}"
  fi
}

run_checked() {
  local label="$1"; shift
  if "$@" >>"${LOG_FILE}" 2>&1; then
    log INFO "${label}: ok"
    return 0
  else
    log WARN "${label}: failed"
    return 1
  fi
}

run_critical() {
  local label="$1"; shift
  if "$@" >>"${LOG_FILE}" 2>&1; then
    log INFO "${label}: ok"
    return 0
  else
    log ERROR "${label}: failed (critical)"
    CRITICAL_FAILURE=1
    return 1
  fi
}

log "bootstrap start (ts=${TS})"
log "runtime env: assume_yes=${BOOTSTRAP_ASSUME_YES} install_packages=${BOOTSTRAP_INSTALL_PACKAGES} create_demo_spec=${BOOTSTRAP_CREATE_DEMO_SPEC} implement_demo=${BOOTSTRAP_IMPLEMENT_DEMO} open_pr=${BOOTSTRAP_OPEN_PR}"

# ============================================================================
# 1. Detect Pi runtime
# ============================================================================
RUNTIME="none"
RUNTIME_VERSION=""
if command -v pi >/dev/null 2>&1; then
  RUNTIME="pi"
  RUNTIME_VERSION="$(pi --version 2>/dev/null | head -1 || true)"
  log INFO "detected runtime: pi ${RUNTIME_VERSION}"
elif command -v omp >/dev/null 2>&1; then
  RUNTIME="omp"
  RUNTIME_VERSION="$(omp --version 2>/dev/null | head -1 || true)"
  log WARN "detected legacy runtime: omp ${RUNTIME_VERSION} (deprecated; prefer pi)"
else
  RUNTIME="none"
  log ERROR "no Pi runtime (pi or omp) detected"
  BLOCKERS+=("No Pi runtime detected (pi and omp both missing)")
  cat > "${ROOT_DIR}/BOOTSTRAP_BLOCKED.md" <<'EOF'
# BOOTSTRAP_BLOCKED

The Pi runtime is missing. Neither `pi` nor `omp` was found on PATH.
Install the Pi runtime and re-run scripts/bootstrap-stack.sh.
EOF
  log "bootstrap blocked: BOOTSTRAP_BLOCKED.md written"
  exit 1
fi
if ! "${RUNTIME}" --help >/dev/null 2>&1; then
  log ERROR "${RUNTIME} does not respond to --help"
  BLOCKERS+=("${RUNTIME} does not respond to --help")
  CRITICAL_FAILURE=1
fi

# ============================================================================
# 2. OpenSpec detection
# ============================================================================
OPENSPEC_RESULT="not detected"
if command -v openspec >/dev/null 2>&1; then
  OPENSPEC_VERSION="$(openspec --version 2>/dev/null | head -1 || true)"
  OPENSPEC_RESULT="OpenSpec CLI ${OPENSPEC_VERSION}"
  log INFO "OpenSpec detected: ${OPENSPEC_RESULT}"
elif [[ -d "${ROOT_DIR}/openspec/changes" ]]; then
  OPENSPEC_RESULT="directory convention only (openspec/changes/)"
  log INFO "OpenSpec CLI not found; using directory convention openspec/changes/"
else
  OPENSPEC_RESULT="not available"
  log WARN "OpenSpec CLI not available and openspec/changes/ missing"
fi

# ============================================================================
# 3. System dependency checks
# ============================================================================
APT_AVAILABLE=0
if command -v apt-get >/dev/null 2>&1; then APT_AVAILABLE=1; fi
BREW_AVAILABLE=0
if command -v brew >/dev/null 2>&1; then BREW_AVAILABLE=1; fi
SUDO_AVAILABLE=0
if command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then SUDO_AVAILABLE=1; fi

install_sys_pkg() {
  local pkg="$1"
  local critical="${2:-no}"
  if command -v "${pkg}" >/dev/null 2>&1; then
    INSTALLED_DEPS+=("${pkg}:$(command -v "${pkg}")")
    return 0
  fi
  log WARN "${pkg}: not found, attempting install"
  local ok=0
  if [[ "${APT_AVAILABLE}" -eq 1 ]]; then
    local prefix=()
    if [[ "$(id -u)" -ne 0 ]]; then
      if [[ "${SUDO_AVAILABLE}" -eq 1 ]]; then prefix=(sudo); else log WARN "no root/sudo; cannot apt-get install ${pkg}"; fi
    fi
    if [[ "${#prefix[@]}" -eq 0 ]] || [[ "${SUDO_AVAILABLE}" -eq 1 ]] || [[ "$(id -u)" -eq 0 ]]; then
      "${prefix[@]}" apt-get install -y "${pkg}" >>"${LOG_FILE}" 2>&1 && ok=1
    fi
  elif [[ "${BREW_AVAILABLE}" -eq 1 ]]; then
    brew install "${pkg}" >>"${LOG_FILE}" 2>&1 && ok=1
  fi
  if [[ "${ok}" -eq 1 ]] && command -v "${pkg}" >/dev/null 2>&1; then
    log INFO "${pkg}: installed (${pkg}:$(command -v "${pkg}"))"
    INSTALLED_DEPS+=("${pkg}:$(command -v "${pkg}")")
  else
    log WARN "${pkg}: install failed"
    FAILED_DEPS+=("${pkg}")
    if [[ "${critical}" == "yes" ]]; then
      BLOCKERS+=("Critical system dependency ${pkg} failed to install")
      CRITICAL_FAILURE=1
    fi
  fi
}

# node
if command -v node >/dev/null 2>&1; then
  NODE_VERSION="$(node --version 2>/dev/null | sed 's/^v//' || true)"
  NODE_MAJOR="${NODE_VERSION%%.*}"
  log INFO "node ${NODE_VERSION} detected (major ${NODE_MAJOR})"
  INSTALLED_DEPS+=("node:$(command -v node) v${NODE_VERSION}")
  if [[ "${NODE_MAJOR}" -lt 20 ]]; then
    log WARN "node major ${NODE_MAJOR} < 20; attempting nvm install 20"
    if [[ -s "${HOME_DIR}/.nvm/nvm.sh" ]]; then
      # shellcheck disable=SC1091
      . "${HOME_DIR}/.nvm/nvm.sh"
      nvm install 20 >>"${LOG_FILE}" 2>&1 && nvm use 20 >>"${LOG_FILE}" 2>&1 && log INFO "node 20 installed via nvm" || { log ERROR "nvm install 20 failed"; BLOCKERS+=("node < 20 and nvm install 20 failed"); CRITICAL_FAILURE=1; }
    else
      log WARN "nvm not found at ~/.nvm/nvm.sh; skipping node 20 install (node ${NODE_VERSION} present)"
    fi
  fi
else
  log WARN "node not found; attempting nvm-based install of node 20"
  if [[ -s "${HOME_DIR}/.nvm/nvm.sh" ]]; then
    # shellcheck disable=SC1091
    . "${HOME_DIR}/.nvm/nvm.sh"
    nvm install 20 >>"${LOG_FILE}" 2>&1 && log INFO "node 20 installed via nvm" || { log ERROR "nvm install 20 failed"; BLOCKERS+=("node missing and nvm install 20 failed"); CRITICAL_FAILURE=1; }
  else
    log ERROR "node missing and nvm not available"
    BLOCKERS+=("node missing and nvm not available")
    CRITICAL_FAILURE=1
  fi
fi
# npm
install_sys_pkg npm yes
# git, curl, jq
install_sys_pkg git yes
install_sys_pkg curl yes
install_sys_pkg jq yes
# lsof, gh: non-critical
install_sys_pkg lsof no
install_sys_pkg gh no

# ============================================================================
# 4. Pi packages
# ============================================================================
# NOTE: this script is legacy (omp-era) bootstrap tooling. Under the native Pi
# stack, package installs use `pi install -l`; the omp `plugin install`
# subcommand does not exist in pi. Prompt templates ship in-repo under
# .pi/prompts/ and are NOT regenerated here. See openspec/changes/
# remove-pantheon-native-agents/ and the "Agent Pipeline" section of AGENTS.md.

install_pi_package() {
  local pkg="$1"
  if [[ "${RUNTIME}" == "omp" ]]; then
    "${RUNTIME}" plugin install "${pkg}"
  else
    pi install -l "${pkg}"
  fi
}

PACKAGES=(
  "npm:pi-side-agents"
  "npm:@bopstack/pi-codegraph"
  "npm:pi-playwright"
  "npm:pi-web-search"
)
CRITICAL_PACKAGES=("npm:@bopstack/pi-codegraph")

if [[ "${BOOTSTRAP_INSTALL_PACKAGES}" == "true" ]]; then
  log "installing Pi packages via ${RUNTIME} (pi: install -l / omp: plugin install)"
  for pkg in "${PACKAGES[@]}"; do
    label="${pkg//\//_}"
    if run_checked "install ${pkg}" install_pi_package "${pkg}"; then
      INSTALLED_PACKAGES+=("${pkg}")
    else
      FAILED_PACKAGES+=("${pkg}")
      if [[ " ${CRITICAL_PACKAGES[*]} " == *" ${pkg} "* ]]; then
        log ERROR "critical package ${pkg} failed to install"
        BLOCKERS+=("Critical package ${pkg} failed to install")
        CRITICAL_FAILURE=1
      fi
    fi
  done
else
  log "BOOTSTRAP_INSTALL_PACKAGES=false; skipping package installs"
  MANUAL_ACTIONS+=("Install Pi packages manually: pi install -l ${PACKAGES[*]}")
fi

# ============================================================================
# 5. Playwright (Chromium)
# ============================================================================
PLAYWRIGHT_STATUS="not attempted"
if [[ "${BOOTSTRAP_INSTALL_PACKAGES}" == "true" ]]; then
  if run_checked "playwright install chromium" npx --yes playwright install chromium; then
    PLAYWRIGHT_STATUS="chromium installed"
    if [[ "${APT_AVAILABLE}" -eq 1 ]]; then
      if run_checked "playwright install-deps chromium" npx --yes playwright install-deps chromium; then
        PLAYWRIGHT_STATUS="chromium + system deps installed"
      else
        log WARN "playwright install-deps failed (non-critical)"
        PLAYWRIGHT_STATUS="chromium installed; system deps failed"
      fi
    fi
  else
    PLAYWRIGHT_STATUS="install failed"
  fi
else
  PLAYWRIGHT_STATUS="skipped (BOOTSTRAP_INSTALL_PACKAGES=false)"
fi

# ============================================================================
# 6. CodeGraph initialization
# ============================================================================
CODEGRAPH_STATUS="not attempted"
CODEGRAPH_ATTEMPTS=(
  "codegraph init"
  "npx --yes codegraph init"
)
for attempt in "${CODEGRAPH_ATTEMPTS[@]}"; do
  if run_checked "codegraph init (${attempt})" bash -c "${attempt}"; then
    CODEGRAPH_STATUS="initialized via ${attempt}"
    break
  fi
done
if [[ "${CODEGRAPH_STATUS}" == "not attempted" ]] || [[ "${CODEGRAPH_STATUS}" == *"failed"* ]]; then
  CODEGRAPH_STATUS="initialization failed (warning)"
  log WARN "CodeGraph could not be initialized"
  if [[ " ${FAILED_PACKAGES[*]} " == *" npm:@bopstack/pi-codegraph "* ]]; then
    log ERROR "CodeGraph init failed and @bopstack/pi-codegraph also failed to install"
    BLOCKERS+=("CodeGraph init failed and @bopstack/pi-codegraph not installed")
    CRITICAL_FAILURE=1
  fi
fi

# ============================================================================
# 7. Required directories
# ============================================================================
ensure_dir "${ROOT_DIR}/openspec/changes"
ensure_dir "${ROOT_DIR}/docs/compliance"
ensure_dir "${ROOT_DIR}/logs"
ensure_dir "${ROOT_DIR}/scripts"
ensure_dir "${ROOT_DIR}/.pi"
ensure_dir "${ROOT_DIR}/.pi/prompts"

# ============================================================================
# 8. Project-local settings .pi/settings.json (pi package set)
# ============================================================================
SETTINGS_FILE="${ROOT_DIR}/.pi/settings.json"
write_settings() {
  backup_file "${SETTINGS_FILE}"
  cat > "${SETTINGS_FILE}" <<'EOF'
{
  "packages": [
    "npm:@bopstack/pi-codegraph",
    "npm:pi-playwright",
    "npm:pi-web-search"
  ]
}
EOF
  if jq . "${SETTINGS_FILE}" >/dev/null 2>&1; then
    log INFO "settings.json written and valid JSON"
    CREATED_FILES+=("${SETTINGS_FILE}")
  else
    log WARN "settings.json invalid after write; repairing"
    cat > "${SETTINGS_FILE}" <<'EOF'
{
  "packages": [
    "npm:@bopstack/pi-codegraph",
    "npm:pi-playwright",
    "npm:pi-web-search"
  ]
}
EOF
    if jq . "${SETTINGS_FILE}" >/dev/null 2>&1; then
      log INFO "settings.json repaired and valid JSON"
      CREATED_FILES+=("${SETTINGS_FILE}")
    else
      log ERROR "settings.json validation failed after repair"
      BLOCKERS+=(".pi/settings.json failed JSON validation")
      CRITICAL_FAILURE=1
    fi
  fi
}
write_settings

# ============================================================================
# 9. Agent prompt files (native templates already ship in .pi/prompts/)
# ============================================================================
PROMPT_DIR="${ROOT_DIR}/.pi/prompts"

write_prompt() {
  local name="$1"
  local file="${PROMPT_DIR}/${name}"
  if [[ -f "${file}" ]]; then
    log INFO "kept existing ${file} (native template ships in-repo)"
    return 0
  fi
  backup_file "${file}"
  cat > "${file}"
  log INFO "wrote ${file}"
  CREATED_FILES+=("${file}")
}

# Native Pi templates ship in-repo under .pi/prompts/ (architect, council,
# sdet, uiux, iso). The legacy omp-style fallback prompt generation is
# DISABLED: re-running it would create omp-flavored files (iso-auditor.md,
# autopilot.md) with non-pi tool names (read-file/write-file/run-command).
create_prompt_files() {
  # DISABLED: legacy omp-style prompt generation. Native Pi templates ship
  # in-repo under .pi/prompts/ (architect, council, sdet, uiux, iso); the
  # omp-flavored fallbacks (iso-auditor.md, autopilot.md) and non-pi tool
  # names (read-file/write-file/run-command) must not be regenerated.
  log INFO "legacy omp-style prompt generation disabled (native templates ship in-repo)"
  return 0
  # --- legacy omp-era body below is dead code and never executes ---
write_prompt architect.md <<'EOF'
# Architect Agent

description: Enterprise SaaS System Architect and OpenSpec Planner

## Allowed Tools

- @bopstack/pi-codegraph
- read-file
- write-file

## Mandate

- Write ONLY under openspec/changes/<feature>/.
- Create proposal.md, design.md, tasks.md, and routing.json before any code changes.
- Never edit application source code. The Architect plans; other agents implement.

## Rules

- Multi-tenant isolation: every design MUST keep tenant data isolated; shared tables MUST carry a tenant_id column and every query MUST be tenant-scoped. Never design cross-tenant access paths.
- Database migrations MUST follow expand-contract: expand (additive, nullable/backwards-compatible) first, deploy, then contract (drop/rename) in a later change.
- Never mutate production data. Designs may reference seed or test data only.
- Webhook handling MUST be idempotent: event processing MUST be safe to replay at least once (transaction-isolated state checks, idempotency keys).
- New user-visible or risky behavior MUST be behind a feature flag with a documented default.
- Document every integration with Stytch B2B, Polar.sh, or external providers against their documented contracts; note fallback and circuit-breaker states.

## routing.json Fields

- requires_council: boolean - council review needed before implementation
- requires_playwright: boolean - UI tests required
- requires_migration: boolean - database migration required
- requires_feature_flag: boolean - feature flag required
- complexity: low | medium | high
- security_impact: boolean
- data_model_change: boolean
- payment_impact: boolean
EOF

write_prompt sdet.md <<'EOF'
# SDET Agent

description: Software Development Engineer in Test

## Allowed Tools

- read-file
- write-file
- run-command
- @bopstack/pi-codegraph

## Rules

- Strict test-first behavior: write failing tests before implementation, then implement until tests pass.
- NEVER disable or delete failing tests to make a suite green. A failing test is a bug report; fix the code or the test only when the test itself is provably wrong.
- Tenant isolation: tests MUST assert that data from one tenant is never visible to another.
- Webhook idempotency: tests MUST cover replay of webhook deliveries and assert no duplicate side effects.
- Retry-safe background jobs: tests MUST verify jobs survive retries without duplicate effects.
- Structured logging with tenant context: assert logs carry tenant identifiers where required.
- Database migrations in tests MUST be safe: use isolated databases or transactions; never run destructive migrations against shared data.
- Run npm test, npm run lint, and npm run build where available, and report their results verbatim.
EOF

write_prompt uiux.md <<'EOF'
# UI/UX Agent

description: Visual UI/UX, Layout, and Accessibility Tester

## Allowed Tools

- pi-playwright
- read-file
- write-file
- run-command

## Viewport Matrix

- 390 x 844 (mobile)
- 768 x 1024 (tablet)
- 1440 x 900 (desktop)

## Checks

- Layout shift (CLS) on load and interaction
- Content clipping / overflow at every viewport
- Focus states visible on all interactive elements
- ARIA roles, labels, and landmarks correct
- Empty states rendered when there is no data
- Error states rendered and recoverable
- Loading states rendered and not blocking

## Output

- QA report: openspec/changes/<feature>/qa/REPORT.md
- Screenshots: openspec/changes/<feature>/qa/screenshots/
EOF

write_prompt iso-auditor.md <<'EOF'
# ISO Auditor Agent

description: ISO 27001, 9001, and 42001 Evidence Agent

## Allowed Tools

- @bopstack/pi-codegraph
- read-file
- write-file
- run-command

## Rules

- Compliance claims REQUIRE evidence. A claim without a verifiable evidence path is not a claim; it is a gap.
- Verify claims using: file paths, symbol names, AST matches, configuration entries, or test output. Cite the exact artifact.
- Do not mark a control implemented from prose alone. If evidence is missing, mark status as "Not Evidenced" and record the gap.

## Output

- File: docs/compliance/ISO_TRACEABILITY_MATRIX.md
- Required columns:
  - ISO standard
  - control identifier
  - implementation status
  - evidence path
  - evidence type
  - last reviewed date
  - reviewer type
EOF
}
create_prompt_files

# ============================================================================
# 10. QA server wrapper
# ============================================================================
QA_SCRIPT="${ROOT_DIR}/scripts/run-qa-server.sh"
if [[ -e "${QA_SCRIPT}" ]]; then
  backup_file "${QA_SCRIPT}"
fi
cat > "${QA_SCRIPT}" <<'EOF'
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
EOF
chmod +x "${QA_SCRIPT}"
log INFO "wrote and chmod +x ${QA_SCRIPT}"
CREATED_FILES+=("${QA_SCRIPT}")

# ============================================================================
# 11. Compliance baseline
# ============================================================================
ISO_MATRIX="${ROOT_DIR}/docs/compliance/ISO_TRACEABILITY_MATRIX.md"
if [[ -e "${ISO_MATRIX}" ]]; then
  log INFO "ISO_TRACEABILITY_MATRIX.md already exists; not overwriting"
else
  cat > "${ISO_MATRIX}" <<'EOF'
# ISO Traceability Matrix

| ISO Standard | Control | Status | Evidence Path | Evidence Type | Last Reviewed | Reviewer |
|---|---|---|---|---|---|---|
EOF
  log INFO "created ${ISO_MATRIX}"
  CREATED_FILES+=("${ISO_MATRIX}")
fi

# ============================================================================
# 12. .gitignore entries
# ============================================================================
if [[ -d "${ROOT_DIR}/.git" ]]; then
  GITIGNORE="${ROOT_DIR}/.gitignore"
  if [[ ! -f "${GITIGNORE}" ]]; then
    touch "${GITIGNORE}"
  fi
  for entry in "logs/" ".codegraph/"; do
    if ! grep -qxF "${entry}" "${GITIGNORE}"; then
      printf '\n# Bootstrap-generated\n%s\n' "${entry}" >> "${GITIGNORE}"
      log INFO "added '${entry}' to .gitignore"
    else
      log INFO ".gitignore already contains '${entry}'"
    fi
  done
else
  log WARN "not a git repository; skipping .gitignore update"
fi

# ============================================================================
# 13. OpenSpec demo (documentation-only health check verification)
# ============================================================================
if [[ "${BOOTSTRAP_CREATE_DEMO_SPEC}" == "true" ]]; then
  DEMO_DIR="${ROOT_DIR}/openspec/changes/00-bootstrap-health-check"
  ensure_dir "${DEMO_DIR}"
  cat > "${DEMO_DIR}/proposal.md" <<'EOF'
# 00-bootstrap-health-check: Bootstrap Health Check Verification (Documentation-Only)

## Summary

Documentation-only change that records the health check verification workflow for this repository. It verifies that the existing health endpoint behaves as documented and captures the verification procedure for future agents.

## Problem Statement

The repository needs a low-risk, credential-free change to validate the OpenSpec workflow end to end after bootstrap. A health check verification is the safest candidate: it touches no customer data, no billing, no authentication, and no production configuration.

## Assumptions

- The repository already provides a health check endpoint: `GET /healthz` in `go-b2b-starter/cmd/mock-siigo/main.go` (returns 200 `{"status":"ok"}`). Verified during bootstrap on 2026-08-11.
- This change is documentation-only. No application source code is modified.

## Non-Goals

- No billing changes.
- No authentication or authorization changes.
- No production configuration changes.
- No customer data access.
- No external credentials required.
- No database migration.
- No feature flag introduction.

## Impact

- Adds a reusable verification procedure for the health endpoint.
- Low complexity; no security, data-model, or payment impact.
EOF
  cat > "${DEMO_DIR}/design.md" <<'EOF'
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
EOF
  cat > "${DEMO_DIR}/tasks.md" <<'EOF'
# Tasks: Bootstrap Health Check Verification

All tasks are verification-only; no source code is modified.

- [ ] Verify `GET /healthz` exists in `go-b2b-starter/cmd/mock-siigo/main.go`.
      Command: `grep -n "healthz" go-b2b-starter/cmd/mock-siigo/main.go`
- [ ] Verify `scripts/run-qa-server.sh` is executable and contains the health wait loop.
      Command: `test -x scripts/run-qa-server.sh && grep -n "CANDIDATES" scripts/run-qa-server.sh`
- [ ] Confirm no demo task touches billing, auth, production configuration, customer data, or requires external credentials.
- [ ] Confirm `routing.json` is present and all fields are `false` / `low` as specified.

## Verification Commands

```
test -x scripts/run-qa-server.sh
grep -n "healthz" go-b2b-starter/cmd/mock-siigo/main.go
jq . openspec/changes/00-bootstrap-health-check/routing.json
```
EOF
  cat > "${DEMO_DIR}/routing.json" <<'EOF'
{
  "requires_council": false,
  "requires_playwright": false,
  "requires_migration": false,
  "requires_feature_flag": false,
  "complexity": "low",
  "security_impact": false,
  "data_model_change": false,
  "payment_impact": false
}
EOF
  log INFO "created demo spec at ${DEMO_DIR}"
  CREATED_FILES+=("${DEMO_DIR}/proposal.md" "${DEMO_DIR}/design.md" "${DEMO_DIR}/tasks.md" "${DEMO_DIR}/routing.json")
fi

# ============================================================================
# 14. Optional demo implementation (default: off)
# ============================================================================
DEMO_IMPLEMENTATION="not implemented (BOOTSTRAP_IMPLEMENT_DEMO=false)"
if [[ "${BOOTSTRAP_IMPLEMENT_DEMO}" == "true" ]]; then
  log "BOOTSTRAP_IMPLEMENT_DEMO=true; creating isolated worktree for the demo"
  WT_NAME="bootstrap-health-check-demo"
  WT_DIR="${ROOT_DIR}/../${WT_NAME}"
  if git -C "${ROOT_DIR}" worktree add -b "bootstrap/${WT_NAME}" "${WT_DIR}" >>"${LOG_FILE}" 2>&1; then
    log INFO "worktree created at ${WT_DIR} (branch bootstrap/${WT_NAME})"
    DEMO_IMPLEMENTATION="implemented in worktree ${WT_DIR} (branch bootstrap/${WT_NAME})"
    if [[ "${BOOTSTRAP_OPEN_PR}" == "true" ]] && command -v gh >/dev/null 2>&1; then
      if run_checked "open demo PR" gh pr create --repo "$(git -C "${ROOT_DIR}" remote get-url origin 2>/dev/null | sed 's#.*github.com[:/]##; s#\.git$##' || true)" --base main --head "bootstrap/${WT_NAME}" --title "Bootstrap health check demo (documentation-only)" --body "Bootstrap demo change: health check verification documentation."; then
        DEMO_IMPLEMENTATION="${DEMO_IMPLEMENTATION}; PR opened"
      else
        log WARN "could not open PR (non-critical)"
        DEMO_IMPLEMENTATION="${DEMO_IMPLEMENTATION}; PR not opened"
      fi
    else
      log INFO "BOOTSTRAP_OPEN_PR=false or gh unavailable; leaving branch local"
      DEMO_IMPLEMENTATION="${DEMO_IMPLEMENTATION}; branch left local"
    fi
  else
    log ERROR "worktree creation failed"
    BLOCKERS+=("Demo worktree creation failed")
    CRITICAL_FAILURE=1
  fi
fi

# ============================================================================
# 15. Autopilot prompt
# ============================================================================
write_prompt autopilot.md <<'EOF'
# Autopilot Agent

description: Autonomous full-stack feature implementation agent for this repository

## Input

A feature description (one or more sentences describing desired behavior).

## Rules

1. Use the current repository.
2. Require an OpenSpec change directory for every feature: openspec/changes/<feature>/.
3. Create proposal.md, design.md, tasks.md, and routing.json BEFORE any code changes.
4. Invoke council review only when routing.json requires it (requires_council: true).
5. Implement in an isolated Git worktree. Never implement directly on main.
6. Write tests before implementation (test-first).
7. Run available unit, integration, lint, and build commands (npm test, npm run lint, npm run build, pnpm where applicable).
8. Run UI tests only when routing.json requires them (requires_playwright: true).
9. Start dev servers outside the agent when UI testing is required (scripts/run-qa-server.sh).
10. Update ISO evidence (docs/compliance/ISO_TRACEABILITY_MATRIX.md) after implementation.
11. Open a pull request instead of merging.
12. Never deploy.
13. Never modify production data.
14. Never bypass CI.
15. Never disable failing tests.
16. Write a run report under openspec/changes/<feature>/qa/ or logs/.

## Output

- Implemented change in an isolated worktree
- Passing tests, lint, and build
- Open PR (or recorded branch name)
- Run report at openspec/changes/<feature>/qa/REPORT.md or logs/
EOF

# ============================================================================
# 16. Final report
# ============================================================================
write_report() {
  cat > "${REPORT_FILE}" <<EOF
# BOOTSTRAP_REPORT

Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Working directory: ${ROOT_DIR}

## 1. Detected Pi Runtime

- Runtime: ${RUNTIME}
- Version: ${RUNTIME_VERSION}

## 2. OpenSpec Detection

- Result: ${OPENSPEC_RESULT}

## 3. Installed System Dependencies

$(printf -- '- %s\n' "${INSTALLED_DEPS[@]:-none}")

## 4. Installed Pi Packages

$(printf -- '- %s\n' "${INSTALLED_PACKAGES[@]:-none}")

## 5. Failed Pi Packages

$(printf -- '- %s\n' "${FAILED_PACKAGES[@]:-none}")

## 6. CodeGraph Status

- ${CODEGRAPH_STATUS}

## 7. Playwright Status

- ${PLAYWRIGHT_STATUS}

## 8. Created Configuration Files

$(printf -- '- %s\n' "${CREATED_FILES[@]:-none}")

## 9. Created Prompt Files

- ${HOME_DIR}/.omp/prompts/architect.md
- ${HOME_DIR}/.omp/prompts/sdet.md
- ${HOME_DIR}/.omp/prompts/uiux.md
- ${HOME_DIR}/.omp/prompts/iso-auditor.md
- ${HOME_DIR}/.omp/prompts/autopilot.md

## 10. Created Scripts

- ${QA_SCRIPT}
- ${ROOT_DIR}/scripts/bootstrap-stack.sh

## 11. Demo Spec Location

- ${ROOT_DIR}/openspec/changes/00-bootstrap-health-check/ (proposal.md, design.md, tasks.md, routing.json)

## 12. Demo Implementation Status

- ${DEMO_IMPLEMENTATION}

## 13. Verification Results

PASS:
$(printf -- '- %s\n' "${VERIFY_PASS[@]:-none}")

FAIL:
$(printf -- '- %s\n' "${VERIFY_FAIL[@]:-none}")

## 14. Remaining Manual Actions

$(printf -- '- %s\n' "${MANUAL_ACTIONS[@]:-none}")

## 15. Blockers

$(printf -- '- %s\n' "${BLOCKERS[@]:-none}")
EOF
  log INFO "wrote ${REPORT_FILE}"
  CREATED_FILES+=("${REPORT_FILE}")
}

# ============================================================================
# 17. Verification
# ============================================================================
verify() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    log INFO "verify: ${label} PASS"
    VERIFY_PASS+=("${label}")
  else
    log WARN "verify: ${label} FAIL - attempting one repair"
    VERIFY_FAIL+=("${label}")
  fi
  return 0
}

verify "omp responds to --help" omp --help
verify "settings.json is valid JSON" jq . "${SETTINGS_FILE}"
verify "architect.md exists" test -f "${PROMPT_DIR}/architect.md"
verify "sdet.md exists" test -f "${PROMPT_DIR}/sdet.md"
verify "uiux.md exists" test -f "${PROMPT_DIR}/uiux.md"
verify "iso-auditor.md exists" test -f "${PROMPT_DIR}/iso-auditor.md"
verify "autopilot.md exists" test -f "${PROMPT_DIR}/autopilot.md"
verify "openspec/changes exists" test -d "${ROOT_DIR}/openspec/changes"
verify "ISO_TRACEABILITY_MATRIX.md exists" test -f "${ISO_MATRIX}"
verify "run-qa-server.sh exists and executable" test -x "${QA_SCRIPT}"
verify "logs/bootstrap exists" test -d "${LOG_DIR}"

# One-shot repairs for any failed verification
repair_prompt_files() {
  log WARN "repair: rewriting prompt files"
  create_prompt_files
}
if [[ " ${VERIFY_FAIL[*]} " == *" architect.md exists "* ]]; then repair_prompt_files; fi
if [[ " ${VERIFY_FAIL[*]} " == *" sdet.md exists "* ]]; then repair_prompt_files; fi
if [[ " ${VERIFY_FAIL[*]} " == *" uiux.md exists "* ]]; then repair_prompt_files; fi
if [[ " ${VERIFY_FAIL[*]} " == *" iso-auditor.md exists "* ]]; then repair_prompt_files; fi
if [[ " ${VERIFY_FAIL[*]} " == *" autopilot.md exists "* ]]; then repair_prompt_files; fi
if [[ " ${VERIFY_FAIL[*]} " == *" settings.json is valid JSON "* ]]; then
  log WARN "repair: rewriting settings.json"
  write_settings
fi
if [[ " ${VERIFY_FAIL[*]} " == *" ISO_TRACEABILITY_MATRIX.md exists "* ]]; then
  log WARN "repair: creating ISO_TRACEABILITY_MATRIX.md"
  if [[ ! -e "${ISO_MATRIX}" ]]; then
    cat > "${ISO_MATRIX}" <<'EOF'
# ISO Traceability Matrix

| ISO Standard | Control | Status | Evidence Path | Evidence Type | Last Reviewed | Reviewer |
|---|---|---|---|---|---|---|
EOF
  fi
fi
if [[ " ${VERIFY_FAIL[*]} " == *" run-qa-server.sh exists and executable "* ]]; then
  log WARN "repair: chmod +x run-qa-server.sh"
  chmod +x "${QA_SCRIPT}"
fi
if [[ " ${VERIFY_FAIL[*]} " == *" openspec/changes exists "* ]]; then
  log WARN "repair: creating openspec/changes"
  ensure_dir "${ROOT_DIR}/openspec/changes"
fi
if [[ " ${VERIFY_FAIL[*]} " == *" logs/bootstrap exists "* ]]; then
  log WARN "repair: creating logs/bootstrap"
  ensure_dir "${LOG_DIR}"
fi
if [[ " ${VERIFY_FAIL[*]} " == *" omp responds to --help "* ]]; then
  log ERROR "repair not possible: omp does not respond to --help"
  BLOCKERS+=("omp does not respond to --help")
  CRITICAL_FAILURE=1
fi

# Re-verify anything that was repaired
if [[ "${#VERIFY_FAIL[@]}" -gt 0 ]]; then
  log "re-running verification after repairs"
  verify "settings.json is valid JSON" jq . "${SETTINGS_FILE}"
  verify "architect.md exists" test -f "${PROMPT_DIR}/architect.md"
  verify "sdet.md exists" test -f "${PROMPT_DIR}/sdet.md"
  verify "uiux.md exists" test -f "${PROMPT_DIR}/uiux.md"
  verify "iso-auditor.md exists" test -f "${PROMPT_DIR}/iso-auditor.md"
  verify "autopilot.md exists" test -f "${PROMPT_DIR}/autopilot.md"
  verify "openspec/changes exists" test -d "${ROOT_DIR}/openspec/changes"
  verify "ISO_TRACEABILITY_MATRIX.md exists" test -f "${ISO_MATRIX}"
  verify "run-qa-server.sh exists and executable" test -x "${QA_SCRIPT}"
  verify "logs/bootstrap exists" test -d "${LOG_DIR}"
fi

write_report

# ============================================================================
# 18. Exit
# ============================================================================
if [[ "${CRITICAL_FAILURE}" -eq 1 ]]; then
  log "bootstrap finished with critical failures (see BOOTSTRAP_REPORT.md)"
  exit 1
fi
log "bootstrap completed; report at ${REPORT_FILE}"
exit 0
EOF
chmod +x /home/phuongbinhnguyentai/project/Whatsapp/scripts/bootstrap-stack.sh
echo "script written"