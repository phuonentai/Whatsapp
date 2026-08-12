# Fixture: rejected verdict. First STATUS: line must be exactly `STATUS: REJECTED`.

## Persona findings

- Staff Security Engineer: missing tenant isolation on the new endpoint.
- Staff DBA: migration violates expand-contract.
- SRE: no rollback strategy.

## Verdict

STATUS: REJECTED

## Required changes

1. Scope all queries to tenant_id.
2. Split migration into expand then contract.
3. Add rollback notes.
