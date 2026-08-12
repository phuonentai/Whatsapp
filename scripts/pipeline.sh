#!/usr/bin/env bash
# pipeline.sh — deterministic native Pi agent pipeline.
#
# Chains OpenSpec governance stages headlessly via `pi -p --prompt-template`.
# This is the replacement for the omp/Pantheon orchestration layer: every stage
# is a stateless `pi -p` invocation whose shared state lives on disk in the
# change directory (proposal.md, design.md, tasks.md, VERDICT.md, qa/).
#
# Usage:
#   scripts/pipeline.sh <change> [options]
#
# Options:
#   --dry-run        Print the exact commands without executing them (exit 0).
#   --with-council   Force the council stage (adversarial design review).
#   --with-uiux      Force the uiux stage (Playwright visual + a11y QA).
#   --skip-iso       Skip the iso stage (compliance traceability).
#   --skip-revise    REJECTED council verdict halts immediately (exit 2) without
#                    running the revise loop (pre-revision-loop behavior).
#   --max-revisions <N>  Revision-loop cap (positive integer; default 2).
#                    Also configurable via PIPELINE_MAX_REVISIONS env var.
#   --reset-revisions  Delete revision.json before running (restart the revision
#                    cycle after a capped halt; preserves revision.md audit trail).
#   --help           Show this message.
#
# Stage order: architect → council → [revise ⇄ council, bounded] → sdet → uiux → iso
#   architect  unconditional — thin wrapper delegating to the opsx-propose
#              workflow (creates/validates planning artifacts).
#   council    when complexity=high, requires_council=true, requires_market_read=true,
#              or --with-council. Writes VERDICT.md; APPROVED proceeds, REJECTED
#              triggers the bounded revise loop, absent/ambiguous marker halts (exit 3).
#   revise     only after a REJECTED verdict, up to MAX_COUNCIL_REVISIONS passes of
#              revise → openspec validate → council re-review; runs read/write-only
#              with artifact paths provided via a transient .revise-context.json;
#              never edits application source code.
#   sdet       unconditional — thin wrapper delegating to the opsx-apply
#              workflow (task implementation + verification gate).
#   uiux       when requires_playwright=true or --with-uiux.
#   iso        by default; skipped with --skip-iso or requires_iso=false.
#
# Gating metadata: openspec/changes/<change>/routing.json (optional, advisory):
#   {
#     "requires_council": true,
#     "requires_playwright": true,
#     "requires_iso": true,
#     "requires_market_read": true,  # advisory: forces the five-persona council
#                                     # (incl. market-read lens) even at low/medium complexity
#     "complexity": "low" | "medium" | "high"
#   }
#
# Council gate (machine-parseable marker contract):
#   openspec/changes/<change>/VERDICT.md MUST contain exactly one line matching
#   ^STATUS: (APPROVED|REJECTED)$ as its FIRST `STATUS:`-prefixed line.
#     APPROVED  → proceed to the next stage
#     REJECTED  → run the bounded revise loop (revise → validate → re-review)
#                 up to MAX_COUNCIL_REVISIONS; halt exit 2 if still REJECTED
#     absent or ambiguous → inconclusive halt with exit code 3
#   The marker line is parsed (exact match), never substring-grepped, so prose
#   like "rejected items: X" cannot trip the gate.
#
# Revision loop state:
#   revision.json (change dir) — revision counter, last verdict, timestamp.
#     Created by the first revise pass; read before each pass so re-runs resume
#     instead of restarting (cap enforced across invocations); deleted on an
#     APPROVED re-review and by --reset-revisions; never deleted otherwise.
#   revision.md (change dir) — per-pass audit trail of covered/residual items;
#     never deleted by the pipeline.
#   .revise-context.json (change dir) — transient artifact-path context written
#     before each revise pass and removed after it.
#
# Logs: every stage writes logs/pipeline/<change>-<stage>-<ts>.log

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${ROOT_DIR}/logs/pipeline"

CHANGE=""
DRY_RUN=0
WITH_COUNCIL=0
WITH_UIUX=0
SKIP_ISO=0
SKIP_REVISE=0
MAX_COUNCIL_REVISIONS="${PIPELINE_MAX_REVISIONS:-2}"
RESET_REVISIONS=0
REVISION_FILE="revision.json"

usage() {
  awk 'NR > 1 && $0 !~ /^#/ { exit } NR > 1 { sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run) DRY_RUN=1 ;;
      --with-council) WITH_COUNCIL=1 ;;
      --with-uiux) WITH_UIUX=1 ;;
      --skip-iso) SKIP_ISO=1 ;;
      --skip-revise) SKIP_REVISE=1 ;;
      --max-revisions)
        shift
        if [[ $# -eq 0 || ! "$1" =~ ^[1-9][0-9]*$ ]]; then
          echo "pipeline.sh: --max-revisions requires a positive integer" >&2
          usage >&2
          exit 1
        fi
        MAX_COUNCIL_REVISIONS="$1"
        ;;
      --reset-revisions) RESET_REVISIONS=1 ;;
      --help|-h) usage; exit 0 ;;
      -*) echo "pipeline.sh: unknown option: $1" >&2; usage >&2; exit 1 ;;
      *) CHANGE="$1" ;;
    esac
    shift
  done
  if [[ -z "${CHANGE}" ]]; then
    echo "pipeline.sh: missing <change> argument" >&2
    usage >&2
    exit 1
  fi
  # Sanitize: the change name is interpolated into log paths and stage
  # instructions; restrict to kebab-case to prevent path/argument injection.
  if [[ ! "${CHANGE}" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
    echo "pipeline.sh: invalid change name (kebab-case only): ${CHANGE}" >&2
    exit 1
  fi
}

# --- routing.json gating -----------------------------------------------------

routing_get() {
  # routing_get <json-path> <default>
  local file="${CHANGE_DIR}/routing.json"
  if [[ ! -f "${file}" ]]; then
    printf '%s' "$2"
    return
  fi
  local val
  val="$(jq -r "$1" "${file}" 2>/dev/null || true)"
  if [[ -z "${val}" || "${val}" == "null" ]]; then
    printf '%s' "$2"
  else
    printf '%s' "${val}"
  fi
}

council_required() {
  [[ "${WITH_COUNCIL}" == "1" ]] && return 0
  [[ "$(routing_get '.requires_council' 'false')" == "true" ]] && return 0
  [[ "$(routing_get '.requires_market_read' 'false')" == "true" ]] && return 0
  [[ "$(routing_get '.complexity' 'low')" == "high" ]] && return 0
  return 1
}

uiux_required() {
  [[ "${WITH_UIUX}" == "1" ]] && return 0
  [[ "$(routing_get '.requires_playwright' 'false')" == "true" ]] && return 0
  return 1
}

iso_required() {
  [[ "${SKIP_ISO}" == "1" ]] && return 1
  [[ "$(routing_get '.requires_iso' 'true')" == "false" ]] && return 1
  return 0
}

# --- verdict parsing ---------------------------------------------------------

# parse_verdict <file> — prints APPROVED | REJECTED | INCONCLUSIVE.
# Reads the FIRST line starting with "STATUS:" and requires an exact match
# against the marker contract. Fixtures in scripts/tests/fixtures/.
parse_verdict() {
  local file="$1"
  local marker
  if [[ ! -f "${file}" ]]; then
    printf 'INCONCLUSIVE\n'
    return
  fi
  marker="$(grep -m1 '^STATUS:' "${file}" || true)"
  case "${marker}" in
    "STATUS: APPROVED") printf 'APPROVED\n' ;;
    "STATUS: REJECTED") printf 'REJECTED\n' ;;
    *) printf 'INCONCLUSIVE\n' ;;
  esac
}

# --- stage runner ------------------------------------------------------------

run_stage() {
  local stage="$1"
  local instruction="$2"
  local tools="${3:-}"
  local ts log cmd
  ts="$(date +%Y%m%d-%H%M%S)"
  log="${LOG_DIR}/${CHANGE}-${stage}-${ts}.log"
  cmd=(pi -p --approve)
  # --approve: headless `pi -p` never shows a trust prompt and falls back to
  # defaultProjectTrust (default "ask" → project resources ignored), so without
  # it project-local extensions in .pi/settings.json and .pi/prompts/ templates
  # would NOT load on a machine with no saved trust decision. See docs/usage.md.
  # Security note: --approve trusts and executes project-local packages
  # (third-party extension code) with full system access — see AGENTS.md.
  if [[ -n "${tools}" ]]; then
    cmd+=(--tools "${tools}")
  fi
  cmd+=(--prompt-template "${ROOT_DIR}/.pi/prompts/${stage}.md" "${instruction}")

  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '[dry-run] %s\n' "${cmd[*]}"
    printf '          log: %s\n' "${log}"
    return 0
  fi

  mkdir -p "${LOG_DIR}"
  echo "[pipeline] stage ${stage}: ${cmd[*]}"
  if "${cmd[@]}" >"${log}" 2>&1; then
    :
  else
    local rc=$?
    echo "[pipeline] stage ${stage} FAILED (exit ${rc}); log: ${log}" >&2
    exit 1
  fi
  echo "[pipeline] stage ${stage} ok; log: ${log}"
}

# --- revision loop state ----------------------------------------------------

# revision_read — prints the recorded revision count (0 when absent/invalid).
revision_read() {
  if [[ -f "${CHANGE_DIR}/${REVISION_FILE}" ]]; then
    jq -r '.revisions // 0' "${CHANGE_DIR}/${REVISION_FILE}" 2>/dev/null || printf '0\n'
  else
    printf '0\n'
  fi
}

# revision_write <count> <last_verdict> — persists the counter in the change dir
# (jq is a declared pipeline prerequisite per AGENTS.md).
revision_write() {
  local count="$1" verdict="$2"
  jq -n --arg change "${CHANGE}" --argjson revisions "${count}" \
    --arg last_verdict "${verdict}" --arg updated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{change: $change, revisions: $revisions, last_verdict: $last_verdict, updated_at: $updated_at}' \
    > "${CHANGE_DIR}/${REVISION_FILE}"
}

# revision_history — one-line summary of the persisted revision counter (used in
# halt messages).
revision_history() {
  if [[ -f "${CHANGE_DIR}/${REVISION_FILE}" ]]; then
    jq -r '"change=\(.change) revisions=\(.revisions) last_verdict=\(.last_verdict) updated_at=\(.updated_at)"' "${CHANGE_DIR}/${REVISION_FILE}" 2>/dev/null
  else
    printf 'no revision.json (no revision passes recorded)'
  fi
}

# reset_revisions — --reset-revisions escape hatch: delete revision.json before
# execution (dry-run prints "would delete" instead); revision.md audit trail is
# preserved.
reset_revisions() {
  if [[ -f "${CHANGE_DIR}/${REVISION_FILE}" ]]; then
    if [[ "${DRY_RUN}" == "1" ]]; then
      echo "[pipeline] would delete ${CHANGE_DIR}/${REVISION_FILE}"
    else
      rm -f "${CHANGE_DIR}/${REVISION_FILE}"
      echo "[pipeline] deleted ${CHANGE_DIR}/${REVISION_FILE} (revision.md audit trail preserved)"
    fi
  fi
}

# write_revise_context — writes the transient .revise-context.json the
# read/write-only revise stage consumes: changeRoot + per-artifact
# existingOutputPaths from `openspec status --change <name> --json`.
write_revise_context() {
  local status_json
  status_json="$(openspec status --change "${CHANGE}" --json)" || {
    echo "[pipeline] openspec status failed; cannot write .revise-context.json" >&2
    exit 1
  }
  printf '%s' "${status_json}" | jq '{change: .changeName, changeRoot: .changeRoot, artifactPaths: [.artifactPaths | to_entries[] | {id: .key, existingOutputPaths: .value.existingOutputPaths}]}' > "${CHANGE_DIR}/.revise-context.json"
}

# council_revision_loop — bounded revise → validate → re-review loop. Runs only
# after a REJECTED council verdict (first pass or re-review). The cap is read
# from revision.json so re-runs resume instead of restarting.
council_revision_loop() {
  local pass max count verdict
  count="$(revision_read)"
  max="${MAX_COUNCIL_REVISIONS}"
  pass=$((count + 1))
  while [[ "${pass}" -le "${max}" ]]; do
    echo "[pipeline] revision pass ${pass}/${max} (resumed from ${count})"
    # Write the transient artifact-path context the read/write-only revise
    # stage consumes (it has no shell and must not run openspec itself).
    write_revise_context
    run_stage revise "${CHANGE} (verdict-driven revision pass ${pass})" "read,write"
    rm -f "${CHANGE_DIR}/.revise-context.json"
    # Post-pass syntax check the revise agent cannot run itself; a failed
    # validation halts before the council re-review burns a pass.
    echo "[pipeline] openspec validate ${CHANGE}"
    if ! openspec validate "${CHANGE}" >/dev/null 2>&1; then
      echo "[pipeline] openspec validate FAILED after revision pass ${pass}" >&2
      exit 1
    fi
    run_stage council "Re-review the REVISED design of OpenSpec change ${CHANGE} (pass ${pass}). Write VERDICT.md." "read,write"
    verdict="$(parse_verdict "${CHANGE_DIR}/VERDICT.md")"
    revision_write "${pass}" "${verdict}"
    case "${verdict}" in
      APPROVED)
        echo "[pipeline] council re-review APPROVED (pass ${pass}); clearing revision counter"
        rm -f "${CHANGE_DIR}/${REVISION_FILE}"
        return 0 ;;
      REJECTED)
        echo "[pipeline] council re-review REJECTED (pass ${pass})" >&2
        if [[ "${pass}" -eq "${max}" ]]; then
          echo "[pipeline] revision cap (${max}) exhausted — halting pipeline" >&2
          echo "[pipeline] revision history: $(revision_history)" >&2
          echo "[pipeline] logs: ${LOG_DIR}/${CHANGE}-revise-*.log, ${LOG_DIR}/${CHANGE}-council-*.log" >&2
          exit 2
        fi ;;
      *)
        echo "[pipeline] council verdict INCONCLUSIVE — halting" >&2
        exit 3 ;;
    esac
    pass=$((pass + 1))
  done
  # Deterministic halt if the loop exits without a verdict (should not happen
  # for max >= 1).
  echo "[pipeline] revision loop exhausted without a verdict — halting" >&2
  exit 2
}

# --- pipeline ----------------------------------------------------------------

main() {
  parse_args "$@"
  CHANGE_DIR="${ROOT_DIR}/openspec/changes/${CHANGE}"
  if [[ ! -d "${CHANGE_DIR}" ]]; then
    echo "pipeline.sh: change dir not found: ${CHANGE_DIR}" >&2
    exit 1
  fi

  echo "[pipeline] change=${CHANGE} dry_run=${DRY_RUN} with_council=${WITH_COUNCIL} with_uiux=${WITH_UIUX} skip_iso=${SKIP_ISO} skip_revise=${SKIP_REVISE} max_revisions=${MAX_COUNCIL_REVISIONS}"

  # --reset-revisions: delete revision.json before running any stage (escape
  # hatch after a capped halt); revision.md audit trail is preserved.
  if [[ "${RESET_REVISIONS}" == "1" ]]; then
    reset_revisions
  fi

  # 1. architect — unconditional (thin opsx-propose wrapper)
  run_stage architect "${CHANGE}"

  # 2. council — conditional; verdict gate + bounded revision loop
  if council_required; then
    if [[ "${DRY_RUN}" == "1" ]]; then
      # Nothing executes in dry-run, so no fresh VERDICT.md is produced; when a
      # prior REJECTED verdict exists on disk and the loop is enabled, print the
      # revise/re-review command list the loop would run (no execution).
      if [[ -f "${CHANGE_DIR}/VERDICT.md" ]] \
        && [[ "$(parse_verdict "${CHANGE_DIR}/VERDICT.md")" == "REJECTED" ]] \
        && [[ "${SKIP_REVISE}" != "1" ]]; then
        local _pass _count _max
        _count="$(revision_read)"
        _max="${MAX_COUNCIL_REVISIONS}"
        _pass=$((_count + 1))
        while [[ "${_pass}" -le "${_max}" ]]; do
          run_stage revise "${CHANGE} (verdict-driven revision pass ${_pass})" "read,write"
          echo "[dry-run] openspec validate ${CHANGE}"
          run_stage council "Re-review the REVISED design of OpenSpec change ${CHANGE} (pass ${_pass}). Write VERDICT.md." "read,write"
          _pass=$((_pass + 1))
        done
      else
        echo "[pipeline] council verdict gate skipped in dry-run mode"
      fi
    else
      # Council runs with a restricted tool allowlist: read + write only.
      # No bash, no git — the agent cannot mutate the repo beyond VERDICT.md.
      run_stage council "Review the design of OpenSpec change ${CHANGE}. Write VERDICT.md." "read,write"
      local verdict
      verdict="$(parse_verdict "${CHANGE_DIR}/VERDICT.md")"
      case "${verdict}" in
        APPROVED)
          echo "[pipeline] council APPROVED; proceeding" ;;
        REJECTED)
          if [[ "${SKIP_REVISE}" == "1" ]]; then
            echo "[pipeline] council REJECTED — halting pipeline (--skip-revise)" >&2
            echo "[pipeline] halt reason recorded in council stage log under ${LOG_DIR}" >&2
            exit 2
          fi
          council_revision_loop ;;
        *)
          echo "[pipeline] council verdict INCONCLUSIVE (absent or ambiguous STATUS: marker) — halting" >&2
          exit 3 ;;
      esac
    fi
  else
    echo "[pipeline] council skipped (not required by routing.json or flags)"
  fi

  # 3. sdet — unconditional (thin opsx-apply wrapper, owns verification gate)
  run_stage sdet "${CHANGE}"

  # 4. uiux — conditional
  if uiux_required; then
    run_stage uiux "${CHANGE}"
  else
    echo "[pipeline] uiux skipped (not required by routing.json or flags)"
  fi

  # 5. iso — conditional
  if iso_required; then
    run_stage iso "${CHANGE}"
  else
    echo "[pipeline] iso skipped (--skip-iso or requires_iso=false)"
  fi

  echo "[pipeline] all stages complete for ${CHANGE}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
