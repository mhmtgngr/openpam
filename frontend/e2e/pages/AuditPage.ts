import { Page, Locator } from '@playwright/test';

export class AuditPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly exportButton: Locator;
  readonly table: Locator;
  readonly emptyMessage: Locator;
  readonly outcomeFilter: Locator;
  readonly resourceFilter: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: 'Audit Logs' });
    this.searchInput = page.getByPlaceholder('Search audit logs...');
    this.exportButton = page.getByRole('button', { name: /Export/i });
    this.table = page.locator('.card').filter({ hasText: /Timestamp|Actor/ }).locator('table').or(
      page.locator('table')
    );
    this.emptyMessage = page.getByText('No audit events found');
    // Use more specific selectors for filters to avoid strict mode violations
    this.outcomeFilter = page.getByRole('combobox', { name: /outcome/i })
      .or(page.locator('select').nth(0));
    this.resourceFilter = page.getByRole('combobox', { name: /resource/i })
      .or(page.locator('select').nth(1));
  }

  async goto() {
    await this.page.goto('/audit');
  }

  async waitForLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.heading.waitFor({ state: 'visible' });
  }

  async getHeadingText() {
    await this.heading.waitFor({ state: 'visible' });
    return this.heading.textContent();
  }

  async search(query: string) {
    await this.searchInput.fill(query);
  }

  async clickExport() {
    await this.exportButton.click();
  }

  async hasEmptyMessage() {
    return await this.emptyMessage.isVisible().catch(() => false);
  }
}
