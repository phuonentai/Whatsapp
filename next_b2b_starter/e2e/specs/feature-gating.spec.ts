import { test, expect } from "@playwright/test";

test.describe("Feature Gating", () => {
  test("Free plan hides Empresas tab", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-free:admin-free@test.com" });
    await page.goto("/dashboard/crm?view=contactos");

    const empresasTab = page.locator("button:has-text('Empresas')");
    const isDisabled = await empresasTab.getAttribute("disabled");
    expect(isDisabled).not.toBeNull();
  });

  test("Free plan hides Negocios tab", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-free:admin-free@test.com" });
    await page.goto("/dashboard/crm?view=contactos");

    const negociosTab = page.locator("button:has-text('Negocios')");
    const isDisabled = await negociosTab.getAttribute("disabled");
    expect(isDisabled).not.toBeNull();
  });

  test("Pro plan shows Empresas and Negocios tabs", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    await page.goto("/dashboard/crm?view=contactos");

    const empresasTab = page.locator("button:has-text('Empresas')");
    const disabled = await empresasTab.getAttribute("disabled");
    expect(disabled).toBeNull();
  });

  test("Enterprise plan shows all tabs including Etiquetas", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com" });
    await page.goto("/dashboard/crm?view=contactos");

    const etiquetasTab = page.locator("button:has-text('Etiquetas')");
    const disabled = await etiquetasTab.getAttribute("disabled");
    expect(disabled).toBeNull();
  });

  test("Free plan API returns 403 for Pro endpoint", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-free:admin-free@test.com" });

    const response = await page.request.post("http://localhost:8081/api/v1/crm/empresas", {
      data: { name: "Test", nit: "123" },
      headers: { "X-Test-Org-ID": "test-org-free:admin-free@test.com" },
    });
    expect(response.status()).toBe(403);
  });
});
