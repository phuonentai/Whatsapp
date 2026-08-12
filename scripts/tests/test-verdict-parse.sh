#!/usr/bin/env bash
# test-verdict-parse.sh — fixture tests for the council verdict marker parser.
#
# Verifies scripts/pipeline.sh's parse_verdict() against the fixtures in
# scripts/tests/fixtures/. The marker contract is:
#   first ^STATUS: line must be exactly `STATUS: APPROVED` or `STATUS: REJECTED`.
# Prose mentioning "rejected" must NOT trip the gate (see
# verdict-approved-prose-rejected.md).
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PIPELINE="${ROOT_DIR}/scripts/pipeline.sh"
FIXTURES="${ROOT_DIR}/scripts/tests/fixtures"

# shellcheck source=scripts/pipeline.sh
source "${PIPELINE}"

fail=0

check() {
  local expected="$1" file="$2"
  local actual
  actual="$(parse_verdict "${file}")"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "FAIL: $(basename "${file}") -> expected ${expected}, got ${actual}"
    fail=1
  else
    echo "ok:   $(basename "${file}") -> ${actual}"
  fi
}

check APPROVED   "${FIXTURES}/verdict-approved-prose-rejected.md"
check REJECTED   "${FIXTURES}/verdict-rejected.md"
check INCONCLUSIVE "${FIXTURES}/verdict-inconclusive.md"
check INCONCLUSIVE "${FIXTURES}/does-not-exist.md"

if [[ "${fail}" == "1" ]]; then
  echo "verdict-parse fixture tests FAILED"
  exit 1
fi
echo "verdict-parse fixture tests PASSED"
