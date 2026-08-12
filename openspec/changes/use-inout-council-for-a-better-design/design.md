# Council-Driven Design Revision Loop — Design

## Context

The native agent pipeline (`scripts/pipeline.sh`) chains stateless stages via `pi -p --prompt-template .pi/prompts/<stage>.md`: architect → council → sdet → uiux → iso. The council stage reviews `design.md` and writes `VERDICT.md` with a machine-parseable marker (`^STATUS: (APPROVED|REJECTED)$`, first `STATUS:` line, exact-match parse in `parse_verdict`). Today a `REJECTED` verdict halts `main()` with exit 2 immediately — nothing consumes the verdict's findings. Evidence this hurts: `new-client-billing-lifecycle` was REJECTED twice on 2026-08-12 and required two manual revision cycles (its own `design.md`/`VERDICT.md` record both verdicts) before it passed. The council verdict already contains a numbered "Required design changes" list (per the council method) but it is prose that no stage reads.

The `opsx-update` workflow (`.pi/prompts/opsx-update.md`, experimental) already implements artifact revision: read existing artifacts, reconcile in any direction, revise only `existingOutputPaths`, never edit code. Its interactive step ("Confirm every edit with the user") is TUI-oriented; a headless stage overrides that, exactly as `council.md` already overrides interactive defaults for the review (see the council prompt's "Do NOT ask the user anything").

**Constraints (governance):** stages are thin wrappers over OpenSpec workflows (`architect` → opsx-propose, `sdet` → opsx-apply) — `revise` SHALL delegate to opsx-update and never re-implement it. Stages run headless with `pi -p --approve`; the council runs with a restricted tool allowlist (`--tools read,write`) so it cannot mutate the repo beyond `VERDICT.md`. The change touches no auth, billing, webhooks, or data migration; no Stytch contracts; no tenant-scoped data.

## Goals / Non-Goals

**Goals:**

- Convert council output (REJECTED verdict + numbered required changes) into design input (revised planning artifacts) automatically, within one pipeline run.
- Bounded loop: `MAX_COUNCIL_REVISIONS` (default 2) revise→re-review iterations, then the existing halt semantics (exit 2 on REJECTED, exit 3 on inconclusive marker) — the marker contract and exit codes are preserved.
- Machine-parseable coverage: each revise pass records which numbered required changes it addressed (`revision.md`), and the revision count is persisted (`revision.json`) so the cap is enforced across pipeline re-runs and the loop is idempotent.
- Deterministic and explicit: new flags `--skip-revise` (restore immediate halt) and `--max-revisions <N>`; every iteration logged under `logs/pipeline/`.
- Traceability: iso stage records revision-loop outcomes; `AGENTS.md` documents the `revise` stage and the loop.

**Non-Goals:**

- Changing council personas, the marker format, or the three-persona review method (`council.md` unchanged except the numbered-list pin and the conditional re-review read clause).
- Auto-applying code: the revise stage NEVER edits application source; code landing remains sdet/opsx-apply's job, driven by the final approved design + tasks.
- Changing OpenSpec schema artifacts, build order, or lifecycle gates.
- Re-implementing opsx-update; the revise prompt is a thin adapter over it.
- Verdict auto-resolution without a human-visible record: every pass leaves `revision.md` + logs; the cap is a hard stop, not a silent override.

## Decisions

### D1 — `revise` stage as a thin headless wrapper over opsx-update (CLI boundary: pipeline.sh injection)

New prompt template `.pi/prompts/revise.md`, invoked by pipeline stage name `revise` (the existing `run_stage` machinery needs no change — it takes `stage`, `instruction`, `tools`). The stage runs with `--tools read,write` (no bash/git) — the revise agent has NO shell, enforcing least privilege: the LLM reconciles file contents only and cannot execute shell commands. All CLI work lives in the deterministic orchestration layer:

- **`pipeline.sh` writes the context.** Before each revise pass, `pipeline.sh` runs `openspec status --change <name> --json`, extracts `artifactPaths.<id>.existingOutputPaths` + `changeRoot`, and writes them to a transient context file `openspec/changes/<change>/.revise-context.json` (regenerable per pass; consistent with the architecture's "shared state lives on disk in the change dir" pattern). The file is removed when the pass completes, keeping the change dir clean between runs.
- **The revise prompt SHALL NOT run any `openspec` CLI command** (it cannot — no bash). It SHALL: (1) read `openspec/changes/<change>/VERDICT.md` and extract the numbered required-design-changes list as the authoritative input; (2) read `.revise-context.json` for the artifact paths; (3) apply opsx-update's reconciliation rules to those paths — reconcile in ANY direction (`proposal.md`/`design.md`/`tasks.md`/`specs/**/*.md` via the provided `existingOutputPaths`), revise only files that already exist, never create new artifacts; (4) override opsx-update's interactive confirmation with a headless directive (revise all artifacts to address the verdict items — precedent: `council.md` overrides interactivity); (5) write `revision.md`; (6) NEVER edit code or files outside the change directory.
- **Post-pass validation.** `pipeline.sh` runs `openspec validate <change>` after each revise pass — a deterministic syntax check the read/write-restricted agent cannot perform for itself; a failed validation halts the pass (stage failure → exit 1) before the council re-review burns a pass.

The delegation contract is therefore "opsx-update's reconciliation rules applied to wrapper-provided paths", not "run the opsx-update workflow's CLI steps". Alternatives considered: a full new workflow in the openspec CLI — rejected, violates the thin-wrapper architecture and duplicates opsx-update; a revise agent with bash — rejected, weakens the least-privilege posture when the wrapper can do the CLI instead.

### D2 — Verdict input contract: numbered required changes + re-review read contract

`council.md` gains two additions: (a) one pinned sentence — a `REJECTED` verdict SHALL include a numbered "Required design changes" list (it already does in practice; pinning makes it a contract); (b) a **conditional read clause** in the "Your role" section, right after the read-`design.md` step: *"If `revision.md` and a prior `VERDICT.md` already exist in the change directory, read them before evaluating the revised `design.md` — the prior verdict's numbered required changes and the revision's residual items are part of the review input."* Baking the clause into the template (rather than appending it to the re-review instruction) is deliberate: the template's strong prohibitions ("That is the entire task") absorb appended instructions, so the re-review council must be told at template level to read the delta and the residual risks. `revision.md` (written per pass) uses a stable, machine-parseable shape:

```
# Revision N (YYYY-MM-DD)
- [x] item 1 — <artifact(s) revised>
- [x] item 2 — <artifact(s) revised>
- [ ] item 3 — residual (recorded for council re-review)
```

The revise stage SHALL check off each numbered item or explicitly record it as residual risk; the council re-review reads both `VERDICT.md` (prior) and `revision.md` (what changed). Note: each re-review OVERWRITES `VERDICT.md`; the durable record of earlier verdicts is `revision.md` + the timestamped pipeline logs (D5), not a verdict history file. Alternative: extending the VERDICT marker contract (e.g., `REQUIRED: 3`) — rejected: changes the marker grammar that `parse_verdict` and fixtures depend on.

### D3 — Bounded loop in `pipeline.sh` main() (+ recovery flags)

Replaces the REJECTED branch of the existing council gate:

```
REJECTED → loop n = current+1 .. MAX:
  write .revise-context.json (from `openspec status --change <name> --json`)
  run_stage revise "<change> (verdict-driven revision pass N)" "read,write"
  rm .revise-context.json
  openspec validate <change>        # wrapper-side post-pass check; failure → exit 1
  run_stage council "Re-review the REVISED design of OpenSpec change <change> (pass N). Write VERDICT.md." "read,write"
  verdict = parse_verdict
  APPROVED → rm -f revision.json; break, proceed to sdet
  REJECTED → if n == MAX → halt exit 2 (log halt + revision history)
  INCONCLUSIVE → halt exit 3 (unchanged)
```

New flags parsed in `parse_args`: `--skip-revise` (REJECTED → immediate exit 2, today's behavior), `--max-revisions <N>` (positive int; default 2), and `--reset-revisions` (delete `revision.json` before execution — the documented escape hatch for the manual-fix-after-capped-halt scenario; dry-run prints "would delete" instead of deleting; `revision.md` audit trail is preserved). **Auto-clear on APPROVED:** the loop deletes `revision.json` when a re-review approves, so the counter cannot poison a later, unrelated council run (the file can only exist after a revise pass, so no other path needs the cleanup). `council_required()` unchanged; the loop runs only when the council gate runs (i.e., when a verdict exists). Exit codes preserved: APPROVED → proceed, REJECTED after cap → 2, INCONCLUSIVE → 3, stage failure → 1. `--dry-run` prints the revise/re-review command list instead of executing (same no-op treatment as the current gate).

### D4 — `revision.json` persistence (idempotency across re-runs) and lifecycle

Change dir sidecar, e.g. `{"change": "<name>", "revisions": 0, "last_verdict": "REJECTED", "updated_at": "<ts>"}`. The loop SHALL read it before each pass and SHALL write it after each pass (`jq` is already a pipeline prerequisite per AGENTS.md). On a re-run after a partial failure, the loop resumes from the recorded count instead of restarting — the cap is enforced across invocations. The architect stage is unaffected (it must not pre-create the sidecar; only the loop writes it). **Lifecycle:** the sidecar is created by the first revise pass, deleted by the loop on APPROVED (D3), and deleted by `--reset-revisions`; `revision.md` is never deleted by the pipeline (audit trail). Alternative: shell counter only — rejected: a re-run restarts at zero and can exceed the cap.

### D5 — Logging and observability

Each revise pass and each council re-review uses the existing `run_stage` logging (`logs/pipeline/<change>-revise-<ts>.log`, `logs/pipeline/<change>-council-<ts>.log` — the pass number is carried in the instruction string and thus in the logged command, while the existing timestamped log naming disambiguates files). The halt path (D3) prints the revision history line and the log paths, extending the current "halt reason recorded in council stage log" message. The iso stage (`iso.md`) gains a line item to record the revision count and final verdict in `docs/compliance/ISO_TRACEABILITY_MATRIX.md`.

### D6 — Documentation

`AGENTS.md` "Agent Pipeline" section: add the `revise` stage (thin wrapper over opsx-update, verdict-driven, never edits code, runs read/write-only with pipeline-provided artifact paths) and the bounded loop (REJECTED → up to `MAX_COUNCIL_REVISIONS` revise→re-review passes → halt exit 2; flags `--skip-revise`/`--max-revisions`/`--reset-revisions`; counter auto-cleared on approval). The `native-agent-pipeline` living spec is updated at archive via the delta. `routing.json` is untouched (advisory gating unchanged).

## Risks / Trade-offs

- **[Risk] Revise stage could over-edit artifacts (drift from intent)** → Mitigation: it is bound by the verdict's numbered list (D2 coverage check), writes `revision.md` for audit, and runs with `--tools read,write` only; code is out of scope by construction.
- **[Risk] An unaddressed item silently survives to re-review** → Mitigation: coverage check records unaddressed items as residual risk in `revision.md`, which the re-reviewing council reads; a second REJECTED verdict with the same item halts at the cap (exit 2) — the hard stop is preserved.
- **[Risk] Loop could spin across re-runs (cap evasion)** → Mitigation: `revision.json` persists the count (D4); re-runs resume, never restart.
- **[Risk] Headless revise contradicts opsx-update's interactive confirm** → Mitigation: precedent — `council.md` already overrides interactivity for headless stages; the revise prompt documents the override explicitly.
- **[Risk] Existing pipeline fixtures/tests assume immediate halt** → Mitigation: `--skip-revise` preserves today's behavior; fixtures for `parse_verdict` are untouched; new fixtures cover the loop and cap.
- **[Trade-off] Default cap of 2 adds wall-clock time to rejected designs** → the loop is bounded, logged, and idempotent; the alternative (current) behavior is one flag away (`--skip-revise`).

## Migration Plan

0. **Ordering constraint:** apply `add-council-market-read-gate` FIRST (its `council_required()`/persona edits are upstream of this change's REJECTED-branch loop; both touch `council.md`/`pipeline.sh`/`iso.md`/`AGENTS.md` and the `native-agent-pipeline` delta spec — this change rebases onto it).
1. Add `.pi/prompts/revise.md` (verdict-driven, opsx-update-reconciliation-rules-backed, non-interactive; runs with `--tools read,write`; reads `.revise-context.json` written by `pipeline.sh`).
2. Extend `parse_args`/`main()` in `scripts/pipeline.sh`: flags (`--skip-revise`/`--max-revisions`/`--reset-revisions`), loop, `.revise-context.json` write/remove, `openspec validate` post-pass check, `revision.json` read/write/clear, updated halt messages. Keep `parse_verdict` and the marker contract untouched.
3. Pin the numbered-required-changes sentence AND the conditional re-review read clause in `.pi/prompts/council.md`; update the iso prompt line item; update `AGENTS.md`.
4. Verify: `bash -n scripts/pipeline.sh`; `scripts/pipeline.sh <change> --dry-run` prints the revise/re-review commands; `--reset-revisions --dry-run` prints "would delete" and deletes nothing; fixture-based unit checks for `parse_verdict` still pass; shellcheck if available.
5. Rollback: revert the commit; `revision.json`/`revision.md`/`.revise-context.json` are advisory/transient files that may remain or be removed; no OpenSpec schema or marker-contract changes to unwind; no auth/billing/migration impact.

## Decisions recorded (2026-08-12)

Review decisions folded into this design from the explore session on this change:

- **CLI execution boundary (D1):** the revise agent runs `--tools read,write` with NO shell; `pipeline.sh` runs `openspec status --change <name> --json`, writes `.revise-context.json` before each pass (removed after), and runs `openspec validate <change>` after each pass. Least privilege: the LLM reconciles file contents only; all CLI work is deterministic orchestration.
- **Re-review context (D2):** the conditional read clause lives in the `council.md` template (not the re-review instruction) because the template's strong prohibitions absorb appended instructions; each re-review overwrites `VERDICT.md`, so earlier verdicts live in `revision.md` + logs.
- **Capped-halt recovery (D3/D4):** auto-clear `revision.json` on APPROVED; `--reset-revisions` flag as the explicit manual-fix escape hatch; no fragile file-mtime heuristics.
- **Fixed revision cap:** the cap stays fixed (default 2, `PIPELINE_MAX_REVISIONS` env, `--max-revisions` per-run). No item-count scaling — if an agent cannot resolve planning artifacts in two passes, the design needs human intervention; residual marking routes code-fix items to `tasks.md`.
- **Sibling sequencing:** `add-council-market-read-gate` lands FIRST (its `council_required()` edit is upstream of the loop's REJECTED-branch edit; ordering constraint recorded in both changes' tasks).
- **No human-in-the-loop gate:** the audit trail (`revision.md`, timestamped logs, `revision.json` history) is trusted; quality control is the revise coverage check → re-review council (reads `revision.md` per D2) → sdet verification gate → human PR review. A human gate would defeat the one-run-turnaround purpose.

## Open Questions

- *(none outstanding — both prior open questions are resolved above: re-review context is carried by the `council.md` conditional clause (D2), and the cap is fixed with `PIPELINE_MAX_REVISIONS`/`--max-revisions` as overrides; no per-change `routing.json` field. The "confirm during a real first rejection cycle" item becomes a post-implementation validation note, not a design open question.)*
