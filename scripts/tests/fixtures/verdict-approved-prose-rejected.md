# Fixture: verdict whose PROSE mentions "rejected" but whose marker is APPROVED.
# The pipeline MUST parse the first ^STATUS: line exactly — this file must be
# treated as APPROVED (the prose must not trip a substring grep).

## Persona findings

- Staff Security Engineer: **rejected items: none** — no auth surface touched.
- Staff DBA: rejected the idea of a table-locking migration; design uses expand-contract.
- SRE: rejected the need for distributed locks here.

## Verdict

STATUS: APPROVED
