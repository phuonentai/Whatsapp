import { Page, Locator } from "@playwright/test";

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
    await this.page.waitForLoadState("networkidle");
  }

  async gotoSettings(view?: string) {
    const url = view ? `/dashboard/settings?view=${view}` : "/dashboard/settings";
    await this.goto(url);
    await this.page.waitForSelector("text=Workspace settings");
  }

  async sidebarEntry(name: string): Promise<Locator> {
    return this.sidebar.locator(`[aria-label="${name}"]`);
  }

  async hasSidebarEntry(name: string): Promise<boolean> {
    const entry = await this.sidebarEntry(name);
    return (await entry.count()) > 0;
  }

  async openOverviewSection(title: string) {
    await this.goto("/dashboard/settings");
    await this.page.getByRole("button", { name: new RegExp(title) }).first().click();
    await this.page.waitForLoadState("networkidle");
  }

  async editWorkspaceName(name: string) {
    await this.gotoSettings("profile");
    await this.page.getByRole("button", { name: /editar/i }).click();
    await this.page.fill("#workspace-name", name);
    await this.page.getByRole("button", { name: /guardar/i }).click();
  }

  async getAuditLog(): Promise<Locator | null> {
    const list = this.auditLogList;
    return (await list.count()) > 0 ? list : null;
  }

  async filterAuditByType(type: string) {
    await this.page.selectOption('select[aria-label="Filter audit log by type"]', type);
    await this.page.waitForTimeout(300);
  }
}
