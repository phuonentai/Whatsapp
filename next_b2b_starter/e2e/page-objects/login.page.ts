import { Page } from "@playwright/test";
import { OrgType, setMockAuthHeader } from "../fixtures/auth";

export class LoginPage {
  readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async loginAs(orgType: OrgType): Promise<void> {
    await setMockAuthHeader(this.page, orgType);
    await this.page.goto("/dashboard/crm");
    await this.page.waitForLoadState("networkidle");
  }
}
