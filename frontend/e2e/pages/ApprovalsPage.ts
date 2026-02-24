import { Page, Locator } from '@playwright/test';

export class ApprovalsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;
  readonly table: Locator;
  readonly approveButton: Locator;
  readonly denyButton: Locator;
  readonly emptyMessage: Locator;
  readonly modalTitle: Locator;
  readonly modalTextarea: Locator;
  readonly modalCancelButton: Locator;
  readonly modalConfirmButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: 'Pending Approvals' });
    this.searchInput = page.getByPlaceholder('Search requests...');
    this.table = page.locator('.card').filter({ hasText: /Requester|Target/ }).locator('table').or(
      page.locator('table')
    );
    this.approveButton = page.getByRole('button', { name: /Approve/i }).first();
    this.denyButton = page.getByRole('button', { name: /Deny/i }).first();
    this.emptyMessage = page.getByText('No pending approvals');
    // Modal uses modal-backdrop and modal-content classes, not role="dialog"
    this.modalTitle = page.locator('.modal-content').getByRole('heading');
    this.modalTextarea = page.locator('.modal-content textarea');
    this.modalCancelButton = page.locator('.modal-content').getByRole('button', { name: 'Cancel' });
    this.modalConfirmButton = page.locator('.modal-content').getByRole('button', { name: /Approve|Deny/ });
  }

  async goto() {
    await this.page.goto('/approvals');
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

  async clickApprove(index: number = 0) {
    const buttons = this.page.getByRole('button', { name: /Approve/i });
    await buttons.nth(index).click();
  }

  async clickDeny(index: number = 0) {
    const buttons = this.page.getByRole('button', { name: /Deny/i });
    await buttons.nth(index).click();
  }

  async enterModalComment(comment: string) {
    await this.modalTextarea.fill(comment);
  }

  async confirmModalAction() {
    await this.modalConfirmButton.click();
  }

  async cancelModal() {
    await this.modalCancelButton.click();
  }

  async isModalOpen() {
    const modal = this.page.locator('.modal-backdrop');
    return await modal.isVisible();
  }

  async hasEmptyMessage() {
    return await this.emptyMessage.isVisible().catch(() => false);
  }
}
