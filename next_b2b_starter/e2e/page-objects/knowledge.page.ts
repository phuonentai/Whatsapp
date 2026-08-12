import { Page, Locator, expect } from "@playwright/test";
import path from "path";

const FIXTURES = path.join(__dirname, "..", "fixtures");

export class KnowledgePage {
  readonly page: Page;
  readonly chatInput: Locator;

  constructor(page: Page) {
    this.page = page;
    this.chatInput = page.getByPlaceholder("Nueva conversación");
  }

  async goto() {
    await this.page.goto("/dashboard/knowledge");
    await this.page.getByRole("button", { name: "Documentos" }).waitFor({ state: "visible" });
  }

  /** Switch to the Documentos mode where upload + document list live. */
  async openSources() {
    await this.page.getByRole("button", { name: "Documentos" }).click();
  }

  /** Upload a PDF through the upload dialog (admin-only surface). */
  async uploadPdf(fileName: string, title?: string) {
    await this.page.getByRole("button", { name: "Agregar documento" }).first().click();
    const dialog = this.page.getByRole("dialog", { name: "Agregar documento" });
    await expect(dialog).toBeVisible();
    await dialog.locator('input[type="file"]').setInputFiles(path.join(FIXTURES, fileName));
    const titleInput = this.page.getByPlaceholder("My Document");
    await expect(titleInput).toBeVisible();
    if (title !== undefined) {
      await titleInput.fill(title);
    }
    await this.page.getByRole("button", { name: "Upload PDF" }).click();
  }

  /** Drop a rejected (non-PDF) file in the dialog and assert the error. */
  async dropRejectedFile(fileName: string) {
    await this.page.getByRole("button", { name: "Agregar documento" }).first().click();
    const dialog = this.page.getByRole("dialog", { name: "Agregar documento" });
    await expect(dialog).toBeVisible();
    await dialog.locator('input[type="file"]').setInputFiles(path.join(FIXTURES, fileName));
  }

  async assertUploadError(text: string) {
    await expect(this.page.locator(`text="${text}"`).first()).toBeVisible();
  }

  async assertDocumentInList(title: string) {
    await expect(this.page.locator(`text="${title}"`).first()).toBeVisible();
  }

  /** Send a chat message (assumes chat endpoint is route-mocked). */
  async sendChat(text: string) {
    await this.chatInput.fill(text);
    await this.chatInput.press("Enter");
  }

  async assertChatMessage(text: string) {
    await expect(this.page.locator(`text="${text}"`).first()).toBeVisible();
  }
}
