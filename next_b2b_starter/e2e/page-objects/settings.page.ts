import { Page, Locator, expect } from "@playwright/test";

type SectionTitle =
  | "Account & workspace"
  | "Team access"
  | "Subscription & billing"
  | "Modules"
  | "AI Copilot"
  | "Compliance (Ley 1581)"
  | "Messaging"
  | "Audit log"
  | "Integración Siigo"
  | "Onboarding Siigo";

export class SettingsPage {
  readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async goto() {
    await this.page.goto("/dashboard/settings");
    await this.page.getByRole("heading", { name: "Workspace settings" }).waitFor({ state: "visible" });
  }

  async openSection(title: SectionTitle) {
    await this.page.getByRole("button", { name: new RegExp(title) }).click();
  }

  async backToOverview() {
    await this.page.getByRole("button", { name: /back|volver/i }).first().click();
  }

  /** Open the invite-member dialog from the Team access view. */
  async openInviteDialog() {
    await this.openSection("Team access");
    await this.page.getByRole("button", { name: "Add member" }).click();
    const dialog = this.page.locator("#invite-member-dialog");
    await expect(dialog).toBeVisible();
    return dialog;
  }

  /** Fill and submit the invite form. */
  async inviteMember(email: string, role: string, name?: string) {
    const dialog = await this.openInviteDialog();
    await dialog.getByLabel(/name/i).fill(name ?? `Invitee ${Date.now()}`);
    await dialog.getByLabel(/email/i).fill(email);
    await dialog.getByText("Select a role").click();
    await this.page.getByRole("option", { name: new RegExp(role, "i") }).click();
    await dialog.getByRole("button", { name: "Send Invitation" }).click();
  }

  /** Assert a member row with the given email exposes a role control. */
  async assertMemberRole(email: string) {
    const row = this.page.locator(`text="${email}"`).first();
    await expect(row).toBeVisible();
  }

  async assertModuleNameVisible(name: string) {
    await expect(this.page.getByText(name).first()).toBeVisible();
  }

  async assertPlaybookVisible(title: string) {
    await expect(this.page.getByText(title).first()).toBeVisible();
  }

  async assertPlanVisible(plan: string) {
    await expect(this.page.getByText(new RegExp(plan, "i")).first()).toBeVisible();
  }

  async assertWhatsappConfigVisible() {
    await expect(this.page.getByLabel(/active/i).first()).toBeVisible();
  }

  /** Edit the workspace name in the profile view and save. */
  async editWorkspaceName(name: string) {
    await this.openSection("Account & workspace");
    await this.page.getByRole("button", { name: /edit/i }).first().click();
    const input = this.page.locator("#workspace-name");
    await input.fill(name);
    await this.page.getByRole("button", { name: "Guardar" }).click();
  }
  // ---- Siigo onboarding section ----


  async openSiigoSection() {
    await this.goto();
    await this.openSection("Integración Siigo");
  }

  async connectSiigo(opts: { clientId: string; clientSecret: string; nit: string }) {
    await this.page.getByPlaceholder("client_id").fill(opts.clientId);
    await this.page.getByPlaceholder("client_secret").fill(opts.clientSecret);
    await this.page.getByPlaceholder("900.123.456-7").fill(opts.nit);
    await this.page.getByRole("button", { name: "Conectar Siigo" }).click();
  }

  async requestAssistedSetup() {
    // Navigate explicitly so the view re-opens after an auth-header swap
    // (a bare reload can land on the overview while query params settle).
    await this.page.goto("/dashboard/settings?view=siigo");
    await this.page.getByRole("button", { name: /No tengo Siigo/ }).click();
  }

  async confirmNumeration() {
    await this.page.getByRole("button", { name: "Confirmar numeración" }).click();
  }

  async previewImport() {
    await this.page.getByRole("button", { name: "Ver vista previa" }).click();
  }

  async confirmImport() {
    await this.page.getByRole("button", { name: "Confirmar importación" }).click();
  }

  async createTestInvoice() {
    await this.page.getByRole("button", { name: "Crear factura de prueba" }).click();
  }

  async activateInvoicing() {
    await this.page.getByRole("button", { name: "Activar facturación" }).click();
  }

  async pauseInvoicing() {
    await this.page.getByRole("button", { name: "Pausar" }).click();
  }

  async resumeInvoicing() {
    await this.page.getByRole("button", { name: "Reanudar" }).click();
  }

  async assertSiigoBanner(text: string | RegExp) {
    await expect(this.page.getByText(new RegExp(text)).first()).toBeVisible();
}

}
