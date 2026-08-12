import { afterEach, describe, expect, it, vi } from "vitest";

import {
  checkMfaRateLimit,
  createMfaLimiter,
} from "@/lib/auth/mfa-limiter";

const MEMBER_LIMIT_ENV = "MFA_RATE_LIMIT_PER_MEMBER_PER_HOUR";
const IP_LIMIT_ENV = "MFA_RATE_LIMIT_PER_IP_PER_HOUR";
const HOUR_MS = 60 * 60 * 1000;

/** Set an env var for the duration of the test, restoring the prior value. */
function withEnv(envName: string, value: string, fn: () => void) {
  const previous = process.env[envName];
  process.env[envName] = value;
  try {
    fn();
  } finally {
    if (previous === undefined) {
      delete process.env[envName];
    } else {
      process.env[envName] = previous;
    }
  }
}

afterEach(() => {
  vi.useRealTimers();
});

describe("checkMfaRateLimit (default singleton)", () => {
  it("is exported and allows an initial attempt", () => {
    expect(
      checkMfaRateLimit({ memberId: "member-1", ip: "10.0.0.1" }).allowed
    ).toBe(true);
  });
});

describe("createMfaLimiter (per-member + per-IP sliding window)", () => {
  it("allows the 10th attempt within the window and blocks the 11th", () => {
    const limiter = createMfaLimiter();

    for (let i = 1; i <= 10; i++) {
      const result = limiter.check({
        memberId: "member-burst",
        ip: "198.51.100.10",
      });
      expect(result.allowed).toBe(true);
    }

    expect(
      limiter.check({ memberId: "member-burst", ip: "198.51.100.10" }).allowed
    ).toBe(false);
  });

  it("re-allows a throttled member once the hourly window slides", () => {
    vi.useFakeTimers();
    const limiter = createMfaLimiter();

    for (let i = 0; i < 10; i++) {
      expect(
        limiter.check({ memberId: "member-slide", ip: "203.0.113.7" }).allowed
      ).toBe(true);
    }
    expect(
      limiter.check({ memberId: "member-slide", ip: "203.0.113.7" }).allowed
    ).toBe(false);

    vi.advanceTimersByTime(HOUR_MS);

    expect(
      limiter.check({ memberId: "member-slide", ip: "203.0.113.7" }).allowed
    ).toBe(true);
  });

  it("keeps member and IP keys independent (per-member throttle does not block other members on the same IP)", () => {
    const limiter = createMfaLimiter();

    for (let i = 0; i < 10; i++) {
      expect(
        limiter.check({ memberId: "member-a", ip: "198.51.100.20" }).allowed
      ).toBe(true);
    }
    // 'member-a' from this IP is now throttled…
    expect(
      limiter.check({ memberId: "member-a", ip: "198.51.100.20" }).allowed
    ).toBe(false);
    // …but a distinct member from the same IP is still under the IP limit.
    expect(
      limiter.check({ memberId: "member-b", ip: "198.51.100.20" }).allowed
    ).toBe(true);
  });

  it("keeps member and IP keys independent (per-IP throttle blocks a fresh member on that IP)", () => {
    const limiter = createMfaLimiter();

    for (let i = 0; i < 30; i++) {
      expect(
        limiter.check({ memberId: `member-ip-${i}`, ip: "203.0.113.50" })
          .allowed
      ).toBe(true);
    }
    // IP limit (30/h) exhausted: a brand-new member from this IP is throttled.
    expect(
      limiter.check({ memberId: "member-ip-30", ip: "203.0.113.50" }).allowed
    ).toBe(false);
  });

  it("enforces the per-IP window even without a member id (hard backstop)", () => {
    const limiter = createMfaLimiter();

    for (let i = 0; i < 30; i++) {
      expect(limiter.check({ ip: "198.51.100.99" }).allowed).toBe(true);
    }
    expect(limiter.check({ ip: "198.51.100.99" }).allowed).toBe(false);
  });

  it("honors the per-member env override", () => {
    withEnv(MEMBER_LIMIT_ENV, "2", () => {
      const limiter = createMfaLimiter();

      expect(
        limiter.check({ memberId: "member-env", ip: "10.9.8.7" }).allowed
      ).toBe(true);
      expect(
        limiter.check({ memberId: "member-env", ip: "10.9.8.7" }).allowed
      ).toBe(true);
      expect(
        limiter.check({ memberId: "member-env", ip: "10.9.8.7" }).allowed
      ).toBe(false);
    });
  });

  it("honors the per-IP env override", () => {
    withEnv(IP_LIMIT_ENV, "3", () => {
      const limiter = createMfaLimiter();

      for (const memberId of ["m-a", "m-b", "m-c"]) {
        expect(limiter.check({ memberId, ip: "9.9.9.9" }).allowed).toBe(true);
      }
      expect(
        limiter.check({ memberId: "m-d", ip: "9.9.9.9" }).allowed
      ).toBe(false);
    });
  });

  it("falls back to defaults when an env override is unparseable", () => {
    withEnv(MEMBER_LIMIT_ENV, "not-a-number", () => {
      const limiter = createMfaLimiter();

      for (let i = 0; i < 10; i++) {
        expect(
          limiter.check({ memberId: "member-bad", ip: "10.1.1.1" }).allowed
        ).toBe(true);
      }
      expect(
        limiter.check({ memberId: "member-bad", ip: "10.1.1.1" }).allowed
      ).toBe(false);
    });
  });
});
