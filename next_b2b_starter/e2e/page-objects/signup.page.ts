import { Page, Locator } from "@playwright/test";

export class SignupPage {
  readonly page: Page;
  readonly fullNameInput: Locator;
  readonly emailInput: Locator;
  readonly continueButton: Locator;
  readonly organizationNameInput: Locator;
  readonly industrySelect: Locator;
  readonly createAccountButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.fullNameInput = page.getByPlaceholder("Juan Pérez");
    this.emailInput = page.getByPlaceholder("tu@empresa.com");
    this.continueButton = page.getByRole("button", { name: /continuar/i });
    this.organizationNameInput = page.getByPlaceholder("Acme S.A.S.");
    this.industrySelect = page.locator("select");
    this.createAccountButton = page.getByRole("button", { name: /crear cuenta/i });
  }

  async goto() {
    await this.page.goto("/signup");
  }

  async fillAccountStep(fullName: string, email: string) {
    await this.fullNameInput.fill(fullName);
    await this.emailInput.fill(email);
  }

  async goToOrganizationStep() {
    await this.continueButton.click();
  }

  async fillOrganizationStep(organizationName: string, industry: string) {
    await this.organizationNameInput.fill(organizationName);
    await this.industrySelect.selectOption(industry);
  }

  async submit() {
    // Organization step → business context step (goal required to enable submit).
    await this.continueButton.click();
    await this.page
      .getByPlaceholder("Ej: atender consultas de clientes, recibir pedidos, facturar…")
      .fill("Atender consultas por WhatsApp");
    await this.createAccountButton.click();
  }
}
