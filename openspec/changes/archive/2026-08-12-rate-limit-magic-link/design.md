# Rate-Limit Magic Link Sending — Design

## Context

`sendMagicLink` (`next_b2b_starter/lib/actions/auth/send-magic-link.ts`) is a `"use server"` action invoked from `app/auth/page.tsx`. It performs `organizations.members.search` then `magicLinks.email.loginOrSignup` per member org. Neither Stytch's per-project rate limits nor the Go limiter protect this path, and the action is reachable by any unauthenticated visitor.

## Decisions

### D1 — In-process sliding-window limiter, keyed by email and IP

- New module `next_b2b_starter/lib/auth/magic-link-limiter.ts` exporting `checkMagicLinkRateLimit({ email, ip })`.
- Implementation: fixed-size sliding window using a `Map<string, number[]>` of timestamps, pruned on access; no external dependency. Key = `email:<normalized>` and `ip:<ip>`.
- Defaults: 5 sends/email/hour, 20 sends/IP/hour; overridable via env.
- Server-action side: single instance (Next.js dev/prod server process) — sufficient for current deployment; document the single-instance assumption in the module.

### D2 — Neutral failure

- When throttled, return the same `ActionResult` success message the non-member path returns ("If an account exists with that email, a magic link has been sent.") but with a `throttled: true` flag so the UI can show a softer "Too many requests — try again later" hint without revealing whether the email has an account.
- Do NOT call Stytch when throttled (state-transition invariant: throttle check precedes any outbound Stytch call).

### D3 — IP derivation

- `ip = x-forwarded-for` first entry (when present) else `x-real-ip` else `request.ip`; only trust proxies in production behind a known ingress. Local dev: `127.0.0.1` fallback keeps the action usable.

## Stytch Boundary

- Outbound calls (`members.search`, `magicLinks.email.loginOrSignup`) remain unchanged and keep the existing circuit-breaker/fallback behavior defined in the auth adapter. This change only gates whether those calls happen.
- State-transition invariant: throttle hit ⇒ no Stytch call ⇒ `throttled: true` response. Throttle miss ⇒ existing behavior (search → send or neutral message).

## Security Invariants

- Counters hold emails/IPs only — never tokens, credentials, or session material (SSOT compliance).
- Limiter never reveals member existence: throttled responses are indistinguishable in wording from the no-member path.

## Testing Strategy

- Unit: limiter window math (burst within window blocked, window slide allows again, email/IP keys independent).
- Integration: `sendMagicLink` called 6× for one email → 6th returns `throttled: true` and Stytch client spy records 5 sends.
- Existing E2E (`auth-passwordless-e2e`) unaffected: per-email limit (5/h) is above test usage.
