import { type Page, Locator } from '@playwright/test';

export class AnomalyListPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly summaryCards: Locator;
  readonly filtersCard: Locator;
  readonly anomaliesTable: Locator;
  readonly searchInput: Locator;
  readonly severityFilter: Locator;
  readonly statusFilter: Locator;
  readonly typeFilter: Locator;
  readonly periodFilter: Locator;
  readonly exportButton: Locator;
  readonly refreshButton: Locator;
  readonly bulkActionsBar: Locator;
  readonly tableRows: Locator;
  readonly pagination: Locator;
  readonly emptyState: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { level: 1, name: /anomaly detection/i });
    this.summaryCards = page.locator('.card').filter({ hasText: /^(Total|Critical|High|Medium|Low|Resolved)/i });
    this.filtersCard = page.locator('.card').filter({ hasText: /filters/i });
    this.anomaliesTable = page.locator('table');
    this.searchInput = page.getByPlaceholder(/search anomalies/i);
    this.severityFilter = page.locator('select').filter({ hasText: /severity/i });
    this.statusFilter = page.locator('select').filter({ hasText: /status/i });
    this.typeFilter = page.locator('select').filter({ hasText: /type/i });
    this.periodFilter = page.locator('select').filter({ hasText: /Last \d+ days/i });
    this.exportButton = page.getByRole('button', { name: /export/i });
    this.refreshButton = page.getByRole('button', { name: /refresh/i });
    this.bulkActionsBar = page.locator('.bg-primary-500\\/10').or(page.locator('[class*="bg-primary-500/10"]'));
    this.tableRows = page.locator('tbody tr');
    this.pagination = page.locator('[class*="pagination"]').or(page.locator('.card-footer'));
    this.emptyState = page.locator('text=/no anomalies found/i');
  }

  async goto() {
    await this.page.goto('/analytics/anomalies');
    await this.page.waitForLoadState('networkidle');
  }

  async getHeadingText(): Promise<string | null> {
    const heading = this.heading.first();
    if (await heading.isVisible().catch(() => false)) {
      return await heading.textContent();
    }
    return null;
  }

  async getSummaryCardsCount(): Promise<number> {
    return await this.summaryCards.count();
  }

  async getSummaryCardValue(cardTitle: string): Promise<string | null> {
    const card = this.page.locator('.card').filter({ hasText: new RegExp(cardTitle, 'i') }).first();
    const valueElement = card.locator('.text-3xl').or(card.locator('.text-2xl')).first();
    if (await valueElement.isVisible().catch(() => false)) {
      return await valueElement.textContent();
    }
    return null;
  }

  async getAnomalyCount(): Promise<number> {
    return await this.tableRows.count();
  }

  async searchAnomalies(searchTerm: string) {
    await this.searchInput.fill(searchTerm);
    await this.page.keyboard.press('Enter');
    await this.page.waitForLoadState('networkidle');
  }

  async filterBySeverity(severity: string) {
    await this.severityFilter.selectOption({ label: new RegExp(severity, 'i') });
    await this.page.waitForLoadState('networkidle');
  }

  async filterByStatus(status: string) {
    await this.statusFilter.selectOption({ label: new RegExp(status, 'i') });
    await this.page.waitForLoadState('networkidle');
  }

  async filterByType(type: string) {
    await this.typeFilter.selectOption({ label: new RegExp(type, 'i') });
    await this.page.waitForLoadState('networkidle');
  }

  async selectPeriod(days: number) {
    await this.periodFilter.selectOption({ label: new RegExp(`Last ${days} days`, 'i') });
    await this.page.waitForLoadState('networkidle');
  }

  async resetFilters() {
    const clearButton = this.page.getByRole('button', { name: /clear filters/i });
    if (await clearButton.isVisible().catch(() => false)) {
      await clearButton.click();
      await this.page.waitForLoadState('networkidle');
    }
  }

  async clickAnomalyRow(rowIndex: number) {
    const rows = this.tableRows;
    const count = await rows.count();
    if (rowIndex < count) {
      await rows.nth(rowIndex).click();
      await this.page.waitForLoadState('networkidle');
    }
  }

  async clickViewButton(rowIndex: number) {
    const viewButton = this.page.getByRole('button', { name: /view/i }).nth(rowIndex);
    await viewButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async selectAnomaly(rowIndex: number) {
    const checkbox = this.tableRows.nth(rowIndex).locator('input[type="checkbox"]');
    await checkbox.check();
  }

  async selectAllAnomalies() {
    const selectAllCheckbox = this.page.locator('thead input[type="checkbox"]');
    await selectAllCheckbox.check();
  }

  async getSelectedCount(): Promise<number> {
    const selectedText = await this.page.getByText(/anomalies selected/i).textContent();
    const match = selectedText?.match(/(\d+)\s+anomalies/i);
    return match ? parseInt(match[1], 10) : 0;
  }

  async clickBulkStatusUpdate(status: 'investigating' | 'resolved' | 'false_positive') {
    const button = this.page.getByRole('button', { name: new RegExp(status, 'i') });
    await button.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickRefresh() {
    await this.refreshButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickExport() {
    await this.exportButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async isEmptyStateVisible(): Promise<boolean> {
    return await this.emptyState.isVisible().catch(() => false);
  }

  async isBulkActionsBarVisible(): Promise<boolean> {
    return await this.bulkActionsBar.isVisible().catch(() => false);
  }

  async getAnomalySeverity(rowIndex: number): Promise<string | null> {
    const row = this.tableRows.nth(rowIndex);
    const severityBadge = row.locator('[class*="text-danger-400"], [class*="text-warning-400"], [class*="text-success-400"], [class*="text-gray-400"]');
    if (await severityBadge.isVisible().catch(() => false)) {
      const text = await row.textContent();
      if (text) {
        const match = text.match(/(critical|high|medium|low)/i);
        return match ? match[1].toLowerCase() : null;
      }
    }
    return null;
  }

  async getAnomalyTitle(rowIndex: number): Promise<string | null> {
    const row = this.tableRows.nth(rowIndex);
    const titleElement = row.locator('.font-medium').or(row.locator('td:nth-child(3)'));
    if (await titleElement.isVisible().catch(() => false)) {
      return await titleElement.textContent();
    }
    return null;
  }

  async hasPagination(): Promise<boolean> {
    return await this.pagination.isVisible().catch(() => false);
  }

  async goToNextPage() {
    const nextButton = this.page.getByRole('button', { name: /next/i });
    if (await nextButton.isVisible().catch(() => false)) {
      await nextButton.click();
      await this.page.waitForLoadState('networkidle');
    }
  }

  async goToPreviousPage() {
    const prevButton = this.page.getByRole('button', { name: /previous/i }).or(this.page.getByRole('button', { name: /prev/i }));
    if (await prevButton.isVisible().catch(() => false)) {
      await prevButton.click();
      await this.page.waitForLoadState('networkidle');
    }
  }

  async waitForContent(): Promise<void> {
    await Promise.race([
      this.tableRows.first().waitFor({ state: 'visible' }).catch(() => {}),
      this.emptyState.waitFor({ state: 'visible' }).catch(() => {}),
      this.page.waitForTimeout(2000),
    ]);
  }
}
