import { Page, Locator } from '@playwright/test';

export class UsersPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly addUserButton: Locator;
  readonly table: Locator;
  readonly tableRows: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { level: 1 });
    this.searchInput = page.getByPlaceholder('Search users...');
    this.addUserButton = page.getByRole('link', { name: /Add User/ });
    this.table = page.locator('.table');
    this.tableRows = this.table.locator('tbody tr');
  }

  async goto() {
    await this.page.goto('/users');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
  }

  async clickAddUser() {
    await this.addUserButton.click();
  }

  async getTableRowCount() {
    return this.tableRows.count();
  }

  async getUserEmail(rowIndex: number) {
    return this.tableRows.nth(rowIndex).locator('td').nth(0).textContent();
  }
}
