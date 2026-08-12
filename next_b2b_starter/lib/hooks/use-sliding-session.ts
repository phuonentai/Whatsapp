"use client";

import { useEffect } from "react";

import { SESSION_COOKIE_NAME, SESSION_JWT_COOKIE_NAME } from "@/lib/auth/constants";

/**
 * Default sliding-renewal cadence: 10 minutes (Stytch docs pattern for
 * `sessions.authenticate` while the app is open).
 */
export const SLIDING_SESSION_INTERVAL_MINUTES = 10;

/** Server route that exchanges the session token for a fresh JWT and extends
 * the underlying Stytch session lifetime. */
const REFRESH_ENDPOINT = "/api/auth/session/refresh";

export interface UseSlidingSessionOptions {
  /**
   * Session lifetime to slide to, in minutes. Defaults to the same 480
   * (8 hours) used at login; the refresh route falls back to the
   * `NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES` env value when absent.
   */
  durationMinutes?: number;
  /**
   * Only run the renewal loop while a session exists (mount the hook on
   * authenticated pages only; pass `false` on public pages).
   */
  enabled?: boolean;
  /** Renewal cadence in minutes (default 10). */
  intervalMinutes?: number;
}

type RefreshOutcome = "ok" | "rejected" | "unreachable";

/**
 * Renews the session with `session_duration_minutes` so the underlying Stytch
 * session slides instead of hard-expiring. Runs on mount and then every
 * `intervalMinutes` while the document is visible; renewal pauses while the
 * tab is hidden.
 *
 * On a rejected renewal (session revoked or expired, HTTP error) the session
 * cookies are cleared and the browser is redirected to `/auth?returnTo=<path>`.
 * Transient network failures are ignored so a momentary outage does not log
 * the user out; the next interval retries.
 */
export function useSlidingSession(options: UseSlidingSessionOptions = {}): void {
  const {
    durationMinutes = 480,
    enabled = true,
    intervalMinutes = SLIDING_SESSION_INTERVAL_MINUTES,
  } = options;

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let cancelled = false;

    const renew = async () => {
      if (cancelled || document.visibilityState !== "visible") {
        return;
      }

      const outcome = await refreshSession(durationMinutes);
      if (cancelled) {
        return;
      }

      if (outcome === "rejected") {
        clearSessionCookies();
        redirectToLogin();
      }
    };

    // Renew immediately on mount, then on the interval and whenever the tab
    // becomes visible again.
    void renew();
    const timer = setInterval(renew, intervalMinutes * 60 * 1000);

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void renew();
      }
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      cancelled = true;
      clearInterval(timer);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [enabled, intervalMinutes, durationMinutes]);
}

async function refreshSession(durationMinutes: number): Promise<RefreshOutcome> {
  try {
    const response = await fetch(REFRESH_ENDPOINT, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ session_duration_minutes: durationMinutes }),
    });
    return response.ok ? "ok" : "rejected";
  } catch {
    return "unreachable";
  }
}

function clearSessionCookies(): void {
  const expiry = "Thu, 01 Jan 1970 00:00:00 GMT";
  document.cookie = `${SESSION_COOKIE_NAME}=; path=/; expires=${expiry}; max-age=0`;
  document.cookie = `${SESSION_JWT_COOKIE_NAME}=; path=/; expires=${expiry}; max-age=0`;
}

function redirectToLogin(): void {
  const currentPath = window.location.pathname + window.location.search;
  const returnTo = encodeURIComponent(currentPath);
  window.location.href = `/auth?returnTo=${returnTo}`;
}
