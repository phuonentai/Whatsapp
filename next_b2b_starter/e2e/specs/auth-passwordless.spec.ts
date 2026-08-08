import { test, expect } from "@playwright/test";
import { SignupPage } from "../page-objects/signup.page";

test.describe("Passwordless Authentication", () => {
  test("signup page exposes no password input", async ({ page }) => {
    await page.goto("/signup");

    await expect(page.getByPlaceholder("John Doe")).toBeVisible();
    await expect(page.getByPlaceholder("you@company.com")).toBeVisible();
    await expect(page.locator('input[type="password"]')).toHaveCount(0);
  });

  test("login page is email-only with no password input", async ({ page }) => {
    await page.goto("/auth");

    await expect(page.getByPlaceholder("you@company.com")).toBeVisible();
    await expect(page.locator('input[type="password"]')).toHaveCount(0);
  });

  test("signup payload excludes owner_password", async ({ page }) => {
    let capturedBody: Record<string, unknown> | null = null;

    await page.route("**/api/auth/signup", async (route) => {
      const body = route.request().postDataJSON();
      capturedBody = body ?? {};
      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify({
          success: true,
          data: { org_id: "org-test", owner_email: "owner@test.com" },
        }),
      });
    });

    const signupPage = new SignupPage(page);
    await signupPage.goto();
    await signupPage.fillAccountStep("Owner Tester", "owner@test.com");
    await signupPage.goToOrganizationStep();
    await signupPage.fillOrganizationStep("Test Org", "Technology");
    await signupPage.submit();

    expect(capturedBody).not.toBeNull();
    expect(capturedBody).not.toHaveProperty("owner_password");
  });

  test("password login endpoint does not exist", async ({ page }) => {
    const response = await page.request.post(
      "http://localhost:3001/api/auth/login",
      {
        data: { email: "owner@test.com", password: "secret" },
      }
    );
    expect([404, 405]).toContain(response.status());
  });

  test("authenticate page renders without password input", async ({ page }) => {
    await page.goto("/authenticate");

    await expect(page.locator('input[type="password"]')).toHaveCount(0);
  });
});
