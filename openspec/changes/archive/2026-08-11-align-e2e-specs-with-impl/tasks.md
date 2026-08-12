# Tasks

## 1. Feature-gating delta (crm-feature-gating-e2e)

- [x] 1.1 [OPS-GOV] Rewrite the `Free plan restricts Pro features` requirement delta: Empresas, Negocios, and Actividad tabs SHALL render **disabled** (visible, non-interactive, with upgrade hint) for `test-org-free`; navigating to a restricted view SHALL show the upgrade banner. Verify: `grep -n "disabled" openspec/changes/align-e2e-specs-with-impl/specs/crm-feature-gating-e2e/spec.md`.
- [x] 1.2 [OPS-GOV] Rewrite the `Enterprise plan restricts Tags to Enterprise tier` requirement delta: Etiquetas SHALL render disabled for `test-org-pro`; all tabs SHALL be enabled for `test-org-enterprise`. Verify: `grep -n "disabled" openspec/changes/align-e2e-specs-with-impl/specs/crm-feature-gating-e2e/spec.md`.

## 2. Test-infrastructure delta (crm-test-infrastructure)

- [x] 2.1 [OPS-GOV] Rewrite the `Test database seeded with test organizations` requirement delta: seeding SHALL be performed by `cmd/seed-e2e` invoked by the `make test-e2e` bootstrap (`run_e2e.sh`) and by CI after migrations; `global-setup.ts` SHALL validate seeded orgs rather than create them. Verify: `grep -n "seed-e2e" openspec/changes/align-e2e-specs-with-impl/specs/crm-test-infrastructure/spec.md`.

## 3. Verification gate

- [x] 3.1 [OPS-GOV] Run `openspec validate` on the change; fix any schema errors. Verify: command exits 0.
- [x] 3.2 [OPS-GOV] Confirm the corrected wording matches implementation and tests: `grep -n "disabled={tab.disabled}" next_b2b_starter/app/dashboard/crm/crm-page.tsx` and `grep -n "getAttribute(\\"disabled\\")" next_b2b_starter/e2e/specs/feature-gating.spec.ts` both return matches.

- [ ] 3.3 [OPS-GOV] **Archive decision:** archive. All verification greps pass (disabled wording in delta, `disabled={tab.disabled}` in crm-page.tsx:100, `getAttribute("disabled")` in feature-gating.spec.ts:9/18/27/36, seed-e2e wording in delta); `openspec validate --changes` passes 16/16 (bootstrap-health-check renamed from `00-bootstrap-health-check` and given its missing delta to reach conformance). No code impact, no external dependencies. Executed in archive sweep.
