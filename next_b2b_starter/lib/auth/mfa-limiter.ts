/**
 * In-process sliding-window rate limiter for the MFA challenge path
 * (TOTP code + recovery-code attempts).
 *
 * Single-instance assumption: state lives in module memory, so the limiter is
 * only accurate while a single Next.js server process handles the auth path.
 * That holds for the current deployment (one frontend instance). If the app is
 * ever scaled to multiple instances, replace this with a distributed limiter
 * (e.g. Redis) — identical posture to the magic-link limiter.
 *
 * Security: retains only normalized member IDs, IP addresses, and timestamps
 * in memory. NEVER stores tokens, MFA codes, recovery codes, credentials, or
 * any other auth material (SSOT compliance). State is ephemeral: a process
 * restart clears everything.
 *
 * The per-member window is keyed on the member id supplied by the caller,
 * which during a login continuation comes from the client (context only —
 * the intermediate session token remains the authority). An attacker who can
 * rotate member ids can evade the per-member window; the per-IP window is the
 * hard backstop, so both windows are always enforced together.
 */

const WINDOW_MS = 60 * 60 * 1000; // 1 hour sliding window

const MEMBER_LIMIT_ENV = "MFA_RATE_LIMIT_PER_MEMBER_PER_HOUR";
const IP_LIMIT_ENV = "MFA_RATE_LIMIT_PER_IP_PER_HOUR";

const DEFAULT_MEMBER_LIMIT = 10;
const DEFAULT_IP_LIMIT = 30;

export interface MfaRateLimitInput {
  /** Member id (best-effort key — context only, never an authorization input). */
  memberId?: string;
  ip: string;
}

export interface MfaRateLimitResult {
  /** true when the attempt may proceed; false when throttled */
  allowed: boolean;
}

export interface MfaLimiterOptions {
  /**
   * Optional explicit limits; when omitted (or when the corresponding env var
   * is unset/empty/invalid) the defaults apply. Env vars are re-read on every
   * check so tests and runtime config changes take effect without reloads.
   */
  perMemberPerHour?: number;
  perIpPerHour?: number;
}

/** key -> timestamps of attempts inside the current window */
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

export interface MfaLimiter {
  check(input: MfaRateLimitInput): MfaRateLimitResult;
}

/**
 * Create a limiter instance. The module-level default instance backs
 * `checkMfaRateLimit`; factories are used by tests to isolate state.
 */
export function createMfaLimiter(options: MfaLimiterOptions = {}): MfaLimiter {
  const windows: WindowMap = new Map();

  return {
    check({ memberId, ip }: MfaRateLimitInput): MfaRateLimitResult {
      const now = Date.now();
      const ipKey = `ip:${ip.trim() || "127.0.0.1"}`;

      const memberLimit =
        options.perMemberPerHour ??
        readLimit(MEMBER_LIMIT_ENV, DEFAULT_MEMBER_LIMIT);
      const ipLimit =
        options.perIpPerHour ?? readLimit(IP_LIMIT_ENV, DEFAULT_IP_LIMIT);

      // Prune stale windows on every access (keeps the map bounded).
      const ipHits = pruneStale(windows, ipKey, now);

      if (ipHits.length >= ipLimit) {
        // Throttled requests are NOT recorded, so a client whose window has
        // elapsed is not pushed further behind by its blocked attempts.
        return { allowed: false };
      }

      // The member window is only enforced when we have a member id to key on
      // (login continuation supplies it; defense-in-depth on top of the IP
      // window). Without an id the IP window alone still bounds the attack.
      if (memberId) {
        const memberKey = `member:${memberId.trim()}`;
        const memberHits = pruneStale(windows, memberKey, now);
        if (memberHits.length >= memberLimit) {
          return { allowed: false };
        }
        memberHits.push(now);
        windows.set(memberKey, memberHits);
      }

      ipHits.push(now);
      windows.set(ipKey, ipHits);

      return { allowed: true };
    },
  };
}

/** Default singleton used by the MFA server actions. */
export const checkMfaRateLimit = createMfaLimiter().check;
