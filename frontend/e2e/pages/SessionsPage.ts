import { Page, Locator } from '@playwright/test';

export class SessionsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly table: Locator;
  readonly terminateButton: Locator;
  readonly emptyMessage: Locator;
  readonly modalTitle: Locator;
  readonly modalTextarea: Locator;
  readonly modalCancelButton: Locator;
  readonly modalConfirmButton: Locator;
  readonly statusFilter: Locator;
  readonly typeFilter: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /Active Sessions|Sessions/i });
    this.searchInput = page.getByPlaceholder('Search sessions...');
    this.table = page.locator('.card').filter({ hasText: /User|Target/ }).locator('table').or(
      page.locator('table')
    );
    this.terminateButton = page.getByRole('button', { name: /Terminate/i }).first();
    this.emptyMessage = page.getByText('No sessions found');
    this.modalTitle = page.locator('.modal, [role="dialog"]').getByRole('heading');
    this.modalTextarea = page.locator('.modal, [role="dialog"]').locator('textarea');
    this.modalCancelButton = page.locator('.modal, [role="dialog"]').getByRole('button', { name: 'Cancel' });
    this.modalConfirmButton = page.locator('.modal, [role="dialog"]').getByRole('button', { name: 'Terminate Session' });
    this.statusFilter = page.locator('select').first().or(
      page.getByRole('combobox')
    );
    this.typeFilter = page.locator('select').nth(1).or(
      page.getByRole('combobox')
    );
  }

  async goto() {
    await this.page.goto('/sessions');
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

  async clickTerminate(index: number = 0) {
    const buttons = this.page.getByRole('button', { name: /Terminate/i });
    await buttons.nth(index).click();
  }

  async enterTerminateReason(reason: string) {
    await this.modalTextarea.fill(reason);
  }

  async confirmTerminate() {
    await this.modalConfirmButton.click();
  }

  async cancelTerminate() {
    await this.modalCancelButton.click();
  }

  async isModalOpen() {
    const modal = this.page.locator('.modal, [role="dialog"]');
    return await modal.isVisible();
  }

  async hasEmptyMessage() {
    return await this.emptyMessage.isVisible().catch(() => false);
  }
}
