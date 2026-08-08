# ABANDONED

**Date:** 2026-08-08
**Status:** Abandoned — never implemented

## Why

This change planned a password-based `POST /auth/login` flow (Stytch `Passwords.Authenticate`), but the work was never implemented: tasks 1.1–3.1 were marked `[x]` yet no `LoginRequest.OrganizationID`, no `Passwords.Authenticate` call, and no `/auth/login` route exist in the codebase.

The plan also contradicts the living `signup-stytch-compliance` spec, which prohibits an `owner_password` field in the signup payload. Product decision is magic-link-only authentication.

## Decision

Rejected as part of `openspec/changes/abandon-password-auth/`. The delta spec in this directory MUST NOT be merged into `openspec/specs/`.

## Reopening

File a new change proposal that first reconciles with `signup-stytch-compliance` and demonstrates the password flow works end-to-end.
