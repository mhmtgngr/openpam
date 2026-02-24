import type { Page, Locator } from '@playwright/test';

export class TargetsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly addTargetButton: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /targets/i });
    this.searchInput = page.getByPlaceholder(/search targets/i);
    this.addTargetButton = page.getByRole('link', { name: /add target/i });
    this.table = page.locator('.table tbody');
  }

  async goto() {
    await this.page.goto('/targets');
    await this.page.waitForLoadState('networkidle');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async clickAddTarget() {
    await this.addTargetButton.click();
  }

  async getTableRowCount(): Promise<number> {
    await this.page.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty
    });
    return await this.table.locator('tr').count();
  }

  async clickConnectButton(rowIndex: number = 0) {
    const row = this.table.locator('tr').nth(rowIndex);
    await row.getByRole('button', { name: /connect/i }).click();
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

  async selectEnvironmentFilter(environment: string) {
    const select = this.page.getByLabel(/environment/i).or(
      this.page.locator('select').filter({ hasText: /all environments/i })
    ).first();
    await select.selectOption(environment);
    await this.page.waitForTimeout(500);
  }
}
