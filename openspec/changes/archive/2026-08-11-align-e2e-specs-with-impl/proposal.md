# Align e2e specs with implemented behavior

## Why

Two living e2e specs contradict the implemented, test-verified behavior, so the specs cannot be satisfied as written:

1. `crm-feature-gating-e2e` requires restricted tabs (Empresas, Negocios, Actividad, Etiquetas) to "SHALL NOT be visible", but the implementation (`next_b2b_starter/app/dashboard/crm/crm-page.tsx`) renders those tabs **visible but `disabled`**, with an upgrade hint, and shows an upgrade banner when navigating to a restricted view. The e2e test (`next_b2b_starter/e2e/specs/feature-gating.spec.ts`) asserts the `disabled` attribute, not absence. Disabled-but-visible is the deliberate UX (upgrade affordance); the spec demands the opposite of what is implemented and tested.

2. `crm-test-infrastructure` requires `global-setup.ts` to "SHALL seed the test database". The actual seed lives in `go-b2b-starter/cmd/seed-e2e` (4 orgs + accounts + subscriptions + quotas + agent settings), invoked by `go-b2b-starter/scripts/run_e2e.sh` and both CI workflows *before* the suite boots. `next_b2b_starter/e2e/global-setup.ts` is a stub that only logs preconditions. Having global-setup run the seed would double-seed and skip the DB reset that makes runs deterministic.

## What Changes

- **`crm-feature-gating-e2e` (modified):** Restricted feature tabs SHALL render **disabled** (visible, non-interactive, with upgrade hint) instead of being hidden. Navigating to a restricted view SHALL show the upgrade banner. API-level 403 enforcement is unchanged.
- **`crm-test-infrastructure` (modified):** Seeding SHALL be performed by the `seed-e2e` command, invoked by the canonical `make test-e2e` bootstrap (`run_e2e.sh`) and by CI, which runs migrations then seeds the test database before booting the stack. `global-setup.ts` SHALL validate seeded orgs exist, not create them.

## Capabilities

### Modified Capabilities

- `crm-feature-gating-e2e`
- `crm-test-infrastructure`

## Impact

- **Specs only.** No application code, test code, or infrastructure changes. The implementation and e2e assertions already match the corrected wording.
- **Dev workflow:** none — this change only reconciles the OpenSpec source of truth with existing behavior so the verification gate can pass.

## Assumptions

- Restricted tabs are intentionally kept visible-but-disabled as an upgrade affordance; hiding them (the old spec wording) would require a separate feature change.
- `make test-e2e` / `run_e2e.sh` remains the canonical offline e2e bootstrap (migrations → seed → boot backend `:8080` + frontend `:3001` → Playwright). The GitHub Actions `ci.yml` and `.gitlab-ci.yml` run-e2e job follow the same sequence.

## Non-Goals

- No change to feature-gating behavior, the API 403 gate, or any test assertions.
- No change to which orgs/accounts `seed-e2e` creates.
- No implementation of global-setup-driven seeding.
