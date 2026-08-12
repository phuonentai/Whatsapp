import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  useSlidingSession,
  SLIDING_SESSION_INTERVAL_MINUTES,
} from "./use-sliding-session";

const REFRESH_ENDPOINT = "/api/auth/session/refresh";
const SESSION_COOKIE = "stytch_session";
const SESSION_JWT_COOKIE = "stytch_session_jwt";

function okResponse() {
  return { ok: true, status: 200, json: async () => ({ sessionJwt: "jwt" }) } as Response;
}

function rejectedResponse() {
  return { ok: false, status: 401, json: async () => ({ error: "session_invalid" }) } as Response;
}

describe("useSlidingSession", () => {
  const fetchMock = vi.fn();

  const originalLocation = window.location;
  const originalVisibility = document.visibilityState;

  function setVisibility(state: DocumentVisibilityState) {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: state,
    });
  }

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);

    // jsdom navigation is not implemented; stub location so we can observe the
    // redirect target and reset it per test.
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, pathname: "/dashboard/crm", search: "?tab=1", href: "" },
    });

    setVisibility("visible");
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: originalVisibility,
    });
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
  });

  it("refreshes immediately on mount and then on the 10-minute interval", async () => {
    fetchMock.mockResolvedValue(okResponse());
    renderHook(() => useSlidingSession());

    // Immediate renewal on mount.
    await act(async () => {});
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      REFRESH_ENDPOINT,
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ session_duration_minutes: 480 }),
      })
    );

    // No renewal before the interval elapses.
    await act(async () => {
      vi.advanceTimersByTime(SLIDING_SESSION_INTERVAL_MINUTES * 60 * 1000 - 1000);
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // Renewal on the interval tick.
    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("honors a custom duration in the refresh request", async () => {
    fetchMock.mockResolvedValue(okResponse());
    renderHook(() => useSlidingSession({ durationMinutes: 240 }));

    await act(async () => {});
    expect(fetchMock).toHaveBeenCalledWith(
      REFRESH_ENDPOINT,
      expect.objectContaining({
        body: JSON.stringify({ session_duration_minutes: 240 }),
      })
    );
  });

  it("does not renew while the document is hidden, and resumes on visibility change", async () => {
    fetchMock.mockResolvedValue(okResponse());
    renderHook(() => useSlidingSession());

    await act(async () => {});
    expect(fetchMock).toHaveBeenCalledTimes(1);

    setVisibility("hidden");
    await act(async () => {
      vi.advanceTimersByTime(SLIDING_SESSION_INTERVAL_MINUTES * 60 * 1000 * 3);
    });
    // Interval ticks fired but were gated by visibility.
    expect(fetchMock).toHaveBeenCalledTimes(1);

    setVisibility("visible");
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("clears cookies and redirects to /auth with returnTo on refresh rejection", async () => {
    // Seed an existing session so we can observe the cleanup.
    document.cookie = `${SESSION_COOKIE}=tok123; path=/`;
    document.cookie = `${SESSION_JWT_COOKIE}=jwt456; path=/`;
    expect(document.cookie).toContain(`${SESSION_COOKIE}=tok123`);

    fetchMock.mockResolvedValue(rejectedResponse());
    renderHook(() => useSlidingSession());

    await act(async () => {});

    const cookies = document.cookie;
    expect(cookies).not.toContain(`${SESSION_COOKIE}=tok123`);
    expect(cookies).not.toContain(`${SESSION_JWT_COOKIE}=jwt456`);
    expect(window.location.href).toBe(
      `/auth?returnTo=${encodeURIComponent("/dashboard/crm?tab=1")}`
    );
  });

  it("ignores transient network failures and retries on the next interval", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    fetchMock.mockResolvedValue(okResponse());

    renderHook(() => useSlidingSession());
    await act(async () => {});

    // First attempt unreachable: no redirect, no cookie clearing.
    expect(window.location.href).toBe("");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      vi.advanceTimersByTime(SLIDING_SESSION_INTERVAL_MINUTES * 60 * 1000);
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("does nothing when disabled (no session)", async () => {
    fetchMock.mockResolvedValue(okResponse());
    renderHook(() => useSlidingSession({ enabled: false }));

    await act(async () => {});
    await act(async () => {
      vi.advanceTimersByTime(SLIDING_SESSION_INTERVAL_MINUTES * 60 * 1000 * 2);
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("stops renewing after unmount", async () => {
    fetchMock.mockResolvedValue(okResponse());
    const { unmount } = renderHook(() => useSlidingSession());

    await act(async () => {});
    expect(fetchMock).toHaveBeenCalledTimes(1);

    unmount();
    await act(async () => {
      vi.advanceTimersByTime(SLIDING_SESSION_INTERVAL_MINUTES * 60 * 1000 * 2);
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
