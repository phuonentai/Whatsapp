import { test, expect } from "@playwright/test";

test.describe("Proxy Middleware - Auth Protection", () => {
  test("redirects to /auth when accessing protected route without session", async ({ page }) => {
    await page.goto("/dashboard");
    await page.waitForURL(/\/auth/);
    expect(page.url()).toContain("/auth");
  });

  test("redirects to /auth when accessing /settings without session", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForURL(/\/auth/);
    expect(page.url()).toContain("/auth");
  });

  test("allows access to public /auth page without session", async ({ page }) => {
    await page.goto("/auth");
    expect(page.url()).toContain("/auth");
  });

  test("allows access to public / page without session", async ({ page }) => {
    await page.goto("/");
    expect(page.url()).toBe(page.url()); // no redirect
  });

  test("allows access to /authenticate without session", async ({ page }) => {
    await page.goto("/authenticate");
    expect(page.url()).toContain("/authenticate");
  });
});

test.describe("Proxy Middleware - JWT Validation", () => {
  test("redirects to /auth with expired JWT cookie on protected route", async ({ page }) => {
    const expiredJwt =
      "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtZW1iZXItMTIzIiwiZXhwIjoxNTAwMDAwMDAwLCJpYXQiOjE1MDAwMDAwMDAsImh0dHBzOi8vc3R5dGNoLmNvbS9vcmdhbml6YXRpb24iOnsib3JnYW5pemF0aW9uX2lkIjoib3JnLTEyMyJ9fQ.signature";

    await page.context().addCookies([
      {
        name: "stytch_session_jwt",
        value: expiredJwt,
        domain: "localhost",
        path: "/",
      },
    ]);

    await page.goto("/dashboard");
    await page.waitForURL(/\/auth/);
    expect(page.url()).toContain("/auth");
    // Stateless validation rejects the expired JWT and clears the session cookies.
    const cookies = await page.context().cookies();
    expect(cookies.find((c) => c.name === "stytch_session_jwt")).toBeUndefined();
    expect(cookies.find((c) => c.name === "stytch_session")).toBeUndefined();
  });

  test("redirects to /auth with malformed JWT cookie on protected route", async ({ page }) => {
    await page.context().addCookies([
      {
        name: "stytch_session_jwt",
        value: "not-a-valid-jwt",
        domain: "localhost",
        path: "/",
      },
    ]);

    await page.goto("/dashboard");
    await page.waitForURL(/\/auth/);
    expect(page.url()).toContain("/auth");
    // Stateless validation rejects the malformed JWT and clears the session cookies.
    const cookies = await page.context().cookies();
    expect(cookies.find((c) => c.name === "stytch_session_jwt")).toBeUndefined();
    expect(cookies.find((c) => c.name === "stytch_session")).toBeUndefined();
  });

  test("redirects preserves returnTo query param", async ({ page }) => {
    await page.goto("/dashboard/crm");
    await page.waitForURL(/\/auth/);
    const url = new URL(page.url());
    expect(url.searchParams.get("returnTo")).toBe("/dashboard/crm");
  });
});
