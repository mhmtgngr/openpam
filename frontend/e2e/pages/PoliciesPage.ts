import type { Page, Locator } from '@playwright/test';

export class PoliciesPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly addPolicyButton: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /policies/i });
    this.addPolicyButton = page.getByRole('link', { name: /add policy/i });
    this.table = page.locator('.table tbody');
  }

  async goto(type: 'access' | 'password' | 'session' = 'access') {
    await this.page.goto(`/policies/${type}`);
    await this.page.waitForLoadState('networkidle');
  }

  async clickAddPolicy() {
    await this.addPolicyButton.click();
  }

  async getTableRowCount(): Promise<number> {
    await this.page.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty
    });
    return await this.table.locator('tr').count();
  }

  async clickEditButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    const menuButton = row.locator('button').first();
    await menuButton.click();
    await this.page.getByRole('link', { name: /edit/i }).click();
  }

  async clickTestPolicy(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    const menuButton = row.locator('button').first();
    await menuButton.click();
    await this.page.getByRole('link', { name: /test/i }).click();
  }
}
