## 1. Spec Reconciliation

- [x] 1.1 [OPS-GOV] In `openspec/changes/reconcile-auth-spec-drift/specs/stytch-nextjs-components/spec.md`, write MODIFIED requirements: "Login page renders a custom email form" (email-only, membership pre-validation via Stytch `members.search`, neutral unknown-member message, no password) and "Settings uses custom member management" (custom `member-list.tsx`/`invite-member.tsx`, no AdminPortal dependency); REMOVED requirements: pre-built StytchB2B login component, AdminPortal member-management/SSO components, "no custom form components". Verification: `openspec validate reconcile-auth-spec-drift` passes.

- [x] 1.2 [OPS-GOV] Update the spec Purpose to note SSO surfacing is deferred until SSO connections are productized (product intent preserved). Verification: `openspec validate` passes.

## 2. Docs Alignment

- [x] 2.1 [OPS-GOV] `STYTCH_CONFIGURATION.md`: replace `/api/auth/magic-link` with the `sendMagicLink` server action and `/api/auth/logout` with the `logout` server action in all references; keep security rationale. Verification: grep shows no stale route references.

## 3. Verification Gate

- [x] 3.1 [OPS-GOV] `openspec validate reconcile-auth-spec-drift` passes; `openspec status --change reconcile-auth-spec-drift` shows tasks complete.
- [x] 3.2 [OPS-GOV] Consistency: grep living spec `openspec/specs/stytch-nextjs-components/spec.md` (after archive) shows no StytchB2B/AdminPortal mandates; `STYTCH_CONFIGURATION.md` has no `/api/auth/magic-link` or `/api/auth/logout` references.

## Gate Results

- 1.1: PASS — `openspec validate reconcile-auth-spec-drift` → "Change 'reconcile-auth-spec-drift' is valid".
- 1.2: PASS — `openspec validate --all` → "Totals: 105 passed, 0 failed (105 items)".
- 2.1: PASS — grep of `next_b2b_starter/STYTCH_CONFIGURATION.md` for `api/auth` and `magic-link/route` → no matches.
- 3.1: PASS — `openspec validate reconcile-auth-spec-drift` valid; `openspec status --change reconcile-auth-spec-drift` → Progress 4/4 artifacts complete.
- 3.2: PARTIAL (archive-time check recorded) — `STYTCH_CONFIGURATION.md` grep clean (verified now); living-spec StytchB2B/AdminPortal grep is an ARCHIVE-TIME check: the living spec `openspec/specs/stytch-nextjs-components/spec.md` is intentionally NOT edited by this change (delta lives under the change folder); its StytchB2B/AdminPortal mandates will be removed when this change is archived and the delta folds into the living spec. Delta correctness verified now (no StytchB2B/AdminPortal mandates in the delta).

## Archive Decision

**Archive deferred:** centralized verification phase per repo practice. On archive, re-run the 3.2 living-spec grep to confirm no StytchB2B/AdminPortal mandates remain after the delta fold.
