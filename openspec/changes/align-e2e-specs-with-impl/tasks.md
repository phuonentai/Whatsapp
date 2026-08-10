# Tasks

## 1. Feature-gating delta (crm-feature-gating-e2e)

- [ ] 1.1 [OPS-GOV] Rewrite the `Free plan restricts Pro features` requirement delta: Empresas, Negocios, and Actividad tabs SHALL render **disabled** (visible, non-interactive, with upgrade hint) for `test-org-free`; navigating to a restricted view SHALL show the upgrade banner. Verify: `grep -n "disabled" openspec/changes/align-e2e-specs-with-impl/specs/crm-feature-gating-e2e/spec.md`.
- [ ] 1.2 [OPS-GOV] Rewrite the `Enterprise plan restricts Tags to Enterprise tier` requirement delta: Etiquetas SHALL render disabled for `test-org-pro`; all tabs SHALL be enabled for `test-org-enterprise`. Verify: `grep -n "disabled" openspec/changes/align-e2e-specs-with-impl/specs/crm-feature-gating-e2e/spec.md`.

## 2. Test-infrastructure delta (crm-test-infrastructure)

- [ ] 2.1 [OPS-GOV] Rewrite the `Test database seeded with test organizations` requirement delta: seeding SHALL be performed by `cmd/seed-e2e` invoked by the `make test-e2e` bootstrap (`run_e2e.sh`) and by CI after migrations; `global-setup.ts` SHALL validate seeded orgs rather than create them. Verify: `grep -n "seed-e2e" openspec/changes/align-e2e-specs-with-impl/specs/crm-test-infrastructure/spec.md`.

## 3. Verification gate

- [ ] 3.1 [OPS-GOV] Run `openspec validate` on the change; fix any schema errors. Verify: command exits 0.
- [ ] 3.2 [OPS-GOV] Confirm the corrected wording matches implementation and tests: `grep -n "disabled={tab.disabled}" next_b2b_starter/app/dashboard/crm/crm-page.tsx` and `grep -n "getAttribute(\\"disabled\\")" next_b2b_starter/e2e/specs/feature-gating.spec.ts` both return matches.
- [ ] 3.3 [OPS-GOV] Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>`. Verify: entry present in this file.
