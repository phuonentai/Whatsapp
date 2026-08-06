import { test, expect } from "@playwright/test";
import { ActivitiesPage } from "../page-objects/activities.page";

test.describe("Actividades", () => {
  let activitiesPage: ActivitiesPage;

  test.beforeEach(async ({ page }) => {
    activitiesPage = new ActivitiesPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("create note activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Note ${Date.now()}`;
    await activitiesPage.create({ type: "nota", subject, content: "Test content" });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });

  test("create call activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Call ${Date.now()}`;
    await activitiesPage.create({ type: "llamada", subject });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });

  test("create task activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Task ${Date.now()}`;
    await activitiesPage.create({ type: "tarea", subject });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });
});
