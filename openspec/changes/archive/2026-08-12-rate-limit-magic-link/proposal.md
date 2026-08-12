# Rate-Limit Magic Link Sending

## Why

The sign-in path sends magic links from the Next.js server action `sendMagicLink` (`next_b2b_starter/lib/actions/auth/send-magic-link.ts`), which calls Stytch `magicLinks.email.loginOrSignup` directly. The action is unthrottled:

- The Go platform rate limiter (`internal/platform/server/middleware/ratelimit.go`, in-process token bucket on the Gin engine) does not cover Next.js server actions.
- `STYTCH_CONFIGURATION.md` itself lists "Add rate limiting to `/api/auth/magic-link` endpoint" as an outstanding security recommendation.
- Attack: an automated client can repeatedly POST the email of any known member (e.g., from CRM data or breached lists) and trigger unbounded magic-link emails — an email-bombing / reputation-abuse vector against members and a Stytch cost driver.

## What Changes

- Add per-email + per-IP rate limiting to the `sendMagicLink` server action (and the `/auth` form path that calls it).
- Use an in-process sliding-window/token-bucket limiter keyed by normalized email and by IP (derived from `x-forwarded-for`/`x-real-ip` with a local-dev fallback), with env-tunable limits and a conservative default (e.g., 5 sends per email per hour, 20 per IP per hour).
- Exceeding the limit returns the same neutral "If an account exists…" message (no enumeration leak) with an error variant surfaced to the UI; the request is NOT forwarded to Stytch.
- Backward compatible: no API surface change; the existing ActionResult contract is preserved.

## Capabilities

### New Capabilities
- `sign-in-rate-limiting`: throttling of the magic-link send path to prevent email bombing.

### Modified Capabilities
- None.

## Impact

- **Frontend (`next_b2b_starter/`):** new limiter util under `lib/auth/`; `sendMagicLink.ts` uses it; `app/auth/page.tsx` renders a throttled-state message; new unit tests for the limiter.
- **Config:** new env vars `MAGIC_LINK_RATE_LIMIT_PER_EMAIL_PER_HOUR` (default 5), `MAGIC_LINK_RATE_LIMIT_PER_IP_PER_HOUR` (default 20).
- **Dependencies:** none new (in-process; no Redis requirement at this scale).
- **Stytch:** no tenant policy changes; reduces outbound `magicLinks.email.loginOrSignup` calls.

## Rollback

- **Git:** revert the change directory and the touched files; the limiter is additive and non-breaking.
- **Stytch tenant policy state:** no Stytch-side state is modified by this change; nothing to roll back. (The limiter only suppresses outbound calls.)

## Non-Goals

- NOT storing credentials, magic-link tokens, or session tokens locally (per SSOT constitution; only email/IP counters in memory).
- NOT replacing Stytch's own platform rate limits or adding distributed (Redis) limiting — in-process is sufficient at current scale; a distributed limiter is a future change if multi-instance deployment demands it.
- NOT rate limiting other auth endpoints (login consumption, signup) in this change.
