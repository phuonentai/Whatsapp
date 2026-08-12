# Tasks: Bootstrap Health Check Verification

All tasks are verification-only; no source code is modified.

- [x] Verify `GET /healthz` exists in `go-b2b-starter/cmd/mock-siigo/main.go`.
      Command: `grep -n "healthz" go-b2b-starter/cmd/mock-siigo/main.go`
- [x] Verify `scripts/run-qa-server.sh` is executable and contains the health wait loop.
      Command: `test -x scripts/run-qa-server.sh && grep -n "CANDIDATES" scripts/run-qa-server.sh`
- [x] Confirm no demo task touches billing, auth, production configuration, customer data, or requires external credentials.
- [x] Confirm `routing.json` is present and all fields are `false` / `low` as specified.

## Verification Commands

```
test -x scripts/run-qa-server.sh
grep -n "healthz" go-b2b-starter/cmd/mock-siigo/main.go
jq . openspec/changes/00-bootstrap-health-check/routing.json
```

- [ ] **Archive decision (2026-08-11):** **Archive** — all verification tasks pass; change renamed from `00-bootstrap-health-check` to letter-prefixed `bootstrap-health-check` and given its missing `qa-health-check` delta to reach `openspec validate` conformance (16/16). No code impact, no external dependencies. Executed in archive sweep.
