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
    // Try multiple selector strategies for robustness
    this.heading = page.getByText(/anomaly detection/i, { exact: false })
      .or(page.getByRole('heading', { level: 1 }));
    this.summaryCards = page.locator('.card');
    this.filtersCard = page.locator('.card').filter({ hasText: /filters|period/i });
    this.anomaliesTable = page.locator('table');
    // Use more specific selectors for input
    this.searchInput = page.locator('input[placeholder*="Search" i]')
      .or(page.locator('input[placeholder*="search" i]'))
      .or(page.locator('.card input[type="text"]').first());
    // For select elements, use data attributes or index-based selection
    this.severityFilter = page.locator('select').nth(1);
    this.statusFilter = page.locator('select').nth(2);
    this.typeFilter = page.locator('select').nth(3);
    this.periodFilter = page.locator('select').first();
    this.exportButton = page.locator('button', { hasText: /export/i });
    this.refreshButton = page.locator('button', { hasText: /refresh/i });
    this.bulkActionsBar = page.locator('[class*="bg-primary-500/10"]').or(page.locator('[class*="bg-primary-500\\/10"]'));
    this.tableRows = page.locator('tbody tr').or(page.locator('table tr'));
    this.pagination = page.locator('[class*="pagination"]').or(page.locator('.card-footer'));
    this.emptyState = page.getByText('no anomalies found', { exact: false })
      .or(page.getByText('No anomalies found'));
  }

  async goto() {
    await this.page.goto('/analytics/anomalies');
    // Wait for page to load
    await this.page.waitForLoadState('domcontentloaded').catch(() => {});
    await this.page.waitForTimeout(500);
  }

  async getHeadingText(): Promise<string | null> {
    await this.page.waitForTimeout(500);
    const elements = await this.heading.all();
    for (const el of elements) {
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text) return text;
      }
    }
    return null;
  }

  async getSummaryCardsCount(): Promise<number> {
    await this.page.waitForTimeout(1000);
    const visibleCards = this.summaryCards.filter({ hasText: /^(Total|Critical|High|Medium|Low|Resolved)/i });
    return await visibleCards.count();
  }

  async getSummaryCardValue(cardTitle: string): Promise<string | null> {
    await this.page.waitForTimeout(500);
    const card = this.page.locator('.card').filter({ hasText: new RegExp(cardTitle, 'i') }).first();
    const valueElement = card.locator('.text-3xl, .text-2xl, h3, h2').first();
    if (await valueElement.isVisible().catch(() => false)) {
      const text = await valueElement.textContent();
      if (text) {
        const numbers = text.match(/\d+/);
        return numbers ? numbers[0] : text;
      }
    }
    return null;
  }

  async getAnomalyCount(): Promise<number> {
    await this.page.waitForTimeout(500);
    const rows = await this.tableRows.all();
    const visibleRows = rows.filter(async (row) => await row.isVisible().catch(() => false));
    let count = 0;
    for (const row of visibleRows) {
      if (await row.isVisible().catch(() => false)) count++;
    }
    return count;
  }

  async searchAnomalies(searchTerm: string) {
    const input = this.searchInput.first();
    await input.fill(searchTerm);
    await this.page.waitForTimeout(500);
  }

  async filterBySeverity(severity: string) {
    await this.severityFilter.selectOption(severity);
    await this.page.waitForTimeout(500);
  }

  async filterByStatus(status: string) {
    await this.statusFilter.selectOption(status);
    await this.page.waitForTimeout(500);
  }

  async filterByType(type: string) {
    await this.typeFilter.selectOption(type);
    await this.page.waitForTimeout(500);
  }

  async selectPeriod(days: number) {
    await this.periodFilter.selectOption(String(days));
    await this.page.waitForTimeout(500);
  }

  async resetFilters() {
    const clearButton = this.page.locator('button', { hasText: /clear|reset/i });
    const count = await clearButton.count();
    if (count > 0) {
      await clearButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickAnomalyRow(rowIndex: number) {
    const rows = await this.tableRows.all();
    const visibleRows = rows.filter(async (row) => await row.isVisible().catch(() => false));
    if (rowIndex < visibleRows.length) {
      await visibleRows[rowIndex].click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickViewButton(rowIndex: number) {
    const viewButtons = this.page.locator('button', { hasText: /view/i });
    const count = await viewButtons.count();
    if (rowIndex < count) {
      await viewButtons.nth(rowIndex).click();
      await this.page.waitForTimeout(500);
    }
  }

  async selectAnomaly(rowIndex: number) {
    const checkboxes = this.page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();
    if (rowIndex < count) {
      await checkboxes.nth(rowIndex).check();
    }
  }

  async selectAllAnomalies() {
    const selectAllCheckbox = this.page.locator('thead input[type="checkbox"], thead .checkbox');
    const count = await selectAllCheckbox.count();
    if (count > 0) {
      await selectAllCheckbox.first().check();
    }
  }

  async getSelectedCount(): Promise<number> {
    const selectedText = await this.page.getByText(/anomalies selected/i).textContent();
    if (selectedText) {
      const match = selectedText?.match(/(\d+)\s+anomalies/i);
      return match ? parseInt(match[1], 10) : 0;
    }
    return 0;
  }

  async clickBulkStatusUpdate(status: 'investigating' | 'resolved' | 'false_positive') {
    const button = this.page.locator('button', { hasText: new RegExp(status, 'i') });
    const count = await button.count();
    if (count > 0) {
      await button.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickRefresh() {
    const count = await this.refreshButton.count();
    if (count > 0) {
      await this.refreshButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickExport() {
    const count = await this.exportButton.count();
    if (count > 0) {
      await this.exportButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async isEmptyStateVisible(): Promise<boolean> {
    return await this.emptyState.isVisible().catch(() => false);
  }

  async isBulkActionsBarVisible(): Promise<boolean> {
    return await this.bulkActionsBar.isVisible().catch(() => false);
  }

  async getAnomalySeverity(rowIndex: number): Promise<string | null> {
    const rows = await this.tableRows.all();
    const visibleRows = rows.filter(async (row) => await row.isVisible().catch(() => false));
    if (rowIndex < visibleRows.length) {
      const row = visibleRows[rowIndex];
      const text = await row.textContent();
      if (text) {
        const match = text.match(/(critical|high|medium|low)/i);
        return match ? match[1].toLowerCase() : null;
      }
    }
    return null;
  }

  async getAnomalyTitle(rowIndex: number): Promise<string | null> {
    const rows = await this.tableRows.all();
    const visibleRows = rows.filter(async (row) => await row.isVisible().catch(() => false));
    if (rowIndex < visibleRows.length) {
      const row = visibleRows[rowIndex];
      const titleElement = row.locator('.font-medium, td:nth-child(3)').first();
      if (await titleElement.isVisible().catch(() => false)) {
        return await titleElement.textContent();
      }
    }
    return null;
  }

  async hasPagination(): Promise<boolean> {
    return await this.pagination.isVisible().catch(() => false);
  }

  async goToNextPage() {
    const nextButton = this.page.locator('button', { hasText: /next|>/i });
    const count = await nextButton.count();
    if (count > 0) {
      await nextButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async goToPreviousPage() {
    const prevButton = this.page.locator('button', { hasText: /previous|prev|</i });
    const count = await prevButton.count();
    if (count > 0) {
      await prevButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async waitForContent(): Promise<void> {
    await Promise.race([
      this.tableRows.first().waitFor({ state: 'visible', timeout: 5000 }).catch(() => {}),
      this.emptyState.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {}),
      this.page.waitForTimeout(3000),
    ]);
  }
}
