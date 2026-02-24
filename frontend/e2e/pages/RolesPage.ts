import type { Page, Locator } from '@playwright/test';

export class RolesPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly addRoleButton: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /roles/i });
    this.searchInput = page.getByPlaceholder(/search roles/i);
    this.addRoleButton = page.getByRole('link', { name: /add role/i });
    this.table = page.locator('.table tbody');
  }

  async goto() {
    await this.page.goto('/roles');
    await this.page.waitForLoadState('networkidle');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async clickAddRole() {
    await this.addRoleButton.click();
  }

  async getTableRowCount(): Promise<number> {
    await this.page.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty
    });
    return await this.table.locator('tr').count();
  }

  async clickViewButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    await row.getByRole('link', { name: /view/i }).click();
  }

  async clickEditButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    const menuButton = row.locator('button').first();
    await menuButton.click();
    await this.page.getByRole('link', { name: /edit/i }).click();
  }
}
