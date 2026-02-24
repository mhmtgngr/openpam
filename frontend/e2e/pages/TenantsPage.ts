import type { Page, Locator } from '@playwright/test';

export class TenantsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly addTenantButton: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /tenants/i });
    this.searchInput = page.getByPlaceholder(/search tenants/i);
    this.addTenantButton = page.getByRole('link', { name: /add tenant/i });
    this.table = page.locator('.table tbody');
  }

  async goto() {
    await this.page.goto('/tenants');
    await this.page.waitForLoadState('networkidle');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async clickAddTenant() {
    await this.addTenantButton.click();
  }

  async getTableRowCount(): Promise<number> {
    await this.page.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty
    });
    return await this.table.locator('tr').count();
  }

  async selectStatusFilter(status: string) {
    const select = this.page.getByLabel(/status/i).or(
      this.page.locator('select').filter({ hasText: /all status/i })
    ).first();
    await select.selectOption(status);
    await this.page.waitForTimeout(500);
  }
}
