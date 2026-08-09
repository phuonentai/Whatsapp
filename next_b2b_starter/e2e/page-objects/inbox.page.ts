import { Page, Locator } from "@playwright/test";

export class InboxPage {
  readonly page: Page;
  readonly inboxHeading: Locator;
  readonly thread: Locator;

  constructor(page: Page) {
    this.page = page;
    this.inboxHeading = page.getByRole("heading", { name: /inbox/i });
    this.thread = page.locator('[data-testid="message-thread"]');
  }

  async goto() {
    await this.page.goto("/dashboard/inbox");
    await this.page.waitForSelector('text=Inbox');
  }

  /** Open a conversation whose row shows the given phone number or display name. */
  async openConversation(phoneOrName: string) {
    await this.page.locator(`button:has-text("${phoneOrName}")`).first().click();
    await this.page.waitForTimeout(500);
  }

  async getConversation(phoneOrName: string): Promise<Locator | null> {
    const row = this.page.locator(`button:has-text("${phoneOrName}")`).first();
    return (await row.count()) > 0 ? row : null;
  }

  async hasMessage(text: string): Promise<boolean> {
    const msg = this.page.locator(`[data-testid="message-thread"] :text("${text}")`).first();
    return (await msg.count()) > 0;
  }

  /** Switch the status filter tab (All / Active / Closed / Archived). */
  async setStatusFilter(label: "All" | "Active" | "Closed" | "Archived") {
    await this.page.getByRole("button", { name: label, exact: true }).click();
    await this.page.waitForTimeout(300);
  }
}
