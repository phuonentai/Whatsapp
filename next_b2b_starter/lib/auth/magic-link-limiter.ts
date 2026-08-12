/**
 * In-process sliding-window rate limiter for the magic-link send path.
 *
 * Single-instance assumption: state lives in module memory, so the limiter is
 * only accurate while a single Next.js server process handles the auth path.
 * That holds for the current deployment (one frontend instance). If the app is
 * ever scaled to multiple instances, replace this with a distributed limiter
 * (e.g. Redis) — see the change proposal's non-goals.
 *
 * Security: retains only normalized email addresses, IP addresses, and
 * timestamps in memory. NEVER stores tokens, credentials, session material, or
 * any other auth data (SSOT compliance). State is ephemeral: a process
 * restart clears everything.
 */

const WINDOW_MS = 60 * 60 * 1000; // 1 hour sliding window

const EMAIL_LIMIT_ENV = "MAGIC_LINK_RATE_LIMIT_PER_EMAIL_PER_HOUR";
const IP_LIMIT_ENV = "MAGIC_LINK_RATE_LIMIT_PER_IP_PER_HOUR";

const DEFAULT_EMAIL_LIMIT = 5;
const DEFAULT_IP_LIMIT = 20;

export interface MagicLinkRateLimitInput {
  email: string;
  ip: string;
}

export interface MagicLinkRateLimitResult {
  /** true when the send may proceed; false when throttled */
  allowed: boolean;
}

export interface MagicLinkLimiterOptions {
  /**
   * Optional explicit limits; when omitted (or when the corresponding env var
   * is unset/empty/invalid) the defaults apply. Env vars are re-read on every
   * check so tests and runtime config changes take effect without reloads.
   */
  perEmailPerHour?: number;
  perIpPerHour?: number;
}

/** key -> timestamps of sends inside the current window */
type WindowMap = Map<string, number[]>;

function readLimit(envName: string, fallback: number): number {
  const raw = process.env[envName];
  if (raw === undefined || raw === "") {
    return fallback;
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return fallback;
  }
  return parsed;
}

function pruneStale(windows: WindowMap, key: string, now: number): number[] {
  const cutoff = now - WINDOW_MS;
  const timestamps = (windows.get(key) ?? []).filter((ts) => ts > cutoff);
  if (timestamps.length === 0) {
    windows.delete(key);
  } else {
    windows.set(key, timestamps);
  }
  return timestamps;
}

export interface MagicLinkLimiter {
  check(input: MagicLinkRateLimitInput): MagicLinkRateLimitResult;
}

/**
 * Create a limiter instance. The module-level default instance backs
 * `checkMagicLinkRateLimit`; factories are used by tests to isolate state.
 */
export function createMagicLinkLimiter(
  options: MagicLinkLimiterOptions = {}
): MagicLinkLimiter {
  const windows: WindowMap = new Map();

  return {
    check({ email, ip }: MagicLinkRateLimitInput): MagicLinkRateLimitResult {
      const now = Date.now();
      const emailKey = `email:${email.trim().toLowerCase()}`;
      const ipKey = `ip:${ip.trim() || "127.0.0.1"}`;

      const emailLimit =
        options.perEmailPerHour ??
        readLimit(EMAIL_LIMIT_ENV, DEFAULT_EMAIL_LIMIT);
      const ipLimit =
        options.perIpPerHour ?? readLimit(IP_LIMIT_ENV, DEFAULT_IP_LIMIT);

      // Prune stale windows on every access (keeps the map bounded).
      const emailHits = pruneStale(windows, emailKey, now);
      const ipHits = pruneStale(windows, ipKey, now);

      if (emailHits.length >= emailLimit || ipHits.length >= ipLimit) {
        // Throttled requests are NOT recorded, so a client whose window has
        // elapsed is not pushed further behind by its blocked attempts.
        return { allowed: false };
      }

      emailHits.push(now);
      windows.set(emailKey, emailHits);
      ipHits.push(now);
      windows.set(ipKey, ipHits);

      return { allowed: true };
    },
  };
}

/** Default singleton used by the `sendMagicLink` server action. */
export const checkMagicLinkRateLimit = createMagicLinkLimiter().check;
