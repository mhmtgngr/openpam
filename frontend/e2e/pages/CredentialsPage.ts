import type { Page, Locator } from '@playwright/test';

export class CredentialsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly addCredentialButton: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /credential vault/i });
    this.searchInput = page.getByPlaceholder(/search credentials/i);
    this.addCredentialButton = page.getByRole('link', { name: /add credential/i });
    this.table = page.locator('.table tbody');
  }

  async goto() {
    await this.page.goto('/credentials');
    await this.page.waitForLoadState('networkidle');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async clickAddCredential() {
    await this.addCredentialButton.click();
  }

  async getTableRowCount(): Promise<number> {
    await this.page.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty
    });
    return await this.table.locator('tr').count();
  }

  async clickRotateButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    await row.getByRole('button', { name: /rotate/i }).click();
  }

  async clickViewButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    await row.getByRole('link', { name: /view/i }).click();
  }

  async selectTypeFilter(type: string) {
    const select = this.page.getByLabel(/type/i).or(
      this.page.locator('select').filter({ hasText: /all types/i })
    ).first();
    await select.selectOption(type);
    await this.page.waitForTimeout(500);
  }

  async selectStatusFilter(status: string) {
    const select = this.page.getByLabel(/status/i).or(
      this.page.locator('select').filter({ hasText: /all status/i })
    ).first();
    await select.selectOption(status);
    await this.page.waitForTimeout(500);
  }
}
