# AGENTS.md

Guidance for AI coding agents (including MonkeyCode) working in this repository. Read this file before starting any task, and follow it for every change you make.

## Source of Truth: OpenSpec

This repository is **spec-driven**. The behavioural source of truth for the software does not live in the code — it lives in the OpenSpec tree:

- `openspec/specs/<capability>/spec.md` — **Living specifications**. Each capability (auth, CRM, billing, WhatsApp, etc.) has an authoritative spec of required behaviour. These are the source of truth for what the system must do.
- `openspec/changes/<change>/` — **Active change proposals**. Each contains `proposal.md`, `design.md`, `tasks.md`, and delta specs under `specs/` describing work that is planned or in progress. When a change is completed and archived, its deltas are folded into the living specs.
- `openspec/changes/archive/` — **Archived changes** whose deltas have already been merged into the living specs.
- `openspec/config.yaml` — OpenSpec configuration: schema, project context, and governance rules that all proposals, specs, designs, and tasks must obey.

## Mandatory Workflow for Every Code Task

1. **Read the relevant specs first.** Before writing or modifying any code, read every `openspec/specs/<capability>/spec.md` that applies to the code you will touch. If the spec describes a feature, endpoint, schema, permission, or status code, the implementation MUST match it.
2. **Check for active changes.** List `openspec/changes/` and read any active change that touches the same capability (its `proposal.md`, `design.md`, `tasks.md`, and delta specs). If your work matches an active change, implement against that change's artifacts and keep them up to date.
3. **Treat OpenSpec Markdown as authoritative.** Do not invent behaviour that contradicts the specs. Where code and spec disagree, the spec wins — adjust the code (or, if the spec is genuinely wrong, propose an OpenSpec change first).
4. **Never edit the OpenSpec tree casually.** `openspec/specs/` and `openspec/changes/` are versioned governance artifacts. Do not delete, move, or rewrite them. Behavioural changes go through a change proposal under `openspec/changes/<change>/`; they are not made by editing code in isolation.

## Project Context

Monorepo with two applications:

- `go-b2b-starter/` — Go 1.25 backend: Gin, PostgreSQL + SQLC (pgvector), Stytch B2B (auth/RBAC), Polar.sh (billing), AI/RAG. Clean Architecture enforced: `domain` -> `app` -> `infrastructure`, dependency injection via uber-go/dig. Key commands: `make dev`, `make server`, `make sqlc`, `make test`.
- `next_b2b_starter/` — Next.js 16 + React 19 frontend: TypeScript (strict), Tailwind, shadcn/ui, TanStack Query, Server Actions. Key commands: `pnpm dev`, `pnpm build`, `pnpm lint`.

### Dual Single Point of Truth (SSOT) Architecture

Extracted from `openspec/config.yaml` — follow these invariants:

- **Static SSOT** — this Git repository: authoritative for code, DB migrations, SQLC query models, OpenAPI schemas, and OpenSpec deltas. Code changes cannot alter these contracts without an archived OpenSpec change proposal.
- **Runtime SSOT** — Stytch B2B: sole authority for member identity, organization multi-tenancy, session validity, and RBAC. Local PostgreSQL MUST NOT store passwords, MFA tokens, or session tokens — store only `stytch_member_id` / `stytch_organization_id` foreign keys.
- Inbound requests are verified via Stytch B2B JWKS (local cache TTL <= 300s); Stytch webhooks are verified via `stytch-signature`/HMAC headers before any DB mutation; outbound Stytch API calls are wrapped in a two-tier circuit breaker (threshold 5, timeout 10s, half-open probe 2). On Stytch degradation, fall back to cached JWKS validation with read-only scope.

### Governance Rules (from `openspec/config.yaml`)

- **Proposals**: Any proposal modifying auth flow or data persistence MUST explicitly reference Stytch B2B API contracts, MUST define rollback strategies for both Git state and Stytch tenant policy state, and MUST include a "Non-Goals" section rejecting local credential storage.
- **Specs**: Define state-transition invariants across the Go/Stytch boundary, specify fallback/circuit-breaker states for every outbound Stytch SDK invocation, and map Stytch RBAC roles to local authorization contexts with strict type definitions.
- **Design**: Go domain models MUST NOT import Stytch SDKs or external transport packages; infrastructure adapters (`infrastructure/auth/stytch`) MUST implement domain interface abstractions; DB operations triggered by Stytch webhooks MUST be idempotent (transaction-isolated state checks).
- **Tasks**: Cap execution at 2 hours per task unit; tag tasks with `[BE-DOMAIN]`, `[BE-INFRA]`, `[DB-SQLC]`, or `[FE-NEXT]`; require verification criteria covering signature-validation tests and mock fallback execution.

## Other Reference Material

- `CLAUDE.md` — project overview and quick navigation (also used by Claude Code).
- `go-b2b-starter/.claude/CLAUDE.md` — backend-specific conventions and commands.
- `next_b2b_starter/.claude/CLAUDE.md` — frontend-specific conventions and commands.
- `DEVELOPMENT.md`, `SETUP.md`, `README.md` — development workflow, setup, and stack details.
- `OPENSPEC_USAGE.md` — human-facing guide for how OpenSpec files are used together with MonkeyCode.

## Golden Rule

When in doubt, the answer is in `openspec/`. Read the specs and the active changes first; implement to match them exactly; and never invent behaviour that contradicts them.
