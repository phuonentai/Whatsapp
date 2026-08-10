import { Page, Locator, expect } from "@playwright/test";

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
    await expect(this.thread).toBeVisible();
  }

  async getConversation(phoneOrName: string): Promise<Locator | null> {
    const row = this.page.locator(`button:has-text("${phoneOrName}")`).first();
    try {
      await row.first().waitFor({ state: "attached", timeout: 3000 });
      return row;
    } catch {
      return null;
    }
  }

  async hasMessage(text: string): Promise<boolean> {
    const msg = this.page.locator(`[data-testid="message-thread"] :text("${text}")`).first();
    try {
      await msg.waitFor({ state: "attached", timeout: 3000 });
      return true;
    } catch {
      return false;
    }
  }

  /** Fill the reply input, submit, and assert the message lands in the thread. */
  async sendReply(text: string) {
    await this.page.getByPlaceholder("Type a message...").fill(text);
    await this.page.getByPlaceholder("Type a message...").press("Enter");
    await expect(this.page.locator(`[data-testid="message-thread"] :text("${text}")`).first()).toBeVisible();
  }

  /** Submit an empty/whitespace reply and assert nothing is sent. */
  async sendEmptyReply() {
    const input = this.page.getByPlaceholder("Type a message...");
    const before = await this.page.locator('[data-testid="message-thread"]').innerText();
    await input.fill("   ");
    await input.press("Enter");
    await expect(input).toHaveValue("   ");
    const after = await this.page.locator('[data-testid="message-thread"]').innerText();
    return { before, after };
  }

  /** Select a status filter tab (All/Active/Closed/Archived). */
  async setStatusFilter(label: "All" | "Active" | "Closed" | "Archived") {
    await this.page.getByRole("button", { name: label, exact: true }).click();
  }

  /** Assert a status badge is visible on the conversation row. */
  async assertConversationStatus(phoneOrName: string, status: string) {
    const row = this.page.locator(`button:has-text("${phoneOrName}")`).first();
    await expect(row.locator(`text="${status}"`).first()).toBeVisible();
  }

  /** Click a quick-reply pill and assert its message fills the reply input. */
  async selectQuickReply(pillTitle: string, expectedMessage: string) {
    await this.page.getByRole("button", { name: pillTitle }).click();
    await expect(this.page.getByPlaceholder("Type a message...")).toHaveValue(expectedMessage);
  }

  async quickRepliesVisible(): Promise<boolean> {
    const panel = this.page.locator("div.border-t.border-gray-100.bg-gray-50\\/60");
    try {
      await panel.first().waitFor({ state: "visible", timeout: 2000 });
      return true;
    } catch {
      return false;
    }
  }

  /** Approve the pending suggestion in the suggestions panel. */
  async approveSuggestion() {
    await this.page.getByRole("button", { name: "Aprobar y enviar" }).click();
    await expect(this.page.getByRole("button", { name: "Aprobar y enviar" })).not.toBeVisible();
  }

  /** Reject the pending suggestion in the suggestions panel. */
  async rejectSuggestion() {
    await this.page.getByRole("button", { name: "Rechazar" }).click();
    await expect(this.page.getByRole("button", { name: "Rechazar" })).not.toBeVisible();
  }
}
