# ABANDONED

**Date:** 2026-08-08
**Status:** Abandoned — planned only, never implemented

## Why

This change proposed replacing the magic-link signup flow with Stytch `Passwords.Create` (`add-password-auth`, 0/23 tasks completed). It directly contradicts the living `signup-stytch-compliance` spec, which requires the owner member to be created with `SendInvite: true` and prohibits an `owner_password` field in the signup payload. No password code was ever written.

## Decision

Rejected as part of `openspec/changes/abandon-password-auth/`. The `password-auth` capability spec in this directory MUST NOT be merged into `openspec/specs/`.

## Reopening

File a new change proposal that first reconciles with `signup-stytch-compliance` (requires a spec change) and demonstrates the password flow works end-to-end.
