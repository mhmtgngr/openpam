import { type Page, Locator } from '@playwright/test';

export class ReportsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly generateReportButton: Locator;
  readonly scheduleReportButton: Locator;
  readonly searchInput: Locator;
  readonly frameworkFilter: Locator;
  readonly statusFilter: Locator;
  readonly sortSelect: Locator;
  readonly reportsTable: Locator;
  readonly reportRows: Locator;
  readonly statCards: Locator;
  readonly emptyState: Locator;

  constructor(page: Page) {
    this.page = page;
    // The page heading
    this.heading = page.getByRole('heading', { level: 1, name: /compliance reports/i });
    // Buttons
    this.generateReportButton = page.getByRole('button', { name: /generate report/i });
    this.scheduleReportButton = page.getByRole('button', { name: /schedule report/i });
    // Filters
    this.searchInput = page.getByPlaceholder(/search reports/i);
    this.frameworkFilter = page.locator('select').filter({ hasText: /all frameworks/i });
    this.statusFilter = page.locator('select').filter({ hasText: /all status/i });
    this.sortSelect = page.locator('select').filter({ hasText: /newest first/i });
    // Table
    this.reportsTable = page.locator('table').or(page.locator('[role="table"]'));
    this.reportRows = page.locator('tr').filter({ hasText: /^/ }); // Will be refined in actual use
    // Stats
    this.statCards = page.locator('.card').filter({ hasText: /total reports|active schedules|pending exceptions|in queue/i });
    // Empty state
    this.emptyState = page.getByText(/no reports found/i);
  }

  async goto() {
    await this.page.goto('/reports');
    await this.page.waitForLoadState('networkidle');
  }

  async waitForLoad() {
    await this.page.waitForLoadState('networkidle');
    // Wait for either content or empty state
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

  async getReportCount(): Promise<number> {
    await this.waitForLoad();
    const rows = this.page.locator('tbody tr');
    return await rows.count();
  }

  async searchReports(query: string) {
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

  async sortBy(sortOption: string) {
    await this.sortSelect.selectOption({ label: sortOption });
    await this.page.waitForLoadState('networkidle');
  }

  async clickGenerateReport() {
    await this.generateReportButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickScheduleReport() {
    await this.scheduleReportButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickReport(reportName: string) {
    const row = this.page.locator('tr').filter({ hasText: reportName }).first();
    await row.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickViewDetails(reportName: string) {
    const button = this.page.getByRole('button', { name: /view details/i })
      .filter({ hasText: reportName });
    await button.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickDownload(reportName: string) {
    const button = this.page.getByRole('button', { name: /download/i })
      .filter({ has: this.page.locator(`tr:has-text("${reportName}")`) });
    await button.click();
  }

  async clickShare(reportName: string) {
    const button = this.page.getByRole('button', { name: /share/i })
      .filter({ has: this.page.locator(`tr:has-text("${reportName}")`) });
    await button.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickDelete(reportName: string) {
    const button = this.page.getByRole('button', { name: /delete/i })
      .filter({ has: this.page.locator(`tr:has-text("${reportName}")`) });
    // Handle confirmation dialog
    this.page.once('dialog', dialog => dialog.accept());
    await button.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clearFilters() {
    await this.page.getByRole('button', { name: /clear filters/i }).click();
    await this.page.waitForLoadState('networkidle');
  }

  async isReportVisible(reportName: string): Promise<boolean> {
    const row = this.page.locator('tr').filter({ hasText: reportName }).first();
    return await row.isVisible().catch(() => false);
  }

  async getReportStatus(reportName: string): Promise<string | null> {
    const row = this.page.locator('tr').filter({ hasText: reportName }).first();
    const statusCell = row.locator('td').nth(2); // Status column
    if (await statusCell.isVisible().catch(() => false)) {
      return await statusCell.textContent();
    }
    return null;
  }

  async getReportFramework(reportName: string): Promise<string | null> {
    const row = this.page.locator('tr').filter({ hasText: reportName }).first();
    const frameworkBadge = row.locator('[class*="badge"]').first();
    if (await frameworkBadge.isVisible().catch(() => false)) {
      return await frameworkBadge.textContent();
    }
    return null;
  }

  async hasEmptyState(): Promise<boolean> {
    return await this.emptyState.isVisible().catch(() => false);
  }

  async navigateToGeneratePage() {
    await this.page.goto('/reports/generate');
    await this.page.waitForLoadState('networkidle');
  }
}
