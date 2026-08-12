import { afterEach, describe, expect, it, vi } from "vitest";

import {
  checkMagicLinkRateLimit,
  createMagicLinkLimiter,
} from "@/lib/auth/magic-link-limiter";

const EMAIL_LIMIT_ENV = "MAGIC_LINK_RATE_LIMIT_PER_EMAIL_PER_HOUR";
const IP_LIMIT_ENV = "MAGIC_LINK_RATE_LIMIT_PER_IP_PER_HOUR";
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

describe("checkMagicLinkRateLimit (default singleton)", () => {
  it("is exported and allows an initial send", () => {
    expect(
      checkMagicLinkRateLimit({ email: "first@example.com", ip: "10.0.0.1" })
        .allowed
    ).toBe(true);
  });
});

describe("createMagicLinkLimiter (per-email + per-IP sliding window)", () => {
  it("allows the 5th send within the window and blocks the 6th", () => {
    const limiter = createMagicLinkLimiter();

    for (let i = 1; i <= 5; i++) {
      const result = limiter.check({
        email: "burst@example.com",
        ip: "198.51.100.10",
      });
      expect(result.allowed).toBe(true);
    }

    expect(
      limiter.check({ email: "burst@example.com", ip: "198.51.100.10" })
        .allowed
    ).toBe(false);
  });

  it("re-allows a throttled email once the hourly window slides", () => {
    vi.useFakeTimers();
    const limiter = createMagicLinkLimiter();

    for (let i = 0; i < 5; i++) {
      expect(
        limiter.check({ email: "slide@example.com", ip: "203.0.113.7" })
          .allowed
      ).toBe(true);
    }
    expect(
      limiter.check({ email: "slide@example.com", ip: "203.0.113.7" }).allowed
    ).toBe(false);

    vi.advanceTimersByTime(HOUR_MS);

    expect(
      limiter.check({ email: "slide@example.com", ip: "203.0.113.7" }).allowed
    ).toBe(true);
  });

  it("keeps email and IP keys independent (per-email throttle does not block other emails on the same IP)", () => {
    const limiter = createMagicLinkLimiter();

    for (let i = 0; i < 5; i++) {
      expect(
        limiter.check({ email: "a@example.com", ip: "198.51.100.20" }).allowed
      ).toBe(true);
    }
    // 'a@example.com' from this IP is now throttled…
    expect(
      limiter.check({ email: "a@example.com", ip: "198.51.100.20" }).allowed
    ).toBe(false);
    // …but a distinct email from the same IP is still under the IP limit.
    expect(
      limiter.check({ email: "b@example.com", ip: "198.51.100.20" }).allowed
    ).toBe(true);
  });

  it("keeps email and IP keys independent (per-IP throttle blocks a fresh email on that IP)", () => {
    const limiter = createMagicLinkLimiter();

    for (let i = 0; i < 20; i++) {
      expect(
        limiter.check({ email: `ip${i}@example.com`, ip: "203.0.113.50" })
          .allowed
      ).toBe(true);
    }
    // IP limit (20/h) exhausted: a brand-new email from this IP is throttled.
    expect(
      limiter.check({ email: "ip20@example.com", ip: "203.0.113.50" }).allowed
    ).toBe(false);
  });

  it("normalizes the email before keying (case/whitespace insensitive)", () => {
    const limiter = createMagicLinkLimiter();

    for (let i = 0; i < 5; i++) {
      const email = i % 2 === 0 ? "  User@Example.COM " : "user@example.com";
      expect(limiter.check({ email, ip: "192.0.2.5" }).allowed).toBe(true);
    }
    expect(
      limiter.check({ email: "USER@EXAMPLE.COM", ip: "192.0.2.5" }).allowed
    ).toBe(false);
  });

  it("honors the per-email env override", () => {
    withEnv(EMAIL_LIMIT_ENV, "2", () => {
      const limiter = createMagicLinkLimiter();

      expect(
        limiter.check({ email: "env@example.com", ip: "10.9.8.7" }).allowed
      ).toBe(true);
      expect(
        limiter.check({ email: "env@example.com", ip: "10.9.8.7" }).allowed
      ).toBe(true);
      expect(
        limiter.check({ email: "env@example.com", ip: "10.9.8.7" }).allowed
      ).toBe(false);
    });
  });

  it("honors the per-IP env override", () => {
    withEnv(IP_LIMIT_ENV, "3", () => {
      const limiter = createMagicLinkLimiter();

      for (const email of ["a@example.com", "b@example.com", "c@example.com"]) {
        expect(limiter.check({ email, ip: "9.9.9.9" }).allowed).toBe(true);
      }
      expect(
        limiter.check({ email: "d@example.com", ip: "9.9.9.9" }).allowed
      ).toBe(false);
    });
  });

  it("falls back to defaults when an env override is unparseable", () => {
    withEnv(EMAIL_LIMIT_ENV, "not-a-number", () => {
      const limiter = createMagicLinkLimiter();

      for (let i = 0; i < 5; i++) {
        expect(
          limiter.check({ email: "bad@example.com", ip: "10.1.1.1" }).allowed
        ).toBe(true);
      }
      expect(
        limiter.check({ email: "bad@example.com", ip: "10.1.1.1" }).allowed
      ).toBe(false);
    });
  });
});
