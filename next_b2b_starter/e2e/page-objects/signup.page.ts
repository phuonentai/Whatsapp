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
    this.fullNameInput = page.getByPlaceholder("John Doe");
    this.emailInput = page.getByPlaceholder("you@company.com");
    this.continueButton = page.getByRole("button", { name: /continue/i });
    this.organizationNameInput = page.getByPlaceholder("Acme Inc");
    this.industrySelect = page.locator("select");
    this.createAccountButton = page.getByRole("button", { name: /create account/i });
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
    await this.industrySelect.selectOption({ label: industry });
  }

  async submit() {
    await this.createAccountButton.click();
  }
}
