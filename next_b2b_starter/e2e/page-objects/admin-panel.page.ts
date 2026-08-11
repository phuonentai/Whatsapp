import { Page, Locator, expect } from "@playwright/test";

const SETTINGS_VIEW_HEADINGS: Record<string, string> = {
  profile: "Account & workspace",
  members: "Team access",
  subscription: "Subscription & billing",
  modules: "Modules",
  ai: "AI Copilot",
  compliance: "Compliance",
  audit: "Audit log",
  whatsapp: "Messaging",
  siigo: "Integración Siigo",
  "siigo-admin": "Onboarding Siigo",
};

export class AdminPanelPage {
  readonly page: Page;
  readonly sidebar: Locator;
  readonly auditLogList: Locator;

  constructor(page: Page) {
    this.page = page;
    this.sidebar = page.locator("nav");
    this.auditLogList = page.locator('[data-testid="audit-log-list"]');
  }

  async goto(path: string) {
    await this.page.goto(path);
    await this.page.waitForLoadState("load");
  }

  async gotoSettings(view?: string) {
    const url = view ? `/dashboard/settings?view=${view}` : "/dashboard/settings";
    await this.goto(url);
    const heading = view ? SETTINGS_VIEW_HEADINGS[view] ?? "Messaging" : "Workspace settings";
    await this.page.waitForSelector(`text=${heading}`);
  }

  async sidebarEntry(name: string): Promise<Locator> {
    return this.sidebar.locator(`[aria-label="${name}"]`);
  }

  async hasSidebarEntry(name: string): Promise<boolean> {
    const entry = await this.sidebarEntry(name);
    try {
      await entry.first().waitFor({ state: "attached", timeout: 3000 });
      return true;
    } catch {
      return false;
    }
  }

  async openOverviewSection(title: string) {
    await this.goto("/dashboard/settings");
    await this.page.getByRole("button", { name: new RegExp(title) }).first().click();
    await this.page.waitForLoadState("load");
  }

  async editWorkspaceName(name: string) {
    await this.gotoSettings("profile");
    await this.page.getByRole("button", { name: /editar/i }).click();
    await this.page.fill("#workspace-name", name);
    await this.page.getByRole("button", { name: /guardar/i }).click();
  }

  async getAuditLog(): Promise<Locator | null> {
    const list = this.auditLogList;
    try {
      await list.first().waitFor({ state: "attached", timeout: 3000 });
      return list;
    } catch {
      return null;
    }
  }

  async filterAuditByType(type: string) {
    await this.page.selectOption('select[aria-label="Filter audit log by type"]', type);
    await expect(this.auditLogList).toBeVisible();
  }
  // ---- Siigo onboarding admin view ----


  async openSiigoOnboarding() {
    await this.gotoSettings("siigo-admin");
    await this.page.getByText("Onboarding de facturación Siigo").first().waitFor({ state: "visible" });
  }

  async assertSiigoRow(orgId: string, status: string) {
    const row = this.page.locator("tr", { has: this.page.getByText(orgId, { exact: true }) });
    await expect(row).toBeVisible();
    await expect(row.getByText(new RegExp(status))).toBeVisible();
    return row;
  }

  async provisionCredentials(opts: { clientId: string; clientSecret: string; nit: string }) {
    await this.page.getByPlaceholder("client_id").fill(opts.clientId);
    await this.page.getByPlaceholder("client_secret").fill(opts.clientSecret);
    await this.page.getByPlaceholder("NIT").fill(opts.nit);
    await this.page.getByRole("button", { name: "Provisionar" }).click();
}

}
