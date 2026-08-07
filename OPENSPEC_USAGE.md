# OPENSPEC_USAGE.md

How the OpenSpec files in this repository are used together with MonkeyCode (and other AI coding agents).

## Overview

This repository is **spec-driven**. The authoritative description of what the software must do lives in the OpenSpec tree — not in the code. AI agents (MonkeyCode, Claude Code, etc.) read these files before making changes; human developers use them to review proposed work and track progress.

The machine-readable contract between the repository and the agents is `AGENTS.md` at the repository root. If you want an agent to work on this repository, simply tell it to **"follow the instructions in AGENTS.md"**.

## OpenSpec Tree Layout

```
openspec/
├── config.yaml            # Schema, project context (SSOT architecture), governance rules
├── specs/                 # Living specifications (source of truth)
│   ├── auth/...           # e.g. stytch-authorization, signup-stytch-compliance
│   ├── crm/...            # e.g. crm-core-data, contact-management, deal-management
│   └── whatsapp/...       # e.g. whatsapp-config-api, whatsapp-webhook-ingress
├── changes/               # Active change proposals (work in progress)
│   ├── <change>/
│   │   ├── proposal.md    # Why / What changes / Capabilities / Impact
│   │   ├── design.md      # Technical design decisions
│   │   ├── tasks.md       # Implementation tasks
│   │   └── specs/         # Delta specs that will be folded into specs/ on archive
│   └── archive/           # Completed changes, already merged into specs/
```

## The Workflow

### 1. Living specs are the source of truth

`openspec/specs/<capability>/spec.md` is the authoritative statement of required behaviour for that capability (endpoints, schemas, permissions, status codes, state transitions). When a spec and the code disagree, the spec wins — the code must be changed to match.

### 2. Changes start as proposals

Any behavioural change goes through a change proposal under `openspec/changes/<change>/`:

1. **Propose** — create a change directory with `proposal.md`, `design.md`, `tasks.md`, and delta specs in `specs/` describing what will change.
2. **Implement** — an agent implements the tasks, treating the delta specs as the requirements. The proposal is kept up to date as implementation proceeds.
3. **Archive** — when complete, the change is moved to `changes/archive/` and its delta specs are folded into the living specs in `specs/`.

### 3. Rules from config.yaml apply to every change

`openspec/config.yaml` encodes governance rules that all proposals, specs, designs, and tasks must obey. These cover the dual SSOT architecture (Git repo + Stytch B2B), Stytch-specific invariants (no local credential storage, circuit breakers, idempotent webhook handling), and task conventions (2-hour units, `[BE-*]`/`[DB-SQLC]`/`[FE-NEXT]` tags). An agent is expected to follow these automatically.

## Working with MonkeyCode

- **Start a task**: tell MonkeyCode to "follow the instructions in AGENTS.md". It will read the relevant `openspec/specs/*/spec.md` files and any active `openspec/changes/<change>/` affecting the same capability before touching code.
- **Propose new behaviour**: use the OpenSpec change workflow (`openspec-propose` in `.opencode/skills/`) to scaffold a change under `openspec/changes/`, then implement against it.
- **Review work**: check the delta specs of the relevant change and the affected living specs to confirm the implementation matches.

## Do's and Don'ts

- **Do** read the relevant specs and active changes before writing code.
- **Do** implement exactly what the specs describe — endpoints, permissions, status codes, and state transitions are all specified.
- **Do** route behavioural changes through `openspec/changes/`.
- **Don't** edit, delete, move, or rewrite the files under `openspec/` casually — they are versioned governance artifacts.
- **Don't** invent behaviour that contradicts the specs.
