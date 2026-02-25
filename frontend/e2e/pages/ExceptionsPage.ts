import { type Page, Locator } from '@playwright/test';

export class ExceptionsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly requestExceptionButton: Locator;
  readonly searchInput: Locator;
  readonly frameworkFilter: Locator;
  readonly statusFilter: Locator;
  readonly exceptionCards: Locator;
  readonly statCards: Locator;
  readonly expiringSoonAlert: Locator;
  readonly emptyState: Locator;

  constructor(page: Page) {
    this.page = page;
    // Heading
    this.heading = page.getByRole('heading', { level: 1, name: /compliance exceptions/i });
    // Buttons
    this.requestExceptionButton = page.getByRole('button', { name: /request exception/i });
    // Filters
    this.searchInput = page.getByPlaceholder(/search exceptions/i);
    this.frameworkFilter = page.locator('select').filter({ hasText: /all frameworks/i });
    this.statusFilter = page.locator('select').filter({ hasText: /all status/i });
    // Content
    this.exceptionCards = page.locator('.card').filter({ hasText: /^/ });
    this.statCards = page.locator('.card').filter({ hasText: /pending requests|active exceptions|expiring soon|total exceptions/i });
    this.expiringSoonAlert = page.locator('.card').filter({ hasText: /exceptions expiring soon/i });
    this.emptyState = page.getByText(/no exceptions found/i);
  }

  async goto() {
    await this.page.goto('/compliance/exceptions');
    await this.page.waitForLoadState('networkidle');
  }

  async waitForLoad() {
    await this.page.waitForLoadState('networkidle');
    await Promise.race([
      this.heading.waitFor({ state: 'visible' }),
      this.emptyState.waitFor({ state: 'visible' }),
    ]).catch(() => {});
  }

  async getHeadingText(): Promise<string> {
    const heading = this.heading.first();
    if (await heading.isVisible().catch(() => false)) {
      return await heading.textContent() || '';
    }
    return '';
  }

  async getStatCount(): Promise<number> {
    return await this.statCards.count();
  }

  async getStatValue(statName: string): Promise<string | null> {
    const card = this.page.locator('.card').filter({ hasText: new RegExp(statName, 'i') }).first();
    const valueElement = card.locator('.text-2xl').first();
    if (await valueElement.isVisible().catch(() => false)) {
      return await valueElement.textContent();
    }
    return null;
  }

  async getExceptionCount(): Promise<number> {
    await this.waitForLoad();
    // Count exception items (could be in a table or card list)
    const rows = this.page.locator('tbody tr');
    const cards = this.page.locator('[class*="divide-y"] > div');
    const rowCount = await rows.count();
    const cardCount = await cards.count();
    return Math.max(rowCount, cardCount);
  }

  async searchExceptions(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForLoadState('networkidle');
  }

  async filterByFramework(framework: string) {
    await this.frameworkFilter.selectOption({ label: framework });
    await this.page.waitForLoadState('networkidle');
  }

  async filterByStatus(status: string) {
    await this.statusFilter.selectOption({ label: status });
    await this.page.waitForLoadState('networkidle');
  }

  async clickRequestException() {
    await this.requestExceptionButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clearFilters() {
    await this.page.getByRole('button', { name: /clear filters/i }).click();
    await this.page.waitForLoadState('networkidle');
  }

  async hasExpiringSoonAlert(): Promise<boolean> {
    return await this.expiringSoonAlert.isVisible().catch(() => false);
  }

  async showExpiringSoon() {
    const showButton = this.expiringSoonAlert.getByRole('button', { name: /show/i });
    await showButton.click();
  }

  async hideExpiringSoon() {
    const hideButton = this.expiringSoonAlert.getByRole('button', { name: /hide/i });
    await hideButton.click();
  }

  async clickApprove(exceptionName: string) {
    const approveButton = this.page.locator('tr, div').filter({ hasText: exceptionName })
      .getByRole('button', { name: /approve/i });
    this.page.once('dialog', dialog => dialog.accept());
    await approveButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickDeny(exceptionName: string) {
    const denyButton = this.page.locator('tr, div').filter({ hasText: exceptionName })
      .getByRole('button', { name: /deny/i });
    this.page.once('dialog', dialog => {
      dialog.type('Test denial reason');
      dialog.accept();
    });
    await denyButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickRevoke(exceptionName: string) {
    const revokeButton = this.page.locator('tr, div').filter({ hasText: exceptionName })
      .getByRole('button', { name: /revoke/i });
    this.page.once('dialog', dialog => {
      dialog.type('Test revocation reason');
      dialog.accept();
    });
    await revokeButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickReview(exceptionName: string) {
    const reviewButton = this.page.locator('tr, div').filter({ hasText: exceptionName })
      .getByRole('button', { name: /review/i });
    await reviewButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickEdit(exceptionName: string) {
    const editButton = this.page.locator('tr, div').filter({ hasText: exceptionName })
      .getByRole('button', { name: /edit/i });
    await editButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async isExceptionVisible(exceptionName: string): Promise<boolean> {
    const item = this.page.locator('tr, div').filter({ hasText: exceptionName }).first();
    return await item.isVisible().catch(() => false);
  }

  async getExceptionStatus(exceptionName: string): Promise<string | null> {
    const item = this.page.locator('tr, div').filter({ hasText: exceptionName }).first();
    const statusBadge = item.locator('[class*="badge"], [class*="Badge"]').first();
    if (await statusBadge.isVisible().catch(() => false)) {
      return await statusBadge.textContent();
    }
    return null;
  }

  async getExceptionFramework(exceptionName: string): Promise<string | null> {
    const item = this.page.locator('tr, div').filter({ hasText: exceptionName }).first();
    const badges = item.locator('[class*="badge"], [class*="Badge"]');
    const count = await badges.count();
    for (let i = 0; i < count; i++) {
      const badge = badges.nth(i);
      const text = await badge.textContent();
      if (text && /SOC 2|ISO 27001|PCI DSS|HIPAA|GDPR/i.test(text)) {
        return text;
      }
    }
    return null;
  }

  async hasEmptyState(): Promise<boolean> {
    return await this.emptyState.isVisible().catch(() => false);
  }

  async isExpiring(exceptionName: string): Promise<boolean> {
    const item = this.page.locator('tr, div').filter({ hasText: exceptionName }).first();
    const expiringBadge = item.locator('text=/Expiring Soon/i');
    return await expiringBadge.isVisible().catch(() => false);
  }
}
